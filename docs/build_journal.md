# SignalOps Build Journal

This journal is the ongoing record of SignalOps build progress. Entries are
append-only unless correcting factual errors. All timestamps are UTC.

## 2026-07-06T20:02:13Z

Summary:

- Established the build documentation trail before implementation begins.
- Added a documentation standard, build journal, and gate audit log.

Files changed:

- `docs/documentation_standards.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Rationale:

- SignalOps is being built as a standalone time/stream intelligence subsystem.
- Every gate passed needs an auditable timestamp and evidence trail.
- The project needs a continuously updated journal before code scaffolding
  starts.

Verification performed:

- Confirmed working tree was clean before adding the documentation trail.
- Captured current UTC timestamp with `date -u +%Y-%m-%dT%H:%M:%SZ`.

Next step:

- Start Phase 1 Platform Foundation with Go core scaffolding, shared
  contracts, and documentation updates for each gate.


## 2026-07-06T20:11:30Z

Summary:

- Started Phase 1 Platform Foundation.
- Added initial Go core platform scaffold for the SignalOps gateway.
- Added shared contract directories and Python plugin worker directory
  structure.

Files changed:

- `go.mod`
- `cmd/gateway/main.go`
- `internal/api/router.go`
- `internal/config/config.go`
- `contracts/README.md`
- `contracts/events/README.md`
- `contracts/*/.gitkeep`
- `pkg/contracts/README.md`
- `python/signalops_plugins/README.md`
- `python/signalops_plugins/*/.gitkeep`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Rationale:

- Phase 1 requires a stable Go core platform boundary and shared contracts
  before implementing domain-specific adapters or algorithm workers.
- The gateway starts with `GET /healthz` and `GET /readyz` because those are
  required operational endpoints in the infrastructure specification.
- Python plugin directories are present to preserve the Go/Python runtime
  boundary before algorithm implementation begins.

Verification performed:

- Confirmed working tree was clean before scaffolding.
- Captured current UTC timestamp with `date -u +%Y-%m-%dT%H:%M:%SZ`.
- Attempted to check Go availability with `go version`; validation could not
  run because `go` is not installed in this environment.
- Performed file creation through patch review and will perform readback before
  commit.

Next step:

- Install or provide Go in the build environment, then run `go test ./...`.
- Add contract schema files for `RawSignalEvent` and `NormalizedSignalEvent`.

## 2026-07-06T20:18:13Z

Summary:

- Added Docker-first development and validation tooling.
- Verified the Go scaffold through Docker instead of relying on host Go.
- Built and smoke-tested the gateway container.

Files changed:

- `.dockerignore`
- `Dockerfile`
- `Makefile`
- `docs/docker_development.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Rationale:

- Host Go is not installed in this environment.
- Docker is available and provides a reproducible toolchain boundary for Go
  tests, builds, and runtime smoke tests.
- The project should standardize on Docker validation early so future gates do
  not depend on workstation-specific tools.

Verification performed:

- Confirmed Docker availability with `docker --version`.
- Ran Dockerized Go tests:
  `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./...`
- Built the gateway image:
  `docker build --target gateway -t signalops-gateway:local .`
- Ran the gateway image and verified:
  `GET /healthz` returned `{"service":"signalops-gateway","status":"ok",...}`.
- Verified:
  `GET /readyz` returned `{"service":"signalops-gateway","status":"ready",...}`.
- Stopped the smoke-test container.

Next step:

- Add versioned JSON Schema event contracts for `RawSignalEvent`,
  `NormalizedSignalEvent`, and `Signal`.
- Use Dockerized validation for schema parsing and Go tests.

## 2026-07-06T20:26:54Z

Summary:

- Added versioned JSON Schema contracts for the first cross-runtime event payloads.
- Added Dockerized schema validation tooling.

Files changed:

