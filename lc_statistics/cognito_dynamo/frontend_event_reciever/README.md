# Frontend Event Receiver

This Lambda captures frontend analytics events from API Gateway and writes them to S3 for later analysis (Athena/QuickSight).

## What this Lambda captures

The ingestion endpoint accepts **both**:
- **batch payloads** (`events[]`) as the preferred format
- **single event payloads** for backward compatibility

Supported `eventType` values:
- `page_access`
- `button_click`
- `api_call`

## Event capture model (important)

This project uses **separate events for separate meanings**:

- `page_access`: user entered/opened a page
- `button_click`: user clicked a button
- `api_call`: frontend executed an API request

If one user action triggers both a click and an API request, send **two events**:
1. `button_click` (no `api` object)
2. `api_call` (with `api` object)

## Current migration behavior

Backend validation is currently configured as:

- `api_call`
  - `api` object is **required**
- `button_click`
  - `api` object is **not allowed**
- `page_access`
  - `api` object is temporarily accepted for migration compatibility
  - target behavior is to move API data to `api_call`

So if frontend sends `button_click` with `api`, backend returns 400:
- `"api object is allowed only when eventType is api_call"`

## Required payload fields

Required in every event:
- `eventId`
- `timestamp` (RFC3339)
- `app`
- `eventType`
- `phase` (`start` or `response`)
- `source` (`page_load` or `user_click`)
- `feature`
- `page`

If `api` is present:
- required: `api.name`, `api.endpoint`, `api.method`
- `api.method` allowed: `GET`, `POST`, `PUT`, `DELETE`
- optional: `api.statusCode`
- optional `api.outcome`: `success` or `error`

## HTTP behavior

- `OPTIONS` -> `204` (CORS preflight)
- `POST` single event valid -> `202` with:
  - `{ "eventId": "...", "status": "accepted" }`
- `POST` single duplicate eventId/key -> `202` with:
  - `{ "eventId": "...", "status": "already_processed" }`
- `POST` batch valid -> `202` with:
  - `{ "status": "accepted", "batchId": "...", "accepted": <n>, "already_processed": <m> }`
- `POST` batch invalid event -> `400` with:
  - `{ "status": "rejected", "error": "invalid_event", "eventId": "...", "details": "..." }`
- invalid payload -> `400` with explicit message
- internal failure -> `500`

## Batch payload format

```json
{
  "batchId": "uuid",
  "sentAt": "2026-02-15T03:40:00.000Z",
  "reason": "max_batch_size | interval_10s | route_change | pagehide | hidden",
  "app": "veet-app-web",
  "events": [
    {
      "eventId": "uuid",
      "timestamp": "2026-02-15T03:39:55.000Z",
      "app": "veet-app-web",
      "eventType": "page_access | button_click | api_call",
      "phase": "start | response",
      "source": "page_load | user_click",
      "feature": "string",
      "page": "string"
    }
  ]
}
```

Each event in `events[]` is validated independently and persisted as one record per event using the same storage model as single-event ingestion.

## CORS behavior

Controlled by `ALLOWED_ORIGINS`.

- empty -> allows `*`
- `*` -> allows `*`
- comma-separated origins -> reflects matching origin

## Storage model

Bucket:
- from env var `events_bucket_name` (fallback: `EVENTS_BUCKET`)

Prefix:
- from `EVENTS_PREFIX` (default `analytics-events`)

Object key format:
- `<prefix>/anomesdia=YYYYMMDD/app=<app>/eventType=<eventType>/<eventId>.json`

Persisted JSON is flattened (no `raw` payload copy), including:
- event and ingestion timestamps
- dimensions like app/eventType/phase/source/feature/page
- optional api fields (`apiName`, `apiEndpoint`, `apiMethod`, ...)
- optional user fields (`userSub`, `userEmail`)

## Frontend guidance

### Correct page-load flow
When a page is opened and it triggers API calls:
1. Send `page_access`
2. Send `api_call` for each API request

Do **not** send `button_click` on page load.

### Correct button flow
On any button click:
1. Send `button_click` (always)
2. If click triggers API, send `api_call` too

## Environment variables

- `events_bucket_name` (required)
- `EVENTS_BUCKET` (fallback)
- `EVENTS_PREFIX` (optional)
- `ALLOWED_ORIGINS` (optional)
- `REQUIRE_AUTH` (optional, extension point)
- `METRICS_NAMESPACE` (optional, default `Veet/AnalyticsEvents`)
- `AWS_REGION` (optional)

## Observability

- Structured JSON logs with `requestId`, `eventId`, `path`, `method`
- CloudWatch embedded metrics:
  - Success
  - Failure
  - Duplicate

## Build and package

```bash
cd lc_statistics/cognito_dynamo/frontend_event_reciever
make build
make package
```

## Internal package layout

- `main/main.go`: Lambda bootstrap
- `packages/handler/handler.go`: orchestration
- `packages/handler/validation.go`: contract validation
- `packages/handler/cors.go`: CORS logic
- `packages/handler/response.go`: response helper
- `packages/handler/auth.go`: auth extension hooks
- `packages/handler/observability.go`: logs + metrics
- `packages/storage/s3.go`: persistence adapter
- `packages/typesandstructs/typesandstructs.go`: DTOs
