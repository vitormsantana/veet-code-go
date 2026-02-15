# Analytics Feature Improvements (Backlog)

Baseline: 2026-02-15  
Sources: `veet-code-app/veet-app/README.analytics-batch.md`, current implementation in both repos.

This file is a practical list of improvements to harden and evolve the analytics event recorder feature (frontend batching -> API Gateway -> Lambda -> S3 -> Athena -> Grafana).

## Goals

- Prevent runaway traffic (no “hundreds of calls on page load”).
- Preserve event semantics (`button_click` != `api_call`).
- Improve delivery reliability without spamming.
- Make S3 layout and Athena queries cheaper and faster.
- Provide a repeatable Grafana dashboard + alerts.

## Frontend Improvements (Angular)

### 1) Reliability and Rate Control

- Add exponential backoff on flush failure (cap to avoid infinite retries).
- Add a hard per-session rate limit (ex: max N flushes/minute) to protect API Gateway/Lambda from accidental loops.
- Add a circuit-breaker: if analytics endpoint returns 429/5xx repeatedly, disable sending for X minutes and log once.
- Consider adding `navigator.sendBeacon` / `fetch(..., { keepalive: true })` on exit-flush paths.

### 2) Flush Strategy Refinements

- Confirm the 10s periodic flush is a “check then flush if queue non-empty”, and log only when useful (avoid noisy console in prod builds).
- Add `minBatchSize` for interval-based flush (ex: only flush on interval if >= 2 events), but still flush on route change/exit.
- Add a `maxQueueLength` cap (drop oldest or newest with a warning) to prevent unbounded memory growth.

### 3) Capture API Conventions (Maintainability)

- Enforce that app code calls only `AnalyticsCaptureService.capture(...)` (single entry point).
- Add a lint rule or grep-based CI check to block direct calls to `EventTrackingService.track*` from components.
- Standardize `label` usage:
  - `button_click.label` should be human-readable button label
  - `api_call.label` should be the user action label that triggered it (optional)

### 4) API Call Tracking Correctness

- Make `ApiEventsInterceptor` set `source` accurately:
  - `page_load` for initial boot calls
  - `user_click` only when the request is known to be click-driven (usually manual tracking)
- Ensure manual-tracked endpoints are consistently added to `manuallyTrackedEndpoints` to avoid double counting.

### 5) Privacy / Security

- Decide whether `userEmail` should be sent at all (PII). Prefer `userSub` only, or hash email client-side.
- Ensure `metadata` never contains tokens, Authorization headers, or request/response bodies.

### 6) Testing

- Add unit tests for batching behavior:
  - flush when `MAX_BATCH_SIZE` reached
  - flush on route change
  - no API call when interval passes with empty queue
- Add a test for the “no synthetic click” rule (`event.isTrusted`).

## Backend Improvements (Go Lambda)

### 1) Batch Behavior Guarantees

- Consider “partial acceptance” mode:
  - validate each event
  - write valid ones
  - return 202 with per-event statuses
  - keep strict “reject entire batch” as default if you prefer simplicity
- Add a max events per batch (ex: 50) to protect Lambda memory/time.
- Add request body size guardrails and clear error messages.

### 2) Idempotency and Dedupe

- If dedupe is implemented via S3 key existence, optimize:
  - avoid extra `HeadObject` roundtrips if possible (still OK at current scale)
- Document idempotency expectation:
  - “same `eventId` should not create new files”

### 3) Observability

- Emit metrics:
  - `EventsReceived` (count)
  - `EventsAccepted`
  - `EventsDuplicate`
  - `EventsInvalid`
  - `BatchesReceived`
  - `BatchSize` (distribution)
  - `S3PutLatencyMs` (distribution)
- Add structured logs with:
  - `batchId`, `reason`, `accepted`, `already_processed`
  - validation error includes `eventId` and a short code

### 4) Schema Evolution

- Version the event schema:
  - add `schemaVersion` to events (frontend)
  - backend accepts multiple versions
- Keep strict semantic rules:
  - `button_click` must not contain `api`
  - `api_call` must contain `api`

## S3 / Athena / Data Model Improvements

### 1) Partitioning and Query Cost

- Current partitions: `anomesdia`, `app`, `eventtype`
  - OK for dashboards filtering by day/type/app
- Consider adding partition projection in Athena to avoid `MSCK REPAIR`:
  - improves automation and reduces Glue calls

### 2) File Format and Compaction (Big Win)

- Today: 1 JSON file per event (simple, but many small files).
- Next step: compaction job (daily/hourly) to Parquet:
  - write `parquet/` alongside raw JSON, or replace after validation
  - huge reduction in Athena scan cost and query latency
  - can be a Glue job, Lambda, or Step Functions + ECS task

### 3) Data Quality

- Validate timestamp parsing and normalize to UTC.
- Add derived columns server-side:
  - `hour`, `minute` (useful for time-series panels)
- Add a field for `environment` (`dev|prod`) in both frontend and backend to avoid mixing.

## Grafana Improvements (Athena Datasource)

### 1) Provisioning and Repeatability

- Store `docker-compose.yml` + provisioning files in a dedicated folder (ex: `analytics/grafana/`).
- Add a “1 command bring-up” script and clear env var requirements.

### 2) Dashboard Design

- Dashboard variables:
  - `app`, `eventtype`, `anomesdia`, `source`
- Panels:
  - Events over time (time series)
  - Event type split (bar)
  - API success/error by endpoint (stacked bar)
  - p95 API statusCode distribution (if you add latency later)
  - Top pages, top buttons

### 3) Alerts (Operational Safety)

- Alert: sudden spike in `events` per minute (possible infinite loop).
- Alert: sustained 4xx/5xx from analytics endpoint (requires backend metrics).

## Infra Improvements (Terraform / API Gateway)

- Add WAF / throttling limits to the analytics endpoint to protect from runaway clients.
- Ensure stage deployments are explicit to avoid drift.
- Add alarms on:
  - API Gateway 4xx/5xx
  - Lambda errors/throttles/duration
  - S3 PutObject errors (rare)

## Suggested Next Steps (Order)

1. Add rate limit + circuit breaker in frontend flush path.
2. Add backend batch size limits + metrics.
3. Add Grafana dashboard provisioning files tracked in repo.
4. Implement Parquet compaction (optional but highest ROI for cost/performance).

