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



Run local PostgreSQL migrations:

```bash
make compose-storage-migrate
```
