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


## 2026-07-07T00:25:36Z

Summary:

- Added the first runnable Python worker runtime for SignalOps.
- Added a `raw-worker` Docker Compose service that consumes
  `signalops.local.raw.v1` from Redpanda.
- Added Python unit tests, worker Docker image, and worker documentation.
- Deployed the long-running worker service locally and verified consumer group
  stability with zero lag.

Files changed:

- `.dockerignore`
- `Makefile`
- `compose.yaml`
- `deploy/docker/python-worker/Dockerfile`
- `python/requirements-worker.txt`
- `python/signalops_workers/__init__.py`
- `python/signalops_workers/__main__.py`
- `python/signalops_workers/broker.py`
- `python/signalops_workers/config.py`
- `python/signalops_workers/worker.py`
- `python/tests/test_config.py`
- `python/tests/test_worker.py`
- `python/signalops_plugins/README.md`
- `docs/docker_development.md`
- `docs/deployment.md`
- `docs/python_worker.md`
- `docs/deployment.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Rationale:

- G009 proved HTTP-to-Redpanda ingestion; G010 proves the Python side of the
  durable boundary can consume and commit raw events without direct Go/Python
  coupling.
- The worker intentionally performs no detector logic yet. It provides the
  runtime seam for later algorithm plugins while keeping the first deployment
  auditable and low risk.
- Invalid raw records are logged and committed in this skeleton so historical
  poison records do not crash-loop the worker before retry/DLQ routing exists.

Verification performed:

- Validated compose syntax with `docker compose config --quiet`.
- Ran Python unit tests with `make docker-test-python`; 7 tests passed.
- Built the worker image with `docker compose build raw-worker`.
- Ran Dockerized Go tests with `go test ./...`; all packages passed.
- Ran Dockerized schema validation with `scripts/validate_json_schemas.py`; all schemas passed.
- Ran one-shot worker validation with `SIGNALOPS_WORKER_MAX_MESSAGES=1`; it
  skipped and committed a historical invalid raw record.
- Ran another one-shot worker validation with the same explicit validation group;
  it processed a valid raw event.
- Started the long-running worker with `docker compose up -d raw-worker`.
- Verified `docker compose ps` showed `signalops-raw-worker-1` running.
- Verified `rpk group describe signalops.raw-worker.v1` showed a stable group
  with one member and total lag `0`.

Issues found and resolved:

- The first one-shot worker run crashed on an older G008 broker-client test
  record that did not contain `event_id`. The worker now logs and commits
  invalid raw records until retry/DLQ behavior is implemented.
- The initial Python adapter commit used a generic synchronous commit and the
  validation group saw the same record again. The adapter now commits explicit
  topic/partition offsets using `TopicPartition(message.offset + 1)`.
- `.dockerignore` excluded `python`, which prevented the worker image from
  containing source code. The Python source is now included in Docker build
  context.

Next step:

- Add retry/DLQ publishing for invalid raw events and processing failures.
- Add detector plugin contracts and a reference no-op detector worker path.


## 2026-07-07T01:40:31Z

Summary:

- Added DLQ publishing for Python raw-worker invalid records and processing
  failures.
- Added `DLQEvent` JSON Schema for durable failed-record payloads.
- Updated the worker to commit source offsets only after processing succeeds or
  the DLQ publish is acknowledged.
- Validated live Redpanda DLQ behavior with an intentionally invalid raw event.

Files changed:

- `compose.yaml`
- `contracts/events/README.md`
- `contracts/events/dlq_event.v1.schema.json`
- `docs/python_worker.md`
- `docs/deployment.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`
- `python/signalops_workers/__main__.py`
- `python/signalops_workers/broker.py`
- `python/signalops_workers/config.py`
- `python/signalops_workers/worker.py`
- `python/tests/test_config.py`
- `python/tests/test_worker.py`

Rationale:

- G010 intentionally logged and committed invalid raw records as a temporary
  skeleton behavior. G011 replaces that with durable DLQ publication so failed
  records remain inspectable and replayable.
- Source offsets are committed only after DLQ acknowledgement to avoid silent
  loss if the DLQ path is unavailable.
- The DLQ payload preserves source topic, partition, offset, key, headers, and
  base64-encoded source value for audit and future replay tooling.

Verification performed:

- Validated compose syntax with `docker compose config --quiet`.
- Ran Python unit tests with `make docker-test-python`; 10 tests passed.
- Built the worker image with `docker compose build raw-worker`.
- Ran Dockerized Go tests with `go test ./...`; all packages passed.
- Ran Dockerized schema validation with `scripts/validate_json_schemas.py`; the
  new `dlq_event.v1.schema.json` passed.
- Redeployed the worker with `docker compose up -d raw-worker`.
- Produced an invalid raw event directly to `signalops.local.raw.v1` with key
  `g011-invalid-live` and correlation ID `g011-correlation-live`.
- Verified worker logs showed `sent raw event to dlq`.
- Consumed `signalops.local.dlq.algorithm.v1` and verified key, error type,
  source topic, source partition, source offset, correlation header, and base64
  source payload.
- Verified `rpk group describe signalops.raw-worker.v1` reported stable group
  state with total lag `0`.

Issue found and resolved:

- The worker needed durable failure handling beyond G010's temporary skip path.
  It now publishes to DLQ before committing the source offset.

Next step:

- Add retry-topic handling for retryable processing failures.
- Add detector plugin interfaces and a no-op detector path that can emit
  normalized success/failure outcomes.


## 2026-07-07T02:01:22Z

Summary:

- Added retry-topic handling for retryable Python worker failures.
- Added `RetryEvent` JSON Schema for durable retry payloads.
- Updated worker routing so `RetryableWorkerError` publishes to
  `signalops.<environment>.retry.algorithm.v1` and non-retryable failures
  continue to DLQ.
- Validated live Redpanda retry publishing with a synthetic retryable failure.

Files changed:

- `compose.yaml`
- `contracts/events/README.md`
- `contracts/events/retry_event.v1.schema.json`
- `docs/python_worker.md`
- `docs/deployment.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`
- `python/signalops_workers/__main__.py`
- `python/signalops_workers/broker.py`
- `python/signalops_workers/config.py`
- `python/signalops_workers/worker.py`
- `python/tests/test_config.py`
- `python/tests/test_worker.py`

Rationale:

- G011 established DLQ handling for invalid and non-retryable failures. G012
  separates retryable failures into the retry topic so transient dependency or
  algorithm failures can be replayed without being treated as terminal DLQ
  records.
- Source offsets are committed only after retry publication is acknowledged,
  preserving at-least-once behavior when the retry path is unavailable.
- Retry payloads preserve source topic, partition, offset, key, headers,
  retry attempt, and base64-encoded source value.

Verification performed:

- Validated compose syntax with `docker compose config --quiet`.
- Ran Python unit tests with `make docker-test-python`; 13 tests passed.
- Ran Dockerized Go tests with `go test ./...`; all packages passed.
- Ran Dockerized schema validation with `scripts/validate_json_schemas.py`; the
  new `retry_event.v1.schema.json` passed.
- Built the worker image with `docker compose build raw-worker`.
- Published a synthetic retryable failure with `RedpandaRetryPublisher` from the
  worker image.
- Consumed `signalops.local.retry.algorithm.v1` and verified key, retry attempt,
  error type, source topic, source partition, source offset, correlation header,
  and base64 source payload.
- Redeployed the long-running worker with `docker compose up -d raw-worker`.
- Verified gateway readiness and `rpk group describe signalops.raw-worker.v1`
  returned stable state with one member and total lag `0` after the old consumer
  session timed out.

Issue found and resolved:

- After worker redeploy, Redpanda briefly reported `PreparingRebalance` with two
  members because the old consumer session had not timed out. The group returned
  to stable state with one member and zero lag after the session timeout window.

Next step:

- Add detector plugin interfaces and a reference no-op detector path.
- Add retry replay tooling that consumes retry records back into worker input or
  a bounded retry processor.


## 2026-07-07T02:09:51Z

Summary:

- Added Python detector plugin contracts and a reference `signalops.noop`
  detector.
- Added detector loading through `SIGNALOPS_WORKER_DETECTOR_ID`.
- Integrated detector invocation into the raw worker for valid raw events.
- Validated live worker execution of the no-op detector path with a fresh raw
  event through the gateway.

Files changed:

- `compose.yaml`
- `docs/Syncratic_SignalOps_Processing_Specification.md`
- `docs/python_worker.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`
- `python/signalops_plugins/README.md`
- `python/signalops_plugins/__init__.py`
- `python/signalops_plugins/detectors/__init__.py`
- `python/signalops_plugins/detectors/base.py`
- `python/signalops_plugins/detectors/noop.py`
- `python/signalops_workers/__main__.py`
- `python/signalops_workers/config.py`
- `python/signalops_workers/detectors.py`
- `python/signalops_workers/worker.py`
- `python/tests/plugins/test_noop_detector.py`
- `python/tests/test_detectors.py`
- `python/tests/test_worker.py`

Rationale:

- The worker needs an explicit algorithm seam before real detectors are added.
- The no-op detector proves lifecycle wiring, deterministic no-signal behavior,
  and retry/DLQ handling around detector failures without introducing model
  complexity.
- Detector plugin types are isolated in `python/signalops_plugins`, while the
  runnable worker remains in `python/signalops_workers`.

Verification performed:

- Validated compose syntax with `docker compose config --quiet`.
- Ran Python unit tests with `make docker-test-python`; 16 tests passed.
- Ran Dockerized Go tests with `go test ./...`; all packages passed.
- Ran Dockerized schema validation with `scripts/validate_json_schemas.py`; all
  schemas passed.
- Built the worker image with `docker compose build raw-worker`.
- Redeployed the worker with `docker compose up -d raw-worker`.
- Posted a fresh valid raw event through `POST /v1/events/raw` with event ID
  `g013-live-event`.
- Verified worker logs contained `detector evaluated raw event` and
  `processed raw event`.
- Verified `rpk group describe signalops.raw-worker.v1` reported stable group
  state with one member and total lag `0` after redeploy rebalance completed.

Issue found and resolved:

- Retryable detector failures initially escaped retry routing because detector
  execution happened after the raw-handler `try` block. The worker now routes
  retryable and non-retryable detector failures through the same retry/DLQ
  publishers before committing source offsets.
- After worker redeploy, Redpanda briefly reported `PreparingRebalance` until
  the old consumer session timed out. The group returned to stable state with
  one member and zero lag.

Next step:

- Add signal/result publishing for detectors that emit signals.
- Add a reference detector that emits a deterministic low-severity test signal.
- Add retry replay tooling and retry attempt limits.

## 2026-07-07T02:25:49Z

Summary:

- Added detector signal/result publishing from Python workers to
  `signalops.<environment>.signal.v1`.
- Added a Redpanda signal publisher with source-topic, partition, offset,
  correlation, and trace headers.
- Added `signalops.static_test`, a deterministic low-severity signal-emitting
  reference detector for contract and deployment validation.
- Added worker mapping from `EmittedSignal` to the existing `signal.v1` event
  schema.
- Validated live signal emission from a gateway-ingested raw event through the
  finite static detector worker.

Files changed:

- `compose.yaml`
- `docs/Syncratic_SignalOps_Processing_Specification.md`
- `docs/python_worker.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`
- `python/signalops_plugins/README.md`
- `python/signalops_plugins/detectors/noop.py`
- `python/signalops_workers/__main__.py`
- `python/signalops_workers/broker.py`
- `python/signalops_workers/config.py`
- `python/signalops_workers/detectors.py`
- `python/signalops_workers/worker.py`
- `python/tests/plugins/test_noop_detector.py`
- `python/tests/test_detectors.py`
- `python/tests/test_worker.py`

Rationale:

- The durable Python-to-Go result boundary needs a concrete signal event path
  before real market, security, IoT, or CRM detector packs are introduced.
- Detectors emit compact algorithm outputs; the worker owns infrastructure
  enrichment, lineage, topic publication, and failure routing.
- Signal-topic publish failures are retryable infrastructure failures and route
  to the retry topic before source offsets are committed.

Verification performed:

- Validated compose syntax with `docker compose config --quiet`.
- Ran Python unit tests with `make docker-test-python`; 21 tests passed.
- Ran Dockerized Go tests with `go test ./...`; all packages passed.
- Ran Dockerized schema validation with `scripts/validate_json_schemas.py`; all
  schemas passed.
- Built the worker image with `docker compose build raw-worker`.
- Stopped the default no-op worker, posted `g014-live-event` through
  `POST /v1/events/raw`, and ran a one-message worker with
  `SIGNALOPS_WORKER_DETECTOR_ID=signalops.static_test`.
- Consumed `signalops.local.signal.v1` and verified signal key
  `signalops.static_test.low`, detector ID `signalops.static_test`, event ID
  `g014-live-event`, source offset `3`, correlation ID
  `g014-correlation-live`, and trace ID `g014-trace-live`.
- Restarted the default no-op worker and verified gateway readiness plus stable
  `signalops.raw-worker.v1` consumer group state with one member and total lag
  `0`.

Issue found and resolved:

- Signal-topic publish failures were initially about to fall through generic
  DLQ handling. The worker now converts signal publish failures into
  `RetryableWorkerError` so the source event can be retried through the durable
  retry topic.

Next step:

- Add retry replay tooling with attempt limits and escalation from retry to DLQ.
- Add schema validation for emitted Python signal payloads before publication.
- Add the first domain-specific market-data detector pack after the replay path
  is safe.

## 2026-07-07T02:54:42Z

Summary:

- Added Python retry replay tooling for `signalops.<environment>.retry.algorithm.v1` records.
- Added an optional `retry-replayer` Docker Compose service under the `retry-replay` profile.
- Added replay attempt limits through `SIGNALOPS_RETRY_REPLAY_MAX_ATTEMPTS`.
- Added exhausted retry escalation to DLQ with `RetryAttemptsExhausted`.
- Added malformed retry-record handling that routes the retry record itself to DLQ.
- Validated live replay and exhausted-DLQ paths against Redpanda using isolated G015 retry topics.

Files changed:

- `compose.yaml`
- `docs/python_worker.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`
- `python/signalops_workers/broker.py`
- `python/signalops_workers/config.py`
- `python/signalops_workers/retry_replay.py`
- `python/signalops_workers/retry_replay_main.py`
- `python/tests/test_config.py`
- `python/tests/test_retry_replay.py`

Rationale:

- Retry records must not become a terminal holding area. The platform needs a bounded replay loop before real detector packs and external source adapters increase retry volume.
- The replayer is a separate process so replay can be scaled, paused, or run as a finite operational task without changing the raw worker.
- The retry replayer commits retry-topic offsets only after replay or DLQ publication is acknowledged.

Verification performed:

- Validated compose syntax with `docker compose config --quiet`.
- Ran Python unit tests with `make docker-test-python`; 31 tests passed.
- Ran Dockerized Go tests with `go test ./...`; all packages passed.
- Ran Dockerized schema validation with `scripts/validate_json_schemas.py`; all schemas passed.
- Built worker/replayer images with `docker compose build raw-worker retry-replayer`.
- Created isolated validation retry topics `signalops.local.retry.g015.replay.v1` and `signalops.local.retry.g015.exhausted.v1`.
- Published synthetic retry records for `g015-replay-live` at attempt `1` and `g015-exhausted-live` at attempt `3`.
- Ran a finite retry replayer against the replay topic and verified `replayed retry event`.
- Ran a finite retry replayer against the exhausted topic and verified `sent exhausted retry event to dlq`.
- Consumed `signalops.local.raw.v1` and verified key `g015-replay-key`, event ID `g015-replay-live`, `signalops_retry_attempt=1`, and `signalops_replayed_from_retry=true`.
- Consumed `signalops.local.dlq.algorithm.v1` and verified key `g015-exhausted-key`, error type `RetryAttemptsExhausted`, source offset `405`, `signalops_retry_attempt=3`, and original source payload preservation.
- Recreated the default raw worker from the rebuilt image and verified gateway readiness plus stable `signalops.raw-worker.v1` group state with one member and total lag `0`.

Issue found and resolved:

- The raw-worker consumer group briefly entered `PreparingRebalance` after recreating the worker image. It returned to stable state with one member and zero lag after the old session expired.

Next step:

- Add schema validation for Python-emitted `signal.v1` payloads before publication.
- Add operational controls for replay windows, tenant/source filtering, and dry-run replay inspection.
- Add the first Massive/Polygon scheduled market-data source adapter and detector pack after signal validation is in place.

## 2026-07-07T03:46:54Z

Summary:

- Added runtime JSON Schema validation for Python-emitted `signal.v1` payloads before signal-topic publication.
- Added a lightweight internal JSON Schema validator that resolves local `$ref` entries against `contracts/events`.
- Packaged `contracts/` into the Python worker image and updated `.dockerignore` accordingly.
- Added DLQ routing for invalid built signal events through `InvalidSignalEventError`.
- Validated live static-detector signal publication after runtime schema validation.

Files changed:

- `.dockerignore`
- `deploy/docker/python-worker/Dockerfile`
- `docs/python_worker.md`
- `docs/deployment.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`
- `python/signalops_workers/schema_validation.py`
- `python/signalops_workers/worker.py`
- `python/tests/test_schema_validation.py`
- `python/tests/test_worker.py`

Rationale:

- Detector output must be checked before it becomes a durable signal event.
- The validator uses the checked-in SignalOps schemas instead of duplicating the signal contract in Python code.
- Invalid detector output is treated as a terminal plugin/output contract failure and routed to DLQ rather than being published or retried as infrastructure failure.

Verification performed:

- Validated compose syntax with `docker compose config --quiet`.
- Ran Python unit tests with `make docker-test-python`; 36 tests passed.
- Ran Dockerized Go tests with `go test ./...`; all packages passed.
- Ran Dockerized schema validation with `scripts/validate_json_schemas.py`; all schemas passed.
- Built worker/replayer images with `docker compose build raw-worker retry-replayer` after packaging `contracts/` into the image.
- Brought up Redpanda, topic bootstrap, and gateway after the compose project was found down during live validation.
- Posted `g016-live-event` through `POST /v1/events/raw`.
- Ran a one-message worker with `SIGNALOPS_WORKER_DETECTOR_ID=signalops.static_test` and verified it processed the event.
- Consumed `signalops.local.signal.v1` and verified the G016 signal payload contained event ID `g016-live-event`, tenant `tenant-g016`, detector ID `signalops.static_test`, correlation ID `g016-correlation-live`, and schema header `signalops.signal.v1`.
- Restarted the default raw worker and Redpanda console and verified gateway readiness plus stable raw-worker consumer group state with one member and total lag `0`.

Issue found and resolved:

- The first Docker build failed because `.dockerignore` excluded `contracts/`. The ignore entry was removed so the Python worker image can include runtime schemas.
- The compose project was down during the first live validation request. Redpanda, topic bootstrap, and gateway were started before retrying validation.

Next step:

- Add replay dry-run and filtering controls, or begin the first Massive/Polygon scheduled market-data adapter now that signal publication is contract-validated.

## 2026-07-07T04:04:03Z

Summary:

- Parsed the provided Massive/Polygon top 50 megacap source universe file.
- Added a Go parser that exposes DB-ready megacap seed records from `top50megacap.txt`.
- Generated `top50megacap.normalized.csv` with rank, ticker, company, sector, optional industry, and normalized keys.
- Added tests covering row count, first/last records, exchange-suffix ticker normalization, and CSV generation.

Files changed:

- `internal/adapters/marketdata/massive/top50megacap.txt`
- `internal/adapters/marketdata/massive/top50megacap.normalized.csv`
- `internal/adapters/marketdata/massive/megacap_seed.go`
- `internal/adapters/marketdata/massive/megacap_seed_test.go`
- `internal/adapters/marketdata/massive/README.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Rationale:

