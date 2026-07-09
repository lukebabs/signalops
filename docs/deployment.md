# SignalOps Deployment

SignalOps uses Redpanda as the default broker runtime while keeping the
implementation Kafka API compatible.

## Local Docker Compose

Start the local stack:

```bash
make compose-up
```

Validate the compose file:

```bash
make compose-validate
```

Show services:

```bash
make compose-ps
```

Run the broker integration test against the local Redpanda listener:

```bash
make docker-test-broker-integration
```

Stop the stack:

```bash
make compose-down
```

## Services

- `redpanda`: default Kafka-compatible broker.
- `redpanda-console`: local broker UI on `http://localhost:18080`.
- `topic-bootstrap`: one-shot topic creation job.
- `gateway`: SignalOps gateway on `http://localhost:18000`.
- `normalizer`: Go worker that consumes `signalops.local.raw.v1`, publishes `signalops.local.normalized.v1`, and persists normalized lineage.
- `raw-worker`: Python algorithm worker that consumes `signalops.local.normalized.v1`.
- `signal-persister`: Go worker that validates and persists Python-emitted `signalops.local.signal.v1` events.
- `web`: production-style nginx container for the SignalOps operational UI on `http://localhost:15173`.

## Local Ports

- Gateway: `18000` host port mapped to container port `8080`
- Web UI: `15173` host port mapped to container port `8080`
- Redpanda Kafka external listener: `19092`
- Redpanda Schema Registry: `18081`
- Redpanda HTTP Proxy: `18082`
- Redpanda Admin/metrics: `19644`
- Redpanda Console: `18080`

## Default Topics

- `signalops.local.raw.v1`
- `signalops.local.normalized.v1`
- `signalops.local.signal.v1`
- `signalops.local.artifact.v1`
- `signalops.local.graph_mutation.v1`
- `signalops.local.insight_candidate.v1`
- `signalops.local.retry.algorithm.v1`
- `signalops.local.dlq.algorithm.v1`

## Broker Decision

- Redpanda is the default local and production broker target.
- SignalOps code must depend on Kafka-compatible broker abstractions.
- Apache Kafka remains a deployment alternative for environments that already
  standardize on Kafka.

## Environment

Copy `.env.example` when local overrides are needed.

```bash
cp .env.example .env
```

Runtime config currently reads:

- `SIGNALOPS_HTTP_ADDR`
- `SIGNALOPS_BROKER_PROVIDER`
- `SIGNALOPS_BROKER_BROKERS`
- `SIGNALOPS_ENV`

Broker configuration is loaded now; concrete broker clients will be wired in a
later gate. The shared Go broker boundary and topic constants live under
`pkg/broker`.

## Web UI

The Compose `web` service builds the Vite frontend under `web/` and serves the
static assets with nginx. The same nginx container proxies `/healthz`, `/readyz`,
and `/v1` to the internal `gateway:8080` service, so browser calls remain
same-origin and the dashboard SSE stream works without gateway CORS headers.

```bash
docker compose build web
docker compose up -d web
curl -fsS http://localhost:15173/healthz
curl -N --max-time 3 'http://localhost:15173/v1/streams/dashboard?channels=health,heartbeat'
```

## Gateway API

The gateway exposes raw event ingestion at `POST /v1/events/raw`. The endpoint
publishes accepted JSON objects to `signalops.<environment>.raw.v1`, atomically persists the
acknowledged broker coordinates to the raw ledger and idempotency store, and then returns
acceptance details. The route requires both broker and PostgreSQL connectivity. See `docs/api.md`
for the required persistence envelope, request mapping, and failure semantics.

## Broker Client

The concrete Kafka-compatible Go client lives under `internal/broker/kafka`.
It uses the shared `pkg/broker` interfaces and preserves SignalOps metadata in
Kafka record headers:

- `correlation_id`
- `causation_id`
- `trace_id`

Dockerized integration tests use host networking because the local Redpanda
compose listener advertises `localhost:19092`.

## Python Worker

The Go normalizer consumes raw events, publishes and persists the canonical normalized contract,
and commits raw offsets only after both durable steps. The Python worker runs from
`python/signalops_workers` and consumes normalized events from Redpanda. It invokes the configured detector, validates emitted signals
against `contracts/events/signal.v1.schema.json`, publishes valid signals to
`signalops.local.signal.v1`, publishes retryable failures to
`signalops.local.retry.algorithm.v1`, and publishes invalid or non-retryable
failures to `signalops.local.dlq.algorithm.v1`. Source offsets are committed
only after successful processing, signal publication, retry publication, or DLQ
publication is acknowledged.

## Retry Replayer

