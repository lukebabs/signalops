# SignalOps API

## Raw Event Ingestion

`POST /v1/events/raw`

Accepts a JSON object and publishes it to the durable raw event topic:

```text
signalops.<environment>.raw.v1
```

The endpoint returns `202 Accepted` only after the broker publish is
acknowledged.

### Request Headers

- `Content-Type: application/json`
- `X-SignalOps-Event-ID`: optional event identifier override.
- `X-Idempotency-Key`: optional broker key override.
- `X-Correlation-ID`: optional correlation identifier.
- `X-Causation-ID`: optional causation identifier.
- `X-Trace-ID`: optional trace identifier.

If identifiers are omitted:

- `event_id` is read from the JSON body, or generated.
- `idempotency_key` is read from the JSON body, or falls back to `event_id`.
- `correlation_id` is read from the JSON body, or generated.

### Broker Mapping

- topic: `signalops.<environment>.raw.v1`
- key: idempotency key
- value: original request JSON bytes
- headers:
  - `content_type`
  - `signalops_event_id`
  - `signalops_idempotency`
  - `signalops_ingest_route`
  - `signalops_ingest_format`
  - `correlation_id`
  - `causation_id`
  - `trace_id`

### Success Response

```json
{
  "status": "accepted",
  "event_id": "evt-123",
  "idempotency_key": "idem-123",
  "correlation_id": "corr-123",
  "topic": "signalops.local.raw.v1",
  "partition": 0,
  "offset": 1
}
```

### Error Responses

- `400 invalid_json`: request body is not a JSON object.
- `502 publish_failed`: broker publish failed.
- `503 broker_unavailable`: ingestion route is not wired with a publisher.


## Dashboard Stream API

`GET /v1/streams/dashboard?channels=health,runs,raw_events,provider_usage,heartbeat`

Streams browser-facing dashboard updates using Server-Sent Events (SSE). This
endpoint is a gateway bridge over the existing query repository; browsers MUST
NOT connect directly to Redpanda.

### Response Headers

```http
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
```

### Channels

The `channels` query parameter is optional. When omitted, all supported channels
are enabled. Values are comma-separated.

Supported channel names:

- `health`
- `runs` or `scheduler_run`
- `raw_events` or `raw_event`
- `provider_usage`
- `heartbeat`

Unknown channels return `400 invalid_channel` as JSON before the stream starts.

### Event Frames

Each SSE message uses this shape:

```text
event: <event_type>
id: <stable_id_when_available>
data: <json_object>
```

Current event types:

- `heartbeat`: emitted when the stream opens and periodically while connected.
- `health`: gateway health payload with service name and UTC timestamp.
- `scheduler_run`: one scheduler run DTO, with `id` set to `run_id`.
- `raw_event`: one raw event ledger DTO, with `id` set to `event_id`.
- `provider_usage`: one provider usage DTO, with `id` set to `usage_id`.
- `error`: stream-level error payload such as `storage_unavailable` or
  `query_failed`.

Example:

```text
event: raw_event
id: evt_5d5a94a0e8ea5d149ec19947
data: {"event_id":"evt_5d5a94a0e8ea5d149ec19947","tenant_id":"tenant-local"}
```

### Limitations

- The initial implementation polls the query repository inside the gateway and
  deduplicates rows per connection by stable id.
- `Last-Event-ID` replay is not implemented yet.
- The stream does not expose broker partitions directly beyond the persisted
  raw-event/provider DTO fields.
- REST query endpoints remain the snapshot/detail fallback for the frontend.

## Stream Catalog API

These endpoints expose tenant-scoped catalog metadata for registered SignalOps
sources. They require gateway storage wiring through `SIGNALOPS_DATABASE_URL`;
when storage is not configured they return `503 storage_unavailable`.

### Sources

`GET /v1/tenants/{tenant_id}/catalog/sources?limit=50`

Returns registered source adapters for a tenant ordered by `source_id`. The first
local catalog migration seeds `tenant-local/src-massive` for the Massive market
data adapter.

Response shape:

```json
{
  "sources": [
    {
      "tenant_id": "tenant-local",
      "source_id": "src-massive",
      "source_domain": "market_data",
      "source_adapter": "market_data.massive",
      "display_name": "Massive Market Data",
      "description": "Scheduled Massive market-data source for equity EOD prices and daily option contracts.",
      "status": "active",
      "ingestion_modes": ["scheduled_pull"],
      "datasets": ["equity_eod_prices", "option_contracts_daily"],
      "metadata": {"provider":"massive","formerly":"polygon.io","streaming":false},
      "created_at": "2026-07-08T00:00:00Z",
      "updated_at": "2026-07-08T00:00:00Z"
    }
  ]
}
```

Current status values: `active`, `inactive`, `deprecated`.

## Operational Query API

These endpoints require gateway storage wiring through `SIGNALOPS_DATABASE_URL`.
When storage is not configured they return `503 storage_unavailable`.

### Scheduler Runs

`GET /v1/scheduler/runs?limit=50`

Returns recent scheduler run audit rows ordered by `started_at DESC`.

`GET /v1/scheduler/runs/{run_id}`

Returns one scheduler run, including datasets, counters, config JSON, report JSON,
status, timestamps, and optional error message.

### Provider Usage

`GET /v1/provider-usage?run_id={run_id}&limit=50`

Returns provider usage rows. `run_id` is optional; when omitted, recent provider
usage across runs is returned.

### Raw Event Ledger

`GET /v1/raw-events?tenant_id={tenant_id}&source_id={source_id}&dataset={dataset}&limit=50`

Returns recent raw event ledger rows. Filters are optional and can be combined.
Each row includes broker acknowledgement details, event timing, payload JSON, and
entity hints.

`GET /v1/raw-events/{event_id}`

Returns one raw event ledger row by event id.

### Idempotency Lookup

`GET /v1/idempotency?tenant_id={tenant_id}&source_id={source_id}&idempotency_key={key}`

Returns one idempotency record. All three query parameters are required.

### Query Errors

- `400 missing_query`: required idempotency lookup parameters are missing.
- `404 *_not_found`: requested run, raw event, or idempotency record was not found.
- `500 query_failed`: storage query failed.
- `503 storage_unavailable`: gateway was not configured with query storage.
