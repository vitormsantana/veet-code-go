# Observability (lc_statistics/cognito_dynamo Lambdas)

## Requirements (keep this list updated)

These requirements come from `lc_statistics/cognito_dynamo/frontend_event_reciever/to_dos_observability.txt` and must stay as the source-of-truth list.

Every time a new requirement is added (in the txt or elsewhere), add it here too.

1. Add observability.
2. My company uses Datadog; check possibility to do it via OpenTelemetry (OTel) because in my current PC I don't have Datadog access.
3. Analyze possibility to use Datadog Vector.
4. Analyze possibility to use Datadog Synthetics.
5. Implement metrics, traces, logs, so we can correlate.
6. For now write docker composes for open source visualization tools.
7. Prefer OTel patterns over CloudWatch-specific ones for metrics/traces (keep logs structured JSON).
8. Document how to see/validate observability signals (AWS and local).
9. Provide Grafana visibility (charts + traces) for local development, starting with `read_metrics_from_users` (Questions page).
10. Allow the frontend to route specific endpoints to a local adapter (so local Grafana gets real traffic).

## Patterns (applied to all lambdas)

These are the standard patterns to apply across every Go lambda under `lc_statistics/cognito_dynamo/*`:

1. Structured JSON logs
   - All log lines are JSON.
   - Include correlation fields in every line:
     - `awsRequestId` (Lambda context)
     - `requestId` (API Gateway request context)
     - `traceId` (X-Ray root id from `_X_AMZN_TRACE_ID` when available)
     - `service`, `function`, `method`, `path`
   - Never log secrets (especially `Authorization` headers, tokens, API keys).

2. Metrics via OTel (baseline)
   - Emit at least:
     - Counter: `invocations_total{result=success|error|unauthorized|bad_request|preflight}`
     - Histogram: `latency_ms{result=...}`
   - Optional domain metrics per lambda (example: `metrics_fetched`, `questions_fetched`).
   - Export target:
     - Local/dev: OTLP -> OpenTelemetry Collector -> OSS backends (Prometheus/Tempo/Loki/Grafana).
     - While OTLP exporters are not available (offline dev), emit OTel-shaped metric events as JSON logs (no EMF).

3. Request lifecycle events
   - Log `request_received` at handler entry.
   - Log `response_sent` on every return path with:
     - `status`
     - `result`

4. Preflight consistency
   - If the lambda is behind API Gateway with CORS, handle `OPTIONS` explicitly and treat it as `preflight`.

5. Traces via OTel (baseline)
   - Create one server span per Lambda invocation.
   - Add span attributes for request/response (`http.method`, `http.route`/`path`, `http.status_code`, `result`).
   - Export target:
     - Local/dev: OTLP -> OpenTelemetry Collector -> Tempo.
     - While OTLP exporters are not available (offline dev), export spans as JSON logs (no X-Ray/EMF coupling).

6. Measure internal work (queries + processing)
   - Add child spans around any expensive block:
     - auth/claims parsing
     - DynamoDB reads/writes
     - data processing (loops, aggregations, marshaling)
   - Add attributes that help debug performance without leaking data:
     - `db.system=dynamodb`, `db.operation=Query|Scan|GetItem|PutItem`, `db.table`, `items.count`, `bytes.out`
     - `result`, `error.type` (classify errors), `retries`
   - For dashboards/alerts, add metrics for:
     - latency histograms per stage (ex: `db_latency_ms`, `processing_latency_ms`)
     - counters for domain events (ex: `metrics_fetched`)

## Reference Implementation: read_metrics_from_users

This section documents the exact patterns currently implemented in:

- Lambda: `lc_statistics/cognito_dynamo/read_metrics_from_users`
- Handler: `lc_statistics/cognito_dynamo/read_metrics_from_users/packages/handler/handler.go`
- Observability lib: `lc_statistics/cognito_dynamo/read_metrics_from_users/packages/observability/observability.go`
- Local adapter: `lc_statistics/cognito_dynamo/read_metrics_from_users/cmd/localserver/main.go`

These are the patterns we will replicate across the other lambdas.

### Logs (JSON, correlated)

What we emit:
- `message=request_received` at handler entry
- `message=response_sent` on every return path
- `message=unauthorized` / `message=dynamodb_fetch_failed` / `message=response_marshal_failed` on error paths
- `type=otel_metric` for metric events (JSON log lines)
- `type=otel_span` for spans (JSON log lines)

Correlation fields included:
- `awsRequestId` (from Lambda context when running in AWS)
- `requestId` (from API Gateway request context)
- `traceId` (from `_X_AMZN_TRACE_ID` Root when available, otherwise generated)
- `service`, `function`, `method`, `path`

Security:
- We never log the `Authorization` header or tokens.

### CORS + Preflight

Pattern:
- Explicit `OPTIONS` handling.
- Treated as `preflight` result and counted in metrics.
- CORS headers returned consistently for all responses.