- The initial Massive market-data program needs a deterministic company universe before scheduled contract and EOD-price ingestion is wired.
- Keeping the provided text file as source of truth plus a parser and normalized CSV gives us both auditability and DB-seed readiness.
- Ticker, company, sector, and industry keys are normalized now to avoid ad hoc parsing when persistence is added.

Verification performed:

- Parsed 50 company rows from `top50megacap.txt`.
- Verified generated CSV starts with `NVDA,NVIDIA,Technology,Semiconductors` and ends with `GEV,GE Vernova,Energy,Industrials`.
- Ran targeted Go tests for `internal/adapters/marketdata/massive`; passed.
- Ran Dockerized Go tests with `go test ./...`; all packages passed.
- Ran Python unit tests with `make docker-test-python`; 36 tests passed.
- Ran Dockerized schema validation with `scripts/validate_json_schemas.py`; all schemas passed.

Issue found and resolved:

- The user referenced `50topmegacap.txt`, but the file present in the repository is `top50megacap.txt`. The implementation uses the actual file name.
- Some source lines include market cap before the sector after `|`; some include only sector after the dash. The parser handles both formats.

Next step:

- Wire the Massive scheduled adapter event builder for this universe: daily option contracts and EOD prices first, still without intraday streaming.

## 2026-07-07T04:12:26Z

Summary:

- Added the first Massive scheduled event builder for market-data records.
- Added deterministic mapping from already-fetched daily option contract records to `RawSignalEvent`.
- Added deterministic mapping from already-fetched equity EOD price records to `RawSignalEvent`.
- Preserved the no-network boundary: this gate does not call Massive APIs or require credentials.
- Added tests for envelope fields, entity hints, stable event IDs, stable idempotency keys, optional metrics, and validation errors.

Files changed:

- `internal/adapters/marketdata/massive/event_builder.go`
- `internal/adapters/marketdata/massive/event_builder_test.go`
- `internal/adapters/marketdata/massive/README.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Rationale:

- The Massive connector needs a stable mapping contract before HTTP fetching, scheduling, persistence, or broker publication is introduced.
- Event builders keep external provider shapes outside the rest of the platform and emit the existing raw event contract used by the gateway and workers.
- Stable event IDs and idempotency keys are required before scheduled pulls can be replayed safely.

Verification performed:

- Ran targeted Go tests for `internal/adapters/marketdata/massive`; passed.
- Ran Dockerized Go tests with `go test ./...`; all packages passed.
- Ran Python unit tests with `make docker-test-python`; 36 tests passed.
- Ran Dockerized schema validation with `scripts/validate_json_schemas.py`; all schemas passed.
- Validated Compose syntax with `docker compose config --quiet`.
- Verified the running stack remained healthy and `signalops.raw-worker.v1` stayed stable with one member and total lag `0`.

Issue found and resolved:

- No code issues were found during this gate. The implementation stayed intentionally offline to avoid coupling tests to Massive API availability or credentials.

Next step:

- Add a Massive HTTP client abstraction and fixture-backed parser for daily option contracts and EOD price responses.
- Then wire a scheduled pull runner that publishes built raw events to Redpanda.

## 2026-07-07T04:24:55Z

Summary:

- Added a Massive HTTP client abstraction for selected scheduled market-data endpoints.
- Added fixture-backed response parsers for option contract listings and daily aggregate bars.
- Added environment-based client configuration with API key precedence: `SIGNALOPS_MASSIVE_API_KEY`, `MASSIVE_API_KEY`, then local `API_KEY` fallback.
- Added tests that verify request paths, query parameters, response parsing, env precedence, required key validation, and no API-key leakage in errors.
- Documented that option aggregate bars must be paired with contract listing metadata before canonical option events are built.
- Kept all tests offline using `httptest`; no live Massive API call is required for validation.

Files changed:

- `internal/adapters/marketdata/massive/client.go`
- `internal/adapters/marketdata/massive/responses.go`
- `internal/adapters/marketdata/massive/client_test.go`
- `internal/adapters/marketdata/massive/README.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Rationale:

- The scheduled pull runner needs a tested client/parser layer before it can fetch provider data and publish raw events.
- API-key handling must be explicit and safe before any live validation is attempted.
- Tests remain fixture-backed so CI and local gates are deterministic and do not depend on provider availability or account limits.

Verification performed:

- Ran targeted Go tests for `internal/adapters/marketdata/massive`; passed.
- Ran Dockerized Go tests with `go test ./...`; all packages passed.
- Ran Python unit tests with `make docker-test-python`; 36 tests passed.
- Ran Dockerized schema validation with `scripts/validate_json_schemas.py`; all schemas passed.
- Validated Compose syntax with `docker compose config --quiet`.
- Verified the running stack remained healthy and `signalops.raw-worker.v1` stayed stable with one member and total lag `0`.

Issue found and resolved:

- The local `.env` currently uses a generic `API_KEY` name. The client supports that as a fallback while preferring `SIGNALOPS_MASSIVE_API_KEY` and `MASSIVE_API_KEY` for clearer production configuration.
- A test syntax typo was caught by the first targeted Go test run and corrected before the full gate.

Next step:

- Add a scheduled pull runner that uses the megacap seed universe, Massive client, event builders, and broker publisher to emit raw events.
- Optionally add a manual live-validation command that uses `.env` without logging the API key.



## 2026-07-07T04:46:21Z

Summary:

- Added the Massive scheduled pull runner that combines the megacap seed universe, Massive HTTP client, event builders, and broker publisher.
- Added dry-run support that builds canonical raw events and reports counts without broker publication.
- Added publish mode that writes JSON `RawSignalEvent` values to `signalops.<env>.raw.v1` using idempotency keys as broker keys.
- Added the `cmd/massive-puller` CLI with dataset, observation-date, company-limit, option-limit, dry-run, and continue-on-error controls.
- Added Docker image target and Compose profile for running the puller without starting it by default.
- Kept validation deterministic; no live Massive API call was made during this gate.

Files changed:

- `internal/adapters/marketdata/massive/scheduled_pull.go`
- `internal/adapters/marketdata/massive/scheduled_pull_test.go`
- `cmd/massive-puller/main.go`
- `cmd/massive-puller/main_test.go`
- `Dockerfile`
- `compose.yaml`
- `Makefile`
- `internal/adapters/marketdata/massive/README.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Rationale:

- The adapter now needs an executable scheduled ingestion boundary after the seed universe, client/parser layer, and event builders were established.
- Dry-run is the default so the local Massive key can validate provider fetch/build behavior before publishing into Redpanda.
- Publish mode uses the same durable broker abstraction as the gateway while preserving scheduled-pull headers for audit and downstream workers.

Verification performed:

- Ran targeted Go tests for `internal/adapters/marketdata/massive`; passed.
- Ran Dockerized Go tests with `go test ./...`; all packages passed.
- Ran Python unit tests with `python -m unittest discover -s python/tests`; 36 tests passed.
- Ran Dockerized schema validation with `scripts/validate_json_schemas.py`; all schemas passed.
- Validated Compose syntax with `docker compose config --quiet` and `docker compose --profile massive-pull config --quiet`.
- Built the new Docker image target with `docker build --target massive-puller -t signalops-massive-puller:local .`.
- Verified the running stack remained healthy and `signalops.raw-worker.v1` stayed stable with one member and total lag `0`.

Issue found and resolved:

- The first CLI flag implementation used the package-level flag helpers before creating the command-specific `FlagSet`; this was corrected before full validation.
- The Compose profile now passes the optional Massive base URL override as well as the supported API key variable names without committing secret values.

Next step:

- Run a controlled Massive dry-run against the local `.env` key for one or two tickers, review provider response compatibility, then switch to publish mode for a small broker-backed smoke test.


## 2026-07-07T04:49:43Z

Summary:

- Ran the first controlled Massive provider-backed validation using the local ignored `.env` key.
- Validated one-ticker equity dry-run: built 1 canonical raw event, published 0, failures 0.
- Validated one-ticker options dry-run with option limit 5: built 5 canonical raw events, published 0, failures 0.
- Ran one-ticker equity publish smoke test: built 1 event and published 1 raw event to `signalops.local.raw.v1`.
- Confirmed the Python raw worker consumed and processed the published event.
- Confirmed the raw-worker consumer group remained stable with total lag `0`.

Files changed:

- `docs/build_journal.md`
- `docs/gate_audit.md`

Rationale:

- G020 added the executable puller, but it intentionally avoided live provider calls. The next risk was provider response compatibility and the end-to-end scheduled pull publish path.
- The validation used dry-run mode before publish mode to prove external fetch/build behavior without immediately producing broker messages.
- The publish smoke test was limited to one equity event to verify broker delivery and worker consumption with minimal blast radius.

Verification performed:

- `docker compose --profile massive-pull run --rm massive-puller --max-companies 1 --datasets equity --dry-run=true --continue-on-error=true`
- `docker compose --profile massive-pull run --rm massive-puller --max-companies 1 --datasets options --options-limit 5 --dry-run=true --continue-on-error=true`
- `docker compose --profile massive-pull run --rm massive-puller --max-companies 1 --datasets equity --dry-run=false --continue-on-error=false`
- `docker compose ps`
- `docker compose exec redpanda rpk group describe signalops.raw-worker.v1`
- `docker compose logs --tail=80 raw-worker`

Issue found and resolved:

- No implementation issues were found. The Massive client parsed the live equity and option-contract responses used in this constrained validation.
- Secret handling remained clean: `.env` stayed ignored and no API key values were logged or committed.

Next step:

- Expand provider-backed validation to a small multi-ticker dry-run and then add scheduler/orchestrator integration for repeatable execution.


## 2026-07-07T05:00:57Z

Summary:

- Added provider request pacing and retry/backoff controls to the Massive scheduled pull runner.
- Added usage counters to scheduled pull reports: provider requests and provider retries.
- Exposed request delay, max retries, and retry backoff through the `cmd/massive-puller` CLI and Compose profile.
- Added unit coverage for transient provider failure retry and retry exhaustion behavior.
- Ran a controlled multi-ticker live dry-run using the ignored local `.env` key: 3 companies, equity plus options, option limit 2, request delay 250ms, max retries 1.
- Live dry-run built 9 events, published 0, made 6 provider requests, retried 0, and failed 0.

Files changed:

- `internal/adapters/marketdata/massive/scheduled_pull.go`
- `internal/adapters/marketdata/massive/scheduled_pull_test.go`
- `cmd/massive-puller/main.go`
- `cmd/massive-puller/main_test.go`
- `compose.yaml`
- `internal/adapters/marketdata/massive/README.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Rationale:

- The one-ticker live smoke test proved provider compatibility and broker delivery, but broad runs need explicit provider usage controls before expanding the universe.
- Pacing and retry/backoff are needed for respectful API usage and transient provider reliability.
- Report counters make each run auditable for provider request volume and retry behavior.

Verification performed:

- Ran Dockerized Go tests with `go test ./...`; all packages passed.
- Ran Python unit tests with `python -m unittest discover -s python/tests`; 36 tests passed.
- Ran Dockerized schema validation with `scripts/validate_json_schemas.py`; all schemas passed.
- Validated Compose syntax with `docker compose --profile massive-pull config --quiet`.
- Built the Massive puller image with `docker build --target massive-puller -t signalops-massive-puller:local .`.
- Rebuilt the Compose service image with `docker compose --profile massive-pull build massive-puller`.
- Ran live multi-ticker dry-run with `docker compose --profile massive-pull run --rm massive-puller --max-companies 3 --datasets equity,options --options-limit 2 --request-delay 250ms --max-retries 1 --retry-backoff 1s --dry-run=true --continue-on-error=true`.
- Verified the running stack remained healthy and `signalops.raw-worker.v1` stayed stable with one member and total lag `0`.

Issue found and resolved:

- The first live dry-run attempt used an older Compose-built puller image and failed because the new flags were not present. Rebuilding the Compose service image resolved the deployment artifact mismatch.
- Secret handling remained clean: `.env` stayed ignored and no API key values were logged or committed.

Next step:

- Add scheduler/orchestrator integration for repeatable one-shot execution, keeping dry-run as the safe default.
- After scheduler wiring, run a constrained scheduled publish validation before expanding the full megacap universe.


## 2026-07-07T05:09:56Z

Summary:

- Added a reusable Massive scheduled loop for repeatable pull execution.
- Added the `cmd/massive-scheduler` entrypoint that wraps the existing Massive pull runner.
- Added schedule controls for interval, max runs, immediate startup execution, and continuing after run errors.
- Added a Docker image target and Compose profile for the scheduler while preserving dry-run as the default.
- Added scheduler loop unit tests for immediate execution, max-run exit, stop-on-error, and continue-on-error behavior.
- Ran a bounded live scheduler validation using the ignored local `.env` key: 1 run, 1 company, equity dry-run, 1 provider request, 1 event built, 0 published, 0 failures.

Files changed:

