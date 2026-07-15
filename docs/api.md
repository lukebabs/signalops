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

`GET /v1/normalized-events?tenant_id={tenant_id}&app_id={app_id}&domain={domain}&use_case={use_case}&source_id={source_id}&dataset={dataset}&limit=50`

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

The Go normalizer applies MarketOps feature enrichment for Massive `equity_eod_prices` and
`options_contracts_daily` records (`app_id=marketops`, `source_adapter=market_data.massive`).
Both datasets can carry a deterministic `normalized_payload.features` map with price-derived
features: `open_close_move_pct`, `intraday_range_pct`, `vwap_distance_pct`, and
`daily_return_pct` when the required source fields are present. Option records also carry
option-interest features: `open_interest`, `volume`, `volume_open_interest_ratio`, and
`days_to_expiration`.

The options canonicalization strategy requires `option_ticker`, `underlying_symbol`,
`contract_type` (`call` or `put`), `expiration_date`, `observation_date`, and positive
`strike_price`; it validates optional OHLC/VWAP as non-negative numbers and volume/open interest
as non-negative integers while preserving provider contract ID and raw provider metadata when
present. Invalid option contracts follow the existing normalization DLQ path.


## Signal Ledger

`GET /v1/signals?tenant_id={tenant_id}&app_id={app_id}&domain={domain}&use_case={use_case}&source_id={source_id}&dataset={dataset}&detector_id={detector_id}&severity={severity}&limit=50`

Lists persisted Python-emitted signals. Filters are optional and pagination is currently limit-only.
Each response includes detector/model versions, normalized source `event_ids`, confidence, severity,
window times, evidence, recommendation, broker coordinates, and the complete validated `signal.v1`
event.

`GET /v1/signals/{signal_id}`

Returns one signal or `404 signal_not_found`.

## MarketOps DSM Artifacts

`GET /v1/marketops/dsm/artifacts?tenant_id={tenant_id}&app_id={app_id}&domain={domain}&use_case={use_case}&signal_type={signal_type}&severity={severity}&subject_symbol={subject_symbol}&limit=50`

Lists first-class MarketOps DSM artifact proposals materialized from persisted signal semantic
evidence. Filters are optional and pagination is currently limit-only. G077 materializes artifacts
idempotently from `semantic_evidence` entries with `type=dsm_artifact_proposal` and embedded
`marketops.dsm.signal_artifact.v1` payloads while preserving the original signal ledger as the
source of truth.

`GET /v1/marketops/dsm/artifacts/{artifact_id}`

Returns one DSM artifact proposal or `404 artifact_not_found`.

Each artifact response includes source/app metadata, linked `signal_id`, `signal_type`, detector,
severity/confidence, `event_ids`, `subject_symbol`, `artifact_type`, the artifact JSON, semantic
evidence JSON, graph target proposals, supporting metrics, and quality issues.

The Go signal persister consumes `signalops.<environment>.signal.v1`. It validates the closed
contract, persists the signal, and only then commits the source offset. Invalid contracts are
published to the algorithm DLQ with the original payload and source coordinates before commit.
Database or broker failures leave the source offset uncommitted for retry.



## Alert and Insight Lifecycle

Signals persisted by the Go signal persister now derive first-class alert and insight ledger rows in
the same database transaction as the signal. The signal topic offset is committed only after signal,
alert, and insight persistence succeeds.

Current derivation rules:

- Every valid signal creates or updates one active insight with id `insight:{signal_id}`.
- `medium`, `high`, and `critical` signals create or update one open alert with id `alert:{signal_id}`.
- `info` and `low` signals do not create alerts.
- Reprocessing the same signal is idempotent and does not reset existing alert/insight lifecycle
  status fields.
- Lifecycle mutation endpoints are available with an explicit operator placeholder. The gateway reads
  `X-SignalOps-Actor`, then request body `actor`, and finally defaults to `operator-local` until formal
  authentication is wired.

### Alerts

`GET /v1/alerts?tenant_id={tenant_id}&app_id={app_id}&domain={domain}&use_case={use_case}&source_id={source_id}&dataset={dataset}&severity={severity}&status={status}&limit=50`

Lists alert ledger rows. Filters are optional and pagination is currently limit-only. Alerts include
source identity, linked `signal_id`, detector identity, severity, lifecycle status, title, summary,
confidence, event lineage, entities, evidence, recommendation, correlation id, observed times,
optional acknowledgement/resolution fields, metadata, and audit timestamps.

