package observability

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambdacontext"
)

// This package intentionally avoids external OTel dependencies.
// The environment currently blocks module downloads, so we emit OTel-shaped
// span/metric events as structured JSON logs as a stepping stone.

type Obs struct {
	service string
	base    map[string]interface{}
}

// In-process Prometheus aggregation (for local adapter use).
//
// This lets us expose stage-level metrics (like DynamoDB latency) on /metrics
// without bringing extra dependencies or requiring OTLP exporters.
//
// Enabled when OBS_INPROC_PROM=1 is set.
type promAgg struct {
	mu    sync.Mutex
	count map[promSeriesKey]float64
	sum   map[promSeriesKey]float64
	n     map[promSeriesKey]float64
	gauge map[promSeriesKey]float64
}

var inproc = &promAgg{
	count: map[promSeriesKey]float64{},
	sum:   map[promSeriesKey]float64{},
	n:     map[promSeriesKey]float64{},
	gauge: map[promSeriesKey]float64{},
}

func inprocEnabled() bool { return strings.TrimSpace(os.Getenv("OBS_INPROC_PROM")) == "1" }

func New(ctx context.Context, req events.APIGatewayProxyRequest, service string) *Obs {
	if strings.TrimSpace(service) == "" {
		service = os.Getenv("AWS_LAMBDA_FUNCTION_NAME")
	}
	correlationID := firstNonEmpty(
		headerValue(req.Headers, "x-correlation-id"),
		req.RequestContext.RequestID,
		newTraceID(),
	)
	env := firstNonEmpty(
		headerValue(req.Headers, "x-env"),
		strings.TrimSpace(os.Getenv("APP_ENV")),
		"dev",
	)
	appVersion := firstNonEmpty(
		headerValue(req.Headers, "x-app-version"),
		strings.TrimSpace(os.Getenv("APP_VERSION")),
		"unknown",
	)

	base := map[string]interface{}{
		"service":        service,
		"function":       os.Getenv("AWS_LAMBDA_FUNCTION_NAME"),
		"awsRequestId":   awsRequestID(ctx),
		"requestId":      req.RequestContext.RequestID,
		"traceId":        traceRootID(),
		"path":           req.Path,
		"resource":       req.Resource,
		"method":         req.HTTPMethod,
		"correlation_id": correlationID,
		"env":            env,
		"app_version":    appVersion,
	}

	return &Obs{service: service, base: base}
}

func (o *Obs) CorrelationID() string {
	return asString(o.base["correlation_id"])
}

func (o *Obs) Info(message string, fields map[string]interface{})  { o.log("info", message, fields) }
func (o *Obs) Warn(message string, fields map[string]interface{})  { o.log("warn", message, fields) }
func (o *Obs) Error(message string, fields map[string]interface{}) { o.log("error", message, fields) }

func (o *Obs) log(level, message string, fields map[string]interface{}) {
	payload := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"level":     level,
		"message":   message,
	}

	for k, v := range o.base {
		if v != "" && v != nil {
			payload[k] = v
		}
	}
	for k, v := range fields {
		payload[k] = v
	}

	line, _ := json.Marshal(payload)
	fmt.Println(string(line))
}

// EmitMetric emits an OTel-shaped metric event as JSON logs (no CloudWatch EMF).
func (o *Obs) EmitMetric(name, unit string, value float64, attrs map[string]string) {
	payload := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"type":      "otel_metric",
		"name":      name,
		"unit":      unit,
		"value":     value,
		"attrs":     attrs,
	}

	for k, v := range o.base {
		if v != "" && v != nil {
			payload[k] = v
		}
	}

	line, _ := json.Marshal(payload)
	fmt.Println(string(line))

	// For local development, also aggregate selected metrics so Prometheus can scrape them.
	// We intentionally skip the baseline Invocation/LatencyMs because the local adapter already exposes those.
	if !inprocEnabled() {
		return
	}
	switch name {
	case "Invocation", "LatencyMs":
		return
	}

	// Convention: *_last are gauges (latest observed value).
	if strings.HasSuffix(name, "_last") {
		inproc.setGauge(sanitizeMetricName(name), sanitizeLabels(attrs), value)
		return
	}

	labels := sanitizeLabels(attrs)
	promName := sanitizeMetricName(name)
	switch unit {
	case "Milliseconds":
		inproc.observe(promName, labels, value)
	case "Count":
		inproc.addCounter(promName, labels, value)
	default:
		// Best effort: treat unknown units as counters.
		inproc.addCounter(promName, labels, value)
	}
}

type Span struct {
	mu       sync.Mutex
	name     string
	traceID  string
	spanID   string
	parentID string
	start    time.Time
	attrs    map[string]interface{}
	status   string
}

type traceCtxKey struct{}

type TraceContext struct {
	TraceID  string
	ParentID string
}

// WithTraceContext allows local runners to inject a stable trace id so spans/logs/metrics correlate.
func WithTraceContext(ctx context.Context, traceID, parentID string) context.Context {
	tc := TraceContext{TraceID: strings.TrimSpace(traceID), ParentID: strings.TrimSpace(parentID)}
	return context.WithValue(ctx, traceCtxKey{}, tc)
}

