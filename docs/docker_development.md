# SignalOps Docker Development

SignalOps uses Docker as the default local toolchain boundary. This keeps the
host environment small and makes validation reproducible when Go or Python
tooling is not installed locally.

## Requirements

- Docker 29 or newer is available in the current environment.
- The Go toolchain is provided by the `golang:1.22-bookworm` container image.

## Commands

Run Go tests:

```bash
make docker-test
```

Build the gateway image:

```bash
make docker-build
```

Validate JSON schemas:

```bash
make docker-validate-schemas
```

Run Python worker tests:

```bash
make docker-test-python
```

Open a Go toolchain shell:

```bash
make docker-shell
```

## Notes

- Host Go is optional.
- Docker may need network access the first time it pulls the Go image.
- The production gateway image is built from `Dockerfile` target `gateway`.
- The Massive one-shot image is built from target `massive-puller`; the repeatable scheduler image is built from target `massive-scheduler`.
- The gateway binary listens on `SIGNALOPS_HTTP_ADDR`, defaulting to `:8080`.



Run local relational PostgreSQL migrations:

```bash
make compose-storage-migrate
```

Run local TimescaleDB temporal migrations:

```bash
make compose-temporal-migrate
```

Backfill relational raw, normalized, and signal ledgers into TimescaleDB:

```bash
make compose-temporal-backfill
```

The backfill is idempotent. It uses conflict-aware upserts, so it can be rerun after partial failures or after live services have started writing new temporal rows.

## Frontend (Operational Dashboard)

The operational UI lives under `web/` (Vite + React + TypeScript). Run it
against the local gateway:

```bash
cd web
npm install
npm run dev      # http://localhost:5173/
```

The Vite dev server proxies `/healthz`, `/readyz`, and `/v1` to the gateway on
`http://localhost:18000`, so no CORS configuration is required. See
`web/README.md` and `docs/frontend_implementation_spec.md` for details.

Build and run the production-style web container through Compose:

```bash
docker compose build web
docker compose up -d web
```

The Compose web service listens on `http://localhost:15173/` and uses nginx to
serve the built frontend and proxy `/healthz`, `/readyz`, and `/v1` to the
`gateway` service. This keeps browser requests same-origin, including the SSE
stream at `/v1/streams/dashboard`.

## Replay Worker

Build the replay worker image:

```bash
docker compose -f compose.yaml -f compose.traefik.yaml build replay-worker
```

Run one queued replay job and exit, capped to one record for validation:

```bash
docker compose --profile replay run --rm \
  -e SIGNALOPS_REPLAY_ONESHOT=true \
  -e SIGNALOPS_REPLAY_MAX_RECORDS=1 \
  -e SIGNALOPS_REPLAY_BATCH_SIZE=1 \
  replay-worker
```

The replay worker requires both `SIGNALOPS_DATABASE_URL` and
`SIGNALOPS_TEMPORAL_DATABASE_URL`. It claims queued PostgreSQL replay jobs, reads
matching Timescale rows in bounded batches, republishes through Redpanda with
configurable publish retries, detects cancellation between batches, and updates
the job status/result metadata.

Replay worker controls:

- `SIGNALOPS_REPLAY_WORKER_ID`: worker identifier written into claim metadata.
- `SIGNALOPS_REPLAY_ONESHOT`: when true, process at most one queued job and exit.
- `SIGNALOPS_REPLAY_MAX_RECORDS`: maximum source records to replay for a job.
- `SIGNALOPS_REPLAY_BATCH_SIZE`: temporal rows read per batch, capped at 200.
- `SIGNALOPS_REPLAY_PUBLISH_MAX_ATTEMPTS`: broker publish attempts per record.
- `SIGNALOPS_REPLAY_POLL_INTERVAL`: wait duration between empty queue polls.