### Traces (nested spans)

Span strategy:
- One top-level span per invocation: `handler`
- Child spans for internal work:
  - `auth.parse_token`
  - `dynamodb.fetch_metrics` with attributes:
    - `db.system=dynamodb`
    - `db.operation=Query`
    - `db.table=hammocker_user_metrics_table`
    - `items.count`
    - `stage.latency_ms`
  - `json.marshal_response`

Span attributes:
- `http.method`, `http.path`, `http.status_code`, `result`
- `stage`, `stage.latency_ms`

Span status:
- Error paths set `status=ERROR` and an error description like `dynamodb_fetch_failed`.

Parenting:
- `StartSpan` uses the current span from context as the parent to create a proper nested trace tree.

### Metrics (baseline + Dynamo/perf)

Baseline (per request):
- `Invocation` (Count) with `Result=success|error|unauthorized|bad_request|preflight`
- `LatencyMs` (Milliseconds) with `Result=...`

Domain metrics:
- `MetricsFetched` (Count)

Dynamo/performance metrics:
- `dynamodb_fetch_latency_ms` (Milliseconds) labeled by `result=success|error`
- `dynamodb_fetch_latency_ms_last` gauge (latest) labeled by `result=...`
- `dynamodb_fetch_errors` (Count)
- `dynamodb_items_fetched` (Count)
- `dynamodb_items_fetched_last` gauge (latest)
- `json_marshal_latency_ms` (Milliseconds)
- `json_marshal_latency_ms_last` gauge (latest)

Notes:
- “avg latency” panels based on `rate()`/`increase()` need steady traffic; we added `*_last` gauges so dashboards still show useful values with sparse dev traffic.

### Local Dev Visibility (Grafana)

Local adapter patterns:
- HTTP adapter exposes: `GET /dev/read_user_metrics`
- Prometheus scrape endpoint: `GET /metrics`
- Adds response header: `X-Local-Trace-Id`
- Exports Zipkin spans to Tempo:
  - via `TEMPO_ZIPKIN_URL=http://localhost:9411/api/v2/spans`
- Prometheus:
  - scrapes `host.docker.internal:8080/metrics`
- Grafana dashboard:
  - `Local Observability -> Questions Page: Read User Metrics (Local)`

In-process Prometheus aggregation:
- `packages/observability` can aggregate selected metrics and expose them on `/metrics` when `OBS_INPROC_PROM=1`.
- The local adapter sets `OBS_INPROC_PROM=1` automatically.

### Response Shape (empty results)

Pattern:
- When there are no items, return `[]` (not `null`).
- This avoids confusing clients and makes local testing clearer.

## How To See It (AWS)

1. Make a request to the API Gateway endpoint (curl or browser).
2. Capture correlation ids from the HTTP response headers:
   - `x-amzn-requestid` (API Gateway)
   - `x-amzn-trace-id` (Root trace id when present)
3. Open the Lambda’s CloudWatch log stream for the invocation and search for:
   - `message=request_received`
   - `message=response_sent`
   - `type=otel_metric` (OTel-shaped metrics emitted as JSON logs)
   - `type=otel_span` (OTel-shaped spans emitted as JSON logs)
4. Confirm correlation fields are present on each log line:
   - `awsRequestId`, `requestId`, `traceId`, `service`, `function`, `method`, `path`

## How To See It (Local)

Infrastructure for a local OSS stack lives in `veet-code-infra/observability`.

### What Exists Today (Local)

This repo currently supports local visualization for **one endpoint** via a local adapter:

- Lambda: `lc_statistics/cognito_dynamo/read_metrics_from_users`
- Adapter HTTP route: `GET http://127.0.0.1:8080/dev/read_user_metrics`
- Frontend page that calls it: `http://localhost:4200/questions` (when local obs is enabled)
- Grafana dashboard: `Local Observability -> Questions Page: Read User Metrics (Local)`

Local adapter responsibilities:
- Exposes `/metrics` (Prometheus text format) so Grafana can chart request counts/latency.
- Exports a Zipkin trace span to Tempo per request so you can search/import traces in Grafana.

### Start Grafana Stack

