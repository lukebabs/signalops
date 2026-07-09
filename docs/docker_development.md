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