The retry replayer runs from `python/signalops_workers.retry_replay_main` and is
available as the optional Compose service `retry-replayer` under the
`retry-replay` profile. It consumes retry records, reconstructs the original
source message, and republishes it to `signalops.local.raw.v1` while retry
attempts remain. Exhausted retries are routed to
`signalops.local.dlq.algorithm.v1` with the original payload and source lineage
preserved.

See `docs/python_worker.md` for worker and retry replayer configuration and
validation commands.


## Massive Scheduled Market Data

The Massive adapter has two optional Compose profiles:

- `massive-pull`: one-shot puller for manual dry-run or publish validation.
- `massive-schedule`: repeatable scheduler around the same pull runner.

Both profiles default to dry-run mode. The scheduler defaults to running once at
startup and then every `24h` until stopped. Use `SIGNALOPS_MASSIVE_SCHEDULE_MAX_RUNS`
or the `--max-runs` flag for bounded validation runs.

Example one-run scheduler dry-run:

```bash
docker compose --profile massive-schedule run --rm massive-scheduler \
  --max-runs 1 \
  --max-companies 1 \
  --datasets equity \
  --dry-run=true
```

Publishing requires `--dry-run=false` and a healthy broker/topic bootstrap path.
Secrets are loaded from the ignored `.env` file through supported Massive API key
environment variables.


## Stream Catalog

Local storage migration `000002_catalog_sources` creates the first source catalog
table and seeds the local Massive source as `tenant-local/src-massive`. Migration
`000003_catalog_pipelines` creates the first pipeline catalog table and seeds
`tenant-local/pipeline-massive-raw-ingest` for the scheduled Massive raw ingest
path. Migration `000004_catalog_rules` creates the first rules catalog table and
seeds `tenant-local/rule-marketdata-eod-price-quality` for Massive EOD equity
price quality checks. The gateway exposes these through:

```bash
curl -fsS 'http://localhost:18000/v1/tenants/tenant-local/catalog/sources?limit=10'
curl -fsS 'http://localhost:18000/v1/tenants/tenant-local/catalog/pipelines?limit=10'
curl -fsS 'http://localhost:18000/v1/tenants/tenant-local/catalog/rules?limit=10'
```

The catalog is intentionally read-only at this stage. Source, pipeline, and rule
registration and management APIs remain future scope.

## Storage

Local Compose includes PostgreSQL for operational metadata and audit storage.
The `postgres` service exposes the database on host port `15432` and stores data
in the `postgres-data` Docker volume.

Run migrations:

```bash
make compose-storage-migrate
```

The first migration creates scheduler run audit, provider usage, idempotency,
raw event ledger, equity EOD, and option contract snapshot tables. TimescaleDB
hypertable conversion remains future scope after the base persistence paths are
proven.

Maturity note: TimescaleDB is expected to become essential as SignalOps moves from
small operational validation into sustained time-series and stream workloads. The
current Compose deployment intentionally uses plain `postgres:16-alpine`; a later
storage gate should replace it with a TimescaleDB PostgreSQL-compatible image,
add `CREATE EXTENSION timescaledb`, and promote high-volume temporal ledgers such
as raw events, normalized events, signals, provider usage, and operational metrics
to hypertables. Catalog, rules, sources, pipelines, and idempotency records should
remain ordinary relational tables unless measured workload requires otherwise.


Scheduler persistence is enabled when `SIGNALOPS_DATABASE_URL` is set. The
local `massive-scheduler` Compose service points at the `postgres` service and
writes scheduler run summaries plus provider usage rows after each run.

Publish-side persistence is also enabled by `SIGNALOPS_DATABASE_URL`. When the
Massive puller or scheduler publishes raw events, successful broker acknowledgements
are written to `raw_event_ledger` and `idempotency_records` with topic, partition,
offset, payload hash, and event metadata. Dry-runs do not write raw event ledger
or idempotency rows because no broker publication occurs.


## G047 alert and insight lifecycle

The existing `signal-persister` service now persists derived alert and insight lifecycle rows in the
same database transaction as each validated `signal.v1` record. No additional worker is required for
G047. The service commits the `signalops.<environment>.signal.v1` source offset only after signal,
alert, and insight persistence succeeds.

Current derivation creates one active insight for every valid signal and one open alert for
`medium`, `high`, or `critical` signals. Low/info signals remain insights only.

## G049 alert and insight lifecycle mutations

The gateway exposes lifecycle mutation endpoints for existing alert and insight ledger rows. No new
migration, worker, topic, or Compose service is required for G049; the endpoints update the columns
created in `000007_alert_insight_lifecycle` and merge a compact `metadata.lifecycle` audit object.

Supported alert actions are `acknowledge`, `resolve`, and `suppress`. Supported insight actions are
`review`, `dismiss`, and `archive`. Operator identity is currently an explicit placeholder supplied by
`X-SignalOps-Actor` or body `actor`, defaulting to `operator-local` until formal authentication is
introduced.