- `internal/adapters/marketdata/massive/scheduled_loop.go`
- `internal/adapters/marketdata/massive/scheduled_loop_test.go`
- `cmd/massive-scheduler/main.go`
- `cmd/massive-scheduler/main_test.go`
- `Dockerfile`
- `Makefile`
- `compose.yaml`
- `internal/adapters/marketdata/massive/README.md`
- `docs/deployment.md`
- `docs/docker_development.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Rationale:

- The one-shot puller proved manual ingestion, but production operation needs a repeatable scheduling surface.
- The scheduler reuses the same runner, rate controls, dry-run behavior, and broker publish path so scheduling does not fork ingestion semantics.
- `max-runs` gives deterministic local and CI-style validation without leaving a long-running scheduler process active.

Verification performed:

- Ran Dockerized Go tests with `go test ./...`; all packages passed.
- Ran Python unit tests with `python -m unittest discover -s python/tests`; 36 tests passed.
- Ran Dockerized schema validation with `scripts/validate_json_schemas.py`; all schemas passed.
- Validated Compose syntax with `docker compose --profile massive-schedule config --quiet`.
- Built the Massive scheduler image with `docker build --target massive-scheduler -t signalops-massive-scheduler:local .`.
- Rebuilt the Compose scheduler service with `docker compose --profile massive-schedule build massive-scheduler`.
- Verified scheduler CLI flags with `docker compose --profile massive-schedule run --rm massive-scheduler --help`.
- Ran bounded live scheduler dry-run with `docker compose --profile massive-schedule run --rm massive-scheduler --max-runs 1 --max-companies 1 --datasets equity --request-delay 250ms --max-retries 1 --retry-backoff 1s --dry-run=true --continue-on-error=true --continue-on-run-error=false`.
- Verified the running stack remained healthy and `signalops.raw-worker.v1` stayed stable with one member and total lag `0`.

Issue found and resolved:

- `--help` returns a nonzero status through Go's flag package after printing usage. This was expected and the output confirmed the scheduler flags were present.
- Secret handling remained clean: `.env` stayed ignored and no API key values were logged or committed.

Next step:

- Run a constrained scheduled publish validation with `max-runs=1` and one equity ticker.
- After scheduler publish validation, add persistent run history/provider usage accounting when the database layer is introduced.


## 2026-07-07T18:19:15Z

Summary:

- Ran the constrained scheduled publish validation for the Massive scheduler.
- Used the ignored local `.env` key without logging or committing secret values.
- Ran `massive-scheduler` with `max-runs=1`, one company, equity-only dataset, provider pacing, retry controls, and publish mode enabled.
- Scheduler completed 1 run successfully, built 1 equity EOD raw event, and published 1 event to `signalops.local.raw.v1`.
- Confirmed the Python raw worker consumed and processed the scheduled-published raw event.
- Confirmed the raw-worker consumer group remained stable with total lag `0`.

Files changed:

- `docs/build_journal.md`
- `docs/gate_audit.md`

Rationale:

- G023 validated scheduler dry-run execution. The remaining scheduler risk was broker publication through the scheduled path and downstream worker consumption.
- The validation stayed constrained to one equity event to confirm scheduled publish behavior without broad provider/API or broker impact.

Verification performed:

- `docker compose --profile massive-schedule run --rm massive-scheduler --max-runs 1 --max-companies 1 --datasets equity --request-delay 250ms --max-retries 1 --retry-backoff 1s --dry-run=false --continue-on-error=false --continue-on-run-error=false`
- `docker compose ps`
- `docker compose exec redpanda rpk group describe signalops.raw-worker.v1`
- `docker compose logs --tail=80 raw-worker`

Live verification result:

- Scheduler run report: 1 company, 1 event built, 1 event published, 1 provider request, 0 provider retries, 0 failures.
- Scheduler loop report: 1 run, 1 succeeded, 0 failed.
- Raw worker logged detector evaluation and successful raw-event processing for the scheduled-published event.
- Redpanda raw-worker group remained `Stable` with one member and total lag `0`.
- Running stack remained healthy.

Issue found and resolved:

- None. Scheduled publish validation passed without code changes.
- Secret handling remained clean: `.env` stayed ignored and no API key values were logged or committed.

Next step:

- Add persistent scheduler run history/provider usage accounting once the database layer exists.
- Add provider usage budgets before any broad scheduled publish run across the full megacap universe.


## 2026-07-07T19:36:33Z

Summary:

- Added hard provider usage budgets to the Massive scheduled pull path.
- Added per-run limits for provider requests, raw events built, and raw events published.
- Exposed the limits through both `cmd/massive-puller` and `cmd/massive-scheduler` CLI flags and environment variables.
- Added Compose defaults for the budget variables, disabled by default with `0` to preserve current local behavior.
- Added unit coverage for provider request budget, built-event budget, and published-event budget enforcement.
- Validated live scheduler budget behavior with the ignored local `.env` key: one dry-run stopped before the second provider request when capped at 1 request, and one dry-run succeeded with a 2-request/2-built-event budget.

Files changed:

- `internal/adapters/marketdata/massive/scheduled_pull.go`
- `internal/adapters/marketdata/massive/scheduled_pull_test.go`
- `cmd/massive-puller/main.go`
- `cmd/massive-scheduler/main.go`
- `compose.yaml`
- `internal/adapters/marketdata/massive/README.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Rationale:

- The scheduler publish path is proven, but broad scheduled publishing needs hard guardrails before expanding beyond constrained validation runs.
- Provider request budgets protect API usage and account limits.
- Build and publish budgets protect downstream broker and worker paths from accidental high-volume runs.

Verification performed:

- Ran Dockerized Go tests with `go test ./...`; all packages passed.
- Ran Python unit tests with `python -m unittest discover -s python/tests`; 36 tests passed.
- Ran Dockerized schema validation with `scripts/validate_json_schemas.py`; all schemas passed.
- Validated Compose syntax with `docker compose --profile massive-schedule config --quiet`.
- Built the Massive puller image with `docker build --target massive-puller -t signalops-massive-puller:local .`.
- Built the Massive scheduler image with `docker build --target massive-scheduler -t signalops-massive-scheduler:local .`.
- Rebuilt the Compose scheduler image with `docker compose --profile massive-schedule build massive-scheduler`.
- Verified scheduler budget flags with `docker compose --profile massive-schedule run --rm massive-scheduler --help`.
- Ran expected-failure budget dry-run with `docker compose --profile massive-schedule run --rm massive-scheduler --max-runs 1 --max-companies 2 --datasets equity --request-delay 250ms --max-retries 0 --retry-backoff 1s --max-provider-requests 1 --dry-run=true --continue-on-error=false --continue-on-run-error=false`.
- Ran positive bounded dry-run with `docker compose --profile massive-schedule run --rm massive-scheduler --max-runs 1 --max-companies 2 --datasets equity --request-delay 250ms --max-retries 0 --retry-backoff 1s --max-provider-requests 2 --max-events-built 2 --dry-run=true --continue-on-error=false --continue-on-run-error=false`.
- Verified the running stack remained healthy and `signalops.raw-worker.v1` stayed stable with one member and total lag `0`.

Live verification result:

- Budget stop run: 2 companies requested, 1 provider request allowed, 1 event built, 0 published, 1 failure, and error `provider request budget exceeded: limit 1` before the second provider request.
- Positive bounded run: 2 companies, 2 provider requests, 2 events built, 0 published, 0 retries, 0 failures.
- Running stack remained healthy and raw-worker lag stayed `0`.

Issue found and resolved:

- None. Budget enforcement behaved as designed.
- Secret handling remained clean: `.env` stayed ignored and no API key values were logged or committed.

Next step:

- Add persistent scheduler run history/provider usage accounting once the database layer exists.
- Start the database/storage layer planning for run audit history, idempotency persistence, normalized market data, and replay/query support.


## 2026-07-07T20:36:30Z

Summary:

- Added the first PostgreSQL storage foundation for SignalOps operational metadata and audit data.
- Added migration `000001_storage_foundation` with tables for scheduler runs, provider usage, idempotency records, raw event ledger, equity EOD prices, and option contract daily snapshots.
- Added a repeatable migration runner script backed by `schema_migrations`.
- Added local Compose `postgres` and `postgres-migrate` services under the `storage` profile.
- Added initial Go storage boundary types and repository interfaces without binding domain code to a concrete database driver yet.
- Documented local storage setup and migration usage.

Files changed:

- `compose.yaml`
- `Makefile`
- `migrations/000001_storage_foundation.up.sql`
- `migrations/000001_storage_foundation.down.sql`
- `scripts/apply_postgres_migrations.sh`
- `internal/storage/storage.go`
- `internal/storage/storage_test.go`
- `docs/deployment.md`
- `docs/docker_development.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Rationale:

- UI/API work needs durable queryable state for scheduler runs, provider usage, idempotency, and market-data snapshots.
- PostgreSQL is the documented canonical store for operational metadata. TimescaleDB conversion remains future scope after the base persistence paths are proven.
- The first storage boundary keeps migrations and contracts in place before wiring write paths into the Massive scheduler.

Verification performed:

- Ran Dockerized Go tests with `go test ./...`; all packages passed.
- Ran Python unit tests with `python -m unittest discover -s python/tests`; 36 tests passed.
- Ran Dockerized schema validation with `scripts/validate_json_schemas.py`; all schemas passed.
- Validated Compose storage profile with `docker compose --profile storage config --quiet`.
- Applied migrations with `docker compose --profile storage run --rm postgres-migrate`.
- Verified created tables with `docker compose exec postgres psql -U signalops -d signalops -Atc "SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename"`.
- Verified migration version with `docker compose exec postgres psql -U signalops -d signalops -Atc "SELECT version FROM schema_migrations ORDER BY version"`.
- Re-ran migrations and confirmed `skip 000001_storage_foundation`.
- Validated migration script syntax with `bash -n scripts/apply_postgres_migrations.sh`.
- Validated Makefile wrappers with `make compose-storage-migrate` and `make compose-validate`.
- Verified the running stack remained healthy and `signalops.raw-worker.v1` stayed stable with one member and total lag `0`.

Live verification result:

- PostgreSQL started healthy on local Compose port `15432`.
- Migration version `000001_storage_foundation` applied successfully.
- Tables present: `schema_migrations`, `scheduler_runs`, `provider_usage_runs`, `idempotency_records`, `raw_event_ledger`, `marketdata_equity_eod_prices`, and `marketdata_option_contracts_daily`.
- Migration rerun skipped the already-applied version successfully.

Issue found and resolved:

- The first write attempt was interrupted before files were created. The migration and storage files were then written in smaller chunks and validated successfully.

Next step:

- Wire Massive scheduler runs to persist scheduler run records and provider usage rows through a concrete PostgreSQL repository.
- After run audit writes are proven, add idempotency and raw event ledger persistence on publish.


## 2026-07-07T20:59:45Z

Summary:

- Completed G027 by adding a concrete PostgreSQL repository for scheduler run audit and provider usage persistence.
- Wired `massive-scheduler` to open the Postgres repository when `SIGNALOPS_DATABASE_URL` is configured.
- Added config loading for `SIGNALOPS_DATABASE_URL` and local Compose scheduler wiring to depend on healthy Postgres.
- Added pgx stdlib as the database driver dependency.
- Persisted each scheduler loop run into `scheduler_runs` and provider usage into `provider_usage_runs` after the Massive scheduled pull completes.
- For single-dataset runs, provider usage is recorded for that dataset. For multi-dataset runs, provider usage is recorded as an aggregate `all` row so request/retry totals remain accurate.
- Documented scheduler persistence in deployment and Massive adapter docs.

Files changed:

- `go.mod`
- `go.sum`
- `compose.yaml`
- `cmd/massive-scheduler/main.go`
- `internal/config/config.go`
- `internal/config/config_test.go`
- `internal/storage/postgres/repository.go`
- `internal/storage/postgres/repository_test.go`
- `docs/deployment.md`
- `internal/adapters/marketdata/massive/README.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Rationale:

- Scheduler runs need durable, queryable audit history before UI/API views can expose ingestion health, budgets, failures, and provider consumption.
- Persisting the scheduler summary first keeps storage integration narrow while proving the database path and deployment wiring.
- The repository implements the existing storage boundary, keeping scheduler code independent of SQL details.

Verification performed:

- Ran Dockerized Go tests with `go test ./...`; all packages passed.
- Built the scheduler image with `docker compose --profile massive-schedule build massive-scheduler`; build completed successfully and Dockerfile test stage passed.
- Ran Python unit tests with `python -m unittest discover -s python/tests`; 36 tests passed.
- Ran schema validation with `scripts/validate_json_schemas.py`; all schemas passed.
- Ran Postgres integration test `TestRepositoryAgainstPostgres`; repository upsert/usage writes passed against local Compose Postgres on port `15432`.
- Ran Redpanda broker integration test `TestPublishConsumeCommitAgainstRedpanda`; publish/consume/commit passed against local Redpanda on port `19092`.
- Ran a live scheduler dry-run through Compose with one company, equity dataset, one provider request, and database persistence enabled.
- Queried Postgres for latest `scheduler_runs` and `provider_usage_runs` rows.
- Verified raw-worker group remained stable with total lag `0`.

Live verification result:

- Scheduler dry-run persisted run `massive:src-massive:20260707T205903.415271355Z` with status `succeeded`, `events_built=1`, `events_published=0`, `provider_requests=1`, `provider_retries=0`, and `failures=0`.
- Provider usage persisted for that run with provider `massive`, dataset `equity_eod_prices`, `request_count=1`, `retry_count=0`, and `event_count=1`.
- Redpanda raw-worker group remained `Stable` with one member and total lag `0`.

Issue found and resolved:

- Local `gofmt` was unavailable, so formatting was run through the Go Docker image.
- The schema validation command initially hit the sandbox loopback failure before Docker started; it was rerun with approved escalation and passed.
- Multi-dataset provider usage initially risked misleading zero request/retry rows. The persistence logic now stores aggregate provider usage as dataset `all` when more than one dataset is selected.
- Final Go validation caught an over-escaped quote in the Postgres array helper; the helper now uses one backslash for quotes and two for literal backslashes, with test coverage.

Next step:

- Add database-backed idempotency and raw event ledger persistence on publish so replay/audit state is durable beyond scheduler summaries.
- Start exposing persisted scheduler history through an API endpoint for UI/API readiness.


## 2026-07-08T00:18:45Z

Summary:

- Completed G028 by adding database-backed idempotency and raw event ledger persistence for Massive raw-event publications.
- Extended the storage boundary with raw event ledger and publish repository interfaces.
- Implemented Postgres upserts for `idempotency_records` and `raw_event_ledger` in the existing repository.
- Wired `massive-puller` and `massive-scheduler` to pass the Postgres repository into the Massive scheduled pull path when `SIGNALOPS_DATABASE_URL` is configured.
- Persisted raw event ledger rows only after broker publish acknowledgement, including topic, partition, offset, payload JSON, entity hints, and event timing.
- Persisted idempotency rows with status `published`, broker acknowledgement details, route metadata, and a SHA-256 hash of the exact published JSON payload.
- Preserved dry-run behavior: no raw ledger or idempotency writes happen when no broker publish occurs.

Files changed:

- `cmd/massive-puller/main.go`
- `cmd/massive-scheduler/main.go`
- `internal/adapters/marketdata/massive/scheduled_pull.go`
- `internal/adapters/marketdata/massive/scheduled_pull_test.go`
- `internal/storage/storage.go`
- `internal/storage/storage_test.go`
- `internal/storage/postgres/repository.go`
- `internal/storage/postgres/repository_test.go`
- `docs/deployment.md`
- `internal/adapters/marketdata/massive/README.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Rationale:

- Scheduler summaries prove that a run happened, but replay, audit, and idempotency require durable per-event state after publication.
- The raw event ledger becomes the queryable source for what was actually emitted to Kafka.
- Idempotency rows provide stable deduplication/audit state keyed by tenant, source, and idempotency key.

Verification performed:

- Ran Dockerized Go tests with `go test ./...`; all packages passed.
- Ran Postgres integration test `TestRepositoryAgainstPostgres`; scheduler, provider usage, raw ledger, and idempotency writes passed against local Compose Postgres on port `15432`.
- Validated Compose scheduler profile with `docker compose --profile massive-schedule config --quiet`.
- Built the scheduler image with `docker compose --profile massive-schedule build massive-scheduler`; build completed successfully and Dockerfile test stage passed.
- Ran Python unit tests with `python -m unittest discover -s python/tests`; 36 tests passed.
- Ran schema validation with `scripts/validate_json_schemas.py`; all schemas passed.
- Ran a bounded live Massive scheduler publish through Compose with one company, equity dataset, one provider request, one built event, one published event, and database persistence enabled.
- Queried Postgres for latest `scheduler_runs`, `raw_event_ledger`, and `idempotency_records` rows.
- Ran Redpanda broker integration test `TestPublishConsumeCommitAgainstRedpanda`; publish/consume/commit passed against local Redpanda on port `19092`.
- Verified raw-worker group remained stable with total lag `0`.

Live verification result:

- Scheduler publish run `massive:src-massive:20260708T001716.692425267Z` persisted with status `succeeded`, `dry_run=false`, `events_built=1`, `events_published=1`, `provider_requests=1`, and `failures=0`.
- Raw event ledger persisted event `evt_5d5a94a0e8ea5d149ec19947` for dataset `equity_eod_prices` on topic `signalops.local.raw.v1`, partition `2`, offset `3`.
- Idempotency record persisted the same event with status `published`, topic `signalops.local.raw.v1`, partition `2`, offset `3`, and SHA-256 payload hash prefix `sha256:22d0af9ad`.
- Redpanda raw-worker group remained `Stable` with one member and total lag `0`.

Issue found and resolved:

- The interrupted turn left only the storage boundary partially edited; the continuation verified the partial state before adding repository and adapter wiring.
- An earlier scoped edit accidentally duplicated production declarations into `internal/storage/storage_test.go`; it was restored to focused constant tests and validated.
- Publish count semantics were corrected so a broker-acknowledged event remains counted as published even if subsequent database persistence fails; a regression test now covers this case.

Next step:

- Add API endpoints for scheduler run history, provider usage, raw event ledger lookup, and idempotency lookup so UI/UX work has durable query surfaces.
- Consider transaction grouping for raw ledger plus idempotency writes if future adapters require stricter all-or-nothing persistence semantics.


## 2026-07-08T00:28:57Z

Summary:

- Completed G029 by adding storage-backed operational query APIs to the gateway.
- Added a read-side storage interface for scheduler runs, provider usage, raw event ledger, and idempotency lookup.
- Implemented Postgres query methods with bounded limits, optional filters, JSONB passthrough, and `storage.ErrNotFound` mapping.
- Added gateway routes for scheduler run history/detail, provider usage, raw event list/detail, and idempotency lookup.
- Wired the gateway to open the Postgres repository when `SIGNALOPS_DATABASE_URL` is configured.
- Updated local Compose so the gateway depends on healthy Postgres and receives the database URL.
- Documented the operational query API surface in `docs/api.md`.

Files changed:

- `cmd/gateway/main.go`
- `compose.yaml`
- `docs/api.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`
- `internal/api/router.go`
- `internal/api/router_test.go`
- `internal/storage/storage.go`
- `internal/storage/postgres/repository.go`
- `internal/storage/postgres/repository_test.go`

Rationale:

- UI/UX work needs stable query surfaces before building screens.
- The persisted G027/G028 data now has HTTP access without coupling the UI directly to Postgres.
- Returning raw JSON payload/config/report fields as JSON preserves the data shape needed for inspection and debugging.

Verification performed:

- Ran Dockerized Go tests with `go test ./...`; all packages passed.
- Ran Postgres integration test `TestRepositoryAgainstPostgres`; write and read/query methods passed against local Compose Postgres on port `15432`.
- Validated Compose configuration with `docker compose config --quiet`.
- Built the gateway image with `docker compose build gateway`; build completed successfully and Dockerfile test stage passed.
- Restarted the gateway with `docker compose up -d gateway`.
- Queried live `GET /healthz` through `localhost:18000`.
- Queried live `GET /v1/scheduler/runs?limit=2` and received persisted scheduler runs.
- Queried live `GET /v1/raw-events?limit=2` and received persisted raw event ledger rows.
- Queried live `GET /v1/raw-events/evt_5d5a94a0e8ea5d149ec19947` and received the raw event payload and broker acknowledgement details.
- Queried live `GET /v1/provider-usage?run_id=massive:src-massive:20260708T001716.692425267Z&limit=5` and received the matching provider usage row.
- Queried live `GET /v1/idempotency?tenant_id=tenant-local&source_id=src-massive&idempotency_key=idem_5d5a94a0e8ea5d149ec19947` and received the published idempotency record.
- Ran Python unit tests with `python -m unittest discover -s python/tests`; 36 tests passed.
- Ran schema validation with `scripts/validate_json_schemas.py`; all schemas passed.
- Ran Redpanda broker integration test `TestPublishConsumeCommitAgainstRedpanda`; publish/consume/commit passed against local Redpanda on port `19092`.
- Verified raw-worker group remained stable with total lag `0`.

Live verification result:

- Gateway health returned status `ok` for service `signalops-gateway`.
- Scheduler API returned run `massive:src-massive:20260708T001716.692425267Z` with `dry_run=false`, `events_built=1`, and `events_published=1`.
- Raw event API returned event `evt_5d5a94a0e8ea5d149ec19947` with topic `signalops.local.raw.v1`, partition `2`, offset `3`, and JSON payload content.
- Provider usage API returned usage id `massive:src-massive:20260708T001716.692425267Z:equity_eod_prices` with one request and one event.
- Idempotency API returned key `idem_5d5a94a0e8ea5d149ec19947` with status `published` and full SHA-256 payload hash.
- Redpanda raw-worker group remained `Stable` with one member and total lag `0`.

Issue found and resolved:

- The gateway initially attempted to open Postgres even when the database URL was empty; this was tightened so storage wiring is strictly optional.
- The first idempotency live check used the wrong key and correctly returned `404`; the check was rerun with the idempotency key returned by the raw event API and passed.

Next step:

- Start UI/UX work against these read endpoints, beginning with an operational dashboard for scheduler runs, provider usage, raw event inspection, and idempotency lookup.
- Add pagination cursors after the initial dashboard shape is proven; current endpoints use bounded `limit` queries.


## 2026-07-08T02:34:21Z

Summary:

- Completed G030 by scaffolding the SignalOps operational dashboard frontend under `web/`.
- Adopted the data-centric stack committed in the revised spec: Vite + React + TypeScript, TanStack Router, TanStack Query, Zustand, Apache ECharts, AG Grid Community, Tailwind CSS, and `lucide-react`.
- Implemented the dashboard shell (app bar + health indicator + navigation) and four operational views: Runs (AG Grid table + detail panel with provider usage + ECharts bar chart), Raw Events (AG Grid table + detail panel with payload/entity JSON), Idempotency (form lookup with 404 handling and raw-event cross-link), and System (health, readiness, storage-availability probe, API base URL, last refresh).
- Added a Vite dev proxy for `/healthz`, `/readyz`, and `/v1` to the gateway, resolving the CORS gap documented in the evaluation (the gateway has no CORS middleware).
- Route-level code splitting lazy-loads AG Grid and ECharts only for the Runs and Raw Events views; vendor chunks are split via `manualChunks`.
- Loading, error, and empty states are implemented for every view; copy-to-clipboard controls cover run/event ids, idempotency key, and payload hash; timestamps render as UTC.
- Preceded implementation by revising `docs/frontend_implementation_spec.md` (stack adoption, CORS/proxy guidance, `raw_events` list envelope, `/readyz` shape, `omitempty` handling, `/frontend` rationale, Future Gates) and adding `docs/frontend/frontend_evaluation.md`.

Files changed:

- `docs/frontend/frontend_evaluation.md`
- `docs/frontend_implementation_spec.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`
- `docs/docker_development.md`
- `web/.env.example`
- `web/.gitignore`
- `web/index.html`
- `web/package.json`
- `web/postcss.config.js`
- `web/README.md`
- `web/tailwind.config.js`
- `web/tsconfig.json`
- `web/vite.config.ts`
- `web/src/App.tsx`
- `web/src/api/client.ts`
- `web/src/api/queries.ts`
- `web/src/components/CopyButton.tsx`
- `web/src/components/DashboardShell.tsx`
- `web/src/components/HealthIndicator.tsx`
- `web/src/components/IdempotencyLookup.tsx`
- `web/src/components/JsonViewer.tsx`
- `web/src/components/MetricTile.tsx`
- `web/src/components/RawEventDetail.tsx`
- `web/src/components/RawEventTable.tsx`
- `web/src/components/RefreshButton.tsx`
- `web/src/components/RunDetail.tsx`
- `web/src/components/RunTable.tsx`
- `web/src/components/RunsBarChart.tsx`
- `web/src/components/States.tsx`
- `web/src/components/StatusBadge.tsx`
- `web/src/lib/format.ts`
- `web/src/main.tsx`
- `web/src/router.tsx`
- `web/src/routes/IdempotencyRoute.tsx`
- `web/src/routes/RawEventsRoute.tsx`
- `web/src/routes/RunsRoute.tsx`
- `web/src/routes/SystemRoute.tsx`
- `web/src/store/ui.ts`
- `web/src/styles/index.css`
- `web/src/types.ts`
- `web/src/vite-env.d.ts`

Rationale:

- The operational UI is the first consumer of the G029 query APIs; a client-side SPA is sufficient for an internal, authenticated dashboard (no SSR needed).
- TanStack Query owns server state and directly satisfies the loading/error/empty and refresh requirements and the `400`/`404`/`500`/`503` error mapping.
- The Vite proxy is the supported dev path because the gateway emits no CORS headers.
- AG Grid Community and Apache ECharts are deferred-loaded per route to keep the initial shell light.

Verification performed:

- `cd web && npm install` (201 packages, no errors).
- `cd web && npm run build` (`tsc && vite build`); type-check passed and production build succeeded with route-level and vendor code splitting.
- Started the Vite dev server on `http://localhost:5173/`.
- Validated the dev proxy forwards same-origin requests to the gateway on `:18000`:
  - `curl -fsS http://localhost:5173/healthz` returned `{"service":"signalops-gateway","status":"ok",...}`.
  - `curl -fsS http://localhost:5173/readyz` returned `status:"ready"`.
  - `curl -fsS 'http://localhost:5173/v1/scheduler/runs?limit=2'` returned persisted scheduler runs with config, counters, and timestamps.
  - `curl -fsS 'http://localhost:5173/v1/raw-events?limit=2'` returned the `raw_events` (plural) list envelope with broker topic/partition/offset.
  - `curl -fsS 'http://localhost:5173/v1/idempotency?tenant_id=tenant-local&source_id=src-massive&idempotency_key=idem_5d5a94a0e8ea5d149ec19947'` returned the published idempotency record with payload hash.
  - `curl -s -w '[HTTP %{http_code}]' '...&idempotency_key=bogus_key_xyz'` returned HTTP 404 `idempotency_not_found`.
  - `curl -s -w '[HTTP %{http_code}]' '.../v1/idempotency'` (no params) returned HTTP 400 `missing_query`.

Live verification result:

- Build passes (`tsc && vite build`).
- All five required views are implemented against the live gateway API through the proxy.
- Response shapes match the TypeScript types in `web/src/types.ts` (including the `raw_events` plural list envelope, `/readyz` `status:"ready"`, and `omitempty` broker/idempotency fields).
- Error mapping confirmed: 404 `idempotency_not_found` and 400 `missing_query` return the documented `{"error","message"}` body.

Issue found and resolved:

- The initial two-tsconfig project-reference setup failed `tsc` with TS6306/TS6310 because the referenced `tsconfig.node.json` could not be `composite` with `noEmit`. Resolved by making `tsconfig.json` self-contained (including `vite.config.ts` directly with `node` + `vite/client` types) and removing `tsconfig.node.json`.

Next step:

- Perform browser validation (console errors, row selection, detail panels, copy buttons, idempotency empty state) as a manual follow-up.
- Add a `web` Compose service and frontend Dockerfile when the gate requires Compose integration; the dev server is sufficient for this gate.
- Defer React Flow, SSE/WebSocket streaming, and client-side time-series evaluation to later gates pending backend topology and streaming endpoints.
- Add Vitest unit tests for `api/client` and formatting helpers when test coverage is prioritized.


## 2026-07-08T03:15:30Z

Summary:

- Closed the G030 manual browser follow-up based on operator confirmation that the browser UI works.
- Completed G031 by adding the first backend-to-frontend SSE stream foundation to the gateway.
- Added `GET /v1/streams/dashboard` with channel filtering for health, scheduler runs, raw events, provider usage, and heartbeat events.
- Kept the stream gateway-backed and query-repository-backed; browsers still do not connect directly to Redpanda.
- Documented the dashboard stream API and its current limitations in `docs/api.md`.

Files changed:

- `docs/api.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`
- `internal/api/router.go`
- `internal/api/router_test.go`

Rationale:

- G030 established the operational UI using polling; the next backend boundary needed is a browser-safe subscription transport.
- SSE is sufficient for one-way dashboard updates and avoids WebSocket complexity until operator control flows require bidirectional messaging.
- Starting from query storage gives the frontend a stable stream contract without exposing broker internals or designing replay prematurely.

Verification performed:

- Ran Dockerized formatting with `gofmt -w internal/api/router.go internal/api/router_test.go`.
- Ran Dockerized focused API tests with `go test ./internal/api -count=1 -v`; all API route and SSE tests passed.
- Ran Dockerized full Go tests with `go test ./...`; all packages passed.
- Validated Compose configuration with `docker compose config --quiet`.
- Built the gateway image with `docker compose build gateway`; build completed successfully and Dockerfile test stage passed.
- Restarted the gateway with `docker compose up -d gateway`.
- Queried live `GET /healthz` through `localhost:18000`.
- Queried live `GET /v1/scheduler/runs?limit=1` and received persisted scheduler data.
- Queried live `GET /v1/streams/dashboard?channels=health,runs,raw_events,provider_usage` with bounded `curl -N --max-time 3`.
- Queried live `GET /v1/streams/dashboard?channels=bogus` and received `400 invalid_channel`.
- Queried live `GET /v1/streams/dashboard?channels=heartbeat` with bounded `curl -N --max-time 6` and observed periodic heartbeat frames.

Live verification result:

- Gateway health returned status `ok` for service `signalops-gateway`.
- Scheduler REST endpoint remained operational after the route change.
- Dashboard SSE stream emitted `heartbeat`, `health`, `scheduler_run`, `raw_event`, and `provider_usage` frames with stable ids where available.
- Unknown channel validation returned `400 invalid_channel` before stream startup.
- Heartbeat-only stream emitted on open and again after the stream interval.

Issue found and resolved:

- Local `gofmt` is not installed; formatting was performed with the Dockerized Go toolchain.
- `apply_patch` failed due to the known sandbox `bwrap: loopback: Failed RTM_NEWADDR` issue; scoped file rewrites were used for the intended files only.

Next step:

- Hand the `/v1/streams/dashboard` contract to the frontend agent for a later UI gate that swaps selected widgets from polling to SSE-backed subscriptions.
- Design `Last-Event-ID` replay and cursor/time-window pagination only after the first frontend stream adoption exposes concrete resume requirements.


## 2026-07-08T03:46:26Z

Summary:

- Completed G032 by adopting the G031 dashboard SSE stream in the frontend.
- Added a browser `EventSource` subscription bridge for `/v1/streams/dashboard`.
- Kept REST/TanStack Query as the snapshot and detail fallback while stream events refresh the existing health, runs, raw-events, and provider-usage caches.
- Added UI stream state for connection status, last stream event time, and stream error display.
- Updated frontend documentation so the spec no longer says the gateway is REST-only.

Files changed:

- `docs/build_journal.md`
- `docs/gate_audit.md`
- `docs/frontend_implementation_spec.md`
- `web/README.md`
- `web/src/App.tsx`
- `web/src/api/client.ts`
- `web/src/api/stream.ts`
- `web/src/components/DashboardStreamBridge.tsx`
- `web/src/components/HealthIndicator.tsx`
- `web/src/routes/SystemRoute.tsx`
- `web/src/store/ui.ts`

Rationale:

- G031 exposed the browser-safe SSE transport; the frontend now needs a narrow adoption layer before individual widgets are rewritten around stream-first data.
- Invalidating existing TanStack Query caches on stream events minimizes UI churn and preserves the proven G030 REST behavior.
- Keeping REST fallback avoids coupling table/detail views to streaming availability.

Verification performed:

- Ran `npm run build` in `web/`; TypeScript and Vite production build passed.
- Confirmed Vite dev server was listening on `:5173` and gateway on `:18000`.
- Queried `http://localhost:5173/healthz` through the dev proxy.
- Queried `http://localhost:5173/v1/streams/dashboard?channels=health,runs,raw_events,provider_usage` with bounded `curl -N --max-time 3` through the dev proxy.
- Queried `http://localhost:5173/v1/streams/dashboard?channels=bogus` and received `400 invalid_channel` through the dev proxy.

Live verification result:

- The proxied stream emitted `heartbeat`, `health`, `scheduler_run`, `raw_event`, and `provider_usage` frames through Vite.
- Invalid stream channel requests returned the documented `invalid_channel` error.
- Production frontend build completed successfully with the stream bridge included.

Issue found and resolved:

- `docs/frontend_implementation_spec.md` still described the gateway as REST-only after G031; it now documents the SSE endpoint and keeps WebSockets as future scope.

Next step:

- Add frontend unit tests for the stream parser/subscription behavior when test coverage is prioritized.
- Move individual dashboard widgets to consume stream-derived state directly only after the cache-invalidation bridge proves stable.


## 2026-07-08T03:57:04Z

Summary:

- Completed G033 by adding production-style Compose integration for the SignalOps web UI.
- Added `web/Dockerfile` with a Node build stage and nginx runtime stage.
- Added nginx proxy configuration for `/healthz`, `/readyz`, and `/v1` so the containerized UI remains same-origin with the gateway API and SSE stream.
- Added a `web` service to `compose.yaml` on host port `15173`.
- Updated Docker/deployment/frontend documentation for the Compose web path.

Files changed:

- `compose.yaml`
- `docs/build_journal.md`
- `docs/deployment.md`
- `docs/docker_development.md`
- `docs/gate_audit.md`
- `web/Dockerfile`
- `web/README.md`
- `web/deploy/nginx.conf`

Rationale:

- The frontend now needs a Dockerized runtime path, not only a Vite dev server.
- nginx can serve the static Vite build and proxy API/SSE traffic to the internal gateway service, avoiding browser CORS requirements.
- Keeping the web service separate preserves SignalOps subsystem independence and does not require gateway changes.

Verification performed:

- Ran `docker compose config --quiet`; Compose config passed.
- Ran `docker compose build web`; image build completed and ran `npm run build` inside the image build.
- Ran `docker compose up -d web`; service started on `http://localhost:15173/`.
- Queried `http://localhost:15173/` and received the built SignalOps HTML shell.
- Queried `http://localhost:15173/healthz` and received gateway health through nginx.
- Queried `http://localhost:15173/v1/scheduler/runs?limit=1` and received persisted scheduler data through nginx.
- Queried `http://localhost:15173/v1/streams/dashboard?channels=health,heartbeat` with bounded `curl -N --max-time 3` and received SSE frames through nginx.
- Ran `docker compose ps web` and confirmed `signalops-web-1` was running with host port `15173` mapped to container port `8080`.

Live verification result:

- Web container served the production Vite app shell.
- nginx proxied REST gateway endpoints correctly.
- nginx proxied the dashboard SSE stream correctly with buffering disabled for `/v1`.
- Compose web service is running at `http://localhost:15173/`.

Issue found and resolved:

- The web image build reported two moderate npm audit findings from the existing dependency tree. Dependency upgrades were not changed in this Compose integration gate and should be assessed separately.

Next step:

- Add frontend test coverage for the stream bridge and consider dependency audit remediation in a dedicated frontend hardening gate.


## 2026-07-08T04:12:01Z

Summary:

- Completed G034 by adding focused frontend tests for the dashboard stream client and formatting helpers.
- Refactored the dashboard stream client to expose small parsing/conversion helpers for deterministic tests.
- Patched the non-major PostCSS audit finding by updating `postcss` to `8.5.16`.
- Re-ran frontend tests, production build, npm audit, and Compose web image build.

Files changed:

- `docs/build_journal.md`
- `docs/gate_audit.md`
- `web/package.json`
- `web/package-lock.json`
- `web/src/api/stream.ts`
- `web/src/api/stream.test.ts`
- `web/src/lib/format.test.ts`

Rationale:

- G032 introduced EventSource behavior but had no unit coverage.
- The stream parser and subscription wiring are small but important enough to lock down before expanding stream-driven widgets.
- The PostCSS audit fix was semver-compatible; the remaining ECharts fix requires a major-version upgrade and should be handled separately.

Verification performed:

- Ran `npm test`; 2 test files and 6 tests passed.
- Ran `npm run build`; TypeScript and Vite production build passed.
- Ran `npm audit --json`; remaining audit result is one moderate ECharts advisory requiring ECharts `6.1.0` semver-major upgrade.
- Ran `docker compose build web`; image build completed and ran `npm run build` inside the image build.

Live verification result:

- Stream parser tests cover JSON, empty, and non-JSON SSE payloads.
- Stream subscription tests cover default channel URL construction, open/error callbacks, event conversion, and subscription close behavior.
- Formatting tests cover UTC timestamps, invalid timestamp fallback, duration formatting, dash fallback, and truncation.
- Compose web image still builds successfully after the dependency update.

Issue found and resolved:

- npm audit reported a moderate PostCSS advisory fixed by `postcss@8.5.16` without a major version upgrade.
- npm audit still reports a moderate ECharts advisory fixed only by `echarts@6.1.0`, a semver-major upgrade. This was not applied in G034 to avoid chart compatibility risk without a dedicated upgrade/test pass.

Next step:

- Evaluate ECharts 6.1.0 compatibility or replace the charting dependency in a dedicated frontend dependency hardening gate.
- Add component-level tests for `DashboardStreamBridge` query invalidation if frontend test infrastructure expands beyond unit-level stream client tests.


## 2026-07-08T04:25:24Z

Summary:

- Completed G035 by upgrading ECharts to `6.1.0`, resolving the remaining npm audit advisory.
- Verified `echarts-for-react` peer dependencies support ECharts 6 before upgrading.
- Re-ran frontend tests, production build, audit, Compose web image build, and live web container smoke checks.

Files changed:

- `docs/build_journal.md`
- `docs/gate_audit.md`
- `web/package.json`
- `web/package-lock.json`

Rationale:

- G034 left one moderate npm audit finding because the ECharts fix required a semver-major upgrade.
- The project uses ECharts only through a simple provider-requests bar chart, and `echarts-for-react` advertises compatibility with ECharts 6.
- Performing the major dependency upgrade in a dedicated gate keeps the compatibility risk explicit and independently validated.

Verification performed:

- Ran `npm view echarts version peerDependencies --json`; latest/fixed ECharts version was `6.1.0`.
- Ran `npm view echarts-for-react version peerDependencies --json`; peer dependency supports `echarts` `^6.0.0`.
- Ran `npm install echarts@6.1.0`; npm audit reported zero vulnerabilities after install.
- Ran `npm test`; 2 test files and 6 tests passed.
- Ran `npm run build`; TypeScript and Vite production build passed.
- Ran `npm audit --json`; zero vulnerabilities reported.
- Ran `docker compose build web`; image build completed and `npm ci` reported zero vulnerabilities during build.
- Ran `docker compose up -d web`; web service restarted successfully.
- Queried `http://localhost:15173/` and received the rebuilt SignalOps HTML shell.
- Queried `http://localhost:15173/v1/scheduler/runs?limit=1` and received persisted scheduler data through nginx.
- Queried `http://localhost:15173/v1/streams/dashboard?channels=health,heartbeat` with bounded `curl -N --max-time 3` and received SSE frames through nginx.
- Ran `docker compose ps web` and confirmed `signalops-web-1` was running on host port `15173`.

Live verification result:

- npm audit is clean: zero vulnerabilities.
- Frontend unit tests still pass after ECharts 6.1.0.
- Production build emitted the updated ECharts vendor chunk successfully.
- Compose web image and running container are healthy after the dependency upgrade.
- REST and SSE proxy paths through the web container remain operational.

Issue found and resolved:

- The remaining moderate ECharts XSS advisory from G034 was resolved by the ECharts 6.1.0 major upgrade.

Next step:

- Add browser-level visual validation for the Runs chart if Playwright or another browser automation tool is introduced.
- Continue backend platform work toward catalog/source registry APIs or pagination, since frontend dependency audit is now clean.


## 2026-07-08T04:54:17Z

Summary:

- Completed G036 by adding the first durable Stream Catalog source registry foundation.
- Added `catalog_sources` migration with a seeded local Massive source for `tenant-local/src-massive`.
- Added storage contracts and Postgres repository methods for source catalog upsert/list behavior.
- Added gateway API `GET /v1/tenants/{tenant_id}/catalog/sources`.
- Documented the Stream Catalog source endpoint and local deployment behavior.

Files changed:

- `docs/api.md`
- `docs/build_journal.md`
- `docs/deployment.md`
- `docs/gate_audit.md`
- `internal/api/router.go`
- `internal/api/router_test.go`
- `internal/storage/storage.go`
- `internal/storage/postgres/repository.go`
- `internal/storage/postgres/repository_test.go`
- `migrations/000002_catalog_sources.up.sql`
- `migrations/000002_catalog_sources.down.sql`

Rationale:

- The UI architecture includes Sources, Pipelines, and Rules; Sources need the first durable catalog boundary before those pages can become real.
- A tenant-scoped source catalog gives operators an explicit registry of adapters, domains, ingestion modes, datasets, and status instead of inferring sources only from scheduler/raw-event rows.
- Seeding the Massive source keeps the current local market-data use case visible immediately after migration.

Verification performed:

- Ran Dockerized formatting with `gofmt` over storage and API files.
- Ran focused Dockerized Go tests for `./internal/api`, `./internal/storage`, and `./internal/storage/postgres`; all passed.
- Ran Dockerized full Go tests with `go test ./...`; all packages passed.
- Ran `docker compose config --quiet`; Compose config passed.
- Ran `make compose-storage-migrate`; migration `000002_catalog_sources` applied and seeded the Massive source.
- Ran `docker compose build gateway`; gateway image build passed and Dockerfile test stage passed.
- Ran `docker compose up -d gateway`; gateway restarted successfully.
- Ran Postgres integration test `TestRepositoryAgainstPostgres`; source catalog upsert/list checks passed.
- Queried live `GET /v1/tenants/tenant-local/catalog/sources?limit=10` through `localhost:18000`.
- Queried the same endpoint through the web proxy at `localhost:15173`.
- Queried Postgres `catalog_sources` rows directly.

Live verification result:

- Gateway catalog API returned `tenant-local/src-massive` with source domain `market_data`, adapter `market_data.massive`, status `active`, ingestion mode `scheduled_pull`, and datasets `equity_eod_prices` plus `option_contracts_daily`.
- Web container proxy forwarded the catalog API correctly.
- Postgres contained the seeded `tenant-local/src-massive` row and the integration-test `tenant-1/src-massive` row.

Issue found and resolved:

- No implementation issues were encountered. The integration test intentionally upserts a separate `tenant-1/src-massive` row, so local Postgres now has both the seeded local source and test tenant source.

Next step:

- Add frontend Sources page consumption of `/v1/tenants/{tenant_id}/catalog/sources`.
- Extend the catalog foundation to pipelines and rules after source visibility lands in the UI.


## 2026-07-08T05:07:40Z

Summary:

- Completed G037 by adding a frontend Sources page wired to the G036 catalog source API.
- Added catalog source TypeScript types, API client method, and TanStack Query hook.
- Added `/sources` route and top navigation entry.
- Rendered tenant-local source catalog metrics, source table, status badges, and metadata JSON.
- Rebuilt and restarted the Compose web container with the new route.

Files changed:

- `docs/build_journal.md`
- `docs/gate_audit.md`
- `web/src/api/client.ts`
- `web/src/api/queries.ts`
- `web/src/components/DashboardShell.tsx`
- `web/src/components/StatusBadge.tsx`
- `web/src/router.tsx`
- `web/src/routes/SourcesRoute.tsx`
- `web/src/types.ts`

Rationale:

- G036 created the backend source catalog; exposing it in the UI makes the Sources subsystem boundary visible to operators.
- The initial UI uses `tenant-local` until tenant selection and auth exist.
- The page remains read-only and avoids fake pipeline/rule records.

Verification performed:

- Ran `npm test`; 2 files and 6 tests passed.
- Ran `npm run build`; TypeScript and Vite production build passed.
- Queried the catalog API through the web proxy at `http://localhost:15173/v1/tenants/tenant-local/catalog/sources?limit=10`.
- Ran `docker compose build web`; image build passed.
- Ran `docker compose up -d web`; web service restarted successfully.
- Queried `http://localhost:15173/sources` and received the built SPA shell.
- Queried the catalog API again through the rebuilt web container proxy.
- Ran `docker compose ps web`; service was running on host port `15173`.
- Ran `npm audit --json`; zero vulnerabilities reported.

Live verification result:

- `/sources` is routable through the production-style web container.
- Catalog API returned the seeded `tenant-local/src-massive` row through the web proxy.
- Frontend audit remains clean.
- Compose web image builds and runs with the new Sources route.

Issue found and resolved:

- No implementation issues were encountered.

Next step:

- Add catalog APIs/pages for pipelines and rules, or add tenant/source selection once auth and tenant context are introduced.


## 2026-07-08T05:21:42Z

Summary:

- Completed G038 by adding the backend pipeline catalog foundation.
- Added `catalog_pipelines` migration with a seeded local Massive raw ingest pipeline for `tenant-local/pipeline-massive-raw-ingest`.
- Added storage contracts and Postgres upsert/list support for catalog pipelines.
- Added gateway API `GET /v1/tenants/{tenant_id}/catalog/pipelines`.
- Updated API and deployment documentation for the new catalog endpoint.

Files changed:

- `docs/api.md`
- `docs/build_journal.md`
- `docs/deployment.md`
- `docs/gate_audit.md`
- `internal/api/router.go`
- `internal/api/router_test.go`
- `internal/storage/storage.go`
- `internal/storage/postgres/repository.go`
- `internal/storage/postgres/repository_test.go`
- `migrations/000003_catalog_pipelines.up.sql`
- `migrations/000003_catalog_pipelines.down.sql`

Rationale:

- The pipeline catalog makes processing topology visible as a durable subsystem boundary before adding the frontend Pipelines page.
- The seed records the current Massive scheduled-pull path without implying full intraday stream ingestion.
- The shape keeps source, pipeline, stages, datasets, output topics, and provider metadata explicit for later heterogeneous ingestion use cases.

Verification performed:

- Ran Dockerized `gofmt` over modified Go files.
- Ran focused Dockerized Go tests for `./internal/api ./internal/storage ./internal/storage/postgres`; all passed.
- Ran Dockerized full Go tests with `go test ./...`; all packages passed.
- Ran `docker compose config --quiet`; Compose config passed.
- Ran `make compose-storage-migrate`; migration `000003_catalog_pipelines` applied and seeded the Massive pipeline.
- Ran `docker compose build gateway`; gateway image build passed and Dockerfile test stage passed.
- Ran `docker compose up -d gateway`; gateway restarted successfully.
- Ran Postgres integration test `TestRepositoryAgainstPostgres`; pipeline upsert/list checks passed.
- Queried live `GET /v1/tenants/tenant-local/catalog/pipelines?limit=10` through `localhost:18000`.
- Queried the same endpoint through the web proxy at `localhost:15173`.
- Queried Postgres `catalog_pipelines` rows directly.

Live verification result:

- Gateway catalog API returned `tenant-local/pipeline-massive-raw-ingest` with source `src-massive`, status `active`, stages `scheduled_pull`, `raw_event_build`, `broker_publish`, `raw_ledger_persist`, and `idempotency_persist`.
- Web container proxy forwarded the pipeline catalog API correctly.
- Postgres contained the seeded local pipeline and the integration-test `tenant-1/pipeline-massive-raw-ingest` row.

Issue found and resolved:

- Host `gofmt` was unavailable, so formatting was performed through the Dockerized Go toolchain.

Next step:

- Add frontend Pipelines page consumption of `/v1/tenants/{tenant_id}/catalog/pipelines`.


## 2026-07-08T05:46:15Z

Summary:

- Started G039 by adding a frontend Pipelines page wired to the backend pipeline catalog API.
- Added catalog pipeline TypeScript types, API client method, and TanStack Query hook.
- Added `/pipelines` route and top navigation entry.
- Rendered tenant-local pipeline catalog metrics, pipeline table, stage flow, inputs, outputs, status badges, and metadata JSON.