- `contracts/events/common.defs.v1.schema.json`
- `contracts/events/raw_signal_event.v1.schema.json`
- `contracts/events/normalized_signal_event.v1.schema.json`
- `contracts/events/signal.v1.schema.json`
- `contracts/events/README.md`
- `scripts/validate_json_schemas.py`
- `Makefile`
- `docs/docker_development.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Rationale:

- Phase 1 requires stable shared contracts before source adapters, broker workers,
  or Python detector workers are implemented.
- The schemas define the domain-neutral fields required for source domains,
  adapters, ingestion modes, datasets, temporal semantics, correlation, and
  idempotency.

Verification performed:

- Captured current UTC timestamp with `date -u +%Y-%m-%dT%H:%M:%SZ`.
- Validated schema JSON metadata through Docker:
  `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace python:3.12-slim python scripts/validate_json_schemas.py`.
- Ran Dockerized Go tests:
  `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./...`.

Next step:

- Add Go contract types that mirror these schemas.
- Add package-level tests for required field serialization and basic API behavior.

## 2026-07-06T20:31:01Z

Summary:

- Added Go contract types for the first shared SignalOps event contracts.
- Added JSON serialization tests for `RawSignalEvent` and `Signal`.

Files changed:

- `pkg/contracts/events.go`
- `pkg/contracts/events_test.go`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Rationale:

- The Go core platform needs typed representations of cross-runtime contracts
  before source adapters, broker publishers, and persistence code are added.
- The structs mirror the v1 JSON Schema names for domain-neutral events and
  preserve the Go/Python boundary through JSON field tags.

Verification performed:

- Captured current UTC timestamp with `date -u +%Y-%m-%dT%H:%M:%SZ`.
- Ran Dockerized Go tests:
  `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./...`.
- Ran Dockerized schema validation:
  `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace python:3.12-slim python scripts/validate_json_schemas.py`.

Next step:

- Add remaining shared output schemas and Go types for `EventArtifact`,
  `GraphMutationProposal`, and `InsightCandidate`.

## 2026-07-06T20:39:12Z

Summary:

- Documented SignalOps internal communication protocol decision.
- Established Kafka/Redpanda as the durable default and gRPC as the bounded
  fast-sync exception.

Files changed:

- `docs/Syncratic_SignalOps_Processing_Specification.md`
- `docs/signalops_extended_capabilities_spec.md`
- `contracts/protocols.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Rationale:

- Algorithm workers and Go infrastructure need an explicit communication rule
  before broker interfaces or plugin runners are implemented.
- Durable, replayable, retryable, and auditable work must use brokered events.
- Fast synchronous calls may use gRPC only when bounded and non-authoritative
  until the Go core persists or republishes the result.

Verification performed:

- Captured current UTC timestamp with `date -u +%Y-%m-%dT%H:%M:%SZ`.
- Read back the updated processing and extended capability specs.
- Ran Dockerized Go tests:
  `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./...`.
- Ran Dockerized schema validation:
  `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace python:3.12-slim python scripts/validate_json_schemas.py`.

Next step:

- Add durable broker topic constants and interfaces for algorithm jobs/results.
- Keep gRPC contracts as future fast-sync scope until a concrete use case needs
  them.

## 2026-07-06T20:45:00Z

Summary:

- Added local Docker Compose deployment for SignalOps with Redpanda as the
  default Kafka-compatible broker.
- Added topic bootstrap job and deployment documentation.
- Started and validated the local deployment stack.

Files changed:

- `.dockerignore`
- `.env.example`
- `Makefile`
- `compose.yaml`
- `deploy/docker/redpanda/create-topics.sh`
- `docs/deployment.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Rationale:

- SignalOps needs a reproducible local deployment before broker interfaces and
  workers are implemented.
- Redpanda is the default broker runtime while preserving Kafka-compatible
  protocol boundaries.
- Default topics should be created deterministically for the durable path.

Verification performed:

- Captured current UTC timestamp with `date -u +%Y-%m-%dT%H:%M:%SZ`.
- Confirmed Docker Compose availability with `docker compose version`.
- Validated compose syntax with `docker compose config --quiet`.
- Ran Dockerized Go tests with `go test ./...`.
- Ran Dockerized schema validation with `scripts/validate_json_schemas.py`.
- Started the local stack with `docker compose up -d --build`.
- Fixed Redpanda healthcheck after `rpk cluster health --brokers` was rejected
  by the bundled `rpk` version.
- Fixed gateway host port after local port `8080` was already allocated;
  gateway now maps host port `18000` to container port `8080`.
- Verified `GET /healthz` returned status `ok` on `http://127.0.0.1:18000`.
- Verified `GET /readyz` returned status `ready` on `http://127.0.0.1:18000`.
- Verified default Redpanda topics exist with 3 partitions and 1 replica.

Next step:

- Add broker topic constants and a Kafka-compatible broker abstraction in Go.
- Keep the local stack available for broker integration tests.


## 2026-07-06T20:56:06Z

Summary:

- Added the initial Go broker boundary for durable SignalOps messaging.
- Added deterministic topic constants matching the Redpanda topic bootstrap.
- Extended runtime config with broker provider, broker addresses, and environment.
- Documented the broker boundary in implementation and protocol docs.

Files changed:

- `pkg/broker/broker.go`
- `pkg/broker/topics.go`
- `pkg/broker/topics_test.go`
- `pkg/broker/README.md`
- `internal/config/config.go`
- `internal/config/config_test.go`
- `contracts/protocols.md`
- `docs/deployment.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Rationale:

- SignalOps needs stable durable messaging interfaces before attaching a
  concrete Redpanda/Kafka client.
- Application code should depend on SignalOps-owned interfaces instead of
  leaking broker client types across the codebase.
- Topic constants must match the local deployment bootstrap so future
  publishers and consumers cannot drift from runtime infrastructure.

Verification performed:

- Formatted Go code with Dockerized `gofmt`.
- Ran Dockerized `go test ./...`; all packages passed.
- Ran Dockerized schema validation with `scripts/validate_json_schemas.py`; all schemas passed.
- Confirmed the running Redpanda stack still exposes the expected durable topics.

Next step:

- Add a concrete Kafka-compatible Redpanda client implementation behind the
  `pkg/broker` interfaces.
- Add integration tests against the local Redpanda stack.


## 2026-07-06T23:22:46Z

Summary:

- Added a concrete Kafka-compatible broker client for Redpanda using franz-go.
- Implemented synchronous publish acknowledgement, manual consumer groups,
  buffered consumption, and explicit offset commits behind `pkg/broker`.
- Added broker integration testing against the local Redpanda compose stack.
- Added a repeatable Dockerized Makefile target for the live broker test.

Files changed:

- `go.mod`
- `go.sum`
- `Makefile`
- `internal/broker/kafka/client.go`
- `internal/broker/kafka/client_test.go`
- `internal/broker/kafka/client_integration_test.go`
- `internal/broker/kafka/README.md`
- `docs/deployment.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Rationale:

- The previous broker gate defined interfaces only; SignalOps now needs a real
  durable messaging implementation to prove Redpanda publish, consume, and
  commit behavior.
- The concrete client is kept under `internal/` so application code depends on
  SignalOps-owned interfaces rather than franz-go types.
- `github.com/twmb/franz-go` is pinned to `v1.18.1` because newer versions in
  the tested series require Go versions newer than the current Go 1.22 toolchain.

Verification performed:

- Formatted Kafka client code with Dockerized `gofmt`.
- Normalized module metadata with Dockerized `go mod tidy`.
- Ran Dockerized `go test ./...`; all regular packages passed.
- Ran Dockerized schema validation with `scripts/validate_json_schemas.py`; all schemas passed.
- Ran live Redpanda integration test with Docker host networking:
  `docker run --rm --network host -e SIGNALOPS_BROKER_INTEGRATION=1 -e SIGNALOPS_BROKER_BROKERS=localhost:19092 -e SIGNALOPS_ENV=local -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./internal/broker/kafka -run TestPublishConsumeCommitAgainstRedpanda -count=1 -v`.
- Verified the repeatable Makefile wrapper with `make docker-test-broker-integration`.

Issue found and resolved:

- A bridge-networked test container timed out because Redpanda advertises
  `localhost:19092`; the repeatable integration target uses Docker host
  networking so the advertised listener resolves correctly.

Next step:

- Wire the gateway ingestion path to publish accepted raw events to
  `signalops.<env>.raw.v1` through the broker client.


## 2026-07-07T00:01:22Z

Summary:

- Added gateway raw event ingestion at `POST /v1/events/raw`.
- Wired the gateway to the concrete Kafka-compatible broker client.
- Published accepted raw events to `signalops.<environment>.raw.v1` with broker
  acknowledgement response details.
- Added API tests and live HTTP-to-Redpanda verification.
- Fixed the Dockerfile to copy `go.sum` before image build tests.

Files changed:

- `cmd/gateway/main.go`
- `internal/api/router.go`
- `internal/api/router_test.go`
- `Dockerfile`
- `docs/api.md`
- `docs/deployment.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Rationale:

- G008 proved the broker client directly; G009 proves the first application path
  from gateway HTTP ingestion into the durable Redpanda raw topic.
- The endpoint keeps the incoming JSON object unchanged as the broker value and
  adds SignalOps metadata in broker headers for downstream consumers.
- The gateway now uses `pkg/broker` abstractions while the concrete Redpanda
  client remains under `internal/broker/kafka`.

Verification performed:

- Formatted gateway/API code with Dockerized `gofmt`.
- Ran Dockerized `go test ./...`; all packages passed.
- Ran Dockerized schema validation with `scripts/validate_json_schemas.py`; all schemas passed.
- Rebuilt and redeployed the gateway with `docker compose up -d --build gateway`.
- Verified `GET /healthz` and `GET /readyz` on `http://127.0.0.1:18000`.
- Posted a live raw event to `POST /v1/events/raw` and received `202 Accepted`
  with topic `signalops.local.raw.v1`, partition `0`, and offset `1`.
- Consumed `signalops.local.raw.v1` partition `0` offset `1` with `rpk` and
  verified key, payload, and metadata headers.

Issue found and resolved:

- The Dockerfile copied `go.mod` but not `go.sum`, which failed image build
  tests after adding franz-go. The Dockerfile now copies both files before
  running `go test ./...` in the build stage.

Next step:

- Add broker-backed readiness checks and an ingestion integration test that
  exercises HTTP publish and consume automatically.
- Begin G010 Python worker skeleton for consuming durable raw or normalized
  work from Redpanda.
