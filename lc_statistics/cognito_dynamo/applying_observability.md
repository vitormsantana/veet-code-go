# Consider current repo dev env

# Observability Standards & Rules (MVP → Scalable)
**Scope:** Angular (frontend) + API Gateway REST + AWS Lambda (Go/Python) + RDS MySQL  
**Primary goal (MVP):** detect issues *before users complain* + enable fast debugging without tribal knowledge.  
**Secondary goal:** create a scalable base for metrics/tracing later.

---

## 0) Definitions (Glossary)

### Correlation ID
A unique ID used to tie together:
- Browser action (user navigation/click)
- API Gateway request
- Lambda execution logs
- Downstream calls
- DB query logs

**Header name (standard):** `x-correlation-id`

### Environment tag
A normalized string to identify environment: `dev | hml | prod`  
**Header name (standard):** `x-env`

### App version tag
A build version string to correlate incidents with releases.  
**Header name (standard):** `x-app-version`

---

## 1) Non-Negotiable Rules (Must Follow)

### R1 — Everything has correlation
- Every frontend request **MUST** include `x-correlation-id`.
- Every backend request **MUST** log `correlation_id`.
- Every downstream request **MUST** propagate `x-correlation-id`.

### R2 — Logs are structured JSON (no free-text-only)
- All Lambdas **MUST** log in JSON format.
- Every request **MUST** emit a final “request completed” log line with standard fields.

### R3 — No PII in logs/telemetry
- **DO NOT** log `funcional` (employee ID) in plain form.
- **DO NOT** log user names, emails, CPF, or any typed input fields.
- **DO NOT** log raw SQL with parameters that can contain PII.
- If “unique user” is needed later, use **pseudonymized** `analytics_user_id` generated server-side.

### R4 — API Gateway latency near 29s must be visible
- API Gateway Stage **MUST** have Access Logs enabled (JSON format).
- We **MUST** be able to query requests with `latency >= 25000ms` (25 seconds) to detect “almost timeout” risk.

### R5 — MySQL slow queries must be visible
- Application layer **MUST** detect and log slow queries over a threshold (start at `500ms`).
- Logs must include correlation_id and duration, but not sensitive SQL payload.

---

## 2) Frontend (Angular) Standards

### 2.1 Headers required for every HTTP request
Frontend must send these headers **on every request**:
- `x-correlation-id: <uuid>`
- `x-env: dev|hml|prod`
- `x-app-version: <build_version>`

**Implementation requirement:** via Angular `HttpInterceptor`.

### 2.2 Correlation ID generation policy
Choose one policy and keep it consistent:

**Option A (simple, OK for MVP):**
- Generate a new `correlation_id` per HTTP request.

**Option B (recommended):**
- Generate a new `correlation_id` per “user action context”:
  - route navigation
  - button click / UI action
- Reuse this id across all HTTP calls triggered by that action.

**Rule:** never re-use the same correlation_id forever. It must change across user flows.

### 2.3 Frontend error logging (MVP)
Capture and emit error events (without PII):
- `window.onerror`
- `unhandledrejection`

**Fields to include:**
- `error_type` (e.g., `JS_ERROR`, `PROMISE_REJECTION`)
- `screen/route`
- `app_version`
- `env`
- `correlation_id` (if present)

### 2.4 Frontend performance markers (optional MVP)
Measure route navigation timing:
- `NavigationStart` → `NavigationEnd`
Emit event/log (no PII):
- `route_load_ms`
- `route`

---

## 3) API Gateway REST Standards (Timeout/Lag Visibility)

### 3.1 Stage Access Logs must be enabled
For each stage: `dev`, `hml`, `prod`
- Enable CloudWatch Logs
- Enable Access Logs
- Use JSON format (below)

### 3.2 Access Log JSON format (recommended)
Paste this as the Access Log format:

```json
{
  "requestId":"$context.requestId",
  "ip":"$context.identity.sourceIp",
  "userAgent":"$context.identity.userAgent",
  "requestTime":"$context.requestTime",
  "httpMethod":"$context.httpMethod",
  "resourcePath":"$context.resourcePath",
  "status":"$context.status",
  "responseLength":"$context.responseLength",
  "latency":"$context.responseLatency",
  "integrationLatency":"$context.integrationLatency",
  "integrationStatus":"$context.integration.status",
  "errorMessage":"$context.error.message",
  "errorResponseType":"$context.error.responseType"
}
```

### 3.3 Operational query requirement
We must be able to filter in CloudWatch Logs Insights:
- `latency >= 25000`
- Group by `resourcePath` and `status`

---

## 4) Telemetry Direction (Mandatory)

### 4.1 OpenTelemetry now
- We **MUST** instrument services using OpenTelemetry conventions from now on.
- Logs, traces, and metrics should keep OTel naming and context fields (`trace_id`, `span_id`, `correlation_id`) whenever applicable.

### 4.2 Datadog later (not yet)
- Datadog is planned as a future backend/export destination.
- Current implementation should stay vendor-neutral and OTel-first, so migration to Datadog is straightforward when enabled.

---

## 5) Feature Status (Per Feature)

Legend:
- `done`: implemented and validated
- `pending`: not implemented yet for this feature

| Feature | Backend | Frontend | Infra |
|---|---|---|---|
| `read_user_metrics` (`read_metrics_from_users`) | `done` | `done` | `done` |
| `read_exercises` | `pending` | `pending` | `pending` |
| `read_statistics_from_exercises` | `pending` | `pending` | `pending` |
| `read_openai_questions_recommendations` | `pending` | `pending` | `pending` |
| `create_exercise` | `pending` | `pending` | `pending` |
| `create_user_metrics` | `pending` | `pending` | `pending` |
| `create_feedback_for_recomendation` | `pending` | `pending` | `pending` |
| `create_user_profile` | `pending` | `pending` | `pending` |
| `read_user_profile` | `pending` | `pending` | `pending` |
| `frontend_event_reciever` | `pending` | `pending` | `pending` |

Notes:
- This status intentionally marks all features as `pending` except `read_user_metrics`, as requested.