Files changed:

- `docs/build_journal.md`
- `docs/gate_audit.md`
- `web/src/api/client.ts`
- `web/src/api/queries.ts`
- `web/src/components/DashboardShell.tsx`
- `web/src/router.tsx`
- `web/src/routes/PipelinesRoute.tsx`
- `web/src/types.ts`

Rationale:

- G038 created the backend pipeline catalog; exposing it in the UI makes processing topology visible to operators.
- The initial page uses `tenant-local` until tenant selection and auth exist.
- The page remains read-only and displays only real catalog pipeline data.

Verification performed:

- Ran `npm test`; 2 files and 6 tests passed.
- Ran `npm run build`; TypeScript and Vite production build passed.
- Ran `docker compose build web`; image build passed and production build ran inside the image.
- Ran `docker compose up -d web`; web service restarted successfully.
- Queried `http://localhost:15173/pipelines` and received the built SPA shell.
- Queried `http://localhost:15173/v1/tenants/tenant-local/catalog/pipelines?limit=10` through the web proxy.
- Ran `docker compose ps web`; service was running on host port `15173`.
- Ran `npm audit --json`; zero vulnerabilities reported.

Live verification result:

- `/pipelines` is routable through the production-style web container.
- Catalog API returned the seeded `tenant-local/pipeline-massive-raw-ingest` row through the web proxy.
- Frontend audit remains clean.
- Compose web image builds and runs with the new Pipelines route.

Issue found and resolved:

- No implementation issues were encountered.

Next step:

- Add rules catalog foundation, then expose Rules in the UI.


## 2026-07-08T05:54:23Z

Summary:

- Started G040 by adding the backend rules catalog foundation.
- Added `catalog_rules` migration with a seeded local Massive EOD price quality rule for `tenant-local/rule-marketdata-eod-price-quality`.
- Added storage contracts and Postgres upsert/list support for catalog rules.
- Added gateway API `GET /v1/tenants/{tenant_id}/catalog/rules`.
- Updated API and deployment documentation for the new rules catalog endpoint.

Files changed:

- `docs/api.md`
- `docs/build_journal.md`
- `docs/deployment.md`
- `docs/gate_audit.md`
- `internal/api/router.go`
- `internal/api/router_test.go`
- `internal/storage/storage.go`
- `internal/storage/postgres/repository.go`
- `internal/storage/postgres/repository_test.go`
- `migrations/000004_catalog_rules.up.sql`
- `migrations/000004_catalog_rules.down.sql`

Rationale:

- The rules catalog makes decision logic discoverable as a durable subsystem boundary before the signal engine executes rules.
- The seeded rule is catalog-only and scoped to Massive EOD equity data quality, matching the current non-intraday ingestion scope.
- The shape keeps rule type, severity, version, source/pipeline linkage, dataset/entity scope, expression JSON, actions, and metadata explicit for later heterogeneous stream use cases.

Verification performed:

- Ran Dockerized `gofmt` over modified Go files.
- Ran `docker compose config --quiet`; Compose config passed.
- Ran focused Dockerized Go tests for `./internal/api ./internal/storage ./internal/storage/postgres`; all passed.
- Ran Dockerized full Go tests with `go test ./...`; all packages passed.
- Ran `make compose-storage-migrate`; migration `000004_catalog_rules` applied and seeded the Massive EOD price quality rule.
- Ran `docker compose build gateway`; gateway image build passed and Dockerfile test stage passed.
- Ran `docker compose up -d gateway`; gateway restarted successfully.
- Ran Postgres integration test `TestRepositoryAgainstPostgres`; rule upsert/list checks passed.
- Queried live `GET /v1/tenants/tenant-local/catalog/rules?limit=10` through `localhost:18000`.
- Queried the same endpoint through the web proxy at `localhost:15173`.
- Queried Postgres `catalog_rules` rows directly.

Live verification result:

- Gateway catalog API returned `tenant-local/rule-marketdata-eod-price-quality` with type `quality_check`, severity `medium`, status `active`, source `src-massive`, pipeline `pipeline-massive-raw-ingest`, dataset scope `equity_eod_prices`, and actions `emit_alert` plus `mark_event_quality_failed`.
- Web container proxy forwarded the rules catalog API correctly.
- Postgres contained the seeded local rule.

Issue found and resolved:

- No implementation issues were encountered.

Next step:

- Write the frontend-agent specification for implementing Rules UI against the new API.


## 2026-07-08T06:02:03Z

Summary:

- Wrote the frontend-agent implementation specification for the Rules catalog UI.
- Placed the specification in `docs/frontend/rules_ui_implementation_spec.md`.
- Grounded the handoff in the G040 backend rules API and the existing Sources/Pipelines frontend patterns.

Files changed:

- `docs/build_journal.md`
- `docs/gate_audit.md`
- `docs/frontend/rules_ui_implementation_spec.md`

Rationale:

- The frontend agent now has a concrete implementation contract for G041 before making UI changes.
- The spec explicitly limits the page to read-only catalog visibility and avoids implying rule execution or management capabilities that do not exist yet.

Verification performed:

- Reviewed the new specification header and backend contract section.
- Searched for G041, rules API, and validation references in the new spec.

Live verification result:

- Not applicable; this was a documentation handoff deliverable.

Next step:

- Hand G041 to the frontend agent for implementation of the Rules catalog page.


## 2026-07-08T06:30:17Z

Summary:

- Implemented G041 by adding the read-only Rules catalog page to the `web/` frontend, backed by the G040 `GET /v1/tenants/{tenant_id}/catalog/rules` API.
- Added `CatalogRule`/`CatalogRulesResponse` types, a `listCatalogRules` client method, and a `useCatalogRules` TanStack Query hook mirroring the Sources/Pipelines conventions.
- Added `web/src/routes/RulesRoute.tsx` — a plain HTML table matching Sources/Pipelines (not AG Grid), with Registered/Active/Rule Types/Critical-High metrics, columns Rule/Type/Severity/Scope/Actions/Status/Updated, and Rule Expressions + Rule Metadata JSON sections.
- Wired `/rules` into the router (lazy-loaded) and added a Rules nav item (`ShieldCheck`) to the shell.
- Tightened `docs/frontend/rules_ui_implementation_spec.md` with minor clarifications: plain-table (not AG Grid), the `400 missing_path` path-segment error, the severity DB-`CHECK`-only contract, the compound Rule-cell layout, and the Compose validation prerequisite (full stack up + migration `000004`).

Files changed:

- `docs/frontend/rules_ui_implementation_spec.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`
- `web/src/types.ts`
- `web/src/api/client.ts`
- `web/src/api/queries.ts`
- `web/src/router.tsx`
- `web/src/components/DashboardShell.tsx`
- `web/src/routes/RulesRoute.tsx`

Rationale:

- The Rules page is the read-only catalog peer of Sources/Pipelines and reuses their exact patterns (plain table, shared components, same client/query/router/nav conventions) to avoid divergence.
- The spec clarifications prevent an implementer from reaching for AG Grid or hitting an unseeded/migrated stack during Compose validation.

Verification performed:

- `cd web && npm test` — 2 files, 6 tests passed.
- `cd web && npm run build` — `tsc && vite build` succeeded; `RulesRoute` is a lazy chunk.
- `cd web && npm audit` — 0 vulnerabilities.
- `curl -fsS http://localhost:18000/healthz` — gateway healthy (G040).
- `curl 'http://localhost:18000/v1/tenants/tenant-local/catalog/rules?limit=10'` — returned seeded rule `rule-marketdata-eod-price-quality`.
- `docker compose build web` — web image built.
- `docker compose up -d web` — web container Up on `:15173`.
- `curl -fsS http://localhost:15173/rules` — served the SPA shell.
- `curl -fsS 'http://localhost:15173/v1/tenants/tenant-local/catalog/rules?limit=10'` — returned the seeded rule through the nginx `/v1/` proxy.
- `docker compose ps web` — container Up, `15173->8080`.

Live verification result:

- Build, tests, and audit all pass; the page adds no new dependencies.
- The rules API response shape matches `CatalogRule` (severity `medium`, status `active`, json_logic expression, actions, metadata).
- Compose web serves `/rules` as the SPA shell and proxies the rules API to the gateway.

Issue found and resolved:

- None. The prescribed client/query/router snippets matched the post-G040 `web/` conventions verbatim.

Next step:

- Browser validation of the Rules page (rendering, console errors) as a manual follow-up.
- Future: rule execution history, expression builder, and edit/management remain out of scope pending backend support.


## 2026-07-08T20:01:14Z

Summary:

- Passed G042 by persisting generic `POST /v1/events/raw` publications to `raw_event_ledger` and `idempotency_records` after broker acknowledgement.
- Added one atomic PostgreSQL transaction for the paired audit records and adopted it in the Massive scheduled publisher.
- Added pre-publish persistence-envelope validation and explicit post-acknowledgement persistence failure semantics.
- Wrote the G043 frontend-agent specification for a first-class operational Dashboard.

Files changed:

- `cmd/gateway/main.go`
- `internal/api/router.go`
- `internal/api/router_test.go`
- `internal/storage/storage.go`
- `internal/storage/postgres/repository.go`
- `internal/adapters/marketdata/massive/scheduled_pull.go`
- `internal/adapters/marketdata/massive/scheduled_pull_test.go`
- `docs/api.md`
- `docs/deployment.md`
- `docs/frontend_implementation_spec.md`
- `docs/frontend/dashboard_ui_implementation_spec.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Rationale:

- Broker acknowledgement alone did not make generic gateway ingestion visible to operational queries. The gateway now returns `202` only after durable publication and atomic audit persistence.
- The paired transaction prevents a raw-ledger row and its idempotency row from diverging during a database failure.
- G043 can now treat accepted generic events as visible in the Dashboard event stream.

Verification performed:

- `make docker-test` - all Go packages passed.
- `docker compose build gateway` - image build and embedded Go tests passed.
- `docker compose up -d postgres redpanda gateway` - rebuilt gateway deployed.
- Posted heterogeneous IoT-shaped event `g042-live-event` to `POST /v1/events/raw`.
- Queried the raw-event and idempotency APIs and joined both rows directly in PostgreSQL.

Live verification result:

- Broker acknowledgement: topic `signalops.local.raw.v1`, partition `2`, offset `5`.
- Raw ledger retained tenant/source/dataset identity, observation and processing times, original payload, entity hints, and broker coordinates.
- Idempotency status was `published` with the same coordinates and hash prefix `sha256:a805706dc`.
- PostgreSQL join returned exactly the paired G042 records.

Residual semantics:

- A `503 persistence_failed` means Redpanda accepted the event but PostgreSQL did not confirm the audit transaction. Retrying can produce broker duplicates; clients and consumers must reuse stable identifiers and remain idempotent.

Next step:

- Frontend agent implements G043 from `docs/frontend/dashboard_ui_implementation_spec.md`.


## 2026-07-08T20:53:18Z

Summary:

- Implemented G043 by promoting `/` into a first-class operational Dashboard that composes the existing health, runs, raw events, provider usage, sources, pipelines, and rules data.
- Added `web/src/routes/DashboardRoute.tsx` composing existing TanStack Query hooks (health/readiness, 10 runs, 10 raw events, the three catalogs) plus a new `useRecentProviderUsage(50)` hook for unfiltered provider usage.
- The Dashboard consumes the already-global SSE subscription (`DashboardStreamBridge` in `App.tsx`) for cache invalidation and reads stream connection/freshness state from `useUi` — it does not mount a second EventSource.
- Swapped the index route `/` from `RunsRoute` to `DashboardRoute` (`/runs` unchanged); prepended a Dashboard nav item (`LayoutDashboard`).
- Layout: page header (UTC updated time, stream state, refresh), metrics strip, Processing Health + Catalog Inventory band, Recent Runs + Provider Usage band, and a full-width Recent Event Stream. Per-widget loading/error/empty states; event rows link to plain `/raw-events`.
- Tightened `docs/frontend/dashboard_ui_implementation_spec.md`: pointed §3 at the existing global `DashboardStreamBridge` (reuse, do not duplicate), corrected the channel-vs-event-name wording, and stated the provider-usage hook fix and the plain `/raw-events` link.

Files changed:

- `docs/frontend/dashboard_ui_implementation_spec.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`
- `web/src/api/queries.ts`
- `web/src/router.tsx`
- `web/src/components/DashboardShell.tsx`
- `web/src/routes/DashboardRoute.tsx`

Rationale:

- The Dashboard is a composition surface over existing endpoints and the existing SSE bridge — no new state library, chart library, component framework, or mock data.
- Reusing the global `DashboardStreamBridge` avoids a second EventSource and keeps one subscription cleaned up in `App.tsx`.

Verification performed:

- `cd web && npm test` — 2 files, 6 tests passed.
- `cd web && npm run build` — `tsc && vite build` succeeded; `DashboardRoute` is a lazy chunk.
- `cd web && npm audit` — 0 vulnerabilities.
- `docker compose build web` — web image rebuilt with the new route.
- `docker compose up -d web` — container Up on `:15173`.
- `curl -fsS http://localhost:15173/` — served the SPA shell (Dashboard at `/`).
- `curl -fsS http://localhost:15173/healthz` — gateway healthy.
- `curl -fsS 'http://localhost:15173/v1/provider-usage?limit=5'` — unfiltered provider usage returned rows.
- `curl -sN --max-time 2 'http://localhost:15173/v1/streams/dashboard?channels=health,heartbeat'` — emitted `event: heartbeat` and `event: health` through the nginx proxy.
- `docker compose ps web` — container Up, `15173->8080`.

Live verification result:

- Build, tests, and audit pass; the page adds no new dependencies.
- `/` serves the Dashboard; all seven data areas are wired to live endpoints through the proxy.
- Unfiltered provider usage and the SSE `health`/`heartbeat` events flow correctly, matching the global bridge's invalidation map.

Issue found and resolved:

- None. The provider-usage hook (`useProviderUsage`) was gated on `run_id`; added `useRecentProviderUsage` (always enabled, reuses the `['provider-usage']` key prefix so the bridge invalidates it). The SSE subscription was already global in `App.tsx`, so no second subscription was needed.

Next step:

- Browser/Playwright validation (rendering, console errors, 375px layout) as a manual follow-up — documented as a residual gap per the spec.
- Future: alerts, timeline/correlation, insights, and rule execution remain out of scope pending backend contracts.


## 2026-07-08T21:18:15Z

Summary:

- Passed G044 by adding a standalone Go normalization service between durable raw ingestion and Python algorithm execution.
- Added migration `000005_normalized_events` and `normalized_event_ledger` with canonical payload, entities, evidence, metadata, complete event JSON, and raw-to-normalized broker lineage.
- Added normalized-event list/detail APIs and moved the Compose Python worker to `signalops.local.normalized.v1` under consumer group `signalops.normalized-worker.v1`.
- Added invalid-contract DLQ handling; infrastructure failures retain uncommitted source offsets for retry.

Files changed:

- `cmd/normalizer/main.go`
- `internal/normalization/processor.go`
- `internal/normalization/processor_test.go`
- `internal/storage/storage.go`
- `internal/storage/postgres/repository.go`
- `internal/api/router.go`
- `internal/api/router_test.go`
- `migrations/000005_normalized_events.up.sql`
- `migrations/000005_normalized_events.down.sql`
- `Dockerfile`
- `compose.yaml`
- `docs/api.md`
- `docs/deployment.md`
- `docs/python_worker.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Runtime contract:

- Go normalizer consumes `signalops.<env>.raw.v1`.
- Identity normalization preserves heterogeneous payloads as `normalized_payload`, converts entity hints to canonical entity references, attaches raw evidence and lineage, and publishes `signalops.<env>.normalized.v1`.
- PostgreSQL persistence completes before the source offset is committed.
- Python algorithms now consume normalized events rather than raw provider envelopes.

Verification performed:

- `make docker-test` - all Go packages passed.
- `make docker-test-python` - 37 Python tests passed.
- `docker compose config --quiet` - passed.
- Applied migration `000005_normalized_events` through the migration container.
- Built and deployed `gateway`, `normalizer`, and `raw-worker` with Compose.
- Posted heterogeneous IoT event `g044-live-event` through the gateway.
- Queried `/v1/normalized-events/g044-live-event`, PostgreSQL, and both consumer groups.

Live verification result:

- Raw acknowledgement: `signalops.local.raw.v1`, partition `2`, offset `6`.
- Normalized publication: `signalops.local.normalized.v1`, partition `2`, offset `2`.
- Persisted schema was `signalops.normalized_signal_event.v1`, confidence `1`, with canonical sensor entity, raw evidence, and the original heterogeneous measurements.
- Normalizer group `signalops.normalizer.v1` was Stable with lag `0`.
- Python group `signalops.normalized-worker.v1` was Stable with lag `0` and processed the live normalized event.
- Historical malformed raw test records were routed to the normalization DLQ instead of blocking partitions.
- The persisted live normalized event passed runtime validation against `normalized_signal_event.v1.schema.json`.

Issue found and resolved:

- The first normalizer restart exited when franz-go reported `ErrDataLoss` after a partition recovery reset. The broker consumer now recognizes that typed recovery notification and continues polling from the broker-selected offset. Rebuild/restart validation showed the container remained Up and `signalops.normalizer.v1` returned to Stable with one member and lag `0`.

Next step:

- Add durable Signal persistence and a read API for Python-emitted `signal.v1` events.


## 2026-07-08T21:41:02Z

Summary:

- Passed G045 by adding a standalone Go signal persister for Python-emitted `signal.v1` events.
- Added migration `000006_signal_ledger`, typed storage/query contracts, strict signal validation, signal list/detail APIs, and durable broker lineage.
- Added invalid-signal DLQ routing and retained uncommitted offsets for infrastructure failures.
- Corrected Python fallback evidence to identify detector inputs as `normalized_event` rather than `raw_event`.

Files changed:

- `cmd/signal-persister/main.go`
- `internal/signals/processor.go`
- `internal/signals/processor_test.go`
- `internal/storage/storage.go`
- `internal/storage/postgres/repository.go`
- `internal/api/router.go`
- `internal/api/router_test.go`
- `migrations/000006_signal_ledger.up.sql`
- `migrations/000006_signal_ledger.down.sql`
- `python/signalops_workers/worker.py`
- `python/tests/test_worker.py`
- `Dockerfile`
- `compose.yaml`
- `docs/api.md`
- `docs/deployment.md`
- `docs/python_worker.md`
- `docs/frontend_implementation_spec.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Runtime contract:

- Python algorithm workers remain the owners of detection and publish validated `signal.v1` events to Redpanda.
- The Go signal persister independently validates the closed contract, persists canonical detector output and normalized `event_ids`, then commits the signal-topic offset.
- Invalid events route to the algorithm DLQ; PostgreSQL failures remain retryable without offset commit.

Verification performed:

- `make docker-test` - all Go packages passed.
- `make docker-test-python` - 37 Python tests passed.
- `docker compose config --quiet` and `git diff --check` passed.
- Applied migration `000006_signal_ledger` through the migration container.
- Built and deployed `gateway` and `signal-persister` with Compose.
- Ran one deterministic `signalops.static_test` Python worker invocation against the normalized topic.
- Queried `/v1/signals/signalops.static_test.low`, PostgreSQL, and the signal-persister consumer group.

Live verification result:

- Python emitted `signalops.static_test.low` to `signalops.local.signal.v1`, partition `0`, offset `3`.
- Go persisted tenant `tenant-local`, detector version `0.1.0`, model `none`, severity `low`, confidence `0.25`, and normalized event ID `evt_5d5a94a0e8ea5d149ec19947`.
- The signal detail API returned the complete canonical event and broker coordinates.
- Consumer group `signalops.signal-persister.v1` was Stable with one member and total lag `0`.
- The final persisted event passed runtime `signal.v1` validation, used `normalized_event` evidence, and the filtered list API returned the expected tenant/detector/severity result.
- Restart validation left `signal-persister` Up and its committed group Stable with lag `0`.

Next step:

- Add a frontend Signals explorer and Dashboard signal metrics, or proceed backend-first with durable alert/insight lifecycle storage.


## 2026-07-08T22:13:05Z

Summary:

- Implemented G046 by adding read-only Normalized Events (`/normalized-events`) and Signals (`/signals`) pages backed by the G044/G045 APIs, plus Dashboard integration.
- Corrected the G046 spec's backend contract before implementing: `schema_name`→`schema_id`, `metrics`→`supporting_metrics`, `signal_topic/partition/offset`→`broker_topic/partition/offset`, removed the non-existent `model_id` (only `model_version`), made broker coords + `window_*` required (no `omitempty`/`NOT NULL`), removed the fabricated `400 invalid_limit` (limit is clamped), and fixed the example envelopes.
- Added corrected types, `listNormalizedEvents`/`getNormalizedEvent`/`listSignals`/`getSignal` client methods, and `useNormalizedEvents`/`useNormalizedEvent`/`useSignals`/`useSignal` hooks.
- Added `NormalizedEventsRoute` and `SignalsRoute` (plain tables + metrics + detail panels with `JsonViewer`, local selection state, loading/error/empty states, severity badge); signal `event_ids` link to plain `/normalized-events`.
- Wired `/normalized-events` + `/signals` routes and nav items (`FileCheck2`, `Radar`); added Normalized + Signals metric tiles and a compact Recent Signals widget to the Dashboard (no second EventSource — the global `DashboardStreamBridge` remains the single subscription).

Files changed:

- `docs/frontend/normalized_signals_ui_implementation_spec.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`
- `web/src/types.ts`
- `web/src/api/client.ts`
- `web/src/api/queries.ts`
- `web/src/router.tsx`
- `web/src/components/DashboardShell.tsx`
- `web/src/routes/NormalizedEventsRoute.tsx`
- `web/src/routes/SignalsRoute.tsx`
- `web/src/routes/DashboardRoute.tsx`

Rationale:

- The spec's prescribed TypeScript interfaces did not match the G044/G045 DTOs; implementing against them would have produced blank columns and detail panels, so the contract was corrected first.
- The pages reuse the existing `RawEventsRoute`/catalog conventions (plain tables, shared components, TanStack Query hooks) — no new libraries or mock data.

Verification performed:

- `cd web && npm test` — 2 files, 6 tests passed.
- `cd web && npm run build` — `tsc && vite build` succeeded; `NormalizedEventsRoute` and `SignalsRoute` are lazy chunks.
- `cd web && npm audit` — 0 vulnerabilities.
- `curl 'http://localhost:18000/v1/normalized-events?tenant_id=tenant-local&limit=3'` — returned normalized event `g044-live-event` with `schema_id` (not `schema_name`), `idempotency_key`, and raw/normalized broker coords.
- `curl 'http://localhost:18000/v1/signals?tenant_id=tenant-local&limit=3'` — returned signal `signalops.static_test.low` with `model_version` (no `model_id`).
- `docker compose build web` + `docker compose up -d web` — container Up on `:15173`.
- `curl -fsS http://localhost:15173/` — served the SPA shell.
- `curl 'http://localhost:15173/v1/normalized-events?…'` and `…/v1/signals?…` — returned live data through the nginx proxy.

Live verification result:

- Build, tests, and audit pass; the pages add no new dependencies.
- Live responses confirm the corrected field names (`schema_id`, `model_version`, no `model_id`) and the extra DTO fields documented in the spec.
- Compose web serves `/`, `/normalized-events`, and `/signals` data through the proxy.

Issue found and resolved:

- The G046 spec documented a non-existent `400 invalid_limit` and wrong field names (`schema_name`, `metrics`, `signal_topic`, `model_id`); verified against `router.go` DTOs + migrations, corrected the spec, and implemented against the real contract.

Next step:

- Browser/Playwright validation (rendering, console errors, 375px layout) as a manual follow-up.
- Alert/insight lifecycle and correlation remain out of scope pending backend contracts.

## 2026-07-08T22:33:32Z

Summary:

- Closed the G046 browser-validation gap after the frontend layout fix for wrapping navigation.
- Validated the deployed web container on `:15173`, the normalized-events and signals proxy APIs, and the Playwright screenshot artifacts from the frontend validation run.

Files changed:

- `web/src/components/DashboardShell.tsx`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Verification performed:

- `cd web && npm test` - 2 files, 6 tests passed.
- `cd web && npm run build` - TypeScript and Vite production build passed.
- `cd web && npm audit --json` - 0 vulnerabilities.
- `docker compose build web` - image build passed.
- `docker compose up -d web` - web container remained Up on `:15173`.
- `curl -fsS http://localhost:15173/` - served the SPA shell.
- `curl -fsS http://localhost:15173/normalized-events` - served the SPA shell for the route.
- `curl -fsS http://localhost:15173/signals` - served the SPA shell for the route.
- `curl -fsS 'http://localhost:15173/v1/normalized-events?tenant_id=tenant-local&limit=3'` - returned live normalized events through the nginx proxy.
- `curl -fsS 'http://localhost:15173/v1/signals?tenant_id=tenant-local&limit=3'` - returned live signals through the nginx proxy.
- Reviewed `/tmp/g046-validate/shots/summary.json` from the Playwright Docker run.
- Confirmed screenshot artifacts exist for desktop and 375px mobile views.
- `git diff --check` - passed.

Live verification result:

- Playwright summary reported no browser console warnings/errors and no page errors.
- Playwright observed exactly one dashboard SSE request: `/v1/streams/dashboard?channels=health%2Cruns%2Craw_events%2Cprovider_usage%2Cheartbeat`.
- SPA navigation reached `/normalized-events`, `/signals`, `/runs`, and `/raw-events`.
- Mobile horizontal overflow was `0px` for Dashboard, Signals, and Normalized Events at 375px width.
- Screenshot artifacts were generated:
  - `/tmp/g046-validate/shots/dashboard-desktop.png`
  - `/tmp/g046-validate/shots/dashboard-mobile.png`
  - `/tmp/g046-validate/shots/normalized-desktop.png`
  - `/tmp/g046-validate/shots/normalized-mobile.png`
  - `/tmp/g046-validate/shots/signals-desktop.png`
  - `/tmp/g046-validate/shots/signals-mobile.png`
  - `/tmp/g046-validate/shots/runs-desktop.png`
  - `/tmp/g046-validate/shots/raw-events-desktop.png`

Validation note:

- The Playwright run generated valid PNG artifacts and machine-checked console, navigation, SSE count, and mobile overflow (all green). The desktop and mobile artifacts were also visually inspected via image analysis: the desktop Dashboard renders the full metrics strip and populated widget bands; the dedicated `/signals` page renders the Signals/Detectors/High-Critical/Avg-Confidence metrics and the full signals table (Signal/Detector/Model/Source-Dataset/Severity/Confidence/Events); and the 375px mobile view shows the navigation wrapping to multiple rows and metric tiles stacking with no overflow. No G046 validation gap remains.

Next step:

- Proceed backend-first with alert/insight lifecycle persistence derived from durable signals.

## 2026-07-08T22:55:26Z

Summary:

- Passed G047 backend foundation by deriving durable alert and insight lifecycle records from validated `signal.v1` events.
- Added alert/insight migrations, storage contracts, transactional Postgres persistence, gateway list/detail APIs, unit tests, and runtime documentation.
- Extended the existing `signal-persister` service; no new worker is required. Signal source offsets remain uncommitted until signal, alert, and insight persistence succeeds.

Files changed:

- `migrations/000007_alert_insight_lifecycle.up.sql`
- `migrations/000007_alert_insight_lifecycle.down.sql`
- `internal/storage/storage.go`
- `internal/storage/postgres/repository.go`
- `internal/signals/processor.go`
- `internal/signals/processor_test.go`
- `internal/api/router.go`
- `internal/api/router_test.go`
- `docs/api.md`
- `docs/deployment.md`
- `docs/python_worker.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Runtime contract:

- Every persisted signal derives one active insight with id `insight:{signal_id}`.
- `medium`, `high`, and `critical` signals derive one open alert with id `alert:{signal_id}`.
- `info` and `low` signals remain insight-only.
- Reprocessing the same signal is idempotent and does not reset existing alert/insight lifecycle status fields.
- Operator mutation endpoints for acknowledgement, resolution, review, dismissal, and suppression remain deferred until auth/operator identity exists.

Verification performed:

- `make docker-test` - all Go packages passed.
- `make docker-test-python` - 37 Python tests passed.
- `docker compose config --quiet` - passed.
- `git diff --check` - passed.
- `docker compose --profile storage run --rm postgres-migrate` - applied `000007_alert_insight_lifecycle`.
- `docker compose build gateway signal-persister` - passed; build stage also ran `go test ./...`.
- `docker compose up -d gateway signal-persister` - both services restarted successfully.
- Published high-severity validation signal `signal-g047-high` to `signalops.local.signal.v1`.
- Queried signal, alert, insight list/detail APIs, direct PostgreSQL rows, service logs, and the signal-persister consumer group.

Live verification result:

- Redpanda accepted `signal-g047-high` at `signalops.local.signal.v1` partition `1`, offset `0`.
- `signal-persister` persisted `signal-g047-high` and derived `alert:signal-g047-high` plus `insight:signal-g047-high`.
- `GET /v1/alerts/alert:signal-g047-high` returned status `open`, severity `high`, recommendation, entities, evidence, and normalized event lineage.
- `GET /v1/insights/insight:signal-g047-high` returned status `active`, severity `high`, supporting metrics, semantic evidence, recommendation, and lineage.
- `GET /v1/alerts?tenant_id=tenant-local&status=open&limit=5` and `GET /v1/insights?tenant_id=tenant-local&status=active&limit=5` returned the expected rows.
- Direct PostgreSQL join confirmed `alert:signal-g047-high` status `open` and `insight:signal-g047-high` status `active`.
- Consumer group `signalops.signal-persister.v1` was Stable with total lag `0`.

Next step:

- Write the frontend-agent G048 specification for Alerts and Active Insights UI backed by the new G047 APIs.

## 2026-07-08T23:32:09Z

Summary:

- Implemented G048 by adding read-only Alerts (`/alerts`) and Active Insights (`/insights`) pages backed by the G047 alert/insight APIs, plus Dashboard integration.
- Evaluated the G048 spec's backend contract against the real G047 implementation before coding; it was accurate (routes, DTO field names, `omitempty`/`NOT NULL` optionality, response envelopes, `queryLimit` default 50/cap 200, error codes, and signal→alert/insight derivation all matched `internal/api/router.go`, migration `000007`, and live data), so the spec was implemented as-is with no contract edits.
- Added typed `AlertRecord`/`InsightRecord` (+filters/responses), `listAlerts`/`getAlert`/`listInsights`/`getInsight` client methods, and `useAlerts`/`useAlert`/`useInsights`/`useInsight` hooks.
- Added `AlertsRoute` and `InsightsRoute` (filters, metrics, plain tables, detail panels with `JsonViewer`, local selection, loading/error/empty states, severity + status text badges); alert/insight `signal_id` links to `/signals` and `event_ids` link to `/normalized-events`. No enabled lifecycle action controls (mutation APIs are deferred per G047).
- Wired `/alerts` + `/insights` routes and nav items (`TriangleAlert`, `Lightbulb`); added Open Alerts + Active Insights metric tiles and compact Open Alerts + Active Insights widgets to the Dashboard (metrics strip widened to a 13-column arbitrary grid; added a caption distinguishing signals vs alerts vs insights). REST is the source of truth — no second EventSource.

Files changed:

- `docs/build_journal.md`
- `docs/gate_audit.md`
- `web/src/types.ts`
- `web/src/api/client.ts`
- `web/src/api/queries.ts`
- `web/src/router.tsx`
- `web/src/components/DashboardShell.tsx`
- `web/src/routes/AlertsRoute.tsx`
- `web/src/routes/InsightsRoute.tsx`
- `web/src/routes/DashboardRoute.tsx`
- `web/src/api/alerts_insights.test.ts`

Rationale:

- The spec author had already verified the G047 DTOs, so unlike G046 there were no field-name or optionality defects to correct; the gate proceeded straight to implementation.
- The pages reuse existing `RawEventsRoute`/`SignalsRoute` conventions (plain tables, shared components, TanStack Query hooks) — no new libraries or mock data.

Verification performed:

- `cd web && npm test` — 3 files, 11 tests passed (incl. 5 new alert/insight client tests asserting filter→URL mapping, default `limit=50`, and path-id encoding).
- `cd web && npm run build` — `tsc && vite build` succeeded; `AlertsRoute` and `InsightsRoute` are lazy chunks.
- `cd web && npm audit` — 0 vulnerabilities.
- `curl -fsS http://localhost:15173/`, `/alerts`, `/insights` — served the SPA shell.
- `curl 'http://localhost:15173/v1/alerts?tenant_id=tenant-local&status=open&limit=10'` — returned `alert:signal-g047-high` (high/open) through the proxy.
- `curl 'http://localhost:15173/v1/insights?tenant_id=tenant-local&status=active&limit=10'` — returned `insight:signal-g047-high` and `insight:signalops.static_test.low`.
- `curl 'http://localhost:15173/v1/alerts/alert:signal-g047-high'` — returned the `{alert}` detail envelope.
- `curl 'http://localhost:15173/v1/insights/insight:signal-g047-high'` — returned the `{insight}` detail envelope.
- Playwright (Docker) browser validation across desktop + 375px mobile.

