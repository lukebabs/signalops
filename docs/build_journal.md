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