`GET /v1/alerts/{alert_id}`

Returns one alert or `404 alert_not_found`.

`POST /v1/alerts/{alert_id}/acknowledge`
`POST /v1/alerts/{alert_id}/resolve`
`POST /v1/alerts/{alert_id}/suppress`

Mutates an existing alert lifecycle row and returns `{ "alert": ... }` with the updated envelope.
Request body is optional and may include `actor`, `note`, and `reason`. The `X-SignalOps-Actor`
header takes precedence over body `actor`; if neither is supplied, the gateway records
`operator-local`. Mutation metadata is merged into the existing alert `metadata.lifecycle` object.
Missing alerts return `404 alert_not_found`; malformed request bodies return `400 invalid_json`.

Current alert statuses: `open`, `acknowledged`, `resolved`, `suppressed`.

### Insights

`GET /v1/insights?tenant_id={tenant_id}&app_id={app_id}&domain={domain}&use_case={use_case}&source_id={source_id}&dataset={dataset}&insight_type={insight_type}&status={status}&limit=50`

Lists insight ledger rows. Filters are optional and pagination is currently limit-only. Insights include
source identity, linked `signal_id`, detector identity, insight type, lifecycle status, title, summary,
confidence, severity, event lineage, entities, supporting metrics, semantic evidence, recommendation,
correlation id, observed/review fields, metadata, and audit timestamps.

`GET /v1/insights/{insight_id}`

Returns one insight or `404 insight_not_found`.

`POST /v1/insights/{insight_id}/review`
`POST /v1/insights/{insight_id}/dismiss`
`POST /v1/insights/{insight_id}/archive`

Mutates an existing insight lifecycle row and returns `{ "insight": ... }` with the updated envelope.
Request body is optional and may include `actor`, `note`, and `reason`. The `X-SignalOps-Actor`
header takes precedence over body `actor`; if neither is supplied, the gateway records
`operator-local`. Mutation metadata is merged into the existing insight `metadata.lifecycle` object.
Missing insights return `404 insight_not_found`; malformed request bodies return `400 invalid_json`.

Current insight statuses: `active`, `reviewed`, `dismissed`, `archived`.

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

## MarketOps Asset Universe API

This endpoint exposes the first first-class MarketOps asset universe from the
checked-in Top 50 mega-cap seed. It requires gateway storage wiring through
`SIGNALOPS_DATABASE_URL`; when storage is not configured it returns
`503 storage_unavailable`.

`GET /v1/tenants/{tenant_id}/marketops/assets?universe_group=top50_megacap&active_only=true&limit=50`

Query parameters:

- `universe_group`: optional universe group. Defaults to `top50_megacap`.
- `active_only`: optional boolean. Defaults to `true`; set `false` to include inactive rows.
- `limit`: optional limit. Defaults to `50` and is clamped by the gateway.

The local migration seeds 50 `tenant-local` MarketOps assets with
`app_id=marketops`, `domain=market_data`, `use_case=daily_market_surveillance`,
`source_id=src-massive`, and `universe_group=top50_megacap`.

Response shape:

```json
{
  "assets": [
    {
      "tenant_id": "tenant-local",
      "app_id": "marketops",
      "domain": "market_data",
      "use_case": "daily_market_surveillance",
      "source_id": "src-massive",
      "universe_group": "top50_megacap",
      "rank": 1,
      "ticker": "NVDA",
      "ticker_key": "nvda",
      "company": "NVIDIA",
      "company_key": "nvidia",
      "asset_type": "equity",
      "exchange": "",
      "sector": "Technology",
      "sector_key": "technology",
      "industry": "Semiconductors",
      "industry_key": "semiconductors",
      "is_active": true,
      "metadata": {"seed":"top50megacap.normalized.csv","provider":"massive"},
      "created_at": "2026-07-10T00:00:00Z",
      "updated_at": "2026-07-10T00:00:00Z"
    }
  ]
}
```

## Operational Query API

These endpoints require gateway storage wiring through `SIGNALOPS_DATABASE_URL`.
When storage is not configured they return `503 storage_unavailable`.

### Scheduler Runs

`GET /v1/scheduler/runs?limit=50`

Returns recent scheduler run audit rows ordered by `started_at DESC`.

`GET /v1/scheduler/runs/{run_id}`

Returns one scheduler run, including datasets, counters, config JSON, report JSON,
status, timestamps, and optional error message.


### Replay Jobs

