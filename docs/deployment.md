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
- `postgres`: relational system-of-record store on `localhost:15432` for scheduler runs, provider usage, idempotency, catalogs, lifecycle state, and operational metadata.
- `timescaledb`: temporal/replay store on `localhost:15433` for raw events, normalized events, signal observations, feature/window data, and market-data history.
- `gateway`: SignalOps gateway on `http://localhost:18000`.
- `normalizer`: Go worker that consumes `signalops.local.raw.v1`, publishes `signalops.local.normalized.v1`, and persists normalized lineage.
- `raw-worker`: Python algorithm worker that consumes `signalops.local.normalized.v1`.
- `signal-persister`: Go worker that validates and persists Python-emitted `signalops.local.signal.v1` events.
- `web`: production-style nginx container for the SignalOps operational UI on `http://localhost:15173`.

## Local Ports

- PostgreSQL: `15432` host port mapped to container port `5432`
- TimescaleDB: `15433` host port mapped to container port `5432`
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
- `SIGNALOPS_DATABASE_URL` for relational PostgreSQL
- `SIGNALOPS_TEMPORAL_DATABASE_URL` for optional separate TimescaleDB temporal storage

Broker configuration is loaded now; concrete broker clients will be wired in a
later gate. The shared Go broker boundary and topic constants live under
`pkg/broker`.

## Storage Roles

SignalOps keeps PostgreSQL and TimescaleDB as separate logical roles. PostgreSQL remains the relational control-plane/system-of-record store. TimescaleDB is PostgreSQL plus the Timescale extension and is used for replayable temporal/event-plane data.

In local Compose these roles run as two services:

- `postgres` / `SIGNALOPS_DATABASE_URL`: scheduler runs, provider usage, idempotency, catalogs, alert/insight lifecycle state, and operational metadata.
- `timescaledb` / `SIGNALOPS_TEMPORAL_DATABASE_URL`: `raw_event_ledger`, `normalized_event_ledger`, `signal_ledger`, and market-data temporal history hypertables.

Apply migrations separately:

```bash
make compose-storage-migrate
make compose-temporal-migrate
```

If `SIGNALOPS_TEMPORAL_DATABASE_URL` is empty, services fall back to the relational DSN for compatibility with single-Postgres deployments and existing tests.

For an existing deployment, apply temporal migrations and backfill or replay existing raw/normalized/signal rows before redeploying services with `SIGNALOPS_TEMPORAL_DATABASE_URL` enabled. Without that step, new temporal writes go to TimescaleDB but historical rows that only exist in relational PostgreSQL will not appear in temporal-backed query endpoints.

Run the idempotent relational-to-Timescale backfill:

```bash
make compose-temporal-backfill
```

The backfill copies `raw_event_ledger`, `normalized_event_ledger`, and `signal_ledger` from relational PostgreSQL into TimescaleDB with conflict-aware upserts. It is safe to rerun after a partial attempt or after live cutover; rows already present in TimescaleDB are updated in place.

After a gateway container is recreated, also recreate or restart the `web` container before validating the public route. The production nginx container resolves `gateway` at startup, so a stale upstream IP can produce `502` even when the gateway is healthy on `localhost:18000`.

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

## Public TLS through Syncratic Traefik

SignalOps can be exposed through the parent Syncratic core Traefik edge without
running a second reverse proxy. The SignalOps overlay `compose.traefik.yaml`
attaches the `web` service to the external Traefik network and adds Docker labels
for the existing `websecure` entrypoint and `letsencrypt` certificate resolver.

Required SignalOps env values:

```bash
SIGNALOPS_PUBLIC_HOST=signalops.syncratic.io
TRAEFIK_NETWORK=syncratic-core_syncratic_net
COMPOSE_FILE=compose.yaml:compose.traefik.yaml
```

`COMPOSE_FILE` is intentional for the public deployment. It makes a plain
`docker compose up -d web` render the Traefik overlay by default, so rebuilds do
not silently recreate `web` without router labels and produce a public 404.

The parent Syncratic core Traefik service must already be running and configured
with its Let's Encrypt resolver credentials, including `LETSENCRYPT_EMAIL`,
`GODADDY_API_KEY`, and `GODADDY_API_SECRET` in the parent stack. DNS for
`SIGNALOPS_PUBLIC_HOST` must point at the same public edge used by Syncratic core.

Start SignalOps with the edge overlay:

```bash
make deploy-web
# equivalent to:
# VITE_SIGNALOPS_AUTH_ENABLED=true docker compose -f compose.yaml -f compose.traefik.yaml up -d --build web
```

`make deploy-web` rebuilds the `web` image **with frontend auth enabled** and
**with the Traefik overlay applied** in one step. The deployment `.env` also sets
`COMPOSE_FILE=compose.yaml:compose.traefik.yaml` so plain Compose operations keep
Traefik labels attached. If `COMPOSE_FILE` is absent, a bare
`docker compose up -d --build web` can recreate `web` without `traefik.*` labels,
which **404s the public host**. Always keep `COMPOSE_FILE` set for this public
deployment and prefer `make deploy-web` when rebuilding the public web image.

Only the `web` service is exposed publicly. The web nginx container proxies API
and SSE paths to the internal gateway, preserving same-origin browser behavior:

- `/healthz`
- `/readyz`
- `/v1/*`

Validate after DNS and certificate issuance:

```bash
curl -fsS https://signalops.syncratic.io/
curl -fsS https://signalops.syncratic.io/healthz
curl -fsS 'https://signalops.syncratic.io/v1/alerts?tenant_id=tenant-local&limit=1'
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

Both profiles default to dry-run mode. Production weekday collection is driven by the installed user timer at **18:01:55 America/New_York** and invokes `scripts/marketops_daily_postclose.sh`. That workflow uses the durable `marketops_equity_reconciliation_tasks` queue, claims missing symbols sequentially, and retries provider failures before downstream processing. A run fails closed unless all 50 active-universe symbols normalize for the session. Configure bounded recovery with `MARKETOPS_DAILY_RECONCILIATION_DEADLINE`, `MARKETOPS_DAILY_RECONCILIATION_MAX_ATTEMPTS`, and `MARKETOPS_DAILY_RECONCILIATION_BACKOFFS`; write mode additionally requires `--write` and reconciliation acknowledgement. The scheduler profile remains useful for isolated validation and defaults to dry-run.

The MarketOps Assets view reads the persisted `marketops_asset_quote_cache`, so the first screen render never waits on Massive. The 15-minute intraday monitor overwrites each active asset’s cached quote during the regular (09:30–16:00 ET) and extended (16:00–20:00 ET) sessions. The browser refetches the cache every 15 minutes while retaining its last response during a refetch. Outside those sessions the monitor exits before requesting Massive, and the API returns the most recent completed value labelled EOD. The Updated column distinguishes EOD, regular, and extended-session context. Native hover help explains the displayed change, entitlement delay, and the asset’s 52-week range position. Stocks Starter provides 15-minute-delayed aggregates; it does not authorize real-time trade, quote, or snapshot endpoints. Display data is read-only and does not enter EOD evidence, reconciliation, or signal pipelines.

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
in the `postgres-data` Docker volume. Local Compose also includes TimescaleDB for
replayable temporal/event-plane storage on host port `15433`, backed by the
`timescaledb-data` Docker volume.

Run relational and temporal migrations:

```bash
make compose-storage-migrate
make compose-temporal-migrate
```

The relational migrations create scheduler run audit, provider usage,
idempotency, catalogs, alert/insight lifecycle state, and other control-plane
tables. Temporal migrations create Timescale hypertables for raw events,
normalized events, signal observations, and market-data history. Catalog, rules,
sources, pipelines, lifecycle state, and idempotency records remain ordinary
relational tables unless measured workload requires a different shape.


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
`review`, `dismiss`, and `archive`. When `SIGNALOPS_AUTH_ENABLED=true`, the gateway validates the
Bearer JWT and derives operator identity from `preferred_username`, then `email`, then `sub`; token
identity overrides `X-SignalOps-Actor` and body `actor`. When auth is disabled for local/frontend
transition work, the legacy placeholder order remains `X-SignalOps-Actor`, body `actor`, then
`operator-local`.

## G052 gateway authentication

Gateway authentication is controlled by `SIGNALOPS_AUTH_ENABLED`. Health endpoints remain public:

- `GET /healthz`
- `GET /readyz`

When enabled, protected `/v1/*` routes require `Authorization: Bearer <jwt>` and validate:

- issuer: `SIGNALOPS_AUTH_ISSUER`
- signature: `SIGNALOPS_AUTH_JWKS_URL`
- audience: `SIGNALOPS_AUTH_AUDIENCE` (`signalops-api`)
- expiry and not-before timestamps
- tenant claim: `tenant_id`

Role enforcement:

- read/protected `/v1/*` routes require one of `signalops:viewer`, `signalops:operator`, or `signalops:admin`.
- alert and insight lifecycle POST routes require `signalops:operator` or `signalops:admin`.
- explicit `tenant_id` query values or `/v1/tenants/{tenant_id}/...` paths must match the token `tenant_id` claim.

Keep `SIGNALOPS_AUTH_ENABLED=false` until the frontend login/token attachment gate is deployed, otherwise the current browser UI will receive `401` responses for protected `/v1/*` calls.
### Intraday asset conditions

Assets now show a persisted, asset-specific intraday condition stream. `marketops-intraday-monitor` records one snapshot per active universe asset; it derives session movement and 52-week-range conditions from entitled stock aggregates, while end-of-day Market State hypotheses remain authoritative research records. The installed `signalops-marketops-intraday.timer` runs weekdays at 09:30, 09:45, then every 15 minutes through 20:00 ET. The worker independently rejects all times outside regular and extended sessions, preventing late persistent timer catch-up from calling Massive. Use `scripts/marketops_intraday_monitor.sh` for an on-demand run and `scripts/install_marketops_intraday_user_timer.sh` to install the user timer.

### Post-close algorithm corroboration

The governed post-close workflow now materializes the latest `options_distribution_daily` feature for each active asset, then runs bounded cross-sectional z-score observations for `daily_return_pct` and usable `call_put_open_interest_ratio`. Results are immutable research corroboration records, displayed beside asset evidence; they never alter Market State hypothesis eligibility, triggering, confidence, or lifecycle. Failures are logged and do not block canonical state processing.

### Quantitative asset series

The expanded MarketOps Asset view reads `GET /v1/tenants/{tenant_id}/marketops/assets/{symbol}/quantitative-series`. The read-only chart aligns persisted EOD closes with call/put open interest and a presentation-derived put/call volume ratio, preserving missing observations as gaps. The stored distribution field is call/put, so the view inverts it solely for display. Put/call volume below 1.0 is labelled bullish (calls elevated); above 1.0 is labelled bearish (puts elevated); exactly 1.0 is neutral. Its `10_trade_days`, `30_trade_days`, and `60_trade_days` views draw the neutral baseline and mark only medium-or-higher platform observations; marker color reflects persisted independent adjudication where available. Ratios are descriptive positioning/flow evidence, not trading recommendations.

### Focused quantitative corroboration

The selected-asset view reads `GET /v1/tenants/{tenant_id}/marketops/assets/{symbol}/algorithm-observations`. Its primary corroboration panel shows at most one usable `zscore_anomaly_v1` observation for each of the latest three payload observation dates. Options-ratio z-scores with `partial_zero`, `all_zero`, or `denominator_zero` quality cannot be selected. Raw platform results remain preserved and are available in the read-only **Algorithm Evidence** tab, grouped by observation date and capped to the five newest events. Its put/call volume entries use the same below-1 bullish and above-1 bearish interpretation as the chart. This curation never changes algorithm execution, persisted evidence, hypotheses, recommendations, or lifecycle state.

### Browser session inactivity and renewal

The browser session is inactivity-based, not a fixed access-token lifetime. `VITE_SIGNALOPS_AUTH_IDLE_TIMEOUT_MINUTES` defaults to `30`; pointer, keyboard, scroll, touch, and browser-focus activity keep the local session active. While it remains active, the SPA renews a token that is within `VITE_SIGNALOPS_AUTH_RENEW_BEFORE_EXPIRY_SECONDS` (default `300`) of expiry. After the configured inactivity period the SPA clears its local user/token and requires a new sign-in flow.

`signalops-web` uses refresh-token renewal when the provider grants a refresh token; otherwise it falls back to the OIDC iframe callback at `${public_origin}/auth/silent-renew`. Register that exact redirect URI for the `signalops-web` Keycloak client, in addition to `${public_origin}/auth/callback` and the public-origin logout redirect. The IdP's own maximum session/idle limits remain authoritative and can still require sign-in when they are stricter than the browser setting.

### Risk/Reward temporal research signal

`signalops.algorithms.risk_reward_temporal_v1` runs after Market State materialization in the post-close corroboration workflow. It evaluates persisted technical inputs over a bounded historical window and writes immutable research-only algorithm results. Put/call volume is normalized as puts divided by calls and shown solely as speculative corroboration; it cannot determine technical direction. The selected-asset algorithm-observations response exposes the latest result and up to 60 historical points under `risk_reward`.
