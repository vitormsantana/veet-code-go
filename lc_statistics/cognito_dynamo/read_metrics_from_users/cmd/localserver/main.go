package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/vitormsantana/veet-code-go/lc_statistics/cognito_dynamo/read_metrics_from_users/packages/handler"
	"github.com/vitormsantana/veet-code-go/lc_statistics/cognito_dynamo/read_metrics_from_users/packages/observability"
)

// Local runner:
// - exposes a simple HTTP endpoint that adapts requests to APIGatewayProxyRequest
// - exports one Zipkin span per request to Tempo (so you can see traces in Grafana)
// - exposes /metrics in Prometheus text format (so you can see metrics in Grafana)

type metrics struct {
	invSuccess      atomic.Int64
	invError        atomic.Int64
	invUnauthorized atomic.Int64
	invBadRequest   atomic.Int64
	invPreflight    atomic.Int64
	latencyMsSum    atomic.Int64
	latencyMsCount  atomic.Int64
}

func (m *metrics) inc(result string) {
	switch result {
	case "success":
		m.invSuccess.Add(1)
	case "error":
		m.invError.Add(1)
	case "unauthorized":
		m.invUnauthorized.Add(1)
	case "bad_request":
		m.invBadRequest.Add(1)
	case "preflight":
		m.invPreflight.Add(1)
	default:
		m.invError.Add(1)
	}
}

func (m *metrics) observeLatency(ms int64) {
	m.latencyMsSum.Add(ms)
	m.latencyMsCount.Add(1)
}

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

func main() {
	addr := envDefault("LOCAL_ADDR", ":8080")
	route := envDefault("LOCAL_ROUTE", "/dev/read_user_metrics")
	tempoZipkin := envDefault("TEMPO_ZIPKIN_URL", "http://localhost:9411/api/v2/spans")
	service := envDefault("SERVICE_NAME", "read_metrics_from_users_local")

	// Enable in-process Prometheus aggregation for stage metrics (DynamoDB / processing).
	// This is only used for local adapter mode and is scraped by Prometheus on /metrics.
	if strings.TrimSpace(os.Getenv("OBS_INPROC_PROM")) == "" {
		_ = os.Setenv("OBS_INPROC_PROM", "1")
	}

	var m metrics

	mux := http.NewServeMux()

	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		fmt.Fprintf(w, "# HELP invocations_total Total handler invocations by result\n")
		fmt.Fprintf(w, "# TYPE invocations_total counter\n")
		fmt.Fprintf(w, "invocations_total{result=\"success\"} %d\n", m.invSuccess.Load())
		fmt.Fprintf(w, "invocations_total{result=\"error\"} %d\n", m.invError.Load())
		fmt.Fprintf(w, "invocations_total{result=\"unauthorized\"} %d\n", m.invUnauthorized.Load())
		fmt.Fprintf(w, "invocations_total{result=\"bad_request\"} %d\n", m.invBadRequest.Load())
		fmt.Fprintf(w, "invocations_total{result=\"preflight\"} %d\n", m.invPreflight.Load())
		fmt.Fprintf(w, "\n")
		fmt.Fprintf(w, "# HELP latency_ms_sum Sum of observed latency (ms)\n")
		fmt.Fprintf(w, "# TYPE latency_ms_sum counter\n")
		fmt.Fprintf(w, "latency_ms_sum %d\n", m.latencyMsSum.Load())
		fmt.Fprintf(w, "# HELP latency_ms_count Count of observed latency samples\n")
		fmt.Fprintf(w, "# TYPE latency_ms_count counter\n")
		fmt.Fprintf(w, "latency_ms_count %d\n", m.latencyMsCount.Load())

		// Stage-level metrics emitted by the lambda code (DynamoDB / processing), aggregated in-process.
		fmt.Fprintf(w, "\n")
		observability.WritePrometheus(w)
	})

	mux.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Generate a stable local trace id so:
		// - our Zipkin span shows in Tempo/Grafana
		// - the lambda-style obs logs/spans reuse the same trace id (via WithTraceContext)
		traceID := newHex(16)
		parentID := newHex(8)

		bodyBytes, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()

		hdrs := map[string]string{}
		for k, vals := range r.Header {
			if len(vals) > 0 {
				hdrs[k] = vals[0]
			}
		}

		// Emulate API Gateway proxy request.
		apigwReq := events.APIGatewayProxyRequest{
			HTTPMethod: r.Method,
			Path:       r.URL.Path,
			Headers:    hdrs,
			Body:       string(bodyBytes),
			RequestContext: events.APIGatewayProxyRequestContext{
				RequestID: "local-" + newHex(8),
			},
		}

		ctx := observability.WithTraceContext(context.Background(), traceID, parentID)
		resp, _ := handler.Handler(ctx, apigwReq)

		// Write response
		for k, v := range resp.Headers {
			if strings.TrimSpace(v) != "" {
				w.Header().Set(k, v)
			}
		}
		w.Header().Set("X-Local-Trace-Id", traceID)
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, bytes.NewBufferString(resp.Body))

		// Derive result (match our handler’s usage)
		result := "success"
		if r.Method == http.MethodOptions {
			result = "preflight"
		} else if resp.StatusCode == http.StatusUnauthorized {
			result = "unauthorized"
		} else if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			result = "bad_request"
		} else if resp.StatusCode >= 500 {
			result = "error"
		}

		latMs := time.Since(start).Milliseconds()
		m.inc(result)
		m.observeLatency(latMs)

		// Export a Zipkin span to Tempo so you can see it in Grafana.
		sendZipkinSpan(tempoZipkin, zipkinSpan{
			TraceID:      traceID,
			ID:           parentID,
			Name:         "read_user_metrics",
			Kind:         "SERVER",
			TimestampMic: start.UTC().UnixMicro(),
			DurationMic:  time.Since(start).Microseconds(),
			LocalEndpoint: map[string]string{
				"serviceName": service,
			},
			Tags: map[string]string{
				"http.method":      r.Method,
				"http.path":        r.URL.Path,
				"http.status_code": strconv.Itoa(resp.StatusCode),
				"result":           result,
			},
		})
	})

	log.Printf("localserver listening on %s", addr)
	log.Printf("route=%s  metrics=/metrics  tempo_zipkin=%s", route, tempoZipkin)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func sendZipkinSpan(url string, sp zipkinSpan) {
	payload, _ := json.Marshal([]zipkinSpan{sp})
	req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	// Best-effort; don't fail the request if Tempo isn't up.
	client := &http.Client{Timeout: 750 * time.Millisecond}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	_ = resp.Body.Close()
}

func newHex(nBytes int) string {
	b := make([]byte, nBytes)
	// rand.Read isn't crypto-secure here; good enough for local ids.
	for i := range b {
		b[i] = byte(rand.IntN(256))
	}
	return hex.EncodeToString(b)
}

func envDefault(k, def string) string {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return def
	}
	return v
}