Replay jobs are PostgreSQL control-plane records that request replay of TimescaleDB temporal ledgers. The replay worker claims queued jobs, republishes matching temporal records to the appropriate durable topic, and updates job result metadata.

`POST /v1/replay/jobs`

Creates a queued replay job. The endpoint accepts JSON:

```json
{
  "tenant_id": "tenant-local",
  "source_id": "src-massive",
  "dataset": "equity_eod_prices",
  "source_kind": "raw_events",
  "replay_mode": "original",
  "requested_by": "operator-local",
  "window_start": "2026-07-09T00:00:00Z",
  "window_end": "2026-07-10T00:00:00Z",
  "filters": {"symbol": "AAPL"},
  "options": {"publish": false}
}
```

Required fields: `tenant_id`, `window_start`, and `window_end`. `source_kind` defaults to `raw_events`; `replay_mode` defaults to `original`; `requested_by` defaults from `X-SignalOps-Actor` or `operator-local`.

Supported `source_kind` values: `raw_events`, `normalized_events`, `signals`. Supported `replay_mode` values: `original`, `latest_compatible`, `explicit`. New jobs start with status `queued`.

Response status: `202 Accepted` with `{ "replay_job": ... }`.

`GET /v1/replay/jobs?tenant_id={tenant_id}&source_id={source_id}&dataset={dataset}&source_kind={source_kind}&status={status}&limit=50`

Lists replay jobs ordered by `created_at DESC`. Filters are optional and can be combined.

`GET /v1/replay/jobs/{replay_job_id}`

Returns one replay job.

`POST /v1/replay/jobs/{replay_job_id}/cancel`

Cancels a queued or running replay job. The endpoint accepts an optional lifecycle body:

```json
{
  "actor": "operator-local",
  "reason": "validation window changed",
  "note": "optional operator note"
}
```

Response status: `200 OK` with `{ "replay_job": ... }`. The worker detects cancellation between replay batches, stops publishing new records, and merges partial counters into the replay job `result` JSON.

`GET /v1/replay/status?tenant_id={tenant_id}&limit=20`

Returns replay operations status for dashboards and health views:

```json
{
  "replay_status": {
    "generated_at": "2026-07-10T05:21:33Z",
    "job_counts": {"queued": 0, "running": 0, "succeeded": 2, "failed": 0, "canceled": 0},
    "workers": [
      {
        "worker_id": "signalops-replay-worker",
        "status": "idle",
        "health": "online",
        "process_started_at": "2026-07-10T05:13:35Z",
        "last_seen_at": "2026-07-10T05:21:30Z",
        "last_claimed_replay_job_id": "replay-123",
        "last_completed_replay_job_id": "replay-123",
        "metadata": {"poll_interval": "5s"}
      }
    ],
    "latest_jobs": []
  }
}
```

Worker `health` is derived by the gateway from heartbeat freshness: `online`, `stale`, or `error`.

Execution notes:

- `raw_events` replay publishes stored raw payloads to `signalops.<env>.raw.v1`; the normalizer then reprocesses them.
- `normalized_events` replay publishes stored normalized event envelopes to `signalops.<env>.normalized.v1`.
- `signals` replay publishes stored signal envelopes to `signalops.<env>.signal.v1`.
- Replayed payloads include `replay_job_id`, `ingestion_mode: replay`, and `metadata.replay`.
- The worker reads temporal rows in batches using `SIGNALOPS_REPLAY_BATCH_SIZE` and caps each job at `SIGNALOPS_REPLAY_MAX_RECORDS`.
- Broker publishes are retried up to `SIGNALOPS_REPLAY_PUBLISH_MAX_ATTEMPTS` before the job fails.
- The worker result JSON records scanned, published, failed, batch, cancellation, and sampled per-record publish status fields.

### Provider Usage

`GET /v1/provider-usage?run_id={run_id}&limit=50`

Returns provider usage rows. `run_id` is optional; when omitted, recent provider
usage across runs is returned.

### Raw Event Ledger

`GET /v1/raw-events?tenant_id={tenant_id}&app_id={app_id}&domain={domain}&use_case={use_case}&source_id={source_id}&dataset={dataset}&limit=50`

Returns recent raw event ledger rows. Filters are optional and can be combined.
Each row includes broker acknowledgement details, event timing, payload JSON, and
entity hints.

`GET /v1/raw-events/{event_id}`

Returns one raw event ledger row by event id.

### Idempotency Lookup

