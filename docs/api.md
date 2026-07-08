# SignalOps API

## Raw Event Ingestion

`POST /v1/events/raw`

Accepts a JSON object and publishes it to the durable raw event topic:

```text
signalops.<environment>.raw.v1
```

The endpoint returns `202 Accepted` only after the broker publish is acknowledged and the
published event is atomically recorded in `raw_event_ledger` and `idempotency_records`.

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

### Persistence Envelope

The JSON object must contain non-empty `tenant_id`, `source_id`, `source_adapter`, and `dataset`
fields plus an RFC3339 `observation_time`. `processing_time` is optional; the gateway acceptance
time is used when it is absent. Optional `entity_hints`, when present, must be an array.

After broker acknowledgement, the gateway stores the original JSON payload, entity hints, event
identity, observation/processing times, and broker coordinates in `raw_event_ledger`. It stores the
same coordinates, `published` status, SHA-256 payload hash, correlation metadata, and route metadata
in `idempotency_records`. Both rows commit in one PostgreSQL transaction.

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
- `400 invalid_event`: required persistence fields are absent or invalid.
- `502 publish_failed`: broker publish failed; no persistence was attempted.
- `503 ingest_unavailable`: publisher, topic, or persistence repository is not configured.
- `503 persistence_failed`: broker publication succeeded, but the atomic audit transaction failed.

A `persistence_failed` response is an indeterminate client outcome because the broker has already
accepted the record. Clients must reuse the same event and idempotency identifiers when reconciling
or retrying; broker-level duplicate delivery remains possible and consumers must remain idempotent.


## Normalized Event Ledger

`GET /v1/normalized-events?tenant_id={tenant_id}&source_id={source_id}&dataset={dataset}&limit=50`

Lists normalized events persisted by the Go normalizer. Filters are optional and pagination is
currently limit-only.

`GET /v1/normalized-events/{event_id}`

Returns one normalized event or `404 normalized_event_not_found`. Responses include the canonical
normalized payload, entities, evidence, metadata, complete normalized contract event, and lineage
coordinates for both `raw.v1` and `normalized.v1`.

The normalizer commits a raw source offset only after the normalized broker publish is acknowledged
and `normalized_event_ledger` persistence succeeds. Invalid source contracts are published to the
algorithm DLQ with original payload and source coordinates before their source offset is committed.
Infrastructure failures leave the source offset uncommitted for retry.


## Signal Ledger

`GET /v1/signals?tenant_id={tenant_id}&source_id={source_id}&dataset={dataset}&detector_id={detector_id}&severity={severity}&limit=50`

Lists persisted Python-emitted signals. Filters are optional and pagination is currently limit-only.
Each response includes detector/model versions, normalized source `event_ids`, confidence, severity,
window times, evidence, recommendation, broker coordinates, and the complete validated `signal.v1`
event.

`GET /v1/signals/{signal_id}`

Returns one signal or `404 signal_not_found`.

The Go signal persister consumes `signalops.<environment>.signal.v1`. It validates the closed
contract, persists the signal, and only then commits the source offset. Invalid contracts are
published to the algorithm DLQ with the original payload and source coordinates before commit.
Database or broker failures leave the source offset uncommitted for retry.


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

### Pipelines

`GET /v1/tenants/{tenant_id}/catalog/pipelines?limit=50`

Returns registered processing pipelines for a tenant ordered by `pipeline_id`.
The local pipeline catalog migration seeds `tenant-local/pipeline-massive-raw-ingest`
for the Massive scheduled pull path.

Response shape:

```json
{
  "pipelines": [
    {
      "tenant_id": "tenant-local",
      "pipeline_id": "pipeline-massive-raw-ingest",
      "source_id": "src-massive",
      "source_domain": "market_data",
      "pipeline_name": "Massive Raw Ingest",
      "description": "Scheduled Massive market-data pull through raw event publication, raw ledger persistence, and idempotency tracking.",
      "status": "active",
      "stages": ["scheduled_pull", "raw_event_build", "broker_publish", "raw_ledger_persist", "idempotency_persist"],
      "input_datasets": ["equity_eod_prices", "option_contracts_daily"],
      "output_topics": ["signalops.local.raw.v1"],
      "metadata": {"adapter":"market_data.massive","provider":"massive","formerly":"polygon.io","streaming":false},
      "created_at": "2026-07-08T00:00:00Z",
      "updated_at": "2026-07-08T00:00:00Z"
    }
  ]
}
```

Current status values: `active`, `inactive`, `deprecated`.

### Rules

`GET /v1/tenants/{tenant_id}/catalog/rules?limit=50`

Returns registered rule definitions for a tenant ordered by `rule_id`. The local
rules catalog migration seeds `tenant-local/rule-marketdata-eod-price-quality`
for Massive EOD equity price quality checks.

Response shape:

```json
{
  "rules": [
    {
      "tenant_id": "tenant-local",
      "rule_id": "rule-marketdata-eod-price-quality",
      "rule_name": "Market Data EOD Price Quality",
      "description": "Flags Massive EOD equity records with missing or non-positive close prices before downstream signal evaluation.",
      "rule_type": "quality_check",
      "severity": "medium",
      "status": "active",
      "version": 1,
      "source_id": "src-massive",
      "pipeline_id": "pipeline-massive-raw-ingest",
      "dataset_scope": ["equity_eod_prices"],
      "entity_scope": ["ticker"],
      "expression": {"language":"json_logic","conditions":[{"field":"close","operator":"exists"},{"field":"close","operator":">","value":0}],"mode":"all"},
      "actions": ["emit_alert", "mark_event_quality_failed"],
      "metadata": {"provider":"massive","formerly":"polygon.io","execution":"catalog_only","streaming":false},
      "created_at": "2026-07-08T00:00:00Z",
      "updated_at": "2026-07-08T00:00:00Z"
    }
  ]
}
```

Current status values: `active`, `inactive`, `deprecated`. Current severity
values: `info`, `low`, `medium`, `high`, `critical`.

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