Live verification result:

- Build, tests, and audit pass; the pages add no new dependencies.
- Proxy serves `/`, `/alerts`, `/insights`; list and detail APIs return live G047 rows.
- Playwright: no browser console/page errors; exactly one dashboard SSE connection held across SPA navigation; nav has 12 items (Alerts, Insights added) without overlap; `/alerts` rendered 1 open alert and selecting it loaded the detail panel; `/insights` rendered 2 active insights and selecting one loaded its detail panel; the Dashboard rendered Open Alerts + Active Insights tiles and widgets; mobile horizontal overflow was `0px` for Dashboard, Alerts, and Insights.

Issue found and resolved:

- The Playwright run surfaced a stale `dashboard-desktop.png` (filename collision with the G046 run's CDN upload), which an image read mis-attributed to the G048 dashboard. Resolved by confirming the live DOM via `body.innerText` (`Open Alerts`, `Active Insights`, and the signals/alerts/insights caption all present) and re-capturing under a unique filename, which visually confirmed the G048 dashboard tiles, widgets, and caption.

Next step:

- Add authenticated lifecycle mutation APIs (acknowledge/resolve/review/dismiss/suppress) and wire the corresponding UI controls when operator identity is in place.

## 2026-07-09T00:13:36Z

Summary:

- Implemented G049 backend lifecycle mutation APIs for alert acknowledgement/resolution/suppression and insight review/dismiss/archive.
- Added storage mutation contracts and Postgres update methods that preserve existing rows, set lifecycle audit columns, merge `metadata.lifecycle`, and return the updated alert or insight envelope.
- Added gateway POST routes using explicit action endpoints. Operator identity is read from `X-SignalOps-Actor`, then body `actor`, then `operator-local` until auth is introduced.
- Added router unit tests for alert acknowledgement and insight archive transitions.

Files changed:

- `internal/storage/storage.go`
- `internal/storage/postgres/repository.go`
- `internal/api/router.go`
- `internal/api/router_test.go`
- `docs/api.md`
- `docs/deployment.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Rationale:

- G047 already created the required lifecycle columns, so G049 intentionally avoids a new migration or worker.
- Fixed action endpoints keep lifecycle state changes explicit for future UI controls and eventual auth/audit enforcement.

Verification performed:

- `docker run --rm -v ... golang:1.22-bookworm go test ./internal/api ./internal/storage/postgres ./cmd/gateway` - passed.
- `make docker-test` - all Go packages passed.
- `make docker-test-python` - 37 Python tests passed.
- `docker compose config --quiet` - passed.
- `docker compose build gateway` - passed; build stage also ran `go test ./...`.
- `docker compose up -d gateway` - restarted the gateway with the G049 image.
- `git diff --check` - passed.
- Published validation signal `signal-g049-high` to `signalops.local.signal.v1` partition `2`, offset `0`.
- Queried `GET /v1/signals/signal-g049-high`, `GET /v1/alerts/alert:signal-g049-high`, and `GET /v1/insights/insight:signal-g049-high`.
- Exercised all six lifecycle mutation endpoints against the G049 rows.
- Queried direct PostgreSQL rows and the `signalops.signal-persister.v1` consumer group.

Live verification result:

- `signal-persister` persisted `signal-g049-high` and derived `alert:signal-g049-high` plus `insight:signal-g049-high`.
- `POST /v1/alerts/alert:signal-g049-high/acknowledge` returned status `acknowledged`, `acknowledged_by=operator-g049`, and lifecycle metadata action `acknowledge`.
- `POST /v1/alerts/alert:signal-g049-high/resolve` returned status `resolved`, `resolved_by=operator-g049`, and lifecycle metadata action `resolve`.
- `POST /v1/alerts/alert:signal-g049-high/suppress` returned status `suppressed` and lifecycle metadata action `suppress`.
- `POST /v1/insights/insight:signal-g049-high/review` returned status `reviewed`, `reviewed_by=operator-g049`, and lifecycle metadata action `review`.
- `POST /v1/insights/insight:signal-g049-high/archive` returned status `archived` and lifecycle metadata action `archive`.
- `POST /v1/insights/insight:signal-g049-high/dismiss` returned status `dismissed` and lifecycle metadata action `dismiss`.
- Direct PostgreSQL confirmed final alert status `suppressed` with acknowledged/resolved actors preserved, and final insight status `dismissed` with reviewed actor preserved.
- Consumer group `signalops.signal-persister.v1` was Stable with total lag `0`.

Next step:

- Write the frontend-agent G050 specification for lifecycle action controls on Alerts and Active Insights, using these backend-ready endpoints.

## 2026-07-09T01:00:56Z

Summary:

- Wrote the G050 frontend-agent implementation specification for Alert and Active Insight lifecycle controls backed by the G049 mutation APIs.
- The spec covers alert acknowledge/resolve/suppress and insight review/dismiss/archive controls, placeholder operator identity, typed client mutations, TanStack Query invalidation, dashboard cache impact, tests, live proxy checks, browser validation, and journal/audit requirements.

Files changed:

- `docs/frontend/alerts_insights_lifecycle_controls_spec.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Rationale:

- G049 made lifecycle mutation APIs backend-ready. G050 should be a frontend-only gate that extends the existing G048 pages without adding unsupported auth, streaming, bulk action, or backend behavior.

Verification performed:

- Specification reviewed against `docs/api.md` and the passed G049 gate evidence in `docs/gate_audit.md`.

Next step:

- Frontend-agent implements G050 using `docs/frontend/alerts_insights_lifecycle_controls_spec.md`.

## 2026-07-09T01:41:23Z

Summary:

- Implemented G050 by adding operator lifecycle controls to the existing G048 Alerts and Insights pages, backed by the committed G049 mutation APIs.
- Evaluated the G050 spec against the committed G049 backend (`router.go` mutation routes/handlers, `lifecycleActor` precedence, `lifecycleMetadata` jsonb merge, error codes) before coding; the contract was accurate, so the spec was implemented as-is with no edits.
- Added a shared `post<T>` helper (Content-Type JSON, same `ApiError` parsing) and `mutateAlertLifecycle`/`mutateInsightLifecycle` (action in path, `encodeURIComponent`'d id, placeholder `X-SignalOps-Actor: operator-local` header); `useMutateAlertLifecycle`/`useMutateInsightLifecycle` write the returned record into the detail cache and invalidate the `['alerts']`/`['insights']` list prefix (covers detail, tables, and Dashboard summaries).
- Added Acknowledge/Resolve/Suppress (`AlertsRoute`) and Review/Dismiss/Archive (`InsightsRoute`) controls to the detail panels with spec-compliant disabled logic (e.g., Acknowledge disabled once acknowledged/resolved/suppressed; all disabled while in-flight), an inline error line, a compact lifecycle summary (action/actor/note|reason/mutated_at), and the existing `acknowledged_*`/`resolved_*`/`reviewed_*` field display. No enabled auth, no SSE/audit/bulk, no modals (per Out of Scope).
- The detail body is keyed by record id so in-flight/error state resets cleanly on selection change; after a mutation the record stays selected and renders backend-returned state (even if it leaves the active status filter).

Files changed:

- `docs/build_journal.md`
- `docs/gate_audit.md`
- `web/src/types.ts`
- `web/src/api/client.ts`
- `web/src/api/queries.ts`
- `web/src/routes/AlertsRoute.tsx`
- `web/src/routes/InsightsRoute.tsx`
- `web/src/api/alerts_insights.test.ts`

Rationale:

- The spec author had already verified the G049 DTOs/routes, so (as with G048) there were no contract defects to correct.
- Controls reuse existing inline button styling and TanStack Query conventions — no new libraries or components.

Verification performed:

- `cd web && npm test` — 3 files, 18 tests passed (incl. 7 new lifecycle mutation tests: action→path mapping for all six actions, POST + Content-Type + `X-SignalOps-Actor` header, error-envelope parsing).
- `cd web && npm run build` — `tsc && vite build` succeeded.
- `cd web && npm audit` — 0 vulnerabilities.
- `docker compose config --quiet` — OK.
- `docker compose build web` + `up -d web` — container Up on `:15173`.
- Live mutation curls through the proxy:
  - `POST /v1/alerts/alert:signal-g049-high/acknowledge` (`X-SignalOps-Actor` header) → status `acknowledged`, `acknowledged_at`/`acknowledged_by` set, `metadata.lifecycle` written.
  - `POST /v1/insights/insight:signal-g049-high/review` (body actor) → status `reviewed`, `reviewed_at`/`reviewed_by` set, `metadata.lifecycle` written.
  - `POST /v1/alerts/alert:does-not-exist/acknowledge` → `404 alert_not_found`.
- Playwright (Docker) desktop + 375px mobile browser validation.
- Validation data prep: the G049 demo rows were left in terminal states by the evaluation live-checks, so they were reset to `open`/`active` via `UPDATE ... metadata - 'lifecycle'` before live/browser validation.

Live verification result:

- Build, tests, audit, and compose config pass; the controls add no new dependencies.
- Both actor paths (`X-SignalOps-Actor` header and body `actor`) resolve to `operator-local` as specified.
- Playwright: no console/page errors; exactly one dashboard SSE connection across SPA nav; nav has 12 items; `/alerts` and `/insights` render the controls; clicking Acknowledge and Review updated state from the backend (Acknowledge/Review buttons correctly disabled afterward, lifecycle summary rendered); Dashboard Open Alerts dropped `2→1` and Active Insights dropped `3→2` after the mutations (cache invalidation refreshed summaries); `0px` mobile overflow on `/alerts` and `/insights`. A screenshot of the acknowledged alert detail confirms status `acknowledged`, the disabled Acknowledge button, the lifecycle summary, and the Acknowledged timestamp/by fields.

Issue found and resolved:

- An initial browser assertion used `getByText('acknowledged',{exact:true})`, which returned false despite the transition succeeding; the screenshot confirmed the status did update. Replaced the redundant text assertion with a behavioral one (the Acknowledge/Review button becomes disabled after the action), which directly verifies the spec's disabled-state logic and passes.

Next step:

- Add real operator authentication/identity (replacing the placeholder `operator-local` actor) and full lifecycle audit history when auth lands.

## 2026-07-09T01:52:52Z

Summary:

- Validated the frontend-agent G050 implementation for alert and insight lifecycle controls.
- Confirmed TimescaleDB remains undeployed in the current Compose architecture and documented it as an essential future storage maturity gate.

Files reviewed/validated:

- `web/src/types.ts`
- `web/src/api/client.ts`
- `web/src/api/queries.ts`
- `web/src/routes/AlertsRoute.tsx`
- `web/src/routes/InsightsRoute.tsx`
- `web/src/api/alerts_insights.test.ts`
- `docs/deployment.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Validation performed:

- Reviewed implementation diff against `docs/frontend/alerts_insights_lifecycle_controls_spec.md`.
- `cd web && npm test` - passed: 3 files, 18 tests.
- `cd web && npm run build` - passed.
- `cd web && npm audit --json` - 0 vulnerabilities.
- `docker compose config --quiet` - passed.
- `docker compose build web` - passed.
- `docker compose up -d web` - web service running.
- Published fresh validation signal `signal-g050-high` to `signalops.local.signal.v1` partition `0`, offset `4`.
- Verified `signal-persister` persisted `signal-g050-high` and derived `alert:signal-g050-high` plus `insight:signal-g050-high`.
- Verified browser-facing proxy served `/alerts` and `/insights` SPA routes.
- Verified `GET /v1/alerts/alert:signal-g050-high` and `GET /v1/insights/insight:signal-g050-high` through `localhost:15173` before mutation.
- Exercised `POST /v1/alerts/alert:signal-g050-high/suppress` and `POST /v1/insights/insight:signal-g050-high/archive` through `localhost:15173`.
- Queried direct PostgreSQL lifecycle rows and the `signalops.signal-persister.v1` consumer group.
- Reviewed frontend-agent browser validation summary at `/tmp/g050-validate/shots/summary.json`.

Live verification result:

- Alert lifecycle POST through the web proxy returned status `suppressed`, actor `operator-local`, and `metadata.lifecycle.action=suppress`.
- Insight lifecycle POST through the web proxy returned status `archived`, `reviewed_by=operator-local`, and `metadata.lifecycle.action=archive`.
- Direct PostgreSQL confirmed final alert status `suppressed` and final insight status `archived`.
- Consumer group `signalops.signal-persister.v1` was Stable with total lag `0`.
- Frontend-agent Playwright summary reported no console/page errors, one dashboard SSE connection, visible alert/insight action controls, disabled post-action controls, lifecycle summaries shown, Open Alerts/Active Insights counts dropped after mutation, 12 nav items, and `0px` mobile horizontal overflow for Alerts and Insights.

Issue found and noted:

- A local independent Playwright rerun could not be completed with the available `mcr.microsoft.com/playwright:v1.61.1-jammy` image because the image did not include the Node `playwright` module. The frontend-agent's Playwright summary and screenshots were present under `/tmp/g050-validate/shots`, and independent validation covered tests, build, proxy mutations, database state, and consumer lag.

Next step:

- Decide whether the next backend gate should add authenticated operator identity/audit-history rows, or start the TimescaleDB storage maturity planning gate.

## 2026-07-09T02:43:48Z

Summary:

- Added a SignalOps Traefik overlay for public TLS through the parent Syncratic core edge.
- Updated the environment example and deployment documentation for the public SignalOps host.
- Deployed the web service with the overlay so Traefik can route to the existing nginx frontend/API proxy.

Files changed:

- `compose.traefik.yaml`
- `.env.example`
- `docs/deployment.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Implementation notes:

- The overlay attaches only `web` to the external `syncratic-core_syncratic_net` network.
- Traefik labels use the existing `websecure` entrypoint and `letsencrypt` resolver from Syncratic core.
- The public router host is `signalops.syncratic.io`.
- Gateway remains internal; nginx in `web` continues to proxy `/healthz`, `/readyz`, and `/v1/*` to `gateway:8080`.

Validation performed:

- `docker compose -f compose.yaml -f compose.traefik.yaml config --quiet`
- Verified Docker networks include `signalops_default` and `syncratic-core_syncratic_net`.
- Rendered merged Compose config and confirmed Traefik labels and dual-network attachment.
- `docker compose -f compose.yaml -f compose.traefik.yaml up -d web`
- Inspected `signalops-web-1` labels and network attachments.
- `curl -fsS http://localhost:15173/healthz`
- `curl -k --resolve signalops.syncratic.io:443:127.0.0.1 -fsS https://signalops.syncratic.io/healthz`
- `curl -k --resolve signalops.syncratic.io:443:127.0.0.1 -fsS https://signalops.syncratic.io/`
- Checked Traefik access logs for `signalops@docker` routing to the web container.
- Public validation: `curl -sS -o /tmp/signalops-http-final.txt -w "%{http_code} %{redirect_url}\n" http://signalops.syncratic.io/healthz` returned `301 https://signalops.syncratic.io/healthz`; `curl -sS -o /tmp/signalops-https-final.txt -w "%{http_code} %{remote_ip} %{ssl_verify_result}\n" https://signalops.syncratic.io/healthz` returned `200 45.60.31.46 0`.

Live verification result:

- Local web health endpoint returned gateway health JSON.
- Local Traefik SNI override served `/healthz` and `/` through the `signalops@docker` route.
- Public DNS resolves, HTTP redirects to HTTPS, public HTTPS reaches SignalOps gateway health, and local Traefik SNI HTTPS succeeds.

Next step:

- Continue with the next deployment/auth gate now that public HTTPS health is live.