// StartSpan creates a lightweight span and returns a context carrying it.
// If an X-Ray trace root exists, we reuse it as the trace id to keep basic correlation.
func (o *Obs) StartSpan(ctx context.Context, name string, attrs map[string]interface{}) (context.Context, *Span) {
	traceID := ""
	parentID := ""
	if v := ctx.Value(traceCtxKey{}); v != nil {
		if tc, ok := v.(TraceContext); ok {
			traceID = strings.TrimSpace(tc.TraceID)
			parentID = strings.TrimSpace(tc.ParentID)
		}
	}
	if traceID == "" {
		traceID = strings.TrimSpace(traceRootID())
	}
	if traceID == "" {
		traceID = newTraceID()
	}

	// Prefer the current span in context as the parent (proper nesting).
	if v := ctx.Value(spanKey{}); v != nil {
		if p, ok := v.(*Span); ok && p != nil && strings.TrimSpace(p.spanID) != "" {
			parentID = p.spanID
		}
	}

	sp := &Span{
		name:     name,
		traceID:  traceID,
		spanID:   newSpanID(),
		parentID: parentID,
		start:    time.Now().UTC(),
		attrs:    map[string]interface{}{},
		status:   "OK",
	}

	// Base attributes (OTel-ish)
	sp.attrs["service"] = o.service
	sp.attrs["faas.name"] = os.Getenv("AWS_LAMBDA_FUNCTION_NAME")
	sp.attrs["http.method"] = asString(o.base["method"])
	sp.attrs["http.path"] = asString(o.base["path"])

	for k, v := range attrs {
		sp.attrs[k] = v
	}

	ctx = context.WithValue(ctx, spanKey{}, sp)
	return ctx, sp
}

func (s *Span) SetAttr(k string, v interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.attrs == nil {
		s.attrs = map[string]interface{}{}
	}
	s.attrs[k] = v
}

func (s *Span) SetStatus(status, description string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if strings.TrimSpace(status) == "" {
		return
	}
	if description != "" {
		s.attrs["status.description"] = description
	}
	s.status = status
}

func (s *Span) End() {
	s.mu.Lock()
	defer s.mu.Unlock()

	end := time.Now().UTC()
	payload := map[string]interface{}{
		"timestamp":  end.Format(time.RFC3339Nano),
		"type":       "otel_span",
		"name":       s.name,
		"traceId":    s.traceID,
		"spanId":     s.spanID,
		"parentId":   s.parentID,
		"startTime":  s.start.Format(time.RFC3339Nano),
		"endTime":    end.Format(time.RFC3339Nano),
		"durationMs": float64(end.Sub(s.start).Microseconds()) / 1000.0,
		"status":     s.status,
		"attrs":      s.attrs,
	}

	line, _ := json.Marshal(payload)
	fmt.Println(string(line))

	// Local-only: export spans to Tempo via Zipkin receiver so Grafana can show nested spans.
	zipkinURL := strings.TrimSpace(os.Getenv("TEMPO_ZIPKIN_URL"))
	if zipkinURL == "" {
		zipkinURL = strings.TrimSpace(os.Getenv("ZIPKIN_URL"))
	}
	if zipkinURL != "" {
		sendZipkinSpan(zipkinURL, zipkinSpan{
			TraceID:      s.traceID,
			ID:           s.spanID,
			ParentID:     s.parentID,
			Name:         s.name,
			Kind:         "SERVER",
			TimestampMic: s.start.UTC().UnixMicro(),
			DurationMic:  end.Sub(s.start).Microseconds(),
			LocalEndpoint: map[string]string{
				"serviceName": asString(s.attrs["service"]),
			},
			Tags: zipkinTagsFromAttrs(s.attrs),
		})
	}
}

type spanKey struct{}

func awsRequestID(ctx context.Context) string {
	lc, ok := lambdacontext.FromContext(ctx)
	if !ok || lc == nil {
		return ""
	}
	return lc.AwsRequestID
}

func traceRootID() string {
	v := strings.TrimSpace(os.Getenv("_X_AMZN_TRACE_ID"))
	if v == "" {
		return ""
	}
	parts := strings.Split(v, ";")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, "Root=") {
			return strings.TrimPrefix(p, "Root=")
		}
	}
	return ""
}

func asString(v interface{}) string {
	s, _ := v.(string)
	return s
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}