`GET /v1/idempotency?tenant_id={tenant_id}&source_id={source_id}&idempotency_key={key}`

Returns one idempotency record. All three query parameters are required.

### MarketOps Back-Tests

`POST /v1/marketops/backtests`

Creates and executes a bounded MarketOps back-test run synchronously. The run reads historical normalized MarketOps rows and writes only isolated `marketops_backtest_*` records.

```json
{
  "run_id": "bt-g081-api-smoke",
  "tenant_id": "tenant-local",
  "source_id": "src-massive",
  "dataset": "equity_eod_prices",
  "detector_id": "marketops.dsm.taxonomy_v1",
  "symbols": ["SPY"],
  "window_start": "2026-07-09T00:00:00Z",
  "window_end": "2026-07-10T00:00:00Z",
  "max_records": 5,
  "batch_size": 5,
  "auto_accept_confidence": 0.75
}
```

Response status: `201 Created` with `{ "backtest_run": ..., "metrics": ... }`. The endpoint is bounded by `max_records` and rejects values above `1000`.

`GET /v1/marketops/backtests?tenant_id={tenant_id}&detector_id={detector_id}&status={status}&limit=50`

Returns isolated MarketOps back-test run rows. Filters are optional. Back-test runs are experiments and are separate from replay jobs.

`GET /v1/marketops/backtests/{run_id}`

Returns one back-test run with filters, parameters, aggregate metrics, and terminal error state when applicable.

`GET /v1/marketops/backtests/{run_id}/signals?tenant_id={tenant_id}&signal_type={signal_type}&limit=50`

Returns generated signal records scoped to the back-test run. These are not production `signal_ledger` rows.

`GET /v1/marketops/backtests/{run_id}/graph-proposals?tenant_id={tenant_id}&subject_symbol={symbol}&candidate_type={candidate_type}&recommendation={recommendation}&limit=50`

Returns generated graph proposal records and policy results scoped to the run. Recommendation values are `auto_accept_candidate`, `auto_reject_candidate`, `manual_review_required`, and `supersede_candidate`.

`POST /v1/marketops/backtest-calibration-summaries`

Creates a persisted calibration summary snapshot over a filter-defined set of existing MarketOps back-test runs. The summary stores selected run ids, run counts, zero-input count, aggregate metrics, recommendation counts/shares, and dominant recommendation. It does not mutate production ledgers.

`GET /v1/marketops/backtest-calibration-summaries?tenant_id={tenant_id}&dataset={dataset}&detector_id={detector_id}&limit=50`

Returns persisted calibration summary snapshots. Filters are optional.

`GET /v1/marketops/backtest-calibration-summaries/{summary_id}`

Returns one persisted calibration summary snapshot.

`POST /v1/marketops/backtest-calibration-baselines`

Creates or updates a named calibration baseline that points at an immutable persisted calibration summary. Required fields are `tenant_id`, `name`, and `summary_id`; `baseline_id`, `description`, `scope`, `status`, and `created_by` are optional. Status values are `active` and `archived`.

`GET /v1/marketops/backtest-calibration-baselines?tenant_id={tenant_id}&dataset={dataset}&detector_id={detector_id}&status=active&limit=50`

Returns persisted calibration baselines. Filters are optional.

`GET /v1/marketops/backtest-calibration-baselines/{baseline_id}`

Returns one persisted calibration baseline.

`POST /v1/marketops/backtest-calibration-comparisons`

Creates or updates a stored comparison between a baseline summary and a candidate calibration summary. Required fields are `tenant_id`, `baseline_id`, and `candidate_summary_id`; `comparison_id` and `created_by` are optional. The response includes deterministic aggregate deltas and an advisory recommendation.

`GET /v1/marketops/backtest-calibration-comparisons?tenant_id={tenant_id}&baseline_id={baseline_id}&recommendation={recommendation}&limit=50`

Returns stored baseline comparisons. Recommendation filters use `needs_more_data`, `regression_candidate`, `improvement_candidate`, `neutral_candidate`, or `manual_review_required`.

`GET /v1/marketops/backtest-calibration-comparisons/{comparison_id}`

Returns one stored baseline comparison.

`POST /v1/marketops/backtest-evaluation-labels/sync`

Synchronizes evaluation labels from reviewed MarketOps DSM graph proposal decisions. Required body field: `tenant_id`. Optional body fields: `app_id`, `domain`, `use_case`, `status`, `include_unresolved`, `limit`, and `requested_by`. When `status` is omitted, the sync defaults to `accepted`, `rejected`, and `superseded`; set `include_unresolved=true` to include `proposed` as `unresolved`.