1. Start the stack:
   - `cd /home/vitor/Documents/veet-code/veet-code-infra/observability`
   - This machine does not have `docker compose` / `docker-compose` installed.
   - Use `docker run` (recommended here):
     - Start (containers must be named `grafana`, `prometheus`, `tempo`, `loki`, `otelcol`):
       - `docker network create veet-observability || true`
       - `docker rm -f grafana prometheus tempo loki otelcol || true`
       - `docker run -d --name tempo --network veet-observability -p 3200:3200 -p 9411:9411 -v ./tempo/tempo.yaml:/etc/tempo.yaml:ro grafana/tempo:2.6.1 -config.file=/etc/tempo.yaml`
       - `docker run -d --name prometheus --network veet-observability --add-host=host.docker.internal:host-gateway -p 9090:9090 -v ./prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro prom/prometheus:v2.54.1 --config.file=/etc/prometheus/prometheus.yml`
       - `docker run -d --name loki --network veet-observability -p 3100:3100 -v ./loki/local-config.yaml:/etc/loki/local-config.yaml:ro grafana/loki:2.9.10 -config.file=/etc/loki/local-config.yaml`
       - `docker run -d --name otelcol --network veet-observability -p 4317:4317 -p 4318:4318 -p 9464:9464 -v ./otelcol/config.yaml:/etc/otelcol/config.yaml:ro otel/opentelemetry-collector-contrib:0.104.0 --config=/etc/otelcol/config.yaml`
       - `docker run -d --name grafana --network veet-observability -p 3000:3000 -e GF_SECURITY_ADMIN_USER=admin -e GF_SECURITY_ADMIN_PASSWORD=admin -e GF_AUTH_ANONYMOUS_ENABLED=false -v ./grafana/provisioning:/etc/grafana/provisioning:ro -v ./grafana/dashboards:/etc/grafana/dashboards:ro grafana/grafana:11.2.0`
     - Stop:
       - `docker rm -f grafana prometheus tempo loki otelcol`
2. Open Grafana:
   - `http://127.0.0.1:3000`
   - user: `admin`
   - password: `admin`
3. When OTLP exporters are enabled in code (future step), send OTLP to the local collector:
   - OTLP/gRPC: `localhost:4317`
   - OTLP/HTTP: `localhost:4318`

Local endpoints (from `veet-code-infra/observability/docker-compose.yml`):
- Grafana: `http://localhost:3000`
- Prometheus: `http://localhost:9090`
- Tempo: `http://localhost:3200`
- Loki: `http://localhost:3100`
- OTel Collector OTLP gRPC: `localhost:4317`
- OTel Collector OTLP HTTP: `localhost:4318`

Notes:
- Today (offline dev), lambdas emit `otel_metric` and `otel_span` as JSON logs. Those are visible in CloudWatch immediately.
- Once OTLP export is implemented, metrics and traces can be viewed in Grafana via Prometheus (metrics) and Tempo (traces).

### Run A Local Lambda Adapter (read_metrics_from_users)

This repo includes a local runner that adapts HTTP requests into `APIGatewayProxyRequest` and exports a Zipkin span to Tempo so you can see traces in Grafana, plus a `/metrics` endpoint for Prometheus.

1. Start the local Grafana stack (see section above).
2. Run the local adapter:
   - `cd /home/vitor/Documents/veet-code/veet-code-go/lc_statistics/cognito_dynamo/read_metrics_from_users`
   - `GOTOOLCHAIN=go1.23.4 GOCACHE=/tmp/go-build go run ./cmd/localserver`
3. Call it directly (optional):
   - `curl -i -H "Authorization: Bearer dummy-sub-token" http://localhost:8080/dev/read_user_metrics`
   - Check `X-Local-Trace-Id` response header.
4. Enable the frontend to use the local adapter:
   - Open `http://localhost:4200/questions`
   - In browser DevTools console:
     - `localStorage.setItem('VEET_LOCAL_OBS', '1')`
     - refresh the page
   - The call to `read_user_metrics` will go to `http://127.0.0.1:8080/dev/read_user_metrics` and print `X-Local-Trace-Id` in the console.
5. See it in Grafana:
   - Dashboard:
     - `Dashboards -> Local Observability -> Questions Page: Read User Metrics (Local)`
   - Metrics (Prometheus):
     - Explore -> Prometheus -> query `invocations_total` and `latency_ms_sum` / `latency_ms_count`
   - Traces (Tempo):
     - Explore -> Tempo -> Import trace -> paste the `X-Local-Trace-Id`
     - Or search by service name: `read_metrics_from_users_local`

### Can We See Query / Processing Performance In Grafana?

Yes, but it depends on where you want to see it:

- AWS today (CloudWatch only):
  - You can add child spans around DynamoDB calls and processing functions and you will see `type=otel_span` JSON lines with timings in CloudWatch Logs.

- Local Grafana today (Tempo/Prometheus):
  - Metrics: yes (Prometheus scrapes the local adapter `/metrics`).
  - Traces: currently you see one server span per request (exported as Zipkin to Tempo).
  - Query-level spans are now exported in local adapter mode:
    - `auth.parse_token`
    - `dynamodb.fetch_metrics`
    - `json.marshal_response`
  - Dynamo performance metrics are exposed on `/metrics`:
    - `dynamodb_fetch_latency_ms_sum` / `dynamodb_fetch_latency_ms_count` (label: `result`)
    - `dynamodb_fetch_errors_total`
    - `dynamodb_items_fetched_total`
    - `json_marshal_latency_ms_sum` / `json_marshal_latency_ms_count`