func headerValue(headers map[string]string, name string) string {
	if len(headers) == 0 {
		return ""
	}
	lower := strings.ToLower(strings.TrimSpace(name))
	for k, v := range headers {
		if strings.ToLower(strings.TrimSpace(k)) == lower {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func newTraceID() string {
	// 16 bytes hex -> 32 chars, compatible with OTel trace id length.
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func newSpanID() string {
	// 8 bytes hex -> 16 chars, compatible with OTel span id length.
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// Prometheus export (text format)

func WritePrometheus(w io.Writer) {
	if !inprocEnabled() {
		return
	}
	inproc.mu.Lock()
	defer inproc.mu.Unlock()

	// Gauges
	gkeys := make([]promSeriesKey, 0, len(inproc.gauge))
	for k := range inproc.gauge {
		gkeys = append(gkeys, k)
	}
	sort.Slice(gkeys, func(i, j int) bool { return gkeys[i].String() < gkeys[j].String() })
	for _, k := range gkeys {
		fmt.Fprintf(w, "%s%s %s\n", k.name, k.labels, formatFloat(inproc.gauge[k]))
	}

	// Counters
	keys := make([]promSeriesKey, 0, len(inproc.count))
	for k := range inproc.count {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].String() < keys[j].String() })
	for _, k := range keys {
		fmt.Fprintf(w, "%s_total%s %s\n", k.name, k.labels, formatFloat(inproc.count[k]))
	}

	// Histograms (sum + count)
	hkeys := make([]promSeriesKey, 0, len(inproc.sum))
	for k := range inproc.sum {
		hkeys = append(hkeys, k)
	}
	sort.Slice(hkeys, func(i, j int) bool { return hkeys[i].String() < hkeys[j].String() })
	for _, k := range hkeys {
		fmt.Fprintf(w, "%s_sum%s %s\n", k.name, k.labels, formatFloat(inproc.sum[k]))
		fmt.Fprintf(w, "%s_count%s %s\n", k.name, k.labels, formatFloat(inproc.n[k]))
	}
}

func (p *promAgg) addCounter(name string, labels map[string]string, delta float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	k := promKey(name, labels)
	p.count[k] += delta
}

func (p *promAgg) observe(name string, labels map[string]string, value float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	k := promKey(name, labels)
	p.sum[k] += value
	p.n[k] += 1
}

func (p *promAgg) setGauge(name string, labels map[string]string, value float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	k := promKey(name, labels)
	p.gauge[k] = value
}

type promSeriesKey struct {
	name   string
	labels string // includes surrounding braces, or empty
}

func (k promSeriesKey) String() string { return k.name + k.labels }

func promKey(name string, labels map[string]string) promSeriesKey {
	k := promSeriesKey{name: name}
	if len(labels) == 0 {
		return k
	}

	ks := make([]string, 0, len(labels))
	for k := range labels {
		ks = append(ks, k)
	}
	sort.Strings(ks)

	var b strings.Builder
	b.WriteString("{")
	for i, k := range ks {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(k)
		b.WriteString("=\"")
		b.WriteString(escapeLabelValue(labels[k]))
		b.WriteString("\"")
	}
	b.WriteString("}")
	k.labels = b.String()
	return k
}

func sanitizeMetricName(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	if s == "" {
		return "metric"
	}
	var b strings.Builder
	prevUnderscore := false
	for _, r := range s {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if ok {
			b.WriteRune(r)
			prevUnderscore = false
			continue
		}
		if !prevUnderscore {
			b.WriteByte('_')
			prevUnderscore = true
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "metric"
	}
	return out
}

func sanitizeLabels(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		k2 := sanitizeMetricName(k)
		if k2 == "" {
			continue
		}
		out[k2] = strings.TrimSpace(v)
	}
	return out
}

func escapeLabelValue(v string) string {
	v = strings.ReplaceAll(v, "\\", "\\\\")
	v = strings.ReplaceAll(v, "\"", "\\\"")
	v = strings.ReplaceAll(v, "\n", "\\n")
	return v
}

func formatFloat(f float64) string {
	// Prometheus text format accepts Go float formatting; keep it stable.
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// Zipkin export helpers (no external deps)

type zipkinSpan struct {
	TraceID       string            `json:"traceId"`
	ID            string            `json:"id"`
	ParentID      string            `json:"parentId,omitempty"`
	Name          string            `json:"name"`
	Kind          string            `json:"kind"`
	TimestampMic  int64             `json:"timestamp"` // microseconds
	DurationMic   int64             `json:"duration"`  // microseconds
	LocalEndpoint map[string]string `json:"localEndpoint,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
}

func sendZipkinSpan(url string, sp zipkinSpan) {
	payload, _ := json.Marshal([]zipkinSpan{sp})
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	// Best-effort; don't fail code paths if Tempo isn't up.
	client := &http.Client{Timeout: 750 * time.Millisecond}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	_ = resp.Body.Close()
}

func zipkinTagsFromAttrs(attrs map[string]interface{}) map[string]string {
	if len(attrs) == 0 {
		return nil
	}
	tags := map[string]string{}
	for k, v := range attrs {
		switch vv := v.(type) {
		case string:
			tags[k] = vv
		case int:
			tags[k] = strconv.Itoa(vv)
		case int64:
			tags[k] = strconv.FormatInt(vv, 10)
		case float64:
			tags[k] = strconv.FormatFloat(vv, 'f', -1, 64)
		case bool:
			if vv {
				tags[k] = "true"
			} else {
				tags[k] = "false"
			}
		default:
			// Avoid JSON-encoding arbitrary objects into tags.
		}
	}
	return tags
}