`GET /v1/marketops/backtest-evaluation-labels?tenant_id={tenant_id}&label={label}&decision_status={status}&limit=50`

Returns synchronized evaluation labels. Labels are `positive`, `negative`, `superseded`, and `unresolved`.

`GET /v1/marketops/backtest-evaluation-labels/{label_id}`

Returns one synchronized evaluation label.

`POST /v1/marketops/backtest-evaluations`

Creates a label-aware evaluation for one MarketOps back-test run. Required body fields: `tenant_id` and `run_id`. Optional fields: `evaluation_id`, `label_source`, and `requested_by`. The scorer matches run-scoped generated graph proposals to synchronized evaluation labels by graph fact key and scores automatic policy recommendations against positive/negative labels.

`GET /v1/marketops/backtest-evaluations?tenant_id={tenant_id}&run_id={run_id}&limit=50`

Returns stored label-aware back-test evaluations.

`GET /v1/marketops/backtest-evaluations/{evaluation_id}`

Returns one stored label-aware back-test evaluation.

`POST /v1/marketops/backtest-promotion-candidates`

Creates a stored promotion candidate from G083/G085 evidence. Required fields are `tenant_id`, `baseline_id`, and `comparison_id`; `evaluation_id`, `candidate_id`, `candidate_version`, and `requested_by` are optional. Creation computes conservative readiness status and does not deploy policy or change detector thresholds.

`GET /v1/marketops/backtest-promotion-candidates?tenant_id={tenant_id}&baseline_id={baseline_id}&readiness_status={status}&status=proposed&limit=50`

Returns stored promotion candidates. Filters are optional.

`GET /v1/marketops/backtest-promotion-candidates/{candidate_id}`

Returns one stored promotion candidate.

`POST /v1/marketops/backtest-promotion-candidates/{candidate_id}/decision`

Records an operator review decision on a promotion candidate. Allowed decision statuses are `approved_for_promotion`, `rejected`, `deferred`, and `superseded`. This endpoint only mutates the candidate audit row; it does not deploy runtime policy, edit detector thresholds, or write graph state.

### Algorithm Registry

`POST /v1/algorithms/definitions`

Creates or updates a versioned algorithm definition. Definitions describe algorithm id, type, runtime type, input feature names, input event types, schemas, default config, status, and metadata. Seeded G106 definitions are draft records and are not executable until a runner gate is added.

`GET /v1/algorithms/definitions?tenant_id={tenant_id}&algorithm_type={type}&runtime_type={runtime_type}&status={status}&limit=50`

Returns algorithm definitions for a tenant. Optional filters narrow by algorithm type, runtime type, and status.

`GET /v1/algorithms/definitions/{algorithm_id}?tenant_id={tenant_id}`

Returns one algorithm definition.

`POST /v1/algorithms/execution-requests`

Creates or updates a queued algorithm execution request. Required fields are `tenant_id`, `algorithm_id`, and `algorithm_version`; optional fields include `execution_request_id`, `event_ids`, `feature_refs`, `entity_refs`, `window_ref`, `config`, `correlation_id`, and `requested_by`. This records intent only; G106 does not execute algorithms.

`GET /v1/algorithms/execution-requests?tenant_id={tenant_id}&algorithm_id={algorithm_id}&status={status}&correlation_id={correlation_id}&limit=50`

Returns algorithm execution request ledger rows.

`GET /v1/algorithms/execution-requests/{execution_request_id}?tenant_id={tenant_id}`

Returns one algorithm execution request.

`GET /v1/algorithms/results?tenant_id={tenant_id}&algorithm_id={algorithm_id}&execution_request_id={execution_request_id}&result_type={result_type}&severity={severity}&correlation_id={correlation_id}&limit=50`

Returns immutable algorithm result ledger rows. Results include score, confidence, severity, payload, source event ids, feature value ids, evidence refs, and correlation id.

`GET /v1/algorithms/results/{algorithm_result_id}?tenant_id={tenant_id}`

Returns one algorithm result.

### Query Errors

- `400 missing_query`: required idempotency lookup parameters are missing.
- `404 *_not_found`: requested run, raw event, or idempotency record was not found.
- `500 query_failed`: storage query failed.
- `503 storage_unavailable`: gateway was not configured with query storage.
