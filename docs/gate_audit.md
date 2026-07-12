# SignalOps Gate Audit

This file records implementation gates, status, evidence, and timestamps for
change audit. All timestamps are UTC.

## Gate G000: Build Documentation Trail

Timestamp: `2026-07-06T20:02:13Z`

Status: `passed`

Gate name:

- Establish ongoing documentation and audit process.

Criteria:

- Create an ongoing build journal.
- Create a gate audit log.
- Define timestamp, journal, gate, and verification documentation standards.
- Record the first gate with timestamp and evidence.

Evidence:

- `docs/documentation_standards.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`
- UTC timestamp captured from local environment: `2026-07-06T20:02:13Z`

Actor:

- Codex

Follow-up items:

- Apply this documentation process to every future implementation gate.
- Update the journal and gate audit in the same change set as each gate.


## Gate G001: Phase 1 Core Scaffold

Timestamp: `2026-07-06T20:11:30Z`

Status: `passed`

Gate name:

- Create initial Phase 1 Go core, shared contract, and Python plugin scaffold.

Criteria:

- Add a Go module for `github.com/lukebabs/signalops`.
- Add a gateway entrypoint owned by the Go core platform.
- Add health and readiness routes.
- Add basic environment-driven configuration.
- Add shared contract directories for cross-runtime schemas.
- Add Python plugin directories without embedding Python in Go services.
- Record verification limitations in the build journal.

Evidence:

- `go.mod`
- `cmd/gateway/main.go`
- `internal/api/router.go`
- `internal/config/config.go`
- `contracts/`
- `pkg/contracts/`
- `python/signalops_plugins/`
- `docs/build_journal.md`
- UTC timestamp captured from local environment: `2026-07-06T20:11:30Z`

Verification performed:

- Working tree was clean before scaffold.
- File patch applied successfully.
- Go toolchain check attempted with `go version`.

Verification limitation:

- `go test ./...` was not run because `go` is not installed in this
  environment.

Actor:

- Codex

Follow-up items:

- Install or provide Go and run `go test ./...`.
- Add shared event contract schema files.

## Gate G002: Docker Toolchain And Gateway Smoke Test

Timestamp: `2026-07-06T20:18:13Z`

Status: `passed`

Gate name:

- Establish Docker as the default local toolchain and verify the gateway
  scaffold in a container.

Criteria:

- Add Docker build configuration for the Go gateway.
- Add a Docker-first development guide.
- Add Make targets for Dockerized test, build, and shell workflows.
- Run Go tests through Docker.
- Build the gateway container image.
- Run the gateway container and verify health/readiness endpoints.

Evidence:

- `.dockerignore`
- `Dockerfile`
- `Makefile`
- `docs/docker_development.md`
- Docker version: `Docker version 29.5.0, build 98f1464`
- Dockerized `go test ./...` passed for:
  - `github.com/lukebabs/signalops/cmd/gateway`
  - `github.com/lukebabs/signalops/internal/api`
  - `github.com/lukebabs/signalops/internal/config`
- Docker image built: `signalops-gateway:local`
- `GET /healthz` returned status `ok`.
- `GET /readyz` returned status `ready`.

Verification performed:

- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./...`
- `docker build --target gateway -t signalops-gateway:local .`
- `docker run --rm -d --name signalops-gateway-smoke -p 18080:8080 signalops-gateway:local`
- `curl -sS http://127.0.0.1:18080/healthz`
- `curl -sS http://127.0.0.1:18080/readyz`
- `docker stop signalops-gateway-smoke`

Actor:

- Codex

Follow-up items:

- Continue using Docker for build/test validation.
- Add schema validation tooling through Docker.

## Gate G003: Event Contract Schemas

Timestamp: `2026-07-06T20:26:54Z`

Status: `passed`

Gate name:

- Establish first versioned shared event contracts for Go/Python runtime boundaries.

Criteria:

- Add common JSON Schema definitions for source domains, ingestion modes,
  timestamps, confidence, entity references, and evidence references.
- Add `RawSignalEvent` v1 schema.
- Add `NormalizedSignalEvent` v1 schema.
- Add `Signal` v1 schema.
- Add Dockerized schema validation tooling.
- Verify schemas parse and expose required metadata.
- Re-run Dockerized Go tests.

Evidence:

- `contracts/events/common.defs.v1.schema.json`
- `contracts/events/raw_signal_event.v1.schema.json`
- `contracts/events/normalized_signal_event.v1.schema.json`
- `contracts/events/signal.v1.schema.json`
- `scripts/validate_json_schemas.py`
- `Makefile`
- `docs/docker_development.md`
- Schema validation output showed all four schemas as `ok`.
- Dockerized `go test ./...` passed.

Verification performed:

- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace python:3.12-slim python scripts/validate_json_schemas.py`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./...`

Actor:

- Codex

Follow-up items:

- Add Go contract structs and JSON serialization tests.
- Add remaining schemas for EventArtifact, GraphMutationProposal, and InsightCandidate.

## Gate G004: Go Event Contract Types

Timestamp: `2026-07-06T20:31:01Z`

Status: `passed`

Gate name:

- Add typed Go representations for initial shared event contracts.

Criteria:

- Add Go enum-like constants for source domains, ingestion modes, and severity.
- Add common entity/evidence reference structs.
- Add `RawSignalEvent`, `NormalizedSignalEvent`, and `Signal` structs.
- Use JSON tags matching the v1 schemas.
- Add serialization tests for representative raw event and signal payloads.
- Re-run Dockerized Go tests and schema validation.

Evidence:

- `pkg/contracts/events.go`
- `pkg/contracts/events_test.go`
- Dockerized `go test ./...` passed, including `pkg/contracts`.
- Dockerized schema validation passed for all event schemas.

Verification performed:

- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./...`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace python:3.12-slim python scripts/validate_json_schemas.py`

Actor:

- Codex

Follow-up items:

- Add remaining output contract schemas and Go types for artifacts, graph
  mutation proposals, and insight candidates.

## Gate G005: Internal Communication Protocol Decision

Timestamp: `2026-07-06T20:39:12Z`

Status: `passed`

Gate name:

- Codify durable path, fast sync path, and protocol decision rule.

Criteria:

- Document Kafka/Redpanda as the default durable protocol for Go/Python
  algorithm communication.
- Document JSON + JSON Schema as the v1 durable payload contract.
- Document gRPC + Protobuf as the bounded fast-sync internal exception.
- State that gRPC responses are not canonical truth until persisted or
  republished by the Go core platform.
- State the decision rule for Kafka/Redpanda, gRPC, REST, and prohibited
  in-process Python calls.
- Re-run Dockerized Go tests and schema validation.

Evidence:

- `docs/Syncratic_SignalOps_Processing_Specification.md`
- `docs/signalops_extended_capabilities_spec.md`
- `contracts/protocols.md`
- Dockerized `go test ./...` passed.
- Dockerized schema validation passed.

Verification performed:

- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./...`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace python:3.12-slim python scripts/validate_json_schemas.py`

Actor:

- Codex

Follow-up items:

- Add broker topic constants and durable algorithm job/result contracts.
- Defer gRPC Protobuf definitions until a bounded fast-sync use case is selected.

## Gate G006: Local Redpanda Deployment

Timestamp: `2026-07-06T20:45:00Z`

Status: `passed`

Gate name:

- Add and validate local Docker Compose deployment with Redpanda default broker.

Criteria:

- Add local Docker Compose stack for Redpanda, Redpanda Console, topic
  bootstrap, and SignalOps gateway.
- Add deterministic topic bootstrap for durable SignalOps topics.
- Add deployment documentation and local environment example.
- Validate compose syntax.
- Run Dockerized Go tests and schema validation.
- Start the stack successfully.
- Verify gateway health/readiness endpoints.
- Verify default topics exist.

Evidence:

- `compose.yaml`
- `.env.example`
- `deploy/docker/redpanda/create-topics.sh`
- `docs/deployment.md`
- `Makefile`
- `docker compose config --quiet` passed.
- Dockerized `go test ./...` passed.
- Dockerized schema validation passed.
- `docker compose ps` showed `redpanda` healthy, `redpanda-console` running,
  and `gateway` running.
- `GET /healthz` returned status `ok` from `http://127.0.0.1:18000`.
- `GET /readyz` returned status `ready` from `http://127.0.0.1:18000`.
- `rpk topic list` showed all default SignalOps topics.

Issues found and resolved:

- Initial Redpanda healthcheck used `rpk cluster health --brokers`, but this
  `rpk` version does not support `--brokers` for that command. Healthcheck was
  corrected to `rpk cluster health`.
- Initial gateway host port `8080` conflicted with an existing local service.
  The compose mapping was changed to `18000:8080`.

Verification performed:

- `docker compose version`
- `docker compose config --quiet`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./...`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace python:3.12-slim python scripts/validate_json_schemas.py`
- `docker compose up -d --build`
- `curl -sS http://127.0.0.1:18000/healthz`
- `curl -sS http://127.0.0.1:18000/readyz`
- `docker compose exec redpanda rpk topic list --brokers localhost:9092`

Actor:

- Codex

Follow-up items:

- Add broker topic constants and broker abstraction interfaces.
- Add integration tests that use the local Redpanda stack.


## Gate G007: Broker Boundary And Topic Constants

Timestamp: `2026-07-06T20:56:06Z`

Status: `passed`

Gate name:

- Add durable broker interfaces and canonical topic constants.

Criteria:

- Add Go topic constants that match the local Redpanda topic bootstrap.
- Add publisher and consumer interfaces for Kafka-compatible durable messaging.
- Add message metadata fields required for correlation, causation, traceability,
  partition acknowledgement, and offset acknowledgement.
- Extend process config with broker provider, broker addresses, and environment.
- Document the broker boundary and protocol implications.
- Validate Go tests, schema validation, and running Redpanda topic state.

Evidence:

- `pkg/broker/broker.go`
- `pkg/broker/topics.go`
- `pkg/broker/topics_test.go`
- `pkg/broker/README.md`
- `internal/config/config.go`
- `internal/config/config_test.go`
- `contracts/protocols.md`
- `docs/deployment.md`

Verification performed:

- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm gofmt -w internal/config/config.go internal/config/config_test.go pkg/broker/broker.go pkg/broker/topics.go pkg/broker/topics_test.go`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./...`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace python:3.12-slim python scripts/validate_json_schemas.py`
- `docker compose exec redpanda rpk topic list --brokers localhost:9092`

Actor:

- Codex

Follow-up items:

- Add a concrete Redpanda/Kafka client package implementing `pkg/broker`.
- Add broker integration tests that can run against the Docker Compose stack.


## Gate G008: Concrete Redpanda Kafka Client

Timestamp: `2026-07-06T23:22:46Z`

Status: `passed`

Gate name:

- Add and validate the concrete Kafka-compatible Redpanda broker client.

Criteria:

- Add a concrete Go implementation of `pkg/broker` behind an internal package.
- Preserve Kafka client types behind `internal/` and avoid leaking them into
  public SignalOps contracts.
- Implement synchronous publish acknowledgement with topic, partition, and offset.
- Implement manual-commit consumer groups.
- Preserve `correlation_id`, `causation_id`, and `trace_id` in Kafka headers.
- Add unit tests and a live Redpanda publish/consume/commit integration test.
- Add repeatable Docker documentation for the integration test.

Evidence:

- `internal/broker/kafka/client.go`
- `internal/broker/kafka/client_test.go`
- `internal/broker/kafka/client_integration_test.go`
- `internal/broker/kafka/README.md`
- `Makefile`
- `docs/deployment.md`
- `go.mod`
- `go.sum`

Verification performed:

- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm gofmt -w internal/broker/kafka/client.go internal/broker/kafka/client_test.go internal/broker/kafka/client_integration_test.go`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go mod tidy`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./...`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace python:3.12-slim python scripts/validate_json_schemas.py`
- `docker run --rm --network host -e SIGNALOPS_BROKER_INTEGRATION=1 -e SIGNALOPS_BROKER_BROKERS=localhost:19092 -e SIGNALOPS_ENV=local -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./internal/broker/kafka -run TestPublishConsumeCommitAgainstRedpanda -count=1 -v`
- `make docker-test-broker-integration`

Issues found and resolved:

- `github.com/twmb/franz-go@latest` requires Go 1.25, and `v1.19.5` requires
  Go 1.23.8. The client is pinned to `v1.18.1`, which validates on Go 1.22.
- A bridge-networked Docker integration test could not use the Redpanda
  advertised `localhost:19092` listener. The repeatable integration target uses
  Docker host networking.

Actor:

- Codex

Follow-up items:

- Wire gateway ingestion to publish raw events through the broker client.
- Add application-level publish error handling and readiness checks for broker
  connectivity.


## Gate G009: Gateway Raw Event Ingestion

Timestamp: `2026-07-07T00:01:22Z`

Status: `passed`

Gate name:

- Add and validate gateway raw event ingestion into the durable Redpanda raw topic.

Criteria:

- Add `POST /v1/events/raw` to accept JSON object raw events.
- Publish accepted events through `pkg/broker.Publisher` to
  `signalops.<environment>.raw.v1`.
- Preserve raw JSON bytes as the broker value.
- Use idempotency key as the broker key.
- Preserve or generate event and correlation identifiers.
- Carry SignalOps ingestion metadata and correlation headers into Kafka headers.
- Return broker acknowledgement details on success.
- Add unit tests, API docs, and live Redpanda verification.

Evidence:

- `cmd/gateway/main.go`
- `internal/api/router.go`
- `internal/api/router_test.go`
- `Dockerfile`
- `docs/api.md`
- `docs/deployment.md`

Verification performed:

- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm gofmt -w cmd/gateway/main.go internal/api/router.go internal/api/router_test.go`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./...`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace python:3.12-slim python scripts/validate_json_schemas.py`
- `docker compose up -d --build gateway`
- `curl -sS http://127.0.0.1:18000/healthz`
- `curl -sS http://127.0.0.1:18000/readyz`
- `curl -sS -X POST http://127.0.0.1:18000/v1/events/raw -H 'Content-Type: application/json' -H 'X-Correlation-ID: g009-correlation-live' -H 'X-Causation-ID: g008' -H 'X-Trace-ID: g009-trace-live' -d '{"event_id":"g009-live-event","idempotency_key":"g009-live-key","source_domain":"market_data","source_adapter":"manual-curl","payload":{"symbol":"SPY","close":501.25}}'`
- `docker compose exec redpanda rpk topic consume signalops.local.raw.v1 -p 0 -o 1 -n 1`

Live verification result:

- Gateway response returned `202 Accepted` with topic `signalops.local.raw.v1`,
  partition `0`, offset `1`, event ID `g009-live-event`, idempotency key
  `g009-live-key`, and correlation ID `g009-correlation-live`.
- Redpanda consume returned key `g009-live-key`, the original JSON payload, and
  headers `signalops_ingest_route`, `signalops_ingest_format`,
  `signalops_event_id`, `signalops_idempotency`, `content_type`,
  `correlation_id`, `causation_id`, and `trace_id`.

Issues found and resolved:

- The Dockerfile copied `go.mod` but not `go.sum`, which prevented image build
  tests from resolving franz-go dependencies. The Dockerfile now copies both
  module files before build-stage tests.

Actor:

- Codex

Follow-up items:

- Add a broker-backed readiness check instead of only process readiness.
- Add automated HTTP-to-Redpanda integration tests.
- Begin Python worker skeleton for durable algorithm processing.


## Gate G010: Python Raw Worker Skeleton

Timestamp: `2026-07-07T00:25:36Z`

Status: `passed`

Gate name:

- Add and deploy the initial Python worker skeleton for durable raw event consumption.

Criteria:

- Add a runnable Python worker package separate from Go services.
- Consume raw events from Redpanda using the durable broker boundary.
- Parse raw event JSON and resolve event, idempotency, and correlation metadata.
- Manually commit consumed offsets.
- Avoid detector-specific algorithm behavior in this gate.
- Add Docker image and Compose service for the worker.
- Add Python unit tests and worker documentation.
- Deploy the worker locally and verify consumer group state.

Evidence:

- `python/signalops_workers/config.py`
- `python/signalops_workers/worker.py`
- `python/signalops_workers/broker.py`
- `python/signalops_workers/__main__.py`
- `python/tests/test_config.py`
- `python/tests/test_worker.py`
- `python/requirements-worker.txt`
- `deploy/docker/python-worker/Dockerfile`
- `compose.yaml`
- `Makefile`
- `docs/python_worker.md`
- `docs/deployment.md`
- `docs/docker_development.md`

Verification performed:

- `docker compose config --quiet`
- `make docker-test-python`
- `docker compose build raw-worker`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./...`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace python:3.12-slim python scripts/validate_json_schemas.py`
- `docker compose run --rm -e SIGNALOPS_WORKER_GROUP_ID=signalops.g010.validation -e SIGNALOPS_WORKER_MAX_MESSAGES=1 raw-worker`
- `docker compose run --rm -e SIGNALOPS_WORKER_GROUP_ID=signalops.g010.validation.explicit -e SIGNALOPS_WORKER_MAX_MESSAGES=1 raw-worker`
- `docker compose up -d raw-worker`
- `docker compose ps`
- `docker compose logs --tail=40 raw-worker`
- `docker compose exec redpanda rpk group describe signalops.raw-worker.v1`

Live verification result:

- One-shot worker validation skipped and committed invalid historical raw records
  instead of crash-looping.
- One-shot worker validation processed a valid G009 raw event after explicit
  offset commit handling was added.
- Long-running `signalops-raw-worker-1` is running in Docker Compose.
- Consumer group `signalops.raw-worker.v1` is stable with one member and total
  lag `0` across `signalops.local.raw.v1` partitions.

Issues found and resolved:

- Historical G008 test records lacked `event_id`, causing the first worker run
  to fail. The skeleton now logs and commits invalid records until DLQ routing
  is added.
- Generic synchronous consumer commits did not advance the validation group
  reliably. The worker now commits explicit topic/partition offsets.
- `.dockerignore` excluded the `python` directory. The ignore rule was removed
  so the worker image receives source files.

Actor:

- Codex

Follow-up items:

- Add retry and DLQ publishing for invalid raw records and processing failures.
- Add detector plugin interfaces and a reference no-op detector.
- Add worker health/readiness signaling for orchestration.


## Gate G011: Python Worker DLQ Handling

Timestamp: `2026-07-07T01:40:31Z`

Status: `passed`

Gate name:

- Add durable DLQ publishing for Python raw-worker invalid records and processing failures.

Criteria:

- Add a DLQ publisher for the Python worker.
- Publish invalid raw events and processing failures to
  `signalops.<environment>.dlq.algorithm.v1`.
- Preserve source topic, partition, offset, key, headers, and original broker
  value in a replayable/auditable payload.
- Commit the source offset only after processing succeeds or DLQ publication is
  acknowledged.
- Add DLQ payload schema and schema validation coverage.
- Add unit tests for DLQ success and DLQ publish failure semantics.
- Validate the live worker against Redpanda with a deliberately invalid raw event.

Evidence:

- `python/signalops_workers/worker.py`
- `python/signalops_workers/broker.py`
- `python/signalops_workers/config.py`
- `python/signalops_workers/__main__.py`
- `python/tests/test_worker.py`
- `python/tests/test_config.py`
- `contracts/events/dlq_event.v1.schema.json`
- `contracts/events/README.md`
- `compose.yaml`
- `docs/python_worker.md`
- `docs/deployment.md`

Verification performed:

- `docker compose config --quiet`
- `make docker-test-python`
- `docker compose build raw-worker`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./...`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace python:3.12-slim python scripts/validate_json_schemas.py`
- `docker compose up -d raw-worker`
- `docker compose exec -T redpanda sh -lc "printf '%s\n' '{"payload":{"bad":true}}' | rpk topic produce signalops.local.raw.v1 -k g011-invalid-live -H correlation_id:g011-correlation-live --output-format 'partition=%p offset=%o\n'"`
- `docker compose logs --tail=120 raw-worker`
- `docker compose exec redpanda rpk topic consume signalops.local.dlq.algorithm.v1 -o start -n 1`
- `docker compose exec redpanda rpk group describe signalops.raw-worker.v1`

Live verification result:

- Invalid raw event was produced to `signalops.local.raw.v1`, partition `2`,
  offset `1`.
- Worker log showed `sent raw event to dlq`.
- DLQ topic contained key `g011-invalid-live` with schema ID
  `signalops.dlq.raw_event.v1`, error type `InvalidRawEventError`, error
  message `raw event is missing event_id`, source topic
  `signalops.local.raw.v1`, source partition `2`, source offset `1`, source
  correlation header `g011-correlation-live`, and base64 source payload.
- Worker consumer group `signalops.raw-worker.v1` returned to stable state with
  total lag `0`.

Issues found and resolved:

- G010's temporary invalid-record behavior committed poison records without a
  durable failure artifact. G011 now publishes a DLQ event before committing.
- Unit tests now verify that the source message is not committed when DLQ
  publishing fails.

Actor:

- Codex

Follow-up items:

- Add retry topic handling for retryable failures.
- Add detector plugin interfaces and a no-op detector path.
- Add replay tooling for DLQ records.


## Gate G012: Python Worker Retry Handling

Timestamp: `2026-07-07T02:01:22Z`

Status: `passed`

Gate name:

- Add durable retry-topic publishing for retryable Python worker failures.

Criteria:

- Add a retry publisher for the Python worker.
- Add `RetryableWorkerError` routing to
  `signalops.<environment>.retry.algorithm.v1`.
- Preserve source topic, partition, offset, key, headers, retry attempt, and
  original broker value in a replayable/auditable payload.
- Commit the source offset only after processing succeeds, retry publication is
  acknowledged, or DLQ publication is acknowledged.
- Add retry payload schema and schema validation coverage.
- Add unit tests for retry success and retry publish failure semantics.
- Validate the retry publisher against live Redpanda.

Evidence:

- `python/signalops_workers/worker.py`
- `python/signalops_workers/broker.py`
- `python/signalops_workers/config.py`
- `python/signalops_workers/__main__.py`
- `python/tests/test_worker.py`
- `python/tests/test_config.py`
- `contracts/events/retry_event.v1.schema.json`
- `contracts/events/README.md`
- `compose.yaml`
- `docs/python_worker.md`
- `docs/deployment.md`

Verification performed:

- `docker compose config --quiet`
- `make docker-test-python`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./...`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace python:3.12-slim python scripts/validate_json_schemas.py`
- `docker compose build raw-worker`
- `docker compose run --rm --entrypoint python raw-worker -c "from signalops_workers.broker import RedpandaRetryPublisher; from signalops_workers.worker import BrokerMessage, RetryableWorkerError; p=RedpandaRetryPublisher(brokers='redpanda:9092', retry_topic='signalops.local.retry.algorithm.v1'); p.publish(BrokerMessage(topic='signalops.local.raw.v1', partition=2, offset=99, key='g012-retry-live', value=b'{"event_id":"g012-retry-live"}', headers={'correlation_id':'g012-correlation-live','signalops_retry_attempt':'1'}), RetryableWorkerError('transient dependency unavailable')); p.close(); print('published retry event')"`
- `docker compose exec redpanda rpk topic consume signalops.local.retry.algorithm.v1 -o start -n 1`
- `docker compose up -d raw-worker`
- `curl -sS http://127.0.0.1:18000/readyz`
- `docker compose exec redpanda rpk group describe signalops.raw-worker.v1`

Live verification result:

- Retry topic contained key `g012-retry-live` with schema ID
  `signalops.retry.raw_event.v1`, error type `RetryableWorkerError`, error
  message `transient dependency unavailable`, retry attempt `2`, source topic
  `signalops.local.raw.v1`, source partition `2`, source offset `99`, source
  correlation header `g012-correlation-live`, and base64 source payload.
- Worker service redeployed successfully with retry topic configuration.
- Worker consumer group `signalops.raw-worker.v1` returned to stable state with
  one member and total lag `0` after redeploy rebalance completed.

Issues found and resolved:

- The worker previously routed all failures to DLQ. Retryable failures now have
  a distinct retry topic and contract.
- Unit tests verify that source messages are not committed when retry publishing
  fails.

Actor:

- Codex

Follow-up items:

- Add detector plugin interfaces and a reference no-op detector.
- Add retry replay tooling.
- Add retry attempt limits and escalation from retry to DLQ.


## Gate G013: Detector Plugin Contract And Noop Detector

Timestamp: `2026-07-07T02:09:51Z`

Status: `passed`

Gate name:

- Add Python detector plugin contracts and wire the reference no-op detector into the worker.

Criteria:

- Add Python detector plugin SDK contracts for initialize, detect, explain, and
  emit-signal lifecycle methods.
- Add a deterministic `signalops.noop` reference detector that emits no signals.
- Add worker detector loading through environment configuration.
- Invoke the configured detector for valid raw events.
- Preserve retry/DLQ routing around detector failures.
- Add tests for detector contract, detector loader, worker detector invocation,
  and retryable detector failures.
- Validate the live worker path against Redpanda with a fresh gateway-ingested
  raw event.

Evidence:

- `python/signalops_plugins/detectors/base.py`
- `python/signalops_plugins/detectors/noop.py`
- `python/signalops_workers/detectors.py`
- `python/signalops_workers/worker.py`
- `python/signalops_workers/__main__.py`
- `python/tests/plugins/test_noop_detector.py`
- `python/tests/test_detectors.py`
- `python/tests/test_worker.py`
- `docs/python_worker.md`
- `docs/Syncratic_SignalOps_Processing_Specification.md`
- `compose.yaml`

Verification performed:

- `docker compose config --quiet`
- `make docker-test-python`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./...`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace python:3.12-slim python scripts/validate_json_schemas.py`
- `docker compose build raw-worker`
- `docker compose up -d raw-worker`
- `curl -sS -X POST http://127.0.0.1:18000/v1/events/raw -H 'Content-Type: application/json' -H 'X-Correlation-ID: g013-correlation-live' -H 'X-Trace-ID: g013-trace-live' -d '{"event_id":"g013-live-event","idempotency_key":"g013-live-key","source_domain":"market_data","source_adapter":"manual-curl","payload":{"symbol":"QQQ","close":444.12}}'`
- `docker compose logs --tail=120 raw-worker`
- `docker compose exec redpanda rpk group describe signalops.raw-worker.v1`

Live verification result:

- Gateway accepted `g013-live-event` to `signalops.local.raw.v1`, partition `0`,
  offset `2`.
- Worker logs showed `detector evaluated raw event` and `processed raw event`.
- Worker consumer group `signalops.raw-worker.v1` returned to stable state with
  one member and total lag `0`.

Issues found and resolved:

- Retryable detector failures initially bypassed retry routing. Detector
  failures are now routed through retry/DLQ publishers before source offset
  commit.
- Worker redeploy caused a temporary consumer-group rebalance until the old
  session timed out; final state was stable with zero lag.

Actor:

- Codex

Follow-up items:

- Add detector signal/result publishing to `signalops.<environment>.signal.v1`.
- Add a deterministic reference signal-emitting detector.
- Add retry replay tooling and retry attempt limits.

## Gate G014: Detector Signal Publishing

Timestamp: `2026-07-07T02:25:49Z`

Status: `passed`

Gate name:

- Publish detector-emitted signals from Python workers to the durable signal topic.

Criteria:

- Add a worker signal publisher boundary for `signal.v1` events.
- Add a Redpanda-backed signal publisher for `signalops.<environment>.signal.v1`.
- Map detector `EmittedSignal` output into the existing signal event contract
  with source lineage, timestamps, detector metadata, evidence, and correlation.
- Add a deterministic signal-emitting reference detector for validation.
- Route signal-topic publish failures through retry handling before source
  offset commit.
- Add unit tests for signal mapping, publishing, retry routing, and no-commit
  behavior when retry publication fails.
- Validate the live path against Redpanda with a gateway-ingested raw event.

Evidence:

- `python/signalops_workers/worker.py`
- `python/signalops_workers/broker.py`
- `python/signalops_workers/config.py`
- `python/signalops_workers/__main__.py`
- `python/signalops_workers/detectors.py`
- `python/signalops_plugins/detectors/noop.py`
- `python/tests/test_worker.py`
- `python/tests/test_detectors.py`
- `python/tests/plugins/test_noop_detector.py`
- `docs/python_worker.md`
- `docs/Syncratic_SignalOps_Processing_Specification.md`
- `compose.yaml`

Verification performed:

- `docker compose config --quiet`
- `make docker-test-python`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./...`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace python:3.12-slim python scripts/validate_json_schemas.py`
- `docker compose build raw-worker`
- `docker compose stop raw-worker`
- `curl -sS -X POST http://127.0.0.1:18000/v1/events/raw ... g014-live-event ...`
- `docker compose run --rm -e SIGNALOPS_WORKER_DETECTOR_ID=signalops.static_test -e SIGNALOPS_WORKER_MAX_MESSAGES=1 raw-worker`
- `docker compose exec redpanda rpk topic consume signalops.local.signal.v1 -o start -n 1`
- `docker compose up -d raw-worker`
- `docker compose exec redpanda rpk group describe signalops.raw-worker.v1`
- `curl -sS http://127.0.0.1:18000/readyz`

Live verification result:

- Gateway accepted `g014-live-event` to `signalops.local.raw.v1`, partition `0`,
  offset `3`.
- The finite static detector worker logged `detector evaluated raw event` and
  `processed raw event`.
- `signalops.local.signal.v1` contained signal key `signalops.static_test.low`
  with detector ID `signalops.static_test`, event ID `g014-live-event`, source
  topic `signalops.local.raw.v1`, source partition `0`, source offset `3`,
  correlation ID `g014-correlation-live`, and trace ID `g014-trace-live`.
- The default no-op worker was restarted successfully. The raw-worker consumer
  group returned to stable state with one member and total lag `0`.

Issues found and resolved:

- Signal publish failures are retryable infrastructure failures. The worker now
  wraps signal publisher exceptions in `RetryableWorkerError`, preserving retry
  routing and offset safety.

Actor:

- Codex

Follow-up items:

- Add retry replay tooling with retry attempt limits.
- Add schema validation for emitted Python signal events before publication.
- Add a domain-specific market-data detector pack.

## Gate G015: Retry Replay Tooling

Timestamp: `2026-07-07T02:54:42Z`

Status: `passed`

Gate name:

- Replay retry-topic records with bounded attempts and DLQ escalation.

Criteria:

- Add a retry replayer that consumes retry records and reconstructs original source messages.
- Republish retryable source messages to the configured raw topic while attempts remain.
- Route exhausted retries to DLQ with original payload, headers, source topic, partition, and offset preserved.
- Route malformed retry records to DLQ as retry records without committing until DLQ publication succeeds.
- Commit retry-topic offsets only after replay or DLQ publication is acknowledged.
- Add configuration and Docker Compose support for finite and long-running replay operation.
- Add unit tests for replay decisions, replay publication, exhausted retries, invalid retry records, and no-commit-on-publish-failure semantics.
- Validate replay and exhausted-DLQ behavior against live Redpanda topics.

Evidence:

- `python/signalops_workers/retry_replay.py`
- `python/signalops_workers/retry_replay_main.py`
- `python/signalops_workers/broker.py`
- `python/signalops_workers/config.py`
- `python/tests/test_retry_replay.py`
- `python/tests/test_config.py`
- `compose.yaml`
- `docs/python_worker.md`
- `docs/deployment.md`

Verification performed:

- `docker compose config --quiet`
- `make docker-test-python`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./...`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace python:3.12-slim python scripts/validate_json_schemas.py`
- `docker compose build raw-worker retry-replayer`
- `docker compose exec redpanda rpk topic create signalops.local.retry.g015.replay.v1 signalops.local.retry.g015.exhausted.v1 --brokers redpanda:9092 --partitions 1 --replicas 1 --if-not-exists`
- `docker compose run --rm --entrypoint python raw-worker -c "... published g015 retry validation records ..."`
- `docker compose --profile retry-replay run --rm -e SIGNALOPS_RETRY_REPLAY_INPUT_TOPIC=signalops.local.retry.g015.replay.v1 -e SIGNALOPS_RETRY_REPLAY_GROUP_ID=signalops.g015.replay.validation -e SIGNALOPS_RETRY_REPLAY_MAX_MESSAGES=1 retry-replayer`
- `docker compose --profile retry-replay run --rm -e SIGNALOPS_RETRY_REPLAY_INPUT_TOPIC=signalops.local.retry.g015.exhausted.v1 -e SIGNALOPS_RETRY_REPLAY_GROUP_ID=signalops.g015.exhausted.validation -e SIGNALOPS_RETRY_REPLAY_MAX_MESSAGES=1 retry-replayer`
- `docker compose exec redpanda rpk topic consume signalops.local.raw.v1 -o start -n 20`
- `docker compose exec redpanda rpk topic consume signalops.local.dlq.algorithm.v1 -o start -n 20`
- `docker compose up -d raw-worker`
- `docker compose exec redpanda rpk group describe signalops.raw-worker.v1`
- `curl -sS http://127.0.0.1:18000/readyz`

Live verification result:

- `signalops.local.raw.v1` contained replayed key `g015-replay-key` with event ID `g015-replay-live`, `signalops_retry_attempt=1`, and `signalops_replayed_from_retry=true`.
- `signalops.local.dlq.algorithm.v1` contained key `g015-exhausted-key` with error type `RetryAttemptsExhausted`, source topic `signalops.local.raw.v1`, source offset `405`, `signalops_retry_attempt=3`, and preserved base64 source payload.
- The default raw worker was recreated from the rebuilt image. The raw-worker consumer group returned to stable state with one member and total lag `0`.

Issues found and resolved:

- Worker recreation caused a temporary consumer-group rebalance. The final group state was stable with zero lag.

Actor:

- Codex

Follow-up items:

- Add schema validation for Python-emitted signal events before publication.
- Add replay dry-run and filtering controls.
- Add the first Massive/Polygon scheduled market-data adapter and detector pack.

## Gate G016: Python Signal Schema Validation

Timestamp: `2026-07-07T03:46:54Z`

Status: `passed`

Gate name:

- Validate Python-emitted signal events against the checked-in `signal.v1` contract before publication.

Criteria:

- Add runtime validation for built signal events before publishing to `signalops.<environment>.signal.v1`.
- Use checked-in JSON Schema files under `contracts/events` rather than duplicating the full signal contract in Python logic.
- Package schema files into the Python worker image.
- Route invalid built signal events to DLQ as non-retryable output contract failures.
- Add tests for valid signal events, missing required fields, invalid confidence, unexpected fields, and worker DLQ routing for invalid detector output.
- Validate the live signal path against Redpanda after schema validation is enabled.

Evidence:

- `python/signalops_workers/schema_validation.py`
- `python/signalops_workers/worker.py`
- `python/tests/test_schema_validation.py`
- `python/tests/test_worker.py`
- `deploy/docker/python-worker/Dockerfile`
- `.dockerignore`
- `docs/python_worker.md`
- `docs/deployment.md`

Verification performed:

- `docker compose config --quiet`
- `make docker-test-python`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./...`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace python:3.12-slim python scripts/validate_json_schemas.py`
- `docker compose build raw-worker retry-replayer`
- `docker compose up -d redpanda topic-bootstrap gateway`
- `curl -sS -X POST http://127.0.0.1:18000/v1/events/raw ... g016-live-event ...`
- `docker compose run --rm -e SIGNALOPS_WORKER_DETECTOR_ID=signalops.static_test -e SIGNALOPS_WORKER_MAX_MESSAGES=1 raw-worker`
- `docker compose exec redpanda rpk topic consume signalops.local.signal.v1 -o start -n 5`
- `docker compose up -d raw-worker redpanda-console`
- `docker compose exec redpanda rpk group describe signalops.raw-worker.v1`
- `curl -sS http://127.0.0.1:18000/readyz`

Live verification result:

- Gateway accepted `g016-live-event` to `signalops.local.raw.v1`, partition `0`, offset `4`.
- The finite static detector worker logged `detector evaluated raw event` and `processed raw event`.
- `signalops.local.signal.v1` contained a validated G016 signal with key `signalops.static_test.low`, tenant `tenant-g016`, event ID `g016-live-event`, detector ID `signalops.static_test`, correlation ID `g016-correlation-live`, trace ID `g016-trace-live`, raw source offset `4`, and schema header `signalops.signal.v1`.
- The default raw worker and Redpanda console were restarted. The raw-worker consumer group was stable with one member and total lag `0`.

Issues found and resolved:

- `.dockerignore` excluded `contracts/`, which prevented the Python worker image from copying runtime schemas. The ignore entry was removed.
- The compose project was down when the first live validation request was attempted. Redpanda, topic bootstrap, and gateway were started and the validation was rerun successfully.

Actor:

- Codex

Follow-up items:

- Add replay dry-run and filtering controls.
- Add the first Massive/Polygon scheduled market-data adapter and detector pack.

## Gate G017: Massive Megacap Universe Seed

Timestamp: `2026-07-07T04:04:03Z`

Status: `passed`

Gate name:

- Normalize the initial top 50 megacap company universe for the Massive market-data adapter.

Criteria:

- Parse the provided megacap text file.
- Extract ticker, company, and sector for every listed company.
- Preserve optional industry when the source classification contains `Sector / Industry`.
- Add DB-ready normalized keys for ticker, company, sector, and industry.
- Produce a normalized seed artifact suitable for later persistence.
- Add tests for parser count, representative rows, exchange ticker suffixes, and CSV output.

Evidence:

- `internal/adapters/marketdata/massive/top50megacap.txt`
- `internal/adapters/marketdata/massive/top50megacap.normalized.csv`
- `internal/adapters/marketdata/massive/megacap_seed.go`
- `internal/adapters/marketdata/massive/megacap_seed_test.go`
- `internal/adapters/marketdata/massive/README.md`

Verification performed:

- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./internal/adapters/marketdata/massive`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./...`
- `make docker-test-python`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace python:3.12-slim python scripts/validate_json_schemas.py`
- `head -12 internal/adapters/marketdata/massive/top50megacap.normalized.csv`
- `tail -8 internal/adapters/marketdata/massive/top50megacap.normalized.csv`

Live verification result:

- Parser produced 50 rows.
- First normalized row: `NVDA`, `NVIDIA`, sector `Technology`, industry `Semiconductors`.
- Exchange/class suffixes normalize predictably, including `BRK.B -> brk_b`, `2222.SR -> 2222_sr`, and `005930.KS -> 005930_ks`.
- Final normalized row: `GEV`, `GE Vernova`, sector `Energy`, industry `Industrials`.

Issues found and resolved:

- The repository file is named `top50megacap.txt`, not `50topmegacap.txt`; implementation references the actual file.
- Mixed classification formats are supported: `market cap | sector / industry`, `market cap | sector`, and `sector / industry`.

Actor:

- Codex

Follow-up items:

- Build the Massive scheduled event builder for daily option contracts and EOD prices.
- Add persistence migration once the database layer is introduced.

## Gate G018: Massive Scheduled Event Builder

Timestamp: `2026-07-07T04:12:26Z`

Status: `passed`

Gate name:

- Build canonical raw events from scheduled Massive daily market-data records.

Criteria:

- Add a deterministic builder for daily option contract records.
- Add a deterministic builder for equity end-of-day price records.
- Emit `RawSignalEvent` envelopes with `source_domain=market_data`, `source_adapter=market_data.massive`, and `ingestion_mode=scheduled_pull`.
- Include stable event IDs and idempotency keys for replay-safe scheduled pulls.
- Include entity hints for ticker and option-contract routing.
- Keep external network calls out of the builder layer.
- Add tests for happy paths, stable IDs, JSON envelope fields, and required-field validation.

Evidence:

- `internal/adapters/marketdata/massive/event_builder.go`
- `internal/adapters/marketdata/massive/event_builder_test.go`
- `internal/adapters/marketdata/massive/README.md`

Verification performed:

- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./internal/adapters/marketdata/massive`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./...`
- `make docker-test-python`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace python:3.12-slim python scripts/validate_json_schemas.py`
- `docker compose config --quiet`
- `docker compose ps`
- `docker compose exec redpanda rpk group describe signalops.raw-worker.v1`

Live verification result:

- Targeted Massive adapter tests passed.
- Full Go and Python test gates passed.
- The running Docker stack remained healthy.
- The raw-worker consumer group remained stable with one member and total lag `0`.

Issues found and resolved:

- None. This gate was intentionally offline and fixture-shaped; no Massive API credentials or network calls were introduced.

Actor:

- Codex

Follow-up items:

- Add a Massive HTTP client abstraction and response parser for the selected daily endpoints.
- Add a scheduled pull runner that uses the event builders and publishes raw events to Redpanda.

## Gate G019: Massive HTTP Client And Response Parsers

Timestamp: `2026-07-07T04:24:55Z`

Status: `passed`

Gate name:

- Add fixture-backed Massive HTTP client and parsers for scheduled daily market-data ingestion.

Criteria:

- Add client configuration for Massive base URL and API key.
- Support local `.env` fallback without committing or logging secrets.
- Add request methods for option contract listings, equity daily bars, and option daily bars.
- Parse provider responses into the internal record types consumed by the G018 event builders.
- Document the option aggregate-bar enrichment boundary before event building.
- Keep tests offline and fixture-backed.
- Ensure errors do not leak API key values.

Evidence:

- `internal/adapters/marketdata/massive/client.go`
- `internal/adapters/marketdata/massive/responses.go`
- `internal/adapters/marketdata/massive/client_test.go`
- `internal/adapters/marketdata/massive/README.md`

Implementation notes:

- Option aggregate bars provide price/volume observations only; scheduled option ingestion must enrich them with option contract listing metadata before calling `BuildOptionContractDailyEvent`.

Verification performed:

- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./internal/adapters/marketdata/massive`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./...`
- `make docker-test-python`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace python:3.12-slim python scripts/validate_json_schemas.py`
- `docker compose config --quiet`
- `docker compose ps`
- `docker compose exec redpanda rpk group describe signalops.raw-worker.v1`

Live verification result:

- Fixture-backed client tests passed for option contract listings and equity daily bars.
- API-key precedence tests passed without exposing the key value.
- Error handling test confirmed client errors do not include the API key.
- Full Go and Python test gates passed.
- The running Docker stack remained healthy.
- The raw-worker consumer group remained stable with one member and total lag `0`.

Issues found and resolved:

- The local `.env` uses `API_KEY`; client config supports it as a fallback while documenting more explicit Massive-specific variable names.
- A test syntax typo was found by the targeted Go test and fixed before final validation.

Actor:

- Codex

Follow-up items:

- Add the scheduled pull runner and broker publisher integration.
- Add optional manual live validation using the local Massive API key without logging secrets.



## Gate G020: Massive Scheduled Pull Runner

Timestamp: `2026-07-07T04:46:21Z`

Status: `passed`

Gate name:

- Add the executable scheduled Massive pull path for dry-run and broker publication.

Criteria:

- Combine the megacap seed universe, Massive client, event builders, and broker publisher behind a reusable runner.
- Support equity EOD and option contract scheduled datasets.
- Support dry-run mode that builds events without publishing.
- Support publish mode that writes canonical `RawSignalEvent` JSON to the raw topic with idempotency-key broker keys.
- Add a CLI entrypoint for scheduled execution.
- Add Docker image and Compose profile wiring without starting the puller by default.
- Keep tests fixture-backed and avoid live Massive API calls in automated validation.

Evidence:

- `internal/adapters/marketdata/massive/scheduled_pull.go`
- `internal/adapters/marketdata/massive/scheduled_pull_test.go`
- `cmd/massive-puller/main.go`
- `cmd/massive-puller/main_test.go`
- `Dockerfile`
- `compose.yaml`
- `Makefile`
- `internal/adapters/marketdata/massive/README.md`

Implementation notes:

- The runner defaults to dry-run through the Compose profile.
- Publish mode requires a broker publisher and writes to `signalops.<env>.raw.v1` unless an explicit raw topic is supplied in the runner config.
- Broker messages use the event idempotency key as the record key and include scheduled-pull audit headers.
- The local `.env` API key is consumed through environment variable substitution only; no secret values are committed.

Verification performed:

- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./internal/adapters/marketdata/massive`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./...`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace -e PYTHONPATH=/workspace/python python:3.12-slim python -m unittest discover -s python/tests`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace python:3.12-slim python scripts/validate_json_schemas.py`
- `docker compose config --quiet`
- `docker compose --profile massive-pull config --quiet`
- `docker build --target massive-puller -t signalops-massive-puller:local .`
- `docker compose ps`
- `docker compose exec redpanda rpk group describe signalops.raw-worker.v1`

Live verification result:

- Full Go and Python test gates passed.
- Schema validation passed.
- Docker Compose configuration, including the Massive puller profile, validated successfully.
- The new Massive puller image built successfully.
- The running Docker stack remained healthy.
- The raw-worker consumer group remained stable with one member and total lag `0`.

Issues found and resolved:

- The CLI flag setup was corrected to use a command-specific `FlagSet` rather than package-level flag helpers.
- The Compose profile was extended to expose `SIGNALOPS_MASSIVE_BASE_URL` for provider endpoint overrides.

Actor:

- Codex

Follow-up items:

- Perform a small Massive dry-run using the local `.env` key and one or two seed tickers.
- If provider response compatibility is confirmed, run publish mode for a constrained broker-backed smoke test.
- Add scheduler/orchestrator integration once the one-shot puller is proven.


## Gate G021: Massive Live Dry-Run And Publish Smoke Test

Timestamp: `2026-07-07T04:49:43Z`

Status: `passed`

Gate name:

- Validate the Massive scheduled puller against the live provider API and the local Redpanda raw topic.

Criteria:

- Use the ignored local `.env` key without logging or committing secret values.
- Run a constrained equity dry-run against the provider.
- Run a constrained option-contract dry-run against the provider.
- Run a constrained publish smoke test into `signalops.local.raw.v1`.
- Confirm the Python raw worker consumes the published event.
- Confirm the running stack remains healthy and raw-worker lag returns to `0`.

Evidence:

- `cmd/massive-puller/main.go`
- `internal/adapters/marketdata/massive/scheduled_pull.go`
- `internal/adapters/marketdata/massive/client.go`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Verification performed:

- `docker compose --profile massive-pull run --rm massive-puller --max-companies 1 --datasets equity --dry-run=true --continue-on-error=true`
- `docker compose --profile massive-pull run --rm massive-puller --max-companies 1 --datasets options --options-limit 5 --dry-run=true --continue-on-error=true`
- `docker compose --profile massive-pull run --rm massive-puller --max-companies 1 --datasets equity --dry-run=false --continue-on-error=false`
- `docker compose ps`
- `docker compose exec redpanda rpk group describe signalops.raw-worker.v1`
- `docker compose logs --tail=80 raw-worker`

Live verification result:

- Equity dry-run report: 1 company, 1 event built, 0 events published, 0 failures.
- Option-contract dry-run report: 1 company, 5 events built, 0 events published, 0 failures.
- Equity publish report: 1 company, 1 event built, 1 event published, 0 failures.
- Raw worker logged detector evaluation and successful raw-event processing for the published event.
- Redpanda raw-worker group remained `Stable` with one member and total lag `0`.
- Running stack remained healthy.

Issues found and resolved:

- None. Live response compatibility was confirmed for the constrained equity EOD and option-contract listing paths.

Actor:

- Codex

Follow-up items:

- Expand validation to a small multi-ticker dry-run.
- Add scheduler/orchestrator integration for repeatable Massive pull execution.
- Add production-oriented controls for rate limits, retry/backoff, and provider usage accounting before broad universe runs.


## Gate G022: Massive Rate Controls And Multi-Ticker Dry-Run

Timestamp: `2026-07-07T05:00:57Z`

Status: `passed`

Gate name:

- Add provider usage controls and validate a small multi-ticker Massive dry-run.

Criteria:

- Add request pacing controls for provider calls.
- Add retry/backoff controls for transient provider failures.
- Add report counters for provider requests and retries.
- Expose the controls through the Massive puller CLI and Compose profile.
- Add unit coverage for retry success and retry exhaustion.
- Run a controlled multi-ticker live dry-run without publishing broker messages.
- Confirm the running stack remains healthy and raw-worker lag remains `0`.

Evidence:

- `internal/adapters/marketdata/massive/scheduled_pull.go`
- `internal/adapters/marketdata/massive/scheduled_pull_test.go`
- `cmd/massive-puller/main.go`
- `cmd/massive-puller/main_test.go`
- `compose.yaml`
- `internal/adapters/marketdata/massive/README.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Implementation notes:

- `SIGNALOPS_MASSIVE_REQUEST_DELAY` controls delay before each provider request.
- `SIGNALOPS_MASSIVE_MAX_RETRIES` controls retries per provider request.
- `SIGNALOPS_MASSIVE_RETRY_BACKOFF` controls retry backoff and is multiplied by retry attempt.
- Report output now includes `provider_requests` and `provider_retries` for audit and usage accounting.

Verification performed:

- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./...`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace -e PYTHONPATH=/workspace/python python:3.12-slim python -m unittest discover -s python/tests`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace python:3.12-slim python scripts/validate_json_schemas.py`
- `docker compose --profile massive-pull config --quiet`
- `docker build --target massive-puller -t signalops-massive-puller:local .`
- `docker compose --profile massive-pull build massive-puller`
- `docker compose --profile massive-pull run --rm massive-puller --max-companies 3 --datasets equity,options --options-limit 2 --request-delay 250ms --max-retries 1 --retry-backoff 1s --dry-run=true --continue-on-error=true`
- `docker compose ps`
- `docker compose exec redpanda rpk group describe signalops.raw-worker.v1`

Live verification result:

- Multi-ticker dry-run report: 3 companies, 9 events built, 0 events published, 6 provider requests, 0 provider retries, 0 failures.
- Event counts: 3 `equity_eod_prices` events and 6 `options_contracts_daily` events.
- Redpanda raw-worker group remained `Stable` with one member and total lag `0`.
- Running stack remained healthy.

Issues found and resolved:

- An older Compose-built puller image initially rejected the new pacing/retry flags. Rebuilding `massive-puller` through Compose resolved the artifact mismatch.

Actor:

- Codex

Follow-up items:

- Add scheduler/orchestrator integration for repeatable Massive pull execution.
- Run a constrained scheduled publish validation before broader universe publishing.
- Add provider usage budgeting and persistent run history once the database layer exists.


## Gate G023: Massive Scheduler Integration

Timestamp: `2026-07-07T05:09:56Z`

Status: `passed`

Gate name:

- Add repeatable scheduled execution for the Massive puller.

Criteria:

- Add a reusable scheduled loop around the existing Massive pull runner.
- Add a scheduler CLI entrypoint with interval, max-run, immediate-run, and continue-on-run-error controls.
- Reuse the existing Massive dataset, rate-control, dry-run, and publish configuration.
- Add Docker image target and Compose profile for scheduler execution without starting it by default.
- Keep dry-run enabled by default in Compose.
- Add unit coverage for scheduler loop behavior.
- Validate a bounded one-run live scheduler dry-run without publishing broker messages.
- Confirm the running stack remains healthy and raw-worker lag remains `0`.

Evidence:

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

Implementation notes:

- The scheduler invokes the same `RunScheduledPull` path used by the one-shot puller.
- `SIGNALOPS_MASSIVE_SCHEDULE_INTERVAL` controls interval between runs.
- `SIGNALOPS_MASSIVE_SCHEDULE_MAX_RUNS` bounds runs for validation; `0` means run until stopped.
- `SIGNALOPS_MASSIVE_SCHEDULE_RUN_IMMEDIATELY` runs once at startup before waiting for the interval.
- `SIGNALOPS_MASSIVE_SCHEDULE_CONTINUE_ON_ERROR` controls whether scheduling continues after a pull run returns an error.

Verification performed:

- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./...`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace -e PYTHONPATH=/workspace/python python:3.12-slim python -m unittest discover -s python/tests`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace python:3.12-slim python scripts/validate_json_schemas.py`
- `docker compose --profile massive-schedule config --quiet`
- `docker build --target massive-scheduler -t signalops-massive-scheduler:local .`
- `docker compose --profile massive-schedule build massive-scheduler`
- `docker compose --profile massive-schedule run --rm massive-scheduler --help`
- `docker compose --profile massive-schedule run --rm massive-scheduler --max-runs 1 --max-companies 1 --datasets equity --request-delay 250ms --max-retries 1 --retry-backoff 1s --dry-run=true --continue-on-error=true --continue-on-run-error=false`
- `docker compose ps`
- `docker compose exec redpanda rpk group describe signalops.raw-worker.v1`

Live verification result:

- Scheduler dry-run loop report: 1 run, 1 succeeded, 0 failed.
- Pull report: 1 company, 1 event built, 0 events published, 1 provider request, 0 provider retries, 0 failures.
- Redpanda raw-worker group remained `Stable` with one member and total lag `0`.
- Running stack remained healthy.

Issues found and resolved:

- None. Scheduler dry-run validation passed. The nonzero `--help` status is expected Go flag behavior after usage output.

Actor:

- Codex

Follow-up items:

- Run constrained scheduled publish validation with `max-runs=1` and one equity ticker.
- Add persistent scheduler run history and provider usage accounting once the database layer exists.
- Add Kubernetes CronJob or external orchestrator manifests when deployment targets are finalized.


## Gate G024: Massive Scheduled Publish Validation

Timestamp: `2026-07-07T18:19:15Z`

Status: `passed`

Gate name:

- Validate scheduled Massive publish mode through Redpanda and the raw worker.

Criteria:

- Use the ignored local `.env` key without logging or committing secret values.
- Run `massive-scheduler` with `max-runs=1` and one equity ticker.
- Enable publish mode with `dry-run=false`.
- Confirm one canonical raw event is published to `signalops.local.raw.v1`.
- Confirm the Python raw worker consumes and processes the scheduled-published event.
- Confirm the running stack remains healthy and raw-worker lag returns to `0`.

Evidence:

- `cmd/massive-scheduler/main.go`
- `internal/adapters/marketdata/massive/scheduled_loop.go`
- `internal/adapters/marketdata/massive/scheduled_pull.go`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Verification performed:

- `docker compose --profile massive-schedule run --rm massive-scheduler --max-runs 1 --max-companies 1 --datasets equity --request-delay 250ms --max-retries 1 --retry-backoff 1s --dry-run=false --continue-on-error=false --continue-on-run-error=false`
- `docker compose ps`
- `docker compose exec redpanda rpk group describe signalops.raw-worker.v1`
- `docker compose logs --tail=80 raw-worker`

Live verification result:

- Scheduler run report: 1 company, 1 event built, 1 event published, 1 provider request, 0 provider retries, 0 failures.
- Scheduler loop report: 1 run, 1 succeeded, 0 failed.
- Raw worker logged detector evaluation and successful raw-event processing for the scheduled-published event at `2026-07-07T18:19:07Z`.
- Redpanda raw-worker group remained `Stable` with one member and total lag `0`.
- Running stack remained healthy.

Issues found and resolved:

- None. The constrained scheduled publish path passed without code changes.

Actor:

- Codex

Follow-up items:

- Add persistent scheduler run history and provider usage accounting once the database layer exists.
- Add provider usage budgets and hard run limits before broad scheduled publishing across the full megacap universe.
- Add Kubernetes CronJob or external orchestrator manifests when deployment targets are finalized.


## Gate G025: Massive Provider Usage Budgets

Timestamp: `2026-07-07T19:36:33Z`

Status: `passed`

Gate name:

- Add hard provider and event budgets before broad Massive scheduled runs.

Criteria:

- Add per-run provider request budget enforcement.
- Add per-run built-event budget enforcement.
- Add per-run published-event budget enforcement.
- Expose budgets through one-shot puller and scheduler CLI flags and environment variables.
- Keep Compose defaults disabled with `0` so existing local behavior does not change accidentally.
- Add unit coverage for each budget type.
- Validate a live expected-failure budget stop without publishing broker messages.
- Validate a live positive bounded dry-run without publishing broker messages.
- Confirm the running stack remains healthy and raw-worker lag remains `0`.

Evidence:

- `internal/adapters/marketdata/massive/scheduled_pull.go`
- `internal/adapters/marketdata/massive/scheduled_pull_test.go`
- `cmd/massive-puller/main.go`
- `cmd/massive-scheduler/main.go`
- `compose.yaml`
- `internal/adapters/marketdata/massive/README.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Implementation notes:

- `SIGNALOPS_MASSIVE_MAX_PROVIDER_REQUESTS` stops a run before issuing a provider request that would cross the limit.
- `SIGNALOPS_MASSIVE_MAX_EVENTS_BUILT` stops a run before building a raw event that would cross the limit.
- `SIGNALOPS_MASSIVE_MAX_EVENTS_PUBLISHED` stops a run before publishing a broker message that would cross the limit.
- A value of `0` disables each budget.

Verification performed:

- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./...`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace -e PYTHONPATH=/workspace/python python:3.12-slim python -m unittest discover -s python/tests`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace python:3.12-slim python scripts/validate_json_schemas.py`
- `docker compose --profile massive-schedule config --quiet`
- `docker build --target massive-puller -t signalops-massive-puller:local .`
- `docker build --target massive-scheduler -t signalops-massive-scheduler:local .`
- `docker compose --profile massive-schedule build massive-scheduler`
- `docker compose --profile massive-schedule run --rm massive-scheduler --help`
- `docker compose --profile massive-schedule run --rm massive-scheduler --max-runs 1 --max-companies 2 --datasets equity --request-delay 250ms --max-retries 0 --retry-backoff 1s --max-provider-requests 1 --dry-run=true --continue-on-error=false --continue-on-run-error=false`
- `docker compose --profile massive-schedule run --rm massive-scheduler --max-runs 1 --max-companies 2 --datasets equity --request-delay 250ms --max-retries 0 --retry-backoff 1s --max-provider-requests 2 --max-events-built 2 --dry-run=true --continue-on-error=false --continue-on-run-error=false`
- `docker compose ps`
- `docker compose exec redpanda rpk group describe signalops.raw-worker.v1`

Live verification result:

- Expected-failure budget run stopped with `provider request budget exceeded: limit 1` after 1 provider request, 1 built event, 0 published events, and 1 recorded failure.
- Positive bounded run completed with 2 provider requests, 2 built events, 0 published events, 0 retries, and 0 failures.
- Redpanda raw-worker group remained `Stable` with one member and total lag `0`.
- Running stack remained healthy.

Issues found and resolved:

- None. Budget enforcement passed both stop and allow-path validation.

Actor:

- Codex

Follow-up items:

- Add persistent scheduler run history and provider usage accounting once the database layer exists.
- Add database-backed idempotency, normalized market-data storage, and replay/query support.
- Add Kubernetes CronJob or external orchestrator manifests when deployment targets are finalized.


## Gate G026: PostgreSQL Storage Foundation

Timestamp: `2026-07-07T20:36:30Z`

Status: `passed`

Gate name:

- Add the first durable storage foundation for scheduler audit, provider usage, idempotency, and market-data snapshots.

Criteria:

- Add local PostgreSQL to Docker Compose.
- Add a repeatable migration runner.
- Add a first migration for scheduler run history, provider usage, idempotency records, raw event ledger, equity EOD prices, and option contract daily snapshots.
- Add initial Go storage boundary types and repository interfaces.
- Document local migration usage.
- Validate migration application and idempotent rerun.
- Confirm the running broker/worker stack remains healthy and raw-worker lag remains `0`.

Evidence:

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

Implementation notes:

- PostgreSQL is introduced for operational metadata and audit state.
- TimescaleDB/hypertable conversion remains future scope after base persistence is proven.
- `schema_migrations` records applied migration versions.
- Compose exposes PostgreSQL on host port `15432` and stores data in the `postgres-data` volume.
- Go storage interfaces are intentionally driver-neutral in this gate.

Verification performed:

- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./...`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace -e PYTHONPATH=/workspace/python python:3.12-slim python -m unittest discover -s python/tests`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace python:3.12-slim python scripts/validate_json_schemas.py`
- `docker compose --profile storage config --quiet`
- `docker compose --profile storage run --rm postgres-migrate`
- `docker compose exec postgres psql -U signalops -d signalops -Atc "SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename"`
- `docker compose exec postgres psql -U signalops -d signalops -Atc "SELECT version FROM schema_migrations ORDER BY version"`
- `docker compose --profile storage run --rm postgres-migrate`
- `bash -n scripts/apply_postgres_migrations.sh`
- `make compose-storage-migrate`
- `make compose-validate`
- `docker compose exec redpanda rpk group describe signalops.raw-worker.v1`

Live verification result:

- PostgreSQL became healthy in Compose.
- Migration `000001_storage_foundation` applied successfully.
- Expected tables were present in the `public` schema.
- Migration rerun skipped the already-applied version successfully.
- Redpanda raw-worker group remained `Stable` with one member and total lag `0`.

Issues found and resolved:

- The initial file write was interrupted before migration files were created; files were then written in smaller chunks and validated.

Actor:

- Codex

Follow-up items:

- Add a concrete PostgreSQL repository implementation for scheduler run audit and provider usage writes.
- Persist Massive scheduler run summaries after each scheduled pull run.
- Add database-backed idempotency and raw event ledger writes on publish.


## Gate G027: Massive Scheduler Persistence

Timestamp: `2026-07-07T20:59:45Z`

Status: `passed`

Gate name:

- Persist Massive scheduler run audit and provider usage records to PostgreSQL.

Criteria:

- Add a concrete PostgreSQL repository for `SchedulerRunRepository`.
- Load `SIGNALOPS_DATABASE_URL` from config.
- Wire `massive-scheduler` to persist scheduler run summaries after each scheduled pull run when a database URL is configured.
- Persist provider usage for Massive scheduler runs.
- Enable local Compose scheduler persistence by depending on healthy Postgres and setting the scheduler database URL.
- Document the new persistence behavior.
- Validate unit, schema, image build, Postgres integration, broker integration, and live scheduler dry-run paths.
- Confirm raw-worker lag remains `0`.

Evidence:

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

Implementation notes:

- The repository uses pgx through `database/sql` via `github.com/jackc/pgx/v5/stdlib`.
- Scheduler run rows are upserted by deterministic run id derived from source id and run start timestamp.
- Provider usage rows are upserted by run id and dataset key.
- Single-dataset runs record provider usage against the dataset name.
- Multi-dataset runs record one aggregate provider usage row with dataset `all` to preserve accurate request and retry totals.
- Scheduler persistence is optional and activates only when `SIGNALOPS_DATABASE_URL` is non-empty.

Verification performed:

- `docker compose --profile massive-schedule config --quiet`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm gofmt -w cmd/massive-scheduler/main.go internal/storage/postgres/repository.go internal/storage/postgres/repository_test.go internal/config/config.go internal/config/config_test.go`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./...`
- `docker compose --profile massive-schedule build massive-scheduler`
- `make docker-test-python`
- `make docker-validate-schemas`
- `docker run --rm --network host -e SIGNALOPS_POSTGRES_INTEGRATION=1 -e SIGNALOPS_DATABASE_URL=postgres://signalops:signalops@localhost:15432/signalops?sslmode=disable -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./internal/storage/postgres -run TestRepositoryAgainstPostgres -count=1 -v`
- `docker compose --profile massive-schedule run --rm massive-scheduler --max-runs 1 --max-companies 1 --datasets equity --request-delay 250ms --max-retries 0 --retry-backoff 1s --max-provider-requests 1 --max-events-built 1 --dry-run=true --continue-on-error=false --continue-on-run-error=false`
- `docker compose exec postgres psql -U signalops -d signalops -Atc "SELECT run_id,status,events_built,events_published,provider_requests,provider_retries,failures FROM scheduler_runs ORDER BY started_at DESC LIMIT 3"`
- `docker compose exec postgres psql -U signalops -d signalops -Atc "SELECT run_id,provider,dataset,request_count,retry_count,event_count FROM provider_usage_runs ORDER BY created_at DESC LIMIT 5"`
- `docker run --rm --network host -e SIGNALOPS_BROKER_INTEGRATION=1 -e SIGNALOPS_BROKER_BROKERS=localhost:19092 -e SIGNALOPS_ENV=local -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./internal/broker/kafka -run TestPublishConsumeCommitAgainstRedpanda -count=1 -v`
- `docker compose exec redpanda rpk group describe signalops.raw-worker.v1`

Live verification result:

- Live scheduler dry-run persisted run `massive:src-massive:20260707T205903.415271355Z` with status `succeeded`, `events_built=1`, `events_published=0`, `provider_requests=1`, `provider_retries=0`, and `failures=0`.
- Provider usage persisted for the same run with provider `massive`, dataset `equity_eod_prices`, `request_count=1`, `retry_count=0`, and `event_count=1`.
- Scheduler image build completed successfully.
- Redpanda raw-worker group remained `Stable` with one member and total lag `0`.

Issues found and resolved:

- Local `gofmt` was unavailable; Dockerized `gofmt` was used instead.
- Schema validation initially hit a sandbox loopback failure; rerun with approved escalation passed.
- Multi-dataset usage accounting was adjusted to use aggregate dataset `all` rather than writing misleading zero request/retry counts per dataset.
- Final Go validation caught an over-escaped quote in the Postgres array helper; the helper now uses one backslash for quotes and two for literal backslashes, with test coverage.

Actor:

- Codex

Follow-up items:

- Add database-backed idempotency and raw event ledger persistence on publish.
- Add API access to persisted scheduler run history and provider usage for UI readiness.


## Gate G028: Publish Idempotency And Raw Ledger Persistence

Timestamp: `2026-07-08T00:18:45Z`

Status: `passed`

Gate name:

- Persist broker-acknowledged Massive raw events to PostgreSQL idempotency and raw event ledger tables.

Criteria:

- Extend the storage boundary for raw event ledger persistence.
- Implement Postgres upserts for `idempotency_records` and `raw_event_ledger`.
- Wire Massive scheduled pull publish path to persist after successful broker acknowledgement.
- Record topic, partition, offset, payload JSON, entity hints, timing, route metadata, and payload hash.
- Keep dry-runs free of raw ledger/idempotency writes.
- Wire both `massive-puller` and `massive-scheduler` to use publish persistence when `SIGNALOPS_DATABASE_URL` is configured.
- Validate unit, schema, image build, Postgres integration, broker integration, and live publish paths.
- Confirm raw-worker lag remains `0`.

Evidence:

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

Implementation notes:

- Persistence occurs only after the broker returns a successful publish result.
- `raw_event_ledger.payload` stores the same JSON bytes published to the raw topic.
- `raw_event_ledger.entity_hints` stores the event entity hints separately for query support.
- `idempotency_records.payload_hash` uses `sha256:<hex>` over the published JSON bytes.
- If broker publish succeeds but database persistence fails, the report still counts the event as published and returns the persistence error to the caller.
- The current repository performs raw ledger and idempotency upserts separately; transaction grouping remains future scope if stricter atomicity is required.

Verification performed:

- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm gofmt -w internal/storage/storage.go internal/storage/storage_test.go internal/storage/postgres/repository.go internal/storage/postgres/repository_test.go internal/adapters/marketdata/massive/scheduled_pull.go internal/adapters/marketdata/massive/scheduled_pull_test.go cmd/massive-puller/main.go cmd/massive-scheduler/main.go`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./...`
- `docker run --rm --network host -e SIGNALOPS_POSTGRES_INTEGRATION=1 -e SIGNALOPS_DATABASE_URL=postgres://signalops:signalops@localhost:15432/signalops?sslmode=disable -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./internal/storage/postgres -run TestRepositoryAgainstPostgres -count=1 -v`
- `docker compose --profile massive-schedule config --quiet`
- `docker compose --profile massive-schedule build massive-scheduler`
- `make docker-test-python`
- `make docker-validate-schemas`
- `docker compose --profile massive-schedule run --rm massive-scheduler --max-runs 1 --max-companies 1 --datasets equity --request-delay 250ms --max-retries 0 --retry-backoff 1s --max-provider-requests 1 --max-events-built 1 --max-events-published 1 --dry-run=false --continue-on-error=false --continue-on-run-error=false`
- `docker compose exec postgres psql -U signalops -d signalops -Atc "SELECT run_id,status,dry_run,events_built,events_published,provider_requests,failures FROM scheduler_runs ORDER BY started_at DESC LIMIT 3"`
- `docker compose exec postgres psql -U signalops -d signalops -Atc "SELECT event_id,dataset,broker_topic,broker_partition,broker_offset FROM raw_event_ledger ORDER BY created_at DESC LIMIT 5"`
- `docker compose exec postgres psql -U signalops -d signalops -Atc "SELECT event_id,status,topic,partition,offset_value,left(payload_hash,16) FROM idempotency_records ORDER BY last_seen_at DESC LIMIT 5"`
- `docker run --rm --network host -e SIGNALOPS_BROKER_INTEGRATION=1 -e SIGNALOPS_BROKER_BROKERS=localhost:19092 -e SIGNALOPS_ENV=local -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./internal/broker/kafka -run TestPublishConsumeCommitAgainstRedpanda -count=1 -v`
- `docker compose exec redpanda rpk group describe signalops.raw-worker.v1`

Live verification result:

- Live scheduler publish persisted run `massive:src-massive:20260708T001716.692425267Z` with status `succeeded`, `dry_run=false`, `events_built=1`, `events_published=1`, `provider_requests=1`, and `failures=0`.
- Raw ledger row persisted event `evt_5d5a94a0e8ea5d149ec19947`, dataset `equity_eod_prices`, topic `signalops.local.raw.v1`, partition `2`, offset `3`.
- Idempotency row persisted event `evt_5d5a94a0e8ea5d149ec19947`, status `published`, topic `signalops.local.raw.v1`, partition `2`, offset `3`, hash prefix `sha256:22d0af9ad`.
- Scheduler image build completed successfully.
- Redpanda raw-worker group remained `Stable` with one member and total lag `0`.

Issues found and resolved:

- The previous turn was interrupted after only the storage boundary changed; continuation began by verifying the partial working tree.
- `internal/storage/storage_test.go` was restored after an accidental duplicate-production-content edit.
- Publish count semantics were corrected so broker-acknowledged events are counted before optional database persistence, with a regression test.

Actor:

- Codex

Follow-up items:

- Add API endpoints over scheduler runs, provider usage, raw event ledger, and idempotency records.
- Add UI/UX views once the query APIs expose durable state.
- Evaluate transactional grouping of raw ledger and idempotency writes if future adapter requirements demand atomic persistence.


## Gate G029: Operational Query API

Timestamp: `2026-07-08T00:28:57Z`

Status: `passed`

Gate name:

- Expose persisted scheduler, provider usage, raw event ledger, and idempotency state through gateway query endpoints.

Criteria:

- Add read-side storage contracts for operational query data.
- Implement Postgres query methods for scheduler runs, provider usage, raw event ledger, and idempotency records.
- Add gateway routes for list/detail query surfaces.
- Return JSONB payload/config/report fields as JSON, not base64-encoded bytes.
- Wire local gateway to Postgres through `SIGNALOPS_DATABASE_URL` in Compose.
- Document query endpoints in `docs/api.md`.
- Validate unit, integration, image build, live gateway endpoint, schema, Python, broker, and worker-lag checks.

Evidence:

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

Implementation notes:

- `GET /v1/scheduler/runs` lists recent scheduler runs.
- `GET /v1/scheduler/runs/{run_id}` returns one scheduler run.
- `GET /v1/provider-usage` lists provider usage rows, optionally filtered by `run_id`.
- `GET /v1/raw-events` lists raw event ledger rows, optionally filtered by `tenant_id`, `source_id`, and `dataset`.
- `GET /v1/raw-events/{event_id}` returns one raw event ledger row.
- `GET /v1/idempotency` requires `tenant_id`, `source_id`, and `idempotency_key` and returns one idempotency record.
- Query limits default to `50` and are capped at `200`.
- Gateway query storage is optional; query routes return `503 storage_unavailable` when no repository is configured.

Verification performed:

- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm gofmt -w cmd/gateway/main.go internal/api/router.go internal/api/router_test.go internal/storage/storage.go internal/storage/postgres/repository.go internal/storage/postgres/repository_test.go`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./...`
- `docker run --rm --network host -e SIGNALOPS_POSTGRES_INTEGRATION=1 -e SIGNALOPS_DATABASE_URL=postgres://signalops:signalops@localhost:15432/signalops?sslmode=disable -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./internal/storage/postgres -run TestRepositoryAgainstPostgres -count=1 -v`
- `docker compose config --quiet`
- `docker compose build gateway`
- `docker compose up -d gateway`
- `curl -fsS http://localhost:18000/healthz`
- `curl -fsS 'http://localhost:18000/v1/scheduler/runs?limit=2'`
- `curl -fsS 'http://localhost:18000/v1/raw-events?limit=2'`
- `curl -fsS 'http://localhost:18000/v1/raw-events/evt_5d5a94a0e8ea5d149ec19947'`
- `curl -fsS 'http://localhost:18000/v1/provider-usage?run_id=massive:src-massive:20260708T001716.692425267Z&limit=5'`
- `curl -fsS 'http://localhost:18000/v1/idempotency?tenant_id=tenant-local&source_id=src-massive&idempotency_key=idem_5d5a94a0e8ea5d149ec19947'`
- `make docker-test-python`
- `make docker-validate-schemas`
- `docker run --rm --network host -e SIGNALOPS_BROKER_INTEGRATION=1 -e SIGNALOPS_BROKER_BROKERS=localhost:19092 -e SIGNALOPS_ENV=local -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./internal/broker/kafka -run TestPublishConsumeCommitAgainstRedpanda -count=1 -v`
- `docker compose exec redpanda rpk group describe signalops.raw-worker.v1`

Live verification result:

- Gateway health returned `ok`.
- Scheduler runs endpoint returned persisted run `massive:src-massive:20260708T001716.692425267Z`.
- Raw events endpoint returned persisted event `evt_5d5a94a0e8ea5d149ec19947` with broker topic `signalops.local.raw.v1`, partition `2`, and offset `3`.
- Raw event detail endpoint returned the same event with JSON payload and entity hints.
- Provider usage endpoint returned `massive:src-massive:20260708T001716.692425267Z:equity_eod_prices`.
- Idempotency endpoint returned `idem_5d5a94a0e8ea5d149ec19947` with status `published`.
- Scheduler image build was not part of this gate; gateway image build completed successfully.
- Redpanda raw-worker group remained `Stable` with one member and total lag `0`.

Issues found and resolved:

- Gateway storage open was tightened to only run when `SIGNALOPS_DATABASE_URL` is non-empty.
- A live idempotency check with an incorrect key produced the expected `404`; the correct key from the raw event row was then verified successfully.

Actor:

- Codex

Follow-up items:

- Begin UI/UX dashboard implementation against the G029 query endpoints.
- Add cursor or time-window pagination after initial UI usage patterns are clear.


## Gate G030: Operational Dashboard UI Foundation

Timestamp: `2026-07-08T02:34:21Z`

Status: `passed`

Gate name:

- Scaffold the first SignalOps operational dashboard UI against the G029 query APIs.

Criteria:

- Scaffold `web/` with Vite + React + TypeScript using the adopted stack (TanStack Router/Query, Zustand, ECharts, AG Grid Community, Tailwind, `lucide-react`).
- Implement dashboard shell, health status, Runs, Raw Events, Idempotency, and System views.
- Use the live gateway API by default.
- Resolve browser-to-gateway CORS via a Vite dev proxy.
- Add frontend run instructions to docs.
- Validate with `npm run build` and live API checks through the proxy.

Evidence:

- `web/` (package.json, vite.config.ts, tsconfig.json, tailwind.config.js, postcss.config.js, index.html, src/**)
- `web/README.md`
- `docs/frontend/frontend_evaluation.md`
- `docs/frontend_implementation_spec.md`
- `docs/docker_development.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Implementation notes:

- Dashboard shell renders the `SignalOps` app bar, a gateway health indicator (polls `/healthz` and `/readyz`), and navigation to Runs, Raw Events, Idempotency, and System.
- Runs view: AG Grid table (status, started, source, datasets, dry-run, built, published, provider requests, failures, duration, run id) with a detail panel that fetches provider usage and renders config/report JSON; an ECharts bar chart shows provider requests across recent runs.
- Raw Events view: AG Grid table with optional tenant/source/dataset filters and a detail panel rendering payload and entity-hints JSON with copy controls.
- Idempotency view: form gated until all three fields are present; renders the record on success, shows `No idempotency record found` on 404, and links to the matching raw event.
- System view: `/healthz` and `/readyz` status, API base URL, last refresh, and a storage-availability probe via `/v1/scheduler/runs?limit=1` (200 = available, 503 = unavailable).
- `VITE_SIGNALOPS_API_BASE_URL` defaults to same-origin in dev; the Vite proxy forwards `/healthz`, `/readyz`, `/v1` to `SIGNALOPS_GATEWAY_URL` (default `http://localhost:18000`).
- Route-level `React.lazy` code-splits AG Grid and ECharts; `manualChunks` splits `router`, `echarts`, and `aggrid` vendor chunks.

Verification performed:

- `cd web && npm install`
- `cd web && npm run build`
- `curl -fsS http://localhost:5173/healthz`
- `curl -fsS http://localhost:5173/readyz`
- `curl -fsS 'http://localhost:5173/v1/scheduler/runs?limit=2'`
- `curl -fsS 'http://localhost:5173/v1/raw-events?limit=2'`
- `curl -fsS 'http://localhost:5173/v1/idempotency?tenant_id=tenant-local&source_id=src-massive&idempotency_key=idem_5d5a94a0e8ea5d149ec19947'`
- `curl -s -w '[HTTP %{http_code}]' 'http://localhost:5173/v1/idempotency?tenant_id=tenant-local&source_id=src-massive&idempotency_key=bogus_key_xyz'`
- `curl -s -w '[HTTP %{http_code}]' 'http://localhost:5173/v1/idempotency'`

Live verification result:

- `npm run build` passed (`tsc && vite build`); production build emitted route and vendor chunks.
- Dev server reachable at `http://localhost:5173/`.
- Proxy forwarded all query endpoints to the gateway and returned live persisted data.
- Idempotency 404 and missing-query 400 returned the documented error bodies.

Actor:

- Claude Code

Follow-up items:

- Perform browser validation (console errors, interactions, copy buttons, empty states) as a manual step.
- Add a `web` Compose service and frontend Dockerfile when Compose integration is required.
- Defer React Flow, SSE/WebSocket streaming, and client-side time-series evaluation to later gates pending backend endpoints.
- Add Vitest unit tests for `api/client` and formatting helpers when test coverage is prioritized.


## Gate G030: Operational Dashboard UI Foundation Browser Validation Follow-up

Timestamp: `2026-07-08T03:15:30Z`

Status: `passed`

Gate name:

- Close manual browser validation follow-up for the G030 dashboard UI.

Criteria:

- Confirm the browser app opens and functions after the frontend-agent's build and proxy validation.

Evidence:

- User confirmed: "Broswer works".
- `docs/build_journal.md`
- `docs/gate_audit.md`

Implementation notes:

- No frontend code changes were required for this follow-up.
- The confirmation closes the manual validation item left in the G030 follow-up list.

Verification performed:

- Operator/browser validation was performed outside this agent's tool session and reported as working.

Live verification result:

- Browser UI accepted as working for G030 closeout.

Actor:

- User validation recorded by Codex

Follow-up items:

- Keep frontend adoption of streaming for a later UI gate after G031 exposes the backend SSE contract.


## Gate G031: Backend-to-Frontend Stream Subscription Foundation

Timestamp: `2026-07-08T03:15:30Z`

Status: `passed`

Gate name:

- Add a gateway SSE subscription endpoint for browser-facing dashboard updates.

Criteria:

- Add `GET /v1/streams/dashboard`.
- Support channel filtering for health, scheduler runs, raw events, provider usage, and heartbeat events.
- Emit SSE frames with `event`, optional stable `id`, and JSON `data`.
- Keep Redpanda internal; the browser stream is gateway-owned.
- Deduplicate scheduler run, raw event, and provider usage rows per connection.
- Document the stream endpoint in `docs/api.md`.
- Validate focused API tests, full Go tests, gateway build, and live `curl -N` streaming.

Evidence:

- `docs/api.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`
- `internal/api/router.go`
- `internal/api/router_test.go`

Implementation notes:

- The initial stream uses timed repository polling inside the gateway.
- Omitted `channels` enables all supported channels.
- Unknown channels return `400 invalid_channel` before the stream starts.
- Storage-unavailable state is emitted as an SSE `error` event after the stream opens so clients can keep a live connection and receive heartbeats.
- `Last-Event-ID` replay remains future scope.

Verification performed:

- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm gofmt -w internal/api/router.go internal/api/router_test.go`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./internal/api -count=1 -v`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./...`
- `docker compose config --quiet`
- `docker compose build gateway`
- `docker compose up -d gateway`
- `curl -fsS http://localhost:18000/healthz`
- `curl -fsS 'http://localhost:18000/v1/scheduler/runs?limit=1'`
- `curl -N --max-time 3 'http://localhost:18000/v1/streams/dashboard?channels=health,runs,raw_events,provider_usage'`
- `curl -s -w '[HTTP %{http_code}]' 'http://localhost:18000/v1/streams/dashboard?channels=bogus'`
- `curl -N --max-time 6 'http://localhost:18000/v1/streams/dashboard?channels=heartbeat'`

Live verification result:

- Gateway health returned `ok`.
- Existing scheduler REST query remained operational.
- Dashboard stream emitted `heartbeat`, `health`, `scheduler_run`, `raw_event`, and `provider_usage` events.
- Stable SSE ids were present for scheduler runs, raw events, and provider usage rows.
- Invalid channel requests returned `400 invalid_channel`.
- Heartbeat-only stream emitted the opening heartbeat and a periodic heartbeat after the stream interval.

Actor:

- Codex

Follow-up items:

- Plan frontend SSE adoption as a later UI gate.
- Add `Last-Event-ID` replay only after concrete resume semantics are defined.


## Gate G032: Frontend Dashboard Stream Adoption

Timestamp: `2026-07-08T03:46:26Z`

Status: `passed`

Gate name:

- Adopt the G031 dashboard SSE stream in the frontend through a browser subscription bridge.

Criteria:

- Add a frontend `EventSource` client for `GET /v1/streams/dashboard`.
- Subscribe once at the app shell level and close the subscription on unmount.
- Refresh existing TanStack Query caches for health, scheduler runs, raw events, and provider usage when stream events arrive.
- Surface stream connection state in the existing UI without replacing REST fallback behavior.
- Keep the Vite proxy path working for SSE.
- Update frontend docs/specs to reflect that SSE now exists.

Evidence:

- `docs/frontend_implementation_spec.md`
- `web/README.md`
- `web/src/App.tsx`
- `web/src/api/client.ts`
- `web/src/api/stream.ts`
- `web/src/components/DashboardStreamBridge.tsx`
- `web/src/components/HealthIndicator.tsx`
- `web/src/routes/SystemRoute.tsx`
- `web/src/store/ui.ts`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Implementation notes:

- `buildUrl` is exported from `api/client.ts` so REST and SSE use the same base/proxy behavior.
- `DashboardStreamBridge` owns the app-level `EventSource` subscription.
- Stream events invalidate existing query caches instead of introducing a second state source for operational tables.
- Health and System UI now show stream connection state and last stream event time.

Verification performed:

- `cd web && npm run build`
- `ss -ltnp | rg ':5173|:18000'`
- `curl -fsS http://localhost:5173/healthz`
- `curl -N --max-time 3 'http://localhost:5173/v1/streams/dashboard?channels=health,runs,raw_events,provider_usage'`
- `curl -s -w '[HTTP %{http_code}]' 'http://localhost:5173/v1/streams/dashboard?channels=bogus'`

Live verification result:

- Vite dev proxy forwarded `/healthz` and `/v1/streams/dashboard` to the gateway.
- Dashboard SSE frames arrived through the frontend dev server proxy.
- Invalid channel requests returned `400 invalid_channel` through the frontend dev server proxy.
- Frontend production build passed with TypeScript type checking.

Actor:

- Codex

Follow-up items:

- Add Vitest coverage for stream event parsing and query invalidation.
- Consider direct stream-derived widget state after cache-invalidation adoption has been observed in the browser.


## Gate G033: Compose Web UI Integration

Timestamp: `2026-07-08T03:57:04Z`

Status: `passed`

Gate name:

- Add Docker/Compose runtime integration for the SignalOps operational web UI.

Criteria:

- Add a production-style web image for the Vite frontend.
- Serve the built frontend from a container on a stable local port.
- Proxy `/healthz`, `/readyz`, and `/v1` from the web container to the gateway service.
- Preserve SSE behavior through the proxy.
- Document the Compose web workflow.
- Validate Compose config, image build, service startup, static app serving, REST proxy, and SSE proxy.

Evidence:

- `compose.yaml`
- `docs/deployment.md`
- `docs/docker_development.md`
- `web/Dockerfile`
- `web/README.md`
- `web/deploy/nginx.conf`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Implementation notes:

- `web/Dockerfile` uses `node:22-bookworm-slim` for `npm ci` and `npm run build`, then copies `dist/` into `nginx:1.27-alpine`.
- nginx listens on container port `8080` and serves the SPA with `try_files ... /index.html`.
- nginx proxies `/healthz`, `/readyz`, and `/v1/` to `http://gateway:8080`.
- `/v1/` proxying disables buffering and extends read/send timeouts to support SSE.
- Compose maps host port `15173` to web container port `8080`.

Verification performed:

- `docker compose config --quiet`
- `docker compose build web`
- `docker compose up -d web`
- `curl -fsS http://localhost:15173/ | head -20`
- `curl -fsS http://localhost:15173/healthz`
- `curl -fsS 'http://localhost:15173/v1/scheduler/runs?limit=1'`
- `curl -N --max-time 3 'http://localhost:15173/v1/streams/dashboard?channels=health,heartbeat'`
- `docker compose ps web`

Live verification result:

- Built HTML shell was served from `http://localhost:15173/`.
- Gateway health and scheduler query worked through the web container proxy.
- SSE stream emitted `heartbeat` and `health` frames through the web container proxy.
- `signalops-web-1` is running and exposes `0.0.0.0:15173->8080/tcp`.

Actor:

- Codex

Follow-up items:

- Assess npm audit findings in a dedicated frontend dependency hardening gate.
- Add frontend stream/component tests.


## Gate G034: Frontend Stream Test and Dependency Hardening

Timestamp: `2026-07-08T04:12:01Z`

Status: `passed`

Gate name:

- Add focused frontend test coverage for stream behavior and remediate safe dependency audit findings.

Criteria:

- Add Vitest coverage for dashboard stream parsing and subscription behavior.
- Add Vitest coverage for formatting helpers used across operational tables/details.
- Preserve frontend production build.
- Remediate non-major npm audit findings when safe.
- Record any remaining dependency findings that require larger compatibility work.
- Validate the Compose web image still builds.

Evidence:

- `web/src/api/stream.ts`
- `web/src/api/stream.test.ts`
- `web/src/lib/format.test.ts`
- `web/package.json`
- `web/package-lock.json`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Implementation notes:

- `parseDashboardStreamData` and `toDashboardStreamEvent` expose deterministic stream parsing behavior for unit tests.
- `stream.test.ts` uses a local fake `EventSource`; it does not require browser automation.
- PostCSS was updated to `8.5.16` through npm.
- ECharts remains on the existing major version because the audit fix requires a semver-major upgrade to `6.1.0`.

Verification performed:

- `cd web && npm test`
- `cd web && npm run build`
- `cd web && npm audit --json`
- `docker compose build web`

Live verification result:

- Vitest passed: 2 files, 6 tests.
- Frontend production build passed.
- Compose web image build passed.
- npm audit now reports only one moderate finding: ECharts XSS advisory fixed by semver-major `echarts@6.1.0`.

Actor:

- Codex

Follow-up items:

- Plan an ECharts 6 compatibility gate or charting dependency replacement.
- Add component-level frontend tests for stream-driven query invalidation when test infrastructure is expanded.


## Gate G035: ECharts Security Upgrade Compatibility

Timestamp: `2026-07-08T04:25:24Z`

Status: `passed`

Gate name:

- Upgrade ECharts to the audited fixed major version and validate frontend compatibility.

Criteria:

- Confirm `echarts-for-react` supports ECharts 6.
- Upgrade `echarts` to `6.1.0`.
- Preserve frontend unit tests and production build.
- Confirm npm audit reports zero vulnerabilities.
- Validate the Compose web image and running web service after the upgrade.

Evidence:

- `web/package.json`
- `web/package-lock.json`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Implementation notes:

- `echarts-for-react` peer dependencies include `echarts` `^6.0.0`, so no adapter package change was required.
- The current Runs chart uses a simple ECharts option object through `echarts-for-react`; no code changes were needed for ECharts 6.1.0.

Verification performed:

- `npm view echarts version peerDependencies --json`
- `npm view echarts-for-react version peerDependencies --json`
- `cd web && npm install echarts@6.1.0`
- `cd web && npm test`
- `cd web && npm run build`
- `cd web && npm audit --json`
- `docker compose build web`
- `docker compose up -d web`
- `curl -fsS http://localhost:15173/ | head -20`
- `curl -fsS 'http://localhost:15173/v1/scheduler/runs?limit=1'`
- `curl -N --max-time 3 'http://localhost:15173/v1/streams/dashboard?channels=health,heartbeat'`
- `docker compose ps web`

Live verification result:

- npm audit reported zero vulnerabilities.
- Vitest passed: 2 files, 6 tests.
- Frontend production build passed.
- Compose web image build passed with `npm ci` reporting zero vulnerabilities.
- Web container served the rebuilt app shell on `http://localhost:15173/`.
- Scheduler REST and dashboard SSE proxy paths remained operational through the web container.

Actor:

- Codex

Follow-up items:

- Add browser-level chart rendering validation when frontend browser automation is added.
- Resume backend platform capability work now that frontend dependency audit is clean.


## Gate G036: Source Catalog Foundation

Timestamp: `2026-07-08T04:54:17Z`

Status: `passed`

Gate name:

- Add the first tenant-scoped Stream Catalog source registry and query API.

Criteria:

- Add a durable source catalog table and migration.
- Seed the local Massive market-data source.
- Add storage records and repository methods for catalog sources.
- Add a tenant-scoped gateway API for listing catalog sources.
- Document the API and deployment behavior.
- Validate unit, integration, migration, image build, live gateway, and web-proxy checks.

Evidence:

- `migrations/000002_catalog_sources.up.sql`
- `migrations/000002_catalog_sources.down.sql`
- `internal/storage/storage.go`
- `internal/storage/postgres/repository.go`
- `internal/storage/postgres/repository_test.go`
- `internal/api/router.go`
- `internal/api/router_test.go`
- `docs/api.md`
- `docs/deployment.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Implementation notes:

- `catalog_sources` is keyed by `(tenant_id, source_id)`.
- Source records include domain, adapter, display name, description, status, ingestion modes, datasets, metadata, and timestamps.
- Initial status values are `active`, `inactive`, and `deprecated`.
- Migration `000002_catalog_sources` seeds `tenant-local/src-massive` as an active `market_data.massive` scheduled-pull source.
- `GET /v1/tenants/{tenant_id}/catalog/sources?limit=50` returns `{"sources":[...]}`.
- The endpoint is read-only; source registration APIs remain future scope.

Verification performed:

- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm gofmt -w internal/storage/storage.go internal/storage/postgres/repository.go internal/storage/postgres/repository_test.go internal/api/router.go internal/api/router_test.go`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./internal/api ./internal/storage ./internal/storage/postgres -count=1`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./...`
- `docker compose config --quiet`
- `make compose-storage-migrate`
- `docker compose build gateway`
- `docker compose up -d gateway`
- `docker run --rm --network host -e SIGNALOPS_POSTGRES_INTEGRATION=1 -e SIGNALOPS_DATABASE_URL=postgres://signalops:signalops@localhost:15432/signalops?sslmode=disable -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./internal/storage/postgres -run TestRepositoryAgainstPostgres -count=1 -v`
- `curl -fsS 'http://localhost:18000/v1/tenants/tenant-local/catalog/sources?limit=10'`
- `curl -fsS 'http://localhost:15173/v1/tenants/tenant-local/catalog/sources?limit=10'`
- `curl -fsS http://localhost:18000/healthz`
- `docker compose exec postgres psql -U signalops -d signalops -Atc "SELECT tenant_id,source_id,source_adapter,status,array_to_string(datasets, ',') FROM catalog_sources ORDER BY source_id"`

Live verification result:

- Gateway returned the seeded `tenant-local/src-massive` catalog source.
- Web proxy returned the same catalog response.
- Gateway health remained `ok` after restart.
- Postgres catalog query showed `tenant-local/src-massive` and the integration-test `tenant-1/src-massive` rows.

Actor:

- Codex

Follow-up items:

- Add frontend Sources page wired to the catalog source API.
- Add catalog tables/APIs for pipelines and rules.


## Gate G037: Frontend Sources Catalog Page

Timestamp: `2026-07-08T05:07:40Z`

Status: `passed`

Gate name:

- Add the first frontend Sources page backed by the source catalog API.

Criteria:

- Add TypeScript contracts for catalog sources.
- Add API client and TanStack Query hook for `GET /v1/tenants/{tenant_id}/catalog/sources`.
- Add `/sources` route and navigation entry.
- Render real source catalog data without mock records.
- Validate frontend tests, production build, web image build, running web route, catalog API proxy, and npm audit.

Evidence:

- `web/src/types.ts`
- `web/src/api/client.ts`
- `web/src/api/queries.ts`
- `web/src/router.tsx`
- `web/src/routes/SourcesRoute.tsx`
- `web/src/components/DashboardShell.tsx`
- `web/src/components/StatusBadge.tsx`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Implementation notes:

- The Sources page initially uses tenant `tenant-local` because tenant selection/auth is not implemented.
- The page displays registered source count, active source count, dataset count, source rows, and metadata JSON.
- The page is read-only and backed only by the catalog source API.

Verification performed:

- `cd web && npm test`
- `cd web && npm run build`
- `curl -fsS 'http://localhost:15173/v1/tenants/tenant-local/catalog/sources?limit=10'`
- `docker compose build web`
- `docker compose up -d web`
- `curl -fsS http://localhost:15173/sources | head -20`
- `curl -fsS 'http://localhost:15173/v1/tenants/tenant-local/catalog/sources?limit=10'`
- `docker compose ps web`
- `cd web && npm audit --json`

Live verification result:

- Vitest passed: 2 files, 6 tests.
- Frontend production build passed.
- Web image build passed.
- `/sources` served the SPA shell through the Compose web container.
- Catalog source API returned `tenant-local/src-massive` through the web proxy.
- npm audit reported zero vulnerabilities.

Actor:

- Codex

Follow-up items:

- Add catalog APIs/pages for pipelines and rules.
- Add tenant selection after tenant/auth context exists.


## Gate G038: Backend Pipeline Catalog Foundation

Timestamp: `2026-07-08T05:21:42Z`

Status: `passed`

Gate name:

- Add the backend pipeline catalog for tenant-scoped processing topology visibility.

Criteria:

- Add a durable `catalog_pipelines` migration and seed the local Massive raw ingest pipeline.
- Add storage contracts and Postgres upsert/list methods for catalog pipelines.
- Add `GET /v1/tenants/{tenant_id}/catalog/pipelines?limit=50`.
- Add unit/integration coverage for validation, repository round trip, and API response shape.
- Validate formatting, Go tests, Compose config, migration, gateway rebuild/restart, and live API responses through gateway and web proxy.

Evidence:

- `migrations/000003_catalog_pipelines.up.sql`
- `migrations/000003_catalog_pipelines.down.sql`
- `internal/storage/storage.go`
- `internal/storage/postgres/repository.go`
- `internal/storage/postgres/repository_test.go`
- `internal/api/router.go`
- `internal/api/router_test.go`
- `docs/api.md`
- `docs/deployment.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Implementation notes:

- The seeded pipeline is `tenant-local/pipeline-massive-raw-ingest`.
- The seed captures scheduled pull, raw event build, broker publish, raw ledger persistence, and idempotency persistence stages.
- The metadata explicitly marks the provider as Massive, formerly polygon.io, and `streaming:false` for the current data scope.

Verification performed:

- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm gofmt -w internal/storage/storage.go internal/storage/postgres/repository.go internal/storage/postgres/repository_test.go internal/api/router.go internal/api/router_test.go`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./internal/api ./internal/storage ./internal/storage/postgres -count=1`
- `docker compose config --quiet`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./... -count=1`
- `make compose-storage-migrate`
- `docker compose build gateway`
- `docker compose up -d gateway`
- `docker run --rm --network host -e SIGNALOPS_POSTGRES_INTEGRATION=1 -e SIGNALOPS_DATABASE_URL=postgres://signalops:signalops@localhost:15432/signalops?sslmode=disable -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./internal/storage/postgres -run TestRepositoryAgainstPostgres -count=1 -v`
- `curl -fsS http://localhost:18000/healthz`
- `curl -fsS 'http://localhost:18000/v1/tenants/tenant-local/catalog/pipelines?limit=10'`
- `curl -fsS 'http://localhost:15173/v1/tenants/tenant-local/catalog/pipelines?limit=10'`
- `docker compose exec postgres psql -U signalops -d signalops -Atc "SELECT tenant_id,pipeline_id,status,array_to_string(stages, ',') FROM catalog_pipelines ORDER BY pipeline_id"`

Live verification result:

- Gateway health returned `ok` after restart.
- Gateway returned the seeded `tenant-local/pipeline-massive-raw-ingest` catalog pipeline.
- Web proxy returned the same catalog response.
- Postgres catalog query showed `tenant-local/pipeline-massive-raw-ingest` and the integration-test `tenant-1/pipeline-massive-raw-ingest` rows.

Actor:

- Codex

Follow-up items:

- Add frontend Pipelines page wired to the catalog pipeline API.
- Add rules catalog foundation after pipeline visibility lands.


## Gate G039: Frontend Pipelines Catalog Page

Timestamp: `2026-07-08T05:46:15Z`

Status: `passed`

Gate name:

- Add the first frontend Pipelines page backed by the pipeline catalog API.

Criteria:

- Add TypeScript contracts for catalog pipelines.
- Add API client and TanStack Query hook for `GET /v1/tenants/{tenant_id}/catalog/pipelines`.
- Add `/pipelines` route and navigation entry.
- Render real pipeline catalog data without mock records.
- Validate frontend tests, production build, web image build, running web route, catalog API proxy, and npm audit.

Evidence:

- `web/src/types.ts`
- `web/src/api/client.ts`
- `web/src/api/queries.ts`
- `web/src/router.tsx`
- `web/src/routes/PipelinesRoute.tsx`
- `web/src/components/DashboardShell.tsx`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Implementation notes:

- The Pipelines page initially uses tenant `tenant-local` because tenant selection/auth is not implemented.
- The page displays registered pipeline count, active pipeline count, distinct stage count, output topic count, source linkage, stage flow, inputs, outputs, and metadata JSON.
- The page is read-only and backed only by the catalog pipeline API.

Verification performed:

- `cd web && npm test`
- `cd web && npm run build`
- `docker compose build web`
- `docker compose up -d web`
- `curl -fsS http://localhost:15173/pipelines`
- `curl -fsS 'http://localhost:15173/v1/tenants/tenant-local/catalog/pipelines?limit=10'`
- `docker compose ps web`
- `cd web && npm audit --json`

Live verification result:

- Vitest passed: 2 files, 6 tests.
- Frontend production build passed.
- Web image build passed.
- `/pipelines` served the SPA shell through the Compose web container.
- Catalog pipeline API returned `tenant-local/pipeline-massive-raw-ingest` through the web proxy.
- npm audit reported zero vulnerabilities.

Actor:

- Codex

Follow-up items:

- Add catalog APIs/pages for rules.
- Add tenant selection after tenant/auth context exists.


## Gate G040: Backend Rules Catalog Foundation

Timestamp: `2026-07-08T05:54:23Z`

Status: `passed`

Gate name:

- Add the backend rules catalog for tenant-scoped rule-definition visibility.

Criteria:

- Add a durable `catalog_rules` migration and seed the local Massive EOD price quality rule.
- Add storage contracts and Postgres upsert/list methods for catalog rules.
- Add `GET /v1/tenants/{tenant_id}/catalog/rules?limit=50`.
- Add unit/integration coverage for validation, repository round trip, and API response shape.
- Validate formatting, Go tests, Compose config, migration, gateway rebuild/restart, and live API responses through gateway and web proxy.

Evidence:

- `migrations/000004_catalog_rules.up.sql`
- `migrations/000004_catalog_rules.down.sql`
- `internal/storage/storage.go`
- `internal/storage/postgres/repository.go`
- `internal/storage/postgres/repository_test.go`
- `internal/api/router.go`
- `internal/api/router_test.go`
- `docs/api.md`
- `docs/deployment.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Implementation notes:

- The seeded rule is `tenant-local/rule-marketdata-eod-price-quality`.
- The seed captures a catalog-only quality check for missing or non-positive Massive EOD close prices.
- The rule is linked to `src-massive` and `pipeline-massive-raw-ingest` and explicitly marks `streaming:false` in metadata.

Verification performed:

- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm gofmt -w internal/storage/storage.go internal/storage/postgres/repository.go internal/storage/postgres/repository_test.go internal/api/router.go internal/api/router_test.go`
- `docker compose config --quiet`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./internal/api ./internal/storage ./internal/storage/postgres -count=1`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./... -count=1`
- `make compose-storage-migrate`
- `docker compose build gateway`
- `docker compose up -d gateway`
- `docker run --rm --network host -e SIGNALOPS_POSTGRES_INTEGRATION=1 -e SIGNALOPS_DATABASE_URL=postgres://signalops:signalops@localhost:15432/signalops?sslmode=disable -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./internal/storage/postgres -run TestRepositoryAgainstPostgres -count=1 -v`
- `curl -fsS http://localhost:18000/healthz`
- `curl -fsS 'http://localhost:18000/v1/tenants/tenant-local/catalog/rules?limit=10'`
- `curl -fsS 'http://localhost:15173/v1/tenants/tenant-local/catalog/rules?limit=10'`
- `docker compose exec postgres psql -U signalops -d signalops -Atc "SELECT tenant_id,rule_id,rule_type,severity,status,array_to_string(dataset_scope, ',') FROM catalog_rules ORDER BY rule_id"`

Live verification result:

- Gateway health returned `ok` after restart.
- Gateway returned the seeded `tenant-local/rule-marketdata-eod-price-quality` catalog rule.
- Web proxy returned the same catalog response.
- Postgres catalog query showed `tenant-local/rule-marketdata-eod-price-quality` with type `quality_check`, severity `medium`, status `active`, and dataset scope `equity_eod_prices`.

Actor:

- Codex

Follow-up items:

- Write the frontend-agent implementation specification for Rules UI.
- Add rule execution persistence and signal/insight persistence in later gates.


## Gate G041 Specification: Frontend Rules Catalog Page

Timestamp: `2026-07-08T06:02:03Z`

Status: `ready for implementation`

Gate name:

- Define the frontend-agent implementation contract for the Rules catalog page.

Criteria:

- Place the handoff specification under `docs/frontend/`.
- Ground the specification in the G040 rules catalog API.
- Reuse existing Sources/Pipelines frontend patterns rather than defining a new frontend architecture.
- Define non-goals to prevent mock rule execution, editing, or signal/insight functionality.
- Define validation commands and documentation requirements for the implementation gate.

Evidence:

- `docs/frontend/rules_ui_implementation_spec.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Implementation notes:

- The actual frontend implementation remains future work for G041.
- The spec targets a read-only `/rules` page backed by `GET /v1/tenants/tenant-local/catalog/rules?limit=50`.
- The spec requires the frontend agent to update journal and gate audit after implementation.

Verification performed:

- `head -40 docs/frontend/rules_ui_implementation_spec.md`
- `rg -n "G041|catalog/rules|RulesRoute|Acceptance Criteria|Validation Requirements" docs/frontend/rules_ui_implementation_spec.md`

Live verification result:

- Not applicable; this was a documentation handoff deliverable.

Actor:

- Codex

Follow-up items:

- Frontend agent implements G041 according to `docs/frontend/rules_ui_implementation_spec.md`.


## Gate G041: Frontend Rules Catalog Page

Timestamp: `2026-07-08T06:30:17Z`

Status: `passed`

Gate name:

- Add the first frontend Rules page backed by the rules catalog API.

Criteria:

- Add TypeScript contracts for catalog rules.
- Add API client and TanStack Query hook for `GET /v1/tenants/{tenant_id}/catalog/rules`.
- Add `/rules` route and navigation entry.
- Render real rule catalog data without mock records.
- Validate frontend tests, production build, web image build, running web route, catalog API proxy, and npm audit.

Evidence:

- `web/src/types.ts`
- `web/src/api/client.ts`
- `web/src/api/queries.ts`
- `web/src/router.tsx`
- `web/src/routes/RulesRoute.tsx`
- `web/src/components/DashboardShell.tsx`
- `docs/frontend/rules_ui_implementation_spec.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Implementation notes:

- The Rules page uses tenant `tenant-local` until tenant selection/auth exists.
- The page renders registered/active rule counts, distinct rule types, critical+high count, a plain HTML table (Rule/Type/Severity/Scope/Actions/Status/Updated), and Rule Expressions + Rule Metadata JSON sections.
- Severity renders as restrained colored text (no shared severity badge; no new color-heavy visual system); status uses the existing `StatusBadge`.
- The page is read-only and backed only by the catalog rules API; no edit/create/delete, execution, or test-runner controls.

Verification performed:

- `cd web && npm test`
- `cd web && npm run build`
- `docker compose build web`
- `docker compose up -d web`
- `curl -fsS http://localhost:15173/rules`
- `curl -fsS 'http://localhost:15173/v1/tenants/tenant-local/catalog/rules?limit=10'`
- `docker compose ps web`
- `cd web && npm audit --json`

Live verification result:

- Vitest passed: 2 files, 6 tests.
- Frontend production build passed; `RulesRoute` lazy-loaded.
- Web image build passed; container Up on `:15173`.
- `/rules` served the SPA shell through the Compose web container.
- Catalog rules API returned `tenant-local/rule-marketdata-eod-price-quality` through the web proxy.
- npm audit reported zero vulnerabilities.

Actor:

- Claude Code

Follow-up items:

- Browser validation (rendering, console errors) as a manual step.
- Rule execution history, expression builder, and rule management remain out of scope pending backend support.


## Gate G042: Generic Raw Ingest Persistence

Timestamp: `2026-07-08T20:01:14Z`

Status: `passed`

Gate name:

- Persist generic raw gateway ingestion after durable broker acknowledgement.

Criteria:

- Validate ledger-required event identity before publishing.
- Publish first and retain acknowledged topic, partition, and offset.
- Atomically persist raw ledger and idempotency records.
- Preserve heterogeneous payloads and entity hints without domain-specific mapping.
- Expose explicit broker and post-acknowledgement persistence failures.
- Pass unit, Docker build, deployment, API, and direct database validation.

Evidence:

- `internal/api/router.go`
- `internal/storage/postgres/repository.go`
- `internal/api/router_test.go`
- `docs/api.md`
- `docs/build_journal.md`

Verification performed:

- `make docker-test`
- `docker compose build gateway`
- `docker compose up -d postgres redpanda gateway`
- Live `POST /v1/events/raw` for `g042-live-event`
- Live `GET /v1/raw-events/g042-live-event`
- Live idempotency lookup for `g042-live-key`
- Direct PostgreSQL join of `raw_event_ledger` and `idempotency_records`

Live verification result:

- All Go packages passed.
- Gateway image and deployment passed.
- Event was acknowledged and persisted at topic `signalops.local.raw.v1`, partition `2`, offset `5`.
- Both records shared the identity and coordinates; idempotency status was `published`.

Actor:

- Codex

Follow-up items:

- Broker/database atomicity remains an indeterminate edge when publication succeeds and persistence fails; stable idempotency identifiers and idempotent consumers remain mandatory.


## Gate G043: Frontend First-Class Dashboard

Timestamp: `2026-07-08T20:01:14Z`

Status: `ready for implementation`

Gate name:

- Promote `/` into a first-class operational Dashboard.

Criteria:

- Compose current health, runs, raw events, provider usage, sources, pipelines, and rules.
- Preserve independent widget loading/error/empty states.
- Use one Dashboard SSE subscription to invalidate relevant REST query state.
- Keep unsupported alerts, timeline, correlation, insights, and knowledge capabilities out of the UI until backend contracts exist.
- Validate tests, build, audit, Compose, live proxy data, desktop/mobile rendering, and browser console.

Evidence:

- `docs/frontend/dashboard_ui_implementation_spec.md`

Implementation notes:

- G043 is a frontend-agent handoff and is not yet passed.
- The specification reuses the existing React, TanStack Query, SSE, route, and component architecture.

Actor:

- Codex


## Gate G043 (Implementation): Frontend First-Class Dashboard

Timestamp: `2026-07-08T20:53:18Z`

Status: `passed`

Gate name:

- Promote `/` into a first-class operational Dashboard composing current backend data areas.

Criteria:

- Compose health, runs, raw events, provider usage, sources, pipelines, and rules.
- Preserve independent widget loading/error/empty states.
- Use the existing Dashboard SSE subscription to invalidate relevant REST query state (no second EventSource).
- Keep unsupported alerts, timeline, correlation, insights, and knowledge capabilities out of the UI.
- Validate tests, build, audit, Compose, and live proxy data.

Evidence:

- `web/src/routes/DashboardRoute.tsx`
- `web/src/api/queries.ts`
- `web/src/router.tsx`
- `web/src/components/DashboardShell.tsx`
- `docs/frontend/dashboard_ui_implementation_spec.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Implementation notes:

- `/` renders `DashboardRoute`; `/runs` remains `RunsRoute`; a Dashboard nav item (`LayoutDashboard`) is first.
- The Dashboard consumes the global `DashboardStreamBridge` (mounted in `App.tsx`) for cache invalidation and reads `streamConnected`/`lastStreamEventAt`/`streamError` from `useUi` — no second subscription.
- Added `useRecentProviderUsage` for unfiltered provider usage (the existing `useProviderUsage` is `run_id`-gated).
- Layout: metrics strip, Processing Health, Catalog Inventory, Recent Runs, Provider Usage, and a full-width Recent Event Stream; per-widget failure isolation; event rows link to plain `/raw-events`.

Verification performed:

- `cd web && npm test`
- `cd web && npm run build`
- `cd web && npm audit --json`
- `docker compose build web`
- `docker compose up -d web`
- `curl -fsS http://localhost:15173/`
- `curl -fsS http://localhost:15173/healthz`
- `curl -fsS 'http://localhost:15173/v1/provider-usage?limit=5'`
- `curl -sN --max-time 2 'http://localhost:15173/v1/streams/dashboard?channels=health,heartbeat'`
- `docker compose ps web`

Live verification result:

- Vitest passed: 2 files, 6 tests.
- Frontend production build passed; `DashboardRoute` lazy-loaded.
- Web image rebuilt; container Up on `:15173`; `/` serves the SPA shell.
- Unfiltered provider usage and SSE `health`/`heartbeat` events returned through the web proxy.
- npm audit reported zero vulnerabilities.

Actor:

- Claude Code

Follow-up items:

- Browser/Playwright validation (rendering, console errors, 375px layout) as a manual step.
- Alerts, timeline/correlation, insights, and rule execution remain out of scope pending backend contracts.


## Gate G044: Durable Normalized Event Pipeline

Timestamp: `2026-07-08T21:18:15Z`

Status: `passed`

Gate name:

- Persist normalized events between raw ingestion and Python algorithm processing.

Criteria:

- Consume raw events through a standalone Go infrastructure service.
- Produce the checked-in `NormalizedSignalEvent` v1 shape without domain-specific coupling.
- Publish to the durable normalized topic before committing the raw offset.
- Persist canonical normalized state and raw-to-normalized broker lineage before committing.
- Route invalid source contracts to DLQ and retry infrastructure failures without committing.
- Move Python algorithm consumption to the normalized topic.
- Expose normalized-event list and detail APIs.
- Pass Go/Python tests, migration, Compose build/deployment, API, database, and consumer-group validation.

Evidence:

- `cmd/normalizer/main.go`
- `internal/normalization/processor.go`
- `internal/normalization/processor_test.go`
- `migrations/000005_normalized_events.up.sql`
- `internal/storage/postgres/repository.go`
- `internal/api/router.go`
- `compose.yaml`
- `docs/build_journal.md`

Verification performed:

- `make docker-test`
- `make docker-test-python`
- `docker compose config --quiet`
- `docker compose --profile storage run --rm postgres-migrate`
- `docker compose build gateway normalizer raw-worker`
- `docker compose up -d gateway normalizer raw-worker`
- Live gateway POST, normalized detail API query, direct PostgreSQL query, service logs, and Redpanda group descriptions.

Live verification result:

- `g044-live-event` traversed raw partition/offset `2/6` to normalized partition/offset `2/2`.
- The normalized ledger retained canonical payload, entity, evidence, metadata, complete event, and both broker positions.
- Normalizer and Python normalized-worker groups were Stable with total lag `0`.
- Python worker logged detector evaluation and processing of the live normalized event.
- The persisted event passed `normalized_signal_event.v1.schema.json` runtime validation.
- A rebuilt normalizer remained Up after restart; its group returned to Stable with one member and lag `0` after typed franz-go partition-reset recovery was added.

Actor:

- Codex

Follow-up items:

- Persist Python-emitted signals through a Go signal consumer before adding signal/insight UI.
- Add explicit replay observability for raw-to-normalized duplicate publication after a persistence failure.


## Gate G045: Durable Signal Persistence

Timestamp: `2026-07-08T21:41:02Z`

Status: `passed`

Gate name:

- Persist Python-emitted signals through a Go infrastructure consumer.

Criteria:

- Consume the durable `signal.v1` topic independently of Python workers.
- Validate the closed signal contract at the Go persistence boundary.
- Persist detector/model identity, normalized event lineage, temporal windows, confidence, severity, evidence, recommendation, full event JSON, and broker coordinates.
- Commit source offsets only after successful persistence.
- Route invalid signals to DLQ and retry infrastructure failures without commit.
- Expose signal list/detail APIs with operational filters.
- Pass tests, migration, Docker deployment, live Python emission, API, database, and consumer-group validation.

Evidence:

- `cmd/signal-persister/main.go`
- `internal/signals/processor.go`
- `internal/signals/processor_test.go`
- `migrations/000006_signal_ledger.up.sql`
- `internal/storage/postgres/repository.go`
- `internal/api/router.go`
- `python/signalops_workers/worker.py`
- `compose.yaml`
- `docs/build_journal.md`

Verification performed:

- `make docker-test`
- `make docker-test-python`
- `docker compose --profile storage run --rm postgres-migrate`
- `docker compose build gateway signal-persister`
- `docker compose up -d gateway signal-persister`
- Deterministic Python static detector run with one normalized input.
- Live signal detail API, direct PostgreSQL query, service logs, and Redpanda group description.

Live verification result:

- Signal `signalops.static_test.low` persisted from broker position `0/3` with detector/model metadata and normalized-event lineage.
- API and PostgreSQL values matched.
- `signalops.signal-persister.v1` was Stable with one member and lag `0` after service restart.
- Runtime `signal.v1` validation passed; evidence linked the normalized event and the filtered list API returned the expected record.

Actor:

- Codex

Follow-up items:

- Add Signals UI and Dashboard integration.
- Add alert and insight lifecycle persistence derived from durable signals.


## Gate G046: Frontend Normalized Events and Signals UI

Timestamp: `2026-07-08T22:13:05Z`

Status: `passed`

Gate name:

- Expose the G044 normalized-event and G045 signal ledgers in the web frontend.

Criteria:

- Add Normalized Events (`/normalized-events`) and Signals (`/signals`) read-only pages.
- Use real G044/G045 REST APIs with typed client methods and TanStack Query hooks.
- Support truthful loading, error, empty, list, selection, and detail states.
- Add Dashboard summaries without fabricating alerts or insights.
- Validate tests, build, audit, Compose, and live proxy data.

Evidence:

- `web/src/routes/NormalizedEventsRoute.tsx`
- `web/src/routes/SignalsRoute.tsx`
- `web/src/routes/DashboardRoute.tsx`
- `web/src/types.ts`
- `web/src/api/client.ts`
- `web/src/api/queries.ts`
- `web/src/router.tsx`
- `web/src/components/DashboardShell.tsx`
- `docs/frontend/normalized_signals_ui_implementation_spec.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Implementation notes:

- Corrected the spec's contract before implementing: `schema_name`→`schema_id`, `metrics`→`supporting_metrics`, `signal_*`→`broker_*`, removed `model_id`, required broker coords + `window_*`, removed the fabricated `400 invalid_limit`.
- Both pages use plain HTML tables + detail panels with `JsonViewer`; severity renders as a local badge; signal `event_ids` link to plain `/normalized-events`.
- Dashboard gained Normalized + Signals metric tiles and a Recent Signals widget; the global `DashboardStreamBridge` remains the single SSE subscription.

Verification performed:

- `cd web && npm test`
- `cd web && npm run build`
- `cd web && npm audit --json`
- `curl 'http://localhost:18000/v1/normalized-events?tenant_id=tenant-local&limit=3'`
- `curl 'http://localhost:18000/v1/signals?tenant_id=tenant-local&limit=3'`
- `docker compose build web`
- `docker compose up -d web`
- `curl -fsS http://localhost:15173/`
- `curl 'http://localhost:15173/v1/normalized-events?tenant_id=tenant-local&limit=3'`
- `curl 'http://localhost:15173/v1/signals?tenant_id=tenant-local&limit=3'`

Live verification result:

- Vitest passed: 2 files, 6 tests.
- Frontend production build passed; both new routes lazy-loaded.
- Web image rebuilt; container Up on `:15173`; `/` serves the SPA shell.
- Normalized-events API returned `g044-live-event` with `schema_id`; signals API returned `signalops.static_test.low` with `model_version` (no `model_id`) — confirming the corrected field names.
- npm audit reported zero vulnerabilities.

Actor:

- Claude Code

Follow-up items:

- Browser/Playwright validation (rendering, console errors, 375px layout) as a manual step.
- Alert/insight lifecycle, correlation, and rule execution remain out of scope pending backend contracts.

### G046 Browser Validation Addendum

Timestamp: `2026-07-08T22:33:32Z`

Status: `passed`

Scope:

- Close the prior G046 follow-up item for browser/Playwright validation.
- Validate the frontend-agent navigation wrapping fix in `web/src/components/DashboardShell.tsx`.

Evidence:

- `web/src/components/DashboardShell.tsx`
- `/tmp/g046-validate/validate.js`
- `/tmp/g046-validate/shots/summary.json`
- `/tmp/g046-validate/shots/dashboard-desktop.png`
- `/tmp/g046-validate/shots/dashboard-mobile.png`
- `/tmp/g046-validate/shots/normalized-desktop.png`
- `/tmp/g046-validate/shots/normalized-mobile.png`
- `/tmp/g046-validate/shots/signals-desktop.png`
- `/tmp/g046-validate/shots/signals-mobile.png`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Verification performed:

- `cd web && npm test`
- `cd web && npm run build`
- `cd web && npm audit --json`
- `docker compose build web`
- `docker compose up -d web`
- `curl -fsS http://localhost:15173/`
- `curl -fsS http://localhost:15173/normalized-events`
- `curl -fsS http://localhost:15173/signals`
- `curl -fsS 'http://localhost:15173/v1/normalized-events?tenant_id=tenant-local&limit=3'`
- `curl -fsS 'http://localhost:15173/v1/signals?tenant_id=tenant-local&limit=3'`
- Playwright Docker validation against desktop and 375px mobile routes.
- `git diff --check`

Live verification result:

- Vitest passed: 2 files, 6 tests.
- Production build passed.
- npm audit reported 0 vulnerabilities.
- Web container remained Up on `:15173` after rebuild/restart.
- SPA routes `/`, `/normalized-events`, and `/signals` served successfully.
- Normalized-events and signals APIs returned live data through the web proxy.
- Playwright reported no browser console warnings/errors and no page errors.
- Playwright reported exactly one dashboard SSE connection, so G046 did not introduce duplicate EventSource subscriptions.
- Playwright reached `/normalized-events`, `/signals`, `/runs`, and `/raw-events` through SPA nav clicks.
- Playwright reported `0px` horizontal overflow at 375px for Dashboard, Signals, and Normalized Events.
- Screenshot artifacts were generated for desktop and mobile validation, and were visually inspected via image analysis: the desktop Dashboard metrics strip and widget bands render populated; the dedicated `/signals` page renders its full metrics strip (Signals/Detectors/High-Critical/Avg Confidence) and signals table (Signal/Detector/Model/Source-Dataset/Severity/Confidence/Events); and the 375px mobile view wraps the navigation to rows and stacks tiles with no overflow.

Actor:

- Codex

Follow-up items:

- None for G046. Proceed to alert/insight lifecycle foundation.

## Gate G047: Alert and Insight Lifecycle Foundation

Timestamp: `2026-07-08T22:55:26Z`

Status: `passed`

Gate name:

- Derive durable alert and insight lifecycle rows from persisted signals.

Criteria:

- Add alert and insight storage with lifecycle status fields and audit timestamps.
- Persist signal, derived alerts, and derived insights transactionally before committing signal-topic offsets.
- Preserve lifecycle status on idempotent signal reprocessing.
- Expose alert and insight list/detail APIs with tenant/source/dataset/status filters.
- Validate unit tests, migrations, Docker deployment, live signal ingestion, API, database, and consumer-group state.

Evidence:

- `migrations/000007_alert_insight_lifecycle.up.sql`
- `internal/storage/storage.go`
- `internal/storage/postgres/repository.go`
- `internal/signals/processor.go`
- `internal/api/router.go`
- `docs/api.md`
- `docs/build_journal.md`

Verification performed:

- `make docker-test`
- `make docker-test-python`
- `docker compose config --quiet`
- `docker compose --profile storage run --rm postgres-migrate`
- `docker compose build gateway signal-persister`
- `docker compose up -d gateway signal-persister`
- Published validation signal `signal-g047-high` to Redpanda.
- Queried `/v1/signals/signal-g047-high`.
- Queried `/v1/alerts/alert:signal-g047-high` and filtered alert list API.
- Queried `/v1/insights/insight:signal-g047-high` and filtered insight list API.
- Direct PostgreSQL alert/insight join.
- Redpanda `signalops.signal-persister.v1` group description.
- `git diff --check`.

Live verification result:

- Signal `signal-g047-high` persisted from broker position `1/0`.
- Alert `alert:signal-g047-high` persisted with status `open`, severity `high`, confidence `0.91`, event id `g044-live-event`, entities, evidence, recommendation, and correlation id `corr-g047`.
- Insight `insight:signal-g047-high` persisted with status `active`, severity `high`, confidence `0.91`, supporting metrics, semantic evidence, recommendation, and the same lineage.
- Low-severity historical `signalops.static_test.low` derived an active insight and no alert, matching the severity rule.
- Signal-persister group was Stable with one member and total lag `0`.

Actor:

- Codex

Follow-up items:

- Provide frontend-agent G048 spec for Alerts and Active Insights UI.
- Add authenticated lifecycle mutation APIs for acknowledgement, resolution, review, dismissal, and suppression when auth/operator identity is in place.


## Gate G048: Frontend Alerts and Active Insights UI

Timestamp: `2026-07-08T23:32:09Z`

Status: `passed`

Gate name:

- Expose the G047 alert and insight ledgers in the web frontend as read-only Alerts and Active Insights pages with Dashboard summaries.

Criteria:

- Add Alerts (`/alerts`) and Active Insights (`/insights`) read-only pages.
- Use real G047 REST APIs with typed client methods and TanStack Query hooks.
- Support truthful loading, error, empty, list, selection, and detail states.
- Add Dashboard Open Alerts and Active Insights summaries without fabricating mutation or streaming capability.
- Validate tests, build, audit, Compose, and live proxy data.

Evidence:

- `web/src/routes/AlertsRoute.tsx`
- `web/src/routes/InsightsRoute.tsx`
- `web/src/routes/DashboardRoute.tsx`
- `web/src/types.ts`
- `web/src/api/client.ts`
- `web/src/api/queries.ts`
- `web/src/api/alerts_insights.test.ts`
- `web/src/router.tsx`
- `web/src/components/DashboardShell.tsx`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Implementation notes:

- The spec contract was verified accurate against `internal/api/router.go` (alert/insight routes + DTOs), migration `000007`, and live data before coding, so no spec edits were required.
- Both pages use plain HTML tables + detail panels with `JsonViewer`; severity and status render as local text badges (not color-only); `signal_id` links to `/signals`, `event_ids` link to `/normalized-events`; no enabled lifecycle action controls (mutation deferred in G047).
- Dashboard gained Open Alerts + Active Insights metric tiles (strip widened to a 13-column arbitrary grid) and compact Open Alerts + Active Insights widgets, plus a caption distinguishing signals/alerts/insights; the global `DashboardStreamBridge` remains the single SSE subscription (REST is the source of truth).

Verification performed:

- `cd web && npm test`
- `cd web && npm run build`
- `cd web && npm audit --json`
- `curl -fsS http://localhost:15173/`
- `curl -fsS http://localhost:15173/alerts`
- `curl -fsS http://localhost:15173/insights`
- `curl 'http://localhost:15173/v1/alerts?tenant_id=tenant-local&status=open&limit=10'`
- `curl 'http://localhost:15173/v1/insights?tenant_id=tenant-local&status=active&limit=10'`
- `curl 'http://localhost:15173/v1/alerts/alert:signal-g047-high'`
- `curl 'http://localhost:15173/v1/insights/insight:signal-g047-high'`
- Playwright (Docker) desktop + 375px mobile browser validation.

Live verification result:

- Vitest passed: 3 files, 11 tests (5 new alert/insight client tests).
- Production build passed; both new routes lazy-loaded.
- Web container Up on `:15173`; `/`, `/alerts`, `/insights` serve the SPA shell.
- Alerts API returned `alert:signal-g047-high` (high/open); insights API returned `insight:signal-g047-high` and `insight:signalops.static_test.low`; detail envelopes returned for both.
- npm audit reported zero vulnerabilities.
- Playwright: no console/page errors; exactly one dashboard SSE connection across SPA nav; nav has 12 items without overlap; `/alerts` showed 1 open alert and selection loaded its detail panel; `/insights` showed 2 active insights and selection loaded its detail panel; Dashboard showed Open Alerts + Active Insights tiles/widgets (confirmed via DOM `innerText` and a uniquely-named screenshot after a stale-cache false negative); mobile overflow `0px` on Dashboard, Alerts, Insights.

Actor:

- Claude Code

Follow-up items:

- Add authenticated lifecycle mutation APIs (acknowledge/resolve/review/dismiss/suppress) and wire the corresponding UI controls when operator identity/authentication is in place.
- Consider modest polling or SSE channel additions for alerts/insights only when the backend stream supports them; for now REST refetch is the source of truth.

## Gate G049: Backend Alert and Insight Lifecycle Mutations

Timestamp: `2026-07-09T00:13:36Z`

Status: `passed`

Gate name:

- Add backend lifecycle mutation APIs for durable alert and insight rows.

Criteria:

- Add alert acknowledgement, resolution, and suppression endpoints.
- Add insight review, dismissal, and archive endpoints.
- Persist lifecycle status changes and audit actor/timestamp fields in Postgres.
- Preserve the G047 derivation model and avoid resetting lifecycle state during signal reprocessing.
- Document the API, deployment impact, and verification evidence with timestamped audit entries.

Evidence:

- `internal/storage/storage.go`
- `internal/storage/postgres/repository.go`
- `internal/api/router.go`
- `internal/api/router_test.go`
- `docs/api.md`
- `docs/deployment.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Implementation notes:

- No migration is required; G049 uses the lifecycle columns added by `000007_alert_insight_lifecycle`.
- Alert actions: `POST /v1/alerts/{alert_id}/acknowledge`, `/resolve`, `/suppress`.
- Insight actions: `POST /v1/insights/{insight_id}/review`, `/dismiss`, `/archive`.
- Actor placeholder order: `X-SignalOps-Actor`, request body `actor`, default `operator-local`.
- Mutation metadata is merged into `metadata.lifecycle` for the row and the updated envelope is returned.

Verification performed:

- `docker run --rm -v ... golang:1.22-bookworm go test ./internal/api ./internal/storage/postgres ./cmd/gateway`
- `make docker-test`
- `make docker-test-python`
- `docker compose config --quiet`
- `docker compose build gateway`
- `docker compose up -d gateway`
- `git diff --check`
- `docker exec -i signalops-redpanda-1 rpk topic produce signalops.local.signal.v1 -k signal-g049-high -f '%v'`
- `curl -fsS http://localhost:18000/v1/signals/signal-g049-high`
- `curl -fsS http://localhost:18000/v1/alerts/alert:signal-g049-high`
- `curl -fsS http://localhost:18000/v1/insights/insight:signal-g049-high`
- `curl -fsS -X POST ... /v1/alerts/alert:signal-g049-high/acknowledge`
- `curl -fsS -X POST ... /v1/alerts/alert:signal-g049-high/resolve`
- `curl -fsS -X POST ... /v1/alerts/alert:signal-g049-high/suppress`
- `curl -fsS -X POST ... /v1/insights/insight:signal-g049-high/review`
- `curl -fsS -X POST ... /v1/insights/insight:signal-g049-high/archive`
- `curl -fsS -X POST ... /v1/insights/insight:signal-g049-high/dismiss`
- Direct PostgreSQL lifecycle row queries.
- `docker exec signalops-redpanda-1 rpk group describe signalops.signal-persister.v1`

Live verification result:

- Redpanda accepted `signal-g049-high` at `signalops.local.signal.v1` partition `2`, offset `0`.
- `signal-persister` persisted `signal-g049-high` and derived `alert:signal-g049-high` plus `insight:signal-g049-high`.
- Alert acknowledge, resolve, and suppress endpoints returned updated `{alert}` envelopes with expected status, actor fields, and `metadata.lifecycle.action` values.
- Insight review, archive, and dismiss endpoints returned updated `{insight}` envelopes with expected status, actor fields, and `metadata.lifecycle.action` values.
- Direct PostgreSQL confirmed final alert status `suppressed`, `acknowledged_by=operator-g049`, `resolved_by=operator-g049`, and lifecycle action `suppress`.
- Direct PostgreSQL confirmed final insight status `dismissed`, `reviewed_by=operator-g049`, and lifecycle action `dismiss`.
- Consumer group `signalops.signal-persister.v1` was Stable with total lag `0`.

Actor:

- Codex

Follow-up items:

- Write the frontend-agent spec for lifecycle action controls on Alerts and Active Insights.
- Replace placeholder actor handling with authenticated operator identity when the auth boundary lands.

## Gate G050: Frontend Alert and Insight Lifecycle Controls Specification

Timestamp: `2026-07-09T01:00:56Z`

Status: `handoff ready`

Gate name:

- Define the frontend-agent implementation contract for alert and insight lifecycle controls backed by G049 APIs.

Criteria:

- Specify alert Acknowledge, Resolve, and Suppress controls.
- Specify insight Review, Dismiss, and Archive controls.
- Define client mutation methods, query invalidation, placeholder actor handling, UI states, and validation requirements.
- Keep unsupported auth, streaming, bulk action, and backend changes out of scope.
- Record the handoff in timestamped documentation.

Evidence:

- `docs/frontend/alerts_insights_lifecycle_controls_spec.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Implementation notes:

- The spec extends the existing G048 `/alerts` and `/insights` pages.
- Backend baseline is G049: `POST /v1/alerts/{alert_id}/acknowledge|resolve|suppress` and `POST /v1/insights/{insight_id}/review|dismiss|archive`.
- Operator identity remains a placeholder: `X-SignalOps-Actor`, body `actor`, then `operator-local`; frontend should use `operator-local` until auth lands.
- Query invalidation must refresh list, detail, and Dashboard summary data after successful mutations.

Verification performed:

- Reviewed against `docs/api.md` G049 endpoint contract.
- Reviewed against the passed G049 gate evidence in this audit log.

Live verification result:

- Not applicable; this is a frontend-agent specification handoff, not the implementation gate.

Actor:

- Codex

Follow-up items:

- Frontend-agent implements G050 and records implementation evidence.
- After frontend implementation, validate browser action controls and decide whether the next backend gate should add authenticated operator identity or audit-history rows.


## Gate G050: Frontend Alert and Insight Lifecycle Controls

Timestamp: `2026-07-09T01:41:23Z`

Status: `passed`

Gate name:

- Add operator lifecycle controls (acknowledge/resolve/suppress alerts; review/dismiss/archive insights) to the G048 frontend, backed by the committed G049 mutation APIs.

Criteria:

- Acknowledge/Resolve/Suppress on `/alerts` and Review/Dismiss/Archive on `/insights`, backed by real G049 APIs.
- Truthful loading/disabled/success/error states; no fabricated lifecycle state.
- Query caches refresh list/detail/dashboard data after mutations.
- Placeholder operator identity used consistently (no auth claims).
- Tests, build, audit, compose config, live proxy mutation, and browser validation.

Evidence:

- `web/src/routes/AlertsRoute.tsx`
- `web/src/routes/InsightsRoute.tsx`
- `web/src/types.ts`
- `web/src/api/client.ts`
- `web/src/api/queries.ts`
- `web/src/api/alerts_insights.test.ts`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Implementation notes:

- Spec contract verified accurate against committed G049 (`router.go` mutation routes/handlers, `lifecycleActor` header→body→default precedence, `lifecycleMetadata` jsonb merge, error codes); implemented as-is.
- Added a shared `post<T>` helper + `mutateAlertLifecycle`/`mutateInsightLifecycle` (action in path, `X-SignalOps-Actor: operator-local` header) and `useMutate*` hooks (`setQueryData` detail + invalidate `['alerts']`/`['insights']` list prefix).
- Detail-panel controls with spec disabled logic, inline error, and lifecycle summary; detail body keyed by id so mutation state resets on selection change. No auth/SSE/audit/bulk/modals (Out of Scope).

Verification performed:

- `cd web && npm test`
- `cd web && npm run build`
- `cd web && npm audit --json`
- `docker compose config --quiet`
- `docker compose build web` / `up -d web`
- `POST /v1/alerts/alert:signal-g049-high/acknowledge` (header actor)
- `POST /v1/insights/insight:signal-g049-high/review` (body actor)
- `POST /v1/alerts/alert:does-not-exist/acknowledge` (404)
- Playwright (Docker) desktop + 375px mobile

Live verification result:

- Vitest: 3 files, 18 tests (7 new lifecycle mutation tests).
- Production build passed; npm audit 0 vulnerabilities; `compose config` OK; web Up on `:15173`.
- Acknowledge (header actor) and Review (body actor) both updated status/timestamps and wrote `metadata.lifecycle`; `404 alert_not_found` confirmed.
- Playwright: no console/page errors; one SSE connection; nav 12 items; controls render on `/alerts` + `/insights`; Acknowledge + Review updated backend state (buttons correctly disabled afterward, lifecycle summary rendered); Dashboard Open Alerts `2→1` and Active Insights `3→2` after mutations (summaries refreshed); `0px` mobile overflow. Screenshot confirms acknowledged status + disabled Acknowledge + lifecycle summary + `acknowledged_at`/`acknowledged_by`.

Actor:

- Claude Code

Follow-up items:

- Add real operator authentication/identity (replace placeholder `operator-local`) and full lifecycle audit history when auth lands.
- Consider modest alert/insight SSE/polling only when the backend stream supports it.

## Gate G050: Frontend Alert and Insight Lifecycle Controls Validation

Timestamp: `2026-07-09T01:52:52Z`

Status: `passed`

Gate name:

- Validate the frontend implementation of G050 lifecycle controls for Alerts and Active Insights.

Criteria:

- Alert controls for Acknowledge, Resolve, and Suppress are implemented against G049 APIs.
- Insight controls for Review, Dismiss, and Archive are implemented against G049 APIs.
- Placeholder operator identity is sent as `operator-local`.
- Query invalidation refreshes list/detail/dashboard data after mutations.
- Tests, build, audit, Compose validation, deployment, proxy API checks, database checks, and browser evidence are recorded.
- TimescaleDB future requirement is documented as an essential maturity item while confirming it is not currently deployed.

Evidence:

- `web/src/types.ts`
- `web/src/api/client.ts`
- `web/src/api/queries.ts`
- `web/src/routes/AlertsRoute.tsx`
- `web/src/routes/InsightsRoute.tsx`
- `web/src/api/alerts_insights.test.ts`
- `docs/deployment.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`
- `/tmp/g050-validate/shots/summary.json`

Implementation notes:

- The client adds `mutateAlertLifecycle` and `mutateInsightLifecycle` POST methods with `X-SignalOps-Actor: operator-local` and encoded lifecycle IDs.
- TanStack Query mutation hooks update detail cache and invalidate the `alerts`/`insights` query prefixes.
- Alerts detail panel exposes Acknowledge, Resolve, and Suppress controls with disabled states based on lifecycle status.
- Insights detail panel exposes Review, Dismiss, and Archive controls with disabled states based on lifecycle status.
- Lifecycle metadata summaries render from `metadata.lifecycle` while preserving the full JSON metadata viewer.
- Current storage remains plain PostgreSQL; TimescaleDB is now explicitly documented as an essential future maturity gate for high-volume temporal ledgers.

Verification performed:

- `cd web && npm test`
- `cd web && npm run build`
- `cd web && npm audit --json`
- `docker compose config --quiet`
- `docker compose build web`
- `docker compose up -d web`
- `docker exec -i signalops-redpanda-1 rpk topic produce signalops.local.signal.v1 -k signal-g050-high -f '%v'`
- `curl -fsS http://localhost:15173/v1/alerts/alert:signal-g050-high`
- `curl -fsS http://localhost:15173/v1/insights/insight:signal-g050-high`
- `curl -fsS -X POST ... http://localhost:15173/v1/alerts/alert:signal-g050-high/suppress`
- `curl -fsS -X POST ... http://localhost:15173/v1/insights/insight:signal-g050-high/archive`
- Direct PostgreSQL lifecycle row queries.
- `docker exec signalops-redpanda-1 rpk group describe signalops.signal-persister.v1`
- Reviewed `/tmp/g050-validate/shots/summary.json` from frontend-agent browser validation.

Live verification result:

- Vitest passed: 3 files, 18 tests.
- Production frontend build passed; npm audit reported zero vulnerabilities.
- Compose config and web image build passed; web service restarted successfully.
- Redpanda accepted `signal-g050-high` at partition `0`, offset `4`; `signal-persister` persisted it with total lag `0`.
- Web proxy returned the fresh `open` alert and `active` insight before lifecycle mutation.
- Web proxy mutation changed `alert:signal-g050-high` to `suppressed` with lifecycle action `suppress` and actor `operator-local`.
- Web proxy mutation changed `insight:signal-g050-high` to `archived` with lifecycle action `archive` and actor `operator-local`.
- PostgreSQL confirmed the final lifecycle states and metadata.
- Frontend-agent browser validation summary reported no console/page errors, one dashboard SSE connection, visible controls, disabled post-action controls, lifecycle summaries, Dashboard count drops, 12 nav items, and `0px` mobile overflow for Alerts and Insights.

Issue found and noted:

- Independent local Playwright rerun was blocked by the available Playwright image lacking the Node `playwright` module. Existing frontend-agent Playwright artifacts were used for browser evidence; independent validation covered tests, build, deployed proxy mutations, database state, and consumer lag.

Actor:

- Codex

Follow-up items:

- Add authenticated operator identity and durable lifecycle audit history beyond the latest `metadata.lifecycle` object.
- Plan the TimescaleDB storage maturity gate before sustained high-volume temporal ingestion.

## Gate G051: SignalOps Public TLS via Syncratic Traefik

Timestamp: `2026-07-09T02:43:48Z`

Status: `passed`

Gate name:

- Expose SignalOps through the parent Syncratic core Traefik edge with Let's Encrypt TLS.

Criteria:

- Provide a SignalOps Compose overlay that attaches `web` to the Syncratic Traefik network.
- Use the parent Traefik `websecure` entrypoint and `letsencrypt` certificate resolver.
- Keep the gateway internal and expose browser/API traffic through the existing web nginx proxy.
- Document required SignalOps and parent Traefik environment values.
- Validate merged Compose configuration, deployed labels, network attachment, local proxy health, and Traefik SNI routing.
- Record public DNS/certificate status.

Evidence:

- `compose.traefik.yaml`
- `.env.example`
- `docs/deployment.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Implementation notes:

- `compose.traefik.yaml` adds Traefik labels to `web` only.
- Router rule: `Host("signalops.syncratic.io")`.
- Entrypoint: `websecure`.
- Cert resolver: `letsencrypt`.
- Load balancer target port: `8080` inside the web container.
- External Docker network: `syncratic-core_syncratic_net`.
- Required parent Traefik ACME values remain managed in Syncratic core: `LETSENCRYPT_EMAIL`, `GODADDY_API_KEY`, and `GODADDY_API_SECRET`.

Verification performed:

- `docker compose -f compose.yaml -f compose.traefik.yaml config --quiet`
- Docker network presence check for `signalops_default` and `syncratic-core_syncratic_net`.
- Rendered Compose inspection for Traefik labels and `web` network attachment.
- `SIGNALOPS_PUBLIC_HOST=signalops.syncratic.io TRAEFIK_NETWORK=syncratic-core_syncratic_net docker compose -f compose.yaml -f compose.traefik.yaml up -d web`
- `docker inspect signalops-web-1` for labels and networks.
- `curl -fsS http://localhost:15173/healthz`
- `curl -k --resolve signalops.syncratic.io:443:127.0.0.1 -fsS https://signalops.syncratic.io/healthz`
- `curl -k --resolve signalops.syncratic.io:443:127.0.0.1 -fsS https://signalops.syncratic.io/`
- Traefik log review for `signalops@docker` routing.
- Public validation: `curl -sS -o /tmp/signalops-http-final.txt -w "%{http_code} %{redirect_url}\n" http://signalops.syncratic.io/healthz` returned `301 https://signalops.syncratic.io/healthz`; `curl -sS -o /tmp/signalops-https-final.txt -w "%{http_code} %{remote_ip} %{ssl_verify_result}\n" https://signalops.syncratic.io/healthz` returned `200 45.60.31.46 0`.

Live verification result:

- Compose overlay rendered successfully.
- `signalops-web-1` is attached to both the SignalOps default network and `syncratic-core_syncratic_net`.
- Traefik labels are present on the web container.
- Local health via `localhost:15173` passed.
- Local Traefik SNI override passed for `/healthz` and `/`.
- Public DNS resolves, HTTP redirects to HTTPS, public HTTPS reaches SignalOps gateway health, and local Traefik SNI HTTPS succeeds.

Actor:

- Codex

Follow-up items:

- Keep `SIGNALOPS_PUBLIC_HOST` and `TRAEFIK_NETWORK` defined in deployment `.env` for overlay renders.
- Re-run public HTTPS validation after any DNS, WAF, or Traefik label changes.
- Confirm Traefik ACME storage has issued the certificate for `signalops.syncratic.io`.

### G051 Firewall Forwarding Follow-Up

Timestamp: `2026-07-09T03:12:00Z`

Status: `passed - public access validated`

Evidence:

- Firewall forwarding target stated as `192.168.2.5` on ports `80` and `443`.
- `signalops.syncratic.io` resolves to `108.72.192.62`.
- Direct LAN HTTP request to `192.168.2.5` with Host `signalops.syncratic.io` returned `301` to HTTPS.
- Direct LAN HTTPS request to `192.168.2.5` with SNI `signalops.syncratic.io` returned `200` and SignalOps gateway health.
- Same-host public-domain curls timed out, consistent with NAT hairpin/reflection limitations.

Follow-up:

- Public application access was confirmed by the operator after firewall forwarding was updated.
- Re-run public HTTPS validation after any DNS, WAF, firewall, or Traefik label changes.

### G051 Closure

Timestamp: `2026-07-09T03:20:00Z`

Status: `closed`

Evidence:

- Operator confirmed public access to the SignalOps application works through `https://signalops.syncratic.io`.
- Prior Compose, Traefik label, LAN edge, local SNI, and public HTTPS validation evidence remains recorded above.

Outcome:

- G051 is fully validated and closed.
- Public app access is available through the Syncratic Traefik edge with Let's Encrypt TLS.

Next gate:

- G052 should enforce authentication and operator identity before further public-facing capability expansion.

## Gate G052: Authentication and Operator Identity Readiness

Timestamp: `2026-07-09T03:32:00Z`

Status: `ready for backend implementation`

Gate name:

- Prepare SignalOps for backend OIDC/JWT enforcement using Syncratic IdP.

Criteria satisfied before implementation:

- IdP clients exist for browser login and API resource validation.
- Access tokens include `aud: signalops-api`.
- SignalOps roles and groups exist.
- Initial admin/operator user exists.
- Tenant claim is available for `tenant-local`.
- Backend env contract already documents issuer, JWKS, audience, client id, realm, and auth enablement variables.

Confirmed IdP configuration:

- Realm: `syncratic`.
- Issuer: `https://auth.syncratic.co/realms/syncratic`.
- JWKS: `https://auth.syncratic.co/realms/syncratic/protocol/openid-connect/certs`.
- Browser client: `signalops-web` public OIDC client with Authorization Code + PKCE S256.
- API resource: `signalops-api` bearer-only resource client.
- Roles: `signalops:viewer`, `signalops:operator`, `signalops:admin`.
- Groups: `/signalops/viewers`, `/signalops/operators`, `/signalops/admins`.
- User: `lukeb` / `luke@strategiclabs.io` assigned to `/signalops/admins`.
- Claims: `aud: signalops-api`, `tenant_id: tenant-local`, `preferred_username`, `email`, and roles under `realm_access.roles`.

Backend implementation expectations:

- Keep `/healthz` and `/readyz` unauthenticated.
- When `SIGNALOPS_AUTH_ENABLED=true`, require Bearer JWT for protected `/v1/*` APIs.
- Validate issuer, expiry, signature via JWKS, and audience `signalops-api`.
- Extract tenant from `tenant_id` and reject protected requests without a tenant claim.
- Extract actor from `preferred_username`, then `email`, then `sub`.
- Require `signalops:viewer` for read APIs.
- Require `signalops:operator` or `signalops:admin` for alert/insight lifecycle mutation APIs.
- Preserve disabled-auth local development behavior while preventing `operator-local` fallback when auth is enabled.

Follow-up items:

- Implement and validate G052 backend auth middleware and role checks.
- After backend G052 passes, write the frontend-agent specification for login/logout, token attachment, route guards, and unauthorized states.

## Gate G052: Backend OIDC/JWT Enforcement

Timestamp: `2026-07-09T04:12:00Z`

Status: `passed - deployed with auth disabled pending frontend login gate`

Gate name:

- Enforce Syncratic IdP JWTs and operator identity in the SignalOps gateway.

Criteria:

- Keep `/healthz` and `/readyz` unauthenticated.
- Gate protected `/v1/*` APIs behind Bearer JWT validation when `SIGNALOPS_AUTH_ENABLED=true`.
- Validate issuer, JWKS signature, expiry/not-before, and audience `signalops-api`.
- Require `tenant_id` claim and reject tenant query/path mismatches.
- Extract actor from `preferred_username`, then `email`, then `sub`.
- Require viewer/operator/admin role for read/protected `/v1/*` routes.
- Require operator/admin role for alert and insight lifecycle mutation routes.
- Preserve auth-disabled local/frontend-transition behavior.

Evidence:

- `internal/api/auth.go`
- `internal/api/auth_test.go`
- `internal/api/router.go`
- `internal/config/config.go`
- `internal/config/config_test.go`
- `cmd/gateway/main.go`
- `compose.yaml`
- `.env.example`
- `docs/deployment.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Verification performed:

- Unit tests generate RS256 JWTs against a local JWKS server and validate public health, missing Bearer rejection, viewer read access, tenant mismatch rejection, viewer lifecycle denial, admin lifecycle allowance, and token-derived actor precedence.
- `go test ./internal/api ./internal/config ./cmd/gateway` passed in Docker.
- `go test ./...` passed in Docker.
- `docker compose -f compose.yaml -f compose.traefik.yaml config --quiet` passed.
- `docker compose -f compose.yaml -f compose.traefik.yaml build gateway` passed, including Dockerfile `go test ./...`.
- `docker compose -f compose.yaml -f compose.traefik.yaml up -d gateway web` redeployed the gateway.
- Live local proxy checks passed for `/healthz`, `/readyz`, and `/v1/alerts?tenant_id=tenant-local&limit=1` with auth disabled.

Live verification result:

- Backend G052 enforcement is implemented and validated by tests.
- Running deployment remains `SIGNALOPS_AUTH_ENABLED=false` so the public app stays usable until frontend login/token attachment is implemented.
- Gateway container has Syncratic IdP issuer, JWKS URL, audience, realm, and client id env values ready for enablement.

Actor:

- Codex

Follow-up items:

- Write the frontend-agent G053 specification for OIDC login/logout, token attachment, route guarding, unauthorized states, and role-aware UI behavior.
- After the frontend auth gate passes, set `SIGNALOPS_AUTH_ENABLED=true` and validate live protected API behavior with a real Syncratic IdP token.

## Gate G053: Frontend Auth Integration Specification

Timestamp: `2026-07-09T04:22:00Z`

Status: `handoff ready`

Gate name:

- Specify frontend Syncratic IdP login/logout and token integration for SignalOps.

Criteria:

- The spec must align with G052 backend JWT enforcement.
- The spec must preserve auth-disabled development behavior until backend auth is enabled permanently.
- The spec must require Authorization Code + PKCE for `signalops-web` without client secrets.
- The spec must cover token attachment, route guarding, tenant claim use, role helpers, lifecycle control gating, callback/logout handling, query cache behavior, tests, browser validation, and journal/audit updates.

Evidence:

- `docs/frontend/auth_integration_spec.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Implementation notes:

- Recommended `oidc-client-ts` rather than hand-rolled OAuth/PKCE.
- Required tenant source is token `tenant_id` when auth is enabled.
- Required role sources are `realm_access.roles` and future-compatible `resource_access.signalops-api.roles`.
- Auth-enabled lifecycle mutations must rely on backend token-derived actor and stop sending `X-SignalOps-Actor: operator-local`.
- The frontend should not permanently enable backend auth; live backend auth enablement remains a coordinated validation step after frontend implementation.

Follow-up items:

- Frontend-agent implements G053.
- Codex validates G053 implementation and then coordinates setting `SIGNALOPS_AUTH_ENABLED=true` for live backend enforcement.

## Gate G053: Frontend Auth Integration

Timestamp: `2026-07-09T12:45:11Z`

Status: `implemented — interactive IdP browser login pending`

Gate name:

- Implement browser Syncratic IdP login/logout and Bearer-token API integration for the SignalOps web app.

Criteria:

- Support Authorization Code + PKCE login/logout through the `signalops-web` public client (no client secret).
- Attach `Authorization: Bearer <token>` centrally to protected `/v1/*` calls when auth is enabled; keep `/healthz`/`/readyz` usable unauthenticated.
- Derive tenant for route queries from the token `tenant_id` claim when auth is enabled; preserve `tenant-local` when disabled.
- Gate alert/insight lifecycle controls by role (operator/admin) and stop sending `X-SignalOps-Actor: operator-local` when auth is enabled.
- Preserve auth-disabled development behavior; tests and production build pass; browser validation recorded.
- Keep backend auth disabled until frontend behavior is verified.

Evidence:

- `web/src/auth/{config,oidc,session,claims,LoginScreen}.tsx` and `web/src/auth/{config,claims,auth_client}.test.ts`.
- `web/src/App.tsx`, `web/src/router.tsx`, `web/src/api/client.ts`, `web/src/api/queries.ts`, `web/src/components/{DashboardShell,IdempotencyLookup}.tsx`.
- `web/src/routes/{DashboardRoute,SourcesRoute,PipelinesRoute,RulesRoute,NormalizedEventsRoute,SignalsRoute,AlertsRoute,InsightsRoute}.tsx`.
- `web/.env.example`, `web/Dockerfile`, `compose.yaml`.
- `docs/build_journal.md`.

Implementation notes:

- `oidc-client-ts` `UserManager` configured with `response_type: 'code'`, `S256` PKCE (implicit), `scope: openid profile email`, `extraQueryParams.resource = signalops-api` for the Keycloak audience, `automaticSilentRenew`, and origin-based redirect/post-logout URIs.
- App-level `RootGate` blocks all protected routes (and thus all `/v1/*` queries) until a non-expired token exists; the `/auth/callback` redirect is processed in the gate before the router mounts.
- Module-level token holder updated by the provider lets the non-React API client attach the Bearer token centrally.
- `useTenant()` returns the token `tenant_id` when auth is on (else `tenant-local`); `useCanMutateLifecycle()` returns operator/admin when on (else true for dev).

Verification performed:

- `npm test`: 31/31 pass (config, claims, api-client token/actor/401 behavior).
- `npm run build` (`tsc` + `vite build`): succeeded.
- Auth-disabled `web` rebuild + redeploy: `/healthz` 200, `/readyz` 200, `/v1/alerts` 200, SPA index served, `/auth/callback` SPA fallback 200.
- IdP discovery doc matches the contract (issuer, JWKS URI, `S256`, `authorization_code`, `code`, end-session endpoint).
- Build-only auth-enabled image build via compose args succeeded (production enablement path verified); not redeployed.

Follow-up items:

- Complete the interactive auth-enabled browser login (Imperva WAF blocks headless probing; requires a real browser session with operator credentials) and confirm Bearer attachment, tenant/role resolution, and logout cache clear.
- After browser validation, coordinate setting `SIGNALOPS_AUTH_ENABLED=true` for live backend enforcement.

## Gate G053: Frontend Auth Integration Validation

Timestamp: `2026-07-09T13:14:00Z`

Status: `validated - interactive auth enablement pending`

Gate name:

- Validate the frontend-agent G053 implementation for Syncratic IdP login/logout and token integration.

Criteria validated:

- `oidc-client-ts` is present and configured for Authorization Code + PKCE against `signalops-web`.
- App-level auth gate prevents protected route mounting before authentication when auth is enabled.
- `/auth/callback` route/fallback is present and served by nginx SPA fallback.
- API client centrally attaches Bearer tokens when auth is enabled and token exists.
- Health/readiness remain usable without a token.
- Tenant hook derives `tenant_id` from token claims when auth is enabled and keeps `tenant-local` fallback when auth is disabled.
- Role helpers support `realm_access.roles` and `resource_access.signalops-api.roles`.
- Alert/insight lifecycle controls are role-gated and stop sending `X-SignalOps-Actor: operator-local` when auth is enabled.
- Auth-disabled deployment behavior is preserved.

Evidence:

- `web/src/auth/config.ts`
- `web/src/auth/oidc.ts`
- `web/src/auth/session.tsx`
- `web/src/auth/claims.ts`
- `web/src/auth/LoginScreen.tsx`
- `web/src/App.tsx`
- `web/src/router.tsx`
- `web/src/api/client.ts`
- `web/src/api/stream.ts`
- `web/src/components/DashboardShell.tsx`
- `web/src/routes/AlertsRoute.tsx`
- `web/src/routes/InsightsRoute.tsx`
- `web/Dockerfile`
- `compose.yaml`
- `web/package.json`
- `web/package-lock.json`

Verification performed:

- `cd web && npm test` - 6 files, 31 tests passed.
- `cd web && npm run build` - passed.
- `cd web && npm audit --json` - 0 vulnerabilities.
- `docker compose -f compose.yaml -f compose.traefik.yaml build web` - passed.
- `docker compose -f compose.yaml -f compose.traefik.yaml up -d web` - passed.
- `curl -fsS http://localhost:15173/healthz` - passed.
- `curl -fsS http://localhost:15173/readyz` - passed.
- `curl -fsS 'http://localhost:15173/v1/alerts?tenant_id=tenant-local&limit=1'` - passed with backend auth disabled.
- `curl -fsS -o /tmp/g053-auth-callback.html -w '%{http_code} %{content_type}\n' http://localhost:15173/auth/callback` - returned `200 text/html`.
- Build-only auth-enabled image path passed; default auth-disabled image was rebuilt and redeployed afterward.

Issue noted:

- Authenticated SSE is not closed yet. The current dashboard stream uses native `EventSource`, which cannot send an `Authorization` header to `/v1/streams/dashboard`; G052 protects `/v1/*` when backend auth is enabled. A transport decision is required before permanently enabling backend auth with dashboard streaming.

Outcome:

- G053 frontend implementation is validated for code, tests, build, and auth-disabled deployment compatibility.
- Interactive browser login through Syncratic IdP and the SSE auth transport decision remain pending before `SIGNALOPS_AUTH_ENABLED=true` is made permanent.

Actor:

- Codex

Follow-up items:

- Complete real-browser login/logout validation with `lukeb` at `https://signalops.syncratic.io`.
- Decide and implement the authenticated SSE transport path or explicitly disable SSE with REST fallback while auth is enabled.
- Then enable backend auth and validate live `401`, `403`, viewer read, operator/admin lifecycle mutation, and tenant mismatch behavior with real IdP tokens.

## Gate G054: Authenticated Streaming and Browser Validation Specification

Timestamp: `2026-07-09T13:24:00Z`

Status: `handoff ready`

Gate name:

- Specify frontend handling for authenticated dashboard streaming and real-browser auth validation.

Criteria:

- Address the G053 follow-up that native `EventSource` cannot send `Authorization` headers while G052 protects `/v1/streams/dashboard` under backend auth.
- Keep auth-disabled streaming behavior intact.
- Require a safe auth-enabled fallback that does not put tokens in URLs.
- Require tests for stream mode behavior and fallback refresh.
- Require a real-browser Syncratic IdP validation checklist.
- Keep backend auth enablement out of scope until frontend/browser validation is complete.

Evidence:

- `docs/frontend/authenticated_streaming_and_browser_validation_spec.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Implementation notes:

- Recommended direction is REST fallback under auth instead of query-string tokens, cookie/session edge auth, or a custom fetch-stream parser in this gate.
- The frontend should not permanently enable backend `SIGNALOPS_AUTH_ENABLED=true`; that remains a coordinated post-G054 validation step.
- Browser validation may require a real operator session because Imperva/Incapsula can block headless IdP probing.

Follow-up items:

- Frontend-agent implements G054.
- Codex validates G054 and then coordinates interactive browser login plus backend auth enablement.

## Gate G054: Authenticated Streaming and Browser Validation

Timestamp: `2026-07-09T13:35:30Z`

Status: `closed — browser auth validation passed; backend auth enablement remains a separate gate`

Gate name:

- Implement auth-aware dashboard streaming with a safe REST fallback and add a real-browser Syncratic IdP validation checklist.

Criteria:

- Auth-disabled native `EventSource` streaming remains intact.
- Auth-enabled frontend no longer attempts native `EventSource` to protected `/v1/streams/dashboard`.
- No access token is placed in any stream URL.
- Dashboard uses REST fallback refresh when auth is enabled.
- Stream UI state does not present the intentional fallback as a broken/reconnecting stream.
- Tests cover auth-disabled streaming and auth-enabled fallback behavior; G053 API auth tests still pass.
- `npm test`, `npm run build`, and `npm audit --json` pass.
- Auth-enabled image build path passes.
- Real-browser validation checklist exists.
- Backend auth remains disabled.

Evidence:

- `web/src/api/stream.ts` (`streamMode`, auth-aware `subscribeDashboardStream`, REST fallback primitives).
- `web/src/components/DashboardStreamBridge.tsx` (mode branch + 15s REST invalidation).
- `web/src/store/ui.ts` (`streamMode` state).
- `web/src/components/HealthIndicator.tsx`, `web/src/routes/DashboardRoute.tsx`, `web/src/routes/SystemRoute.tsx` (neutral `REST refresh` wording).
- `web/src/api/stream.test.ts`.
- `docs/frontend/auth_browser_validation_checklist.md`.
- `Makefile` (`deploy-web` auth-enabled public deploy target with Traefik overlay).
- `docs/deployment.md` (public deploy guidance for auth-enabled frontend).
- `web/src/auth/session.tsx` (stable callback/sign-in/sign-out handlers to avoid duplicate PKCE callback processing).
- `web/src/auth/LoginScreen.tsx` (callback failure diagnostics for browser validation).
- `docs/build_journal.md`.

Implementation notes:

- No new runtime dependencies; no backend changes.
- `subscribeDashboardStream` returns an inert no-op under auth (no `EventSource`, no token in URL, no error callback); freshness via a 15s `refreshDashboardViaRest` invalidation of dashboard prefixes (`healthz`/`readyz` excluded since they already poll).
- Added UI `streamMode` so `HealthIndicator`/`Dashboard`/`System` show neutral `REST refresh` wording and the health dot no longer penalizes for the intentionally-off stream.

Verification performed:

- `npm test`: 34/34 pass (3 new G054 stream tests; G053 Bearer/actor/401 tests still pass).
- `npm run build` (`tsc` + `vite build`): succeeded.
- `npm audit --json`: 0 vulnerabilities, exit 0.
- Auth-disabled `web` rebuild + redeploy via Traefik overlay: `https://signalops.syncratic.io/` `/healthz` `/readyz` `/v1/alerts` 200; `/auth/callback` SPA fallback 200; Traefik router label present.
- Auth-enabled image build-only via compose args + Traefik overlay: succeeded; not redeployed.
- Default image tag restored to auth-disabled (running container remains auth-disabled).
- `2026-07-09T17:18:52Z` validation update: frontend-agent completed the real-browser auth checklist; public `https://signalops.syncratic.io/` login path works.
- `npm test`: 34/34 pass after browser-validation fixes.
- `npm run build`: succeeded after browser-validation fixes.
- `npm audit --json`: 0 vulnerabilities after browser-validation fixes.
- `docker compose -f compose.yaml -f compose.traefik.yaml config --quiet`: succeeded.
- Live HTTPS checks after validation: `/healthz` 200, `/readyz` 200, `/` 200, `/auth/callback` 200.

Follow-up items:

- Coordinate the next backend gate for `SIGNALOPS_AUTH_ENABLED=true` live enforcement using the validated frontend auth path.


## Gate G055: Backend Auth Enforcement Enablement

Timestamp: `2026-07-09T17:24:00Z`

Status: `closed — backend auth enforcement and authenticated browser validation passed`

Gate name:

- Enable backend `SIGNALOPS_AUTH_ENABLED=true` in the live deployment and validate protected API boundaries.

Criteria:

- Gateway runs with backend auth enabled.
- Frontend remains auth-enabled and publicly routed through Traefik.
- `/healthz` and `/readyz` remain public.
- Unauthenticated `/v1/*` requests are rejected with `401`.
- Direct gateway access also rejects unauthenticated protected API calls.
- Authenticated browser session can load dashboard data with bearer-token API requests.
- Operator confirms the authenticated dashboard path works as expected after backend auth enforcement.

Evidence:

- Local `.env` deployment flags set to `SIGNALOPS_AUTH_ENABLED=true` and `VITE_SIGNALOPS_AUTH_ENABLED=true`.
- `docker compose -f compose.yaml -f compose.traefik.yaml config` resolves gateway auth and web auth build args to `true`.
- Gateway recreated via `docker compose -f compose.yaml -f compose.traefik.yaml up -d --force-recreate gateway`.
- Web redeployed via `make deploy-web`.
- `docs/build_journal.md`.
- Operator browser validation report at `2026-07-09T17:27:31Z`.

Verification performed:

- `docker compose -f compose.yaml -f compose.traefik.yaml config --quiet`: succeeded.
- Public `GET /healthz`: 200.
- Public `GET /readyz`: 200.
- Public `GET /`: 200 text/html.
- Public unauthenticated `GET /v1/alerts?tenant_id=tenant-local&limit=1`: 401 application/json.
- Public unauthenticated `GET /v1/raw-events?tenant_id=tenant-local&limit=1`: 401 application/json.
- Direct gateway unauthenticated `GET http://localhost:18000/v1/alerts?tenant_id=tenant-local&limit=1`: 401 application/json.
- Gateway restart log observed at `2026-07-09T17:23:05Z` with no startup error.
- `2026-07-09T17:27:31Z` operator browser validation: dashboard data loads and system works as expected after authentication.

Follow-up items:

- None for G055. Continue with the next backend/frontend capability gate.


## Gate G056: TimescaleDB Temporal Storage Foundation

Timestamp: `2026-07-09T20:58:33Z`

Status: `passed — local TimescaleDB foundation and hypertables validated; live temporal cutover pending backfill/replay`

Gate name:

- Add TimescaleDB as the temporal/event-plane store without eliminating PostgreSQL's relational system-of-record role.

Criteria:

- PostgreSQL remains configured for relational control-plane data.
- TimescaleDB is added as a separate Compose service with its own volume and migration runner.
- Temporal migrations create replayable hypertables for raw events, normalized events, signals, and initial market-data history.
- Runtime config supports an optional `SIGNALOPS_TEMPORAL_DATABASE_URL`.
- Repository code falls back to PostgreSQL when no temporal DSN is configured.
- Services that read/write temporal ledgers can use the separate temporal DSN when configured.
- Documentation clearly states storage roles and migration commands.

Evidence:

- `compose.yaml`
- `.env.example`
- `Makefile`
- `temporal_migrations/000001_timescale_temporal_foundation.up.sql`
- `internal/config/config.go`
- `internal/storage/postgres/repository.go`
- `cmd/gateway/main.go`
- `cmd/normalizer/main.go`
- `cmd/signal-persister/main.go`
- `cmd/massive-puller/main.go`
- `cmd/massive-scheduler/main.go`
- `docs/deployment.md`
- `docs/docker_development.md`
- `docs/build_journal.md`

Verification performed:

- Go tests: passed via Docker `go test ./...`.
- Python worker tests: 37 passed.
- JSON schema validation: passed.
- Compose + Traefik overlay config validation: passed.
- TimescaleDB container started successfully.
- `make compose-temporal-migrate`: applied temporal foundation migration successfully.
- Timescale hypertable query returned `marketdata_equity_eod_prices`, `marketdata_option_contracts_daily`, `normalized_event_ledger`, `raw_event_ledger`, and `signal_ledger`.
- `2026-07-09T21:06:45Z` bounded live Massive publish test: `dry_run=false`, `companies=1`, `events_built=1`, `events_published=1`, `provider_requests=1`, `failures=0`.
- TimescaleDB `raw_event_ledger` count increased from `0` to `1`; latest row is tenant `tenant-local`, source `src-massive`, dataset `equity_eod_prices`, topic `signalops.local.raw.v1`, with broker partition/offset present.
- PostgreSQL scheduler run and idempotency records confirmed the relational side remained healthy.

Follow-up items:

- Add a temporal backfill/replay gate for existing relational raw/normalized/signal rows before live cutover.
- After backfill/replay, redeploy live gateway, normalizer, signal persister, and Massive scheduler with `SIGNALOPS_TEMPORAL_DATABASE_URL` enabled.


## Gate G057: Temporal Backfill and Live Cutover

Timestamp: `2026-07-10T02:31:00Z`

Status: `passed — temporal backfill, live cutover, and Timescale write paths validated`

Gate name:

- Backfill existing relational temporal ledgers into TimescaleDB and cut live services over to the separate temporal DSN.

Criteria:

- Existing relational `raw_event_ledger`, `normalized_event_ledger`, and `signal_ledger` rows are copied into TimescaleDB.
- Backfill is idempotent and safe to rerun.
- Live gateway, normalizer, and signal-persister use `SIGNALOPS_TEMPORAL_DATABASE_URL`.
- Public HTTPS health is restored after cutover.
- A post-cutover Massive publish writes raw and normalized temporal rows to TimescaleDB.
- A post-cutover signal-topic publish writes a signal temporal row to TimescaleDB with broker lineage.
- Documentation captures the backfill command, cutover notes, and current PostgreSQL/Timescale storage roles.

Evidence:

- `scripts/backfill_temporal_ledgers.sh`
- `compose.yaml`
- `Makefile`
- `docs/deployment.md`
- `docs/docker_development.md`
- `docs/build_journal.md`

Verification performed:

- Pre-backfill relational counts: `raw=4`, `normalized=8`, `signal=4`.
- Pre-backfill Timescale counts after G056 Massive validation: `raw=1`, `normalized=0`, `signal=0`.
- First successful backfill inserted/upserted relational `raw=4`, `normalized=8`, and `signal=4` into TimescaleDB.
- Backfill rerun completed without duplicate growth.
- Live services rebuilt/recreated: `gateway`, `normalizer`, and `signal-persister`.
- Existing web container initially returned `502` because nginx retained the pre-recreate gateway IP; force-recreating `web` restored proxying.
- `GET http://localhost:18000/healthz`: `200 application/json`.
- `GET http://localhost:15173/healthz`: `200 application/json`.
- `GET https://signalops.syncratic.io/healthz`: `200 application/json`.
- `docker compose -f compose.yaml -f compose.traefik.yaml config --quiet`: passed.
- `make compose-temporal-backfill`: passed after cutover and left Timescale counts stable at `raw=6`, `normalized=9`, `signal=5`.
- Post-cutover Massive bounded publish reported `dry_run=false`, `companies=1`, `events_built=1`, `events_published=1`, `provider_requests=1`, `failures=0`.
- Timescale counts after Massive publish: `raw=6`, `normalized=9`, `signal=4`; raw and normalized increments validated the Massive-to-normalizer temporal path.
- Published `signal-g057-high` to `signalops.local.signal.v1`; Redpanda accepted partition `2`, offset `1`.
- Timescale `signal_ledger` row for `signal-g057-high`: tenant `tenant-local`, detector `signalops.static_test`, severity `high`, broker topic `signalops.local.signal.v1`, partition `2`, offset `1`.
- `signal-persister` log confirmed `signal persisted` for `signal-g057-high`.

Validation boundary:

- Protected `/v1/*` browser-facing reads correctly require bearer authentication after G055. Unauthenticated direct validation of signal and alert detail endpoints returned `401`; backend persistence was validated through Timescale rows and service logs.

Follow-up items:

- Add a first-class replay job model so temporal replay can be requested, audited, and exposed to operators.
- Consider changing nginx upstream resolution to runtime DNS resolution or always forcing a `web` recreate during gateway cutovers.
- Define production Massive ingestion cadence, quota safeguards, and failure alerting now that TimescaleDB is active for temporal ledgers.


## Gate G058: Replay Job Control Plane

Timestamp: `2026-07-10T02:49:00Z`

Status: `passed — replay job persistence and API surface implemented; execution worker pending`

Gate name:

- Add first-class replay job records so temporal replay can be requested, filtered, audited, and later executed by a worker.

Criteria:

- PostgreSQL has a `replay_jobs` table for replay control-plane state.
- Storage contracts and repository code can upsert, list, and get replay jobs.
- Gateway exposes create/list/detail replay job endpoints.
- Replay jobs include tenant, optional source/dataset filters, source kind, replay mode, status, requester, replay window, JSON filters/options/result, timestamps, and error message.
- API and audit docs define the execution boundary.
- Existing auth behavior protects the new `/v1/replay/jobs` routes.

Evidence:

- `migrations/000008_replay_jobs.up.sql`
- `internal/storage/storage.go`
- `internal/storage/postgres/repository.go`
- `internal/api/router.go`
- `docs/api.md`
- `docs/build_journal.md`

Verification performed:

- `docker run --rm -v "$PWD":/src -w /src golang:1.22-bookworm gofmt -w ...` completed.
- `docker run --rm -v "$PWD":/src -w /src golang:1.22-bookworm go test ./internal/storage ./internal/storage/postgres ./internal/api ./cmd/gateway` passed.
- Gateway Docker build ran `go test ./...` and passed.
- `git diff --check` passed.
- `make compose-storage-migrate` applied `000008_replay_jobs`.
- Direct PostgreSQL check confirmed `public.replay_jobs` exists.
- Gateway was rebuilt/redeployed.
- Web was force-recreated after gateway deployment to avoid stale nginx upstream resolution.
- Public health check returned `200 application/json`.
- Public and local unauthenticated `GET /v1/replay/jobs?tenant_id=tenant-local` returned `401 application/json`.

Validation boundary:

- G058 creates the replay job control plane only. It does not claim queued jobs or republish temporal records.
- Positive live API create/list/detail validation through the public route requires an authenticated bearer token; unit tests cover the no-auth handler behavior.

Follow-up items:

- Build the replay worker that claims queued jobs, reads Timescale rows by source kind and window, republishes through Redpanda with replay metadata, and updates job lifecycle/result counters.
- Add frontend replay job views and controls after the backend worker surface is defined.


## Gate G059: Replay Worker Execution

Timestamp: `2026-07-10T03:04:00Z`

Status: `passed — replay worker claims queued jobs and republishes capped temporal rows`

Gate name:

- Execute queued replay jobs against Timescale temporal ledgers through the durable broker pipeline.

Criteria:

- Replay worker can claim one queued job atomically.
- Worker supports raw, normalized, and signal replay source kinds.
- Worker republishes replayed records to the correct Redpanda topic with replay metadata.
- Worker updates replay job status/result counters on success or failure.
- Compose can build and run the worker in one-shot mode for bounded validation.
- Replay output flows through existing downstream services rather than bypassing the broker.

Evidence:

- `cmd/replay-worker/main.go`
- `Dockerfile`
- `compose.yaml`
- `internal/storage/storage.go`
- `internal/storage/postgres/repository.go`
- `docs/api.md`
- `docs/docker_development.md`
- `docs/build_journal.md`

Verification performed:

- Focused Docker Go tests passed for storage, repository, API, gateway, and replay-worker packages.
- Replay-worker Docker image built successfully; build stage ran full `go test ./...`.
- Compose + Traefik overlay config validation passed.
- Queued `replay-g059-raw` with `source_kind=raw_events`, `status=queued`, tenant `tenant-local`, and a window covering existing Timescale raw events.
- Ran replay worker one-shot with `SIGNALOPS_REPLAY_MAX_RECORDS=1`.
- Worker log showed `claimed replay job` and `replay job completed` with `published=1`.
- PostgreSQL `replay_jobs` row showed `status=succeeded`, result `scanned=1`, `published=1`, and no error.
- Normalizer consumed the replayed raw event and persisted a normalized event.
- Timescale contained one normalized event with `replay_job_id=replay-g059-raw`.
- Normalizer consumer group was Stable with total lag `0`.

Validation boundary:

- This gate validates capped single-query replay. Larger historical replay should add pagination/batching and cancellation before production-scale use.
- Authenticated frontend/operator controls are not included in this backend worker gate.

Follow-up items:

- Write the frontend-agent specification for replay job list/detail/create controls.
- Add worker batching, cancellation, and richer per-record failure accounting before high-volume replay.


## Gate G060: Frontend Replay Jobs UI Specification

Timestamp: `2026-07-10T03:12:00Z`

Status: `ready for frontend-agent implementation`

Gate name:

- Define the frontend implementation contract for replay job list/detail/create controls backed by G058/G059.

Criteria:

- Specification is placed under `docs/frontend/`.
- Specification documents backend replay APIs, request/response types, status/source/mode values, and execution semantics.
- Specification defines required TypeScript types, API client methods, query/mutation hooks, route, navigation, Dashboard integration, validation, and acceptance criteria.
- Specification explicitly prevents unsupported UI behavior such as cancel/retry/worker control and fake replay progress.

Evidence:

- `docs/frontend/replay_jobs_ui_implementation_spec.md`
- `docs/build_journal.md`
- `docs/gate_audit.md`

Validation performed:

- Checked against `docs/api.md` replay job contract.
- Checked against existing frontend implementation spec format and current UI architecture expectations.

Follow-up items:

- Frontend-agent should implement G060 from `docs/frontend/replay_jobs_ui_implementation_spec.md`.
- Backend can subsequently add replay cancellation, batching, and per-record failure accounting after the first UI is in place.


## Gate G060: Frontend Replay Jobs UI Implementation

Timestamp: `2026-07-10T03:34:00Z`

Status: `closed — implemented, deployed, and validated`

Gate name:

- Implement the `/replay` list/detail/create UI backed by the G058/G059 replay control plane, per `docs/frontend/replay_jobs_ui_implementation_spec.md`.

Criteria:

- `/replay` route exists and is reachable from shell navigation.
- Replay jobs list reads live backend data via the authenticated API client.
- Filters refetch live backend data via the query key.
- Create form sends valid `POST /v1/replay/jobs` requests and handles success/error states without claiming execution.
- Selected job detail displays lifecycle timestamps, filters, options, result, and errors.
- Queued/running detail polling works without opening a new SSE connection.
- Dashboard includes a compact replay summary/link.
- No unsupported cancel/retry/batch controls are present.
- `npm test`, `npm run build`, and `npm audit --json` pass.

Evidence:

- `web/src/types.ts` (replay types), `web/src/api/client.ts` (`listReplayJobs`/`getReplayJob`/`createReplayJob`).
- `web/src/api/queries.ts` (`useReplayJobs`/`useReplayJob`/`useCreateReplayJob`).
- `web/src/routes/ReplayJobsRoute.tsx` (route: metrics, filters, create form, table, detail panel).
- `web/src/router.tsx` (`/replay` route), `web/src/components/DashboardShell.tsx` (Replay nav link).
- `web/src/routes/DashboardRoute.tsx` (replay summary tile + link), `web/src/components/StatusBadge.tsx` (queued/running styles), `web/src/components/MetricTile.tsx` (`h-full`).
- `web/src/auth/session.tsx` (`useActor`), `web/src/lib/format.ts` + `format.test.ts` (`toRfc3339Utc`/`toDatetimeLocal`).
- `docs/build_journal.md`.

Implementation notes:

- Reconciled spec assumptions against the backend: `DisallowUnknownFields()` on the create body (send exactly the backend fields), RFC3339 window parsing (datetime-local → `…:ssZ`), `202 Accepted` on create, `queryLimit` cap of 200, and `replayActor` not deriving the actor from the JWT (identity sent as `requested_by` via `useActor`).
- Tenant comes from `useTenant()` (token tenant → `tenant-local` fallback) to stay tenant-scoped under auth, matching the rest of the app.
- Success state says `Queued` (not `Started replay`) unless the backend returns `running`; the worker runs separately and status/result come from refetches.
- REST/TanStack Query polling only; no second SSE stream. No cancel/retry/worker controls.

Verification performed:

- `npm test`: 36/36 pass (6 files; 2 new datetime-helper tests).
- `npm run build` (`tsc` + `vite build`): succeeded; `ReplayJobsRoute` chunk emitted.
- `npm audit --json`: 0 vulnerabilities, exit 0.
- `docker compose -f compose.yaml -f compose.traefik.yaml config --quiet`: succeeded.
- `git diff --check`: clean.

Additional validation performed:

- `make deploy-web` rebuilt/deployed the auth-enabled web image through the Traefik overlay.
- Forced web container recreation to ensure nginx serves the current image.
- Local `/replay` route returned `200 text/html`.
- Public `/replay` route returned `200 text/html`.
- Public `/healthz` returned `200 application/json`.
- Public unauthenticated `/v1/replay/jobs?tenant_id=tenant-local` returned `401 application/json`, confirming protected API behavior.
- PostgreSQL confirmed live replay job `replay-g059-raw` is `succeeded` with `published=1`.
- Operator reported G060 cleared after frontend implementation.

Follow-up items:

- Backend may add replay cancellation, retry, batching, and per-record failure accounting as later gates; the UI intentionally omits those controls.


## Gate G061: Backend Replay Hardening

Timestamp: `2026-07-10T04:14:46Z`

Status: `closed — implemented, deployed, and validated`

Gate name:

- Harden replay execution with batching/pagination, cancellation semantics, publish retry behavior, and per-record failure/accounting metadata.

Criteria:

- Replay worker reads temporal ledger rows in bounded batches rather than one capped query.
- Raw, normalized, and signal replay source queries support deterministic `LIMIT/OFFSET` pagination.
- Replay job cancellation can be requested through a backend API endpoint.
- Running replay workers detect cancellation between batches and stop publishing additional records.
- Replay worker retries broker publishes per record with a bounded attempt count.
- Replay job result JSON records scanned, published, failed, batches, cancellation state, and sampled per-record status.
- Replay worker env knobs are documented and included in Compose/example env.
- Backend tests and container build validation pass.

Evidence:

- `cmd/replay-worker/main.go` — batch loop, cancellation checks, retry publishing, structured result JSON.
- `cmd/replay-worker/main_test.go` — tests for paged batches, retries, and cancellation.
- `internal/storage/storage.go` — replay cancellation and paginated replay source contracts.
- `internal/storage/postgres/repository.go` — `CancelReplayJob` and paginated temporal ledger queries.
- `internal/api/router.go` — `POST /v1/replay/jobs/{replay_job_id}/cancel`.
- `internal/api/router_test.go` — replay cancel endpoint regression coverage.
- `compose.yaml`, `.env.example` — replay batch/retry env controls.
- `docs/api.md`, `docs/docker_development.md` — operator/API documentation.
- `docs/build_journal.md` — implementation and validation record.

Verification performed:

- Go formatting through `golang:1.22-bookworm` container succeeded.
- Focused Go tests passed: `go test ./internal/storage ./internal/storage/postgres ./internal/api ./cmd/replay-worker ./cmd/gateway`.
- Full Go suite passed: `go test ./...`.
- Compose config validation passed.
- Replay-worker image build passed and ran `go test ./...` during Docker build.
- Gateway image build passed.
- Gateway service was recreated from the updated image.
- Local gateway health returned `200 OK`.
- Local unauthenticated replay cancel request returned `401 Unauthorized`, confirming the deployed endpoint remains protected by auth.

Follow-up items:

- Perform authenticated live cancellation validation when an operator token is available.
- Optionally give frontend-agent a spec to expose replay job cancel controls and display per-record replay result samples.


## Gate G062: Frontend Replay Cancellation and Result Display Specification

Timestamp: `2026-07-10T04:20:03Z`

Status: `closed — specification written and ready for frontend-agent`

Gate name:

- Define the frontend implementation contract for replay job cancellation controls and G061 replay result accounting display.

Criteria:

- Specification is placed under `docs/frontend/`.
- Specification documents the G061 backend cancel endpoint and result JSON shape.
- Specification defines required frontend types, API client method, mutation hook, optimistic/refetch behavior, disabled states, and rollback expectations.
- Specification scopes UI changes to the existing `/replay` route and Dashboard replay summary without broad redesign.
- Specification requires rendering replay result counters and sampled per-record outcomes while tolerating older G059 result objects.
- Specification includes validation and acceptance criteria for frontend-agent.

Evidence:

- `docs/frontend/replay_jobs_cancel_and_results_spec.md`
- `docs/api.md`, Replay Jobs section
- `docs/build_journal.md`
- `docs/gate_audit.md`

Validation performed:

- Checked against G061 backend implementation: `POST /v1/replay/jobs/{replay_job_id}/cancel`, cancellation metadata, and replay worker result fields.
- Checked against G060 frontend replay UI specification and current route expectations.
- Corrected the result examples to avoid invalid duplicate JSON keys by separating non-canceled and canceled result shapes.

Follow-up items:

- Frontend-agent should implement G062 from `docs/frontend/replay_jobs_cancel_and_results_spec.md`.
- Backend/authenticated live cancellation validation can be performed once a browser-authenticated operator session is available.


## Gate G062: Frontend Replay Cancellation and Result Display Implementation

Timestamp: `2026-07-10T04:42:00Z`

Status: `closed — implemented, deployed, and operator-validated`

Gate name:

- Implement replay job cancellation controls and G061 result accounting display on `/replay`, per `docs/frontend/replay_jobs_cancel_and_results_spec.md`.

Criteria:

- `POST /v1/replay/jobs/{replay_job_id}/cancel` is wired through the authenticated frontend API client.
- Replay cancel mutation exists with optimistic/refetch behavior and safe rollback on failure.
- `/replay` detail panel exposes cancel action only for cancelable statuses.
- Cancel action has disabled/in-flight/error/success states.
- Replay result summary displays G061 counters while tolerating older G059 result objects.
- Per-record replay result samples render when present.
- Canceled status is represented consistently in badges, filters, polling, and metrics.
- Existing replay create/list/detail behavior continues to work.
- Tests/build/audit validations pass and are recorded.

Evidence:

- `web/src/types.ts` (`ReplayResult`, `ReplayRecordResult`, `ReplayCancellationResult`, `ReplayJobCancelRequest`).
- `web/src/api/client.ts` (`cancelReplayJob`), `web/src/api/queries.ts` (`useCancelReplayJob`).
- `web/src/lib/replay.ts` (`parseReplayResult`/`cancellationOf`/`isCancelableStatus`/`replayRecords`).
- `web/src/routes/ReplayJobsRoute.tsx` (cancel control, result summary, per-record table, Canceled metric).
- `web/src/routes/DashboardRoute.tsx` (canceled hint), `web/src/api/replay.test.ts`, `web/src/lib/replay.test.ts`.
- `docs/build_journal.md`.

Implementation notes:

- Reconciled against G061: `lifecycleActor` is JWT-derived under auth (cancel mirrors the alert/insight lifecycle pattern, not replay create); `CancelReplayJob` returns 200-with-existing-state for terminal jobs and 404 only when absent; `result.canceled` is `false` (bool) on completion and an object on cancel (union type models both).
- Optimistic cancel + rollback across all `['replay-jobs']` caches; detail cache seeded on success; list/detail invalidated on settle; duplicate in-flight cancels disabled via `mutation.variables`.
- `canceled` is terminal → detail polling stops; no new SSE stream. No retry/bulk/worker controls.

Verification performed:

- `npm test`: 45/45 pass (9 new — 4 cancel client, 5 result helpers).
- `npm run build` (`tsc` + `vite build`): succeeded.
- `npm audit --json`: 0 vulnerabilities, exit 0.
- `docker compose -f compose.yaml -f compose.traefik.yaml config --quiet`: succeeded.
- `git diff --check`: clean.

Follow-up items:

- Backend may add retry-failed / bulk cancellation as later gates; the UI intentionally omits those controls.

Closure validation:

- Timestamp: `2026-07-10T05:21:33Z`.
- Operator reported G062 browser/deploy validation is good.
- Public `/replay` returned `200 OK`.
- Public `/healthz` returned `200 OK`.
- Public unauthenticated `/v1/replay/jobs?tenant_id=tenant-local` returned `401 Unauthorized`, confirming API protection remains active.


## Gate G063: Replay Worker Always-On Deployment Posture

Timestamp: `2026-07-10T05:16:47Z`

Status: `closed — implemented, deployed, and validated`

Gate name:

- Promote `replay-worker` from manual/profile-only execution to an always-on Compose service.

Criteria:

- `replay-worker` is included in the default Compose service graph.
- Normal deployment can start `replay-worker` without `--profile replay`.
- Existing one-shot validation remains available through environment overrides.
- Documentation no longer instructs operators to use `--profile replay` for normal replay execution.
- Running replay-worker can claim queued jobs and no replay jobs remain stuck in `queued` after validation.

Evidence:

- `compose.yaml` — `replay-worker` no longer has `profiles: ["replay"]`.
- `docs/docker_development.md` — always-on operation documented.
- `docs/frontend/replay_jobs_ui_implementation_spec.md` — validation command updated.
- `docs/build_journal.md` — implementation and validation record.

Verification performed:

- Compose config validation passed.
- Compose services list includes `replay-worker` in the default graph.
- `docker compose -f compose.yaml -f compose.traefik.yaml up -d replay-worker` started the service without profile activation.
- `signalops-replay-worker-1` is running.
- Replay jobs table shows only terminal `succeeded` jobs and no queued jobs.
- `git diff --check` is clean.

Follow-up items:

- Add replay-worker operational health/last activity visibility if container-level status and logs are not enough for operators.


## Gate G064: Replay Operations Observability Backend

Timestamp: `2026-07-10T06:03:00Z`

Status: `closed — implemented, deployed, and validated`

Gate name:

- Add durable replay-worker heartbeat observability and a backend replay operations status endpoint.

Criteria:

- Replay worker writes durable heartbeat/activity state to PostgreSQL.
- Backend can report replay job counts by status.
- Backend can report replay-worker heartbeat status, health, last seen, last claimed job, last completed job, and last error metadata.
- Backend can return latest replay jobs alongside worker status for operational context.
- New API is protected by existing auth enforcement.
- Migration, tests, build, deployment, and live checks pass.

Evidence:

- `migrations/000009_replay_worker_heartbeats.up.sql`, `.down.sql`.
- `internal/storage/storage.go` — heartbeat/status-count types and contracts.
- `internal/storage/postgres/repository.go` — heartbeat and count implementations.
- `cmd/replay-worker/main.go` — heartbeat writes.
- `internal/api/router.go` — `GET /v1/replay/status`.
- `internal/api/router_test.go` — replay status test.
- `docs/api.md` — endpoint documentation.
- `docs/build_journal.md` — implementation and validation record.

Verification performed:

- Focused and full Go test suites passed.
- Compose config validation passed.
- Migration `000009_replay_worker_heartbeats` applied successfully to Postgres.
- Gateway and replay-worker images built successfully; Docker build ran full Go tests.
- Gateway and replay-worker were recreated; web was force-recreated afterward to refresh nginx upstream resolution.
- Local gateway health returned `200 OK`.
- Local unauthenticated replay status returned `401 Unauthorized`.
- Public `/replay` returned `200 OK`.
- Public unauthenticated replay status returned `401 Unauthorized`.
- Postgres heartbeat table shows a fresh `signalops-replay-worker` heartbeat with `idle` status.
- Replay jobs table shows no queued jobs.

Follow-up items:

- Frontend can consume `GET /v1/replay/status` in Dashboard/Health UI.
- Consider fixing nginx upstream resolution so web does not need forced recreation after gateway container replacement.


## Gate G065: Frontend Replay Operations Status UI Specification

Timestamp: `2026-07-10T06:13:33Z`

Status: `closed — specification written and ready for frontend-agent`

Gate name:

- Define the frontend implementation contract for replay-worker health/activity visibility on Dashboard and System/Health.

Criteria:

- Specification is placed under `docs/frontend/`.
- Specification documents G064 `GET /v1/replay/status` request/response behavior.
- Specification defines required frontend types, API client method, query hook, polling behavior, Dashboard integration, System route integration, helpers, tests, validation, and acceptance criteria.
- Specification explicitly excludes worker start/stop/restart controls, retry controls, bulk cancellation, replay SSE, and broad Dashboard redesign.

Evidence:

- `docs/frontend/replay_operations_status_ui_spec.md`.
- `docs/api.md`, Replay Jobs section.
- `docs/build_journal.md`.
- `docs/gate_audit.md`.

Validation performed:

- Checked against G064 backend implementation and documented response shape.
- Checked against current `web/src/routes/DashboardRoute.tsx` and `web/src/routes/SystemRoute.tsx` layout conventions.
- Confirmed validation requires authenticated browser checks and public unauthenticated `401` sanity check.

Follow-up items:

- Frontend-agent should implement G065 from `docs/frontend/replay_operations_status_ui_spec.md`.


## Gate G065: Frontend Replay Operations Status UI Implementation

Timestamp: `2026-07-10T06:27:00Z`

Status: `implemented — automated validations pass; authenticated browser validation pending deploy`

Gate name:

- Surface G064 replay-worker health and operations activity on Dashboard and System, per `docs/frontend/replay_operations_status_ui_spec.md`.

Criteria:

- `getReplayStatus` exposed via the authenticated API client.
- Query hook polls `GET /v1/replay/status` and participates in manual refresh.
- Dashboard surfaces replay worker health concisely (tile hint + Processing Health field).
- System route shows worker health, queue/running/failed counts, last seen, and worker activity.
- UI tolerates no heartbeats, stale/error workers, and missing job-count keys.
- Existing Dashboard, Replay Jobs, and System behavior continues to work.
- Tests/build/audit pass and are recorded.

Evidence:

- `web/src/types.ts` (replay operations status types), `web/src/api/client.ts` (`getReplayStatus`), `web/src/api/queries.ts` (`useReplayStatus`).
- `web/src/lib/replayStatus.ts` (`replayJobCount`/`worstReplayWorkerHealth`/`latestReplayWorkerSeenAt`).
- `web/src/routes/DashboardRoute.tsx` (tile hint from `job_counts`, Processing Health replay field), `web/src/routes/SystemRoute.tsx` (Replay Operations metrics + worker table).
- `web/src/components/StatusBadge.tsx` (online/stale/error), `web/src/api/replayStatus.test.ts`, `web/src/lib/replayStatus.test.ts`.
- `docs/build_journal.md`.

Implementation notes:

- Reconciled against G064: `limit` bounds workers only (latest_jobs hardcoded to 5); workers are not tenant-scoped; `job_counts` is always a full 0-filled map; `health` is backend-derived (online/stale/error) with `unknown` as a frontend fallback.
- Dashboard prefers authoritative `job_counts` totals over the capped list; falls back to list behavior on query error.
- REST polling (5s) only; no SSE. No worker start/stop, retry, bulk-cancel, or new route.

Verification performed:

- `npm test`: 50/50 pass (5 new — 2 status client, 3 helpers).
- `npm run build` (`tsc` + `vite build`): succeeded.
- `npm audit --json`: 0 vulnerabilities, exit 0.
- `docker compose -f compose.yaml -f compose.traefik.yaml config --quiet`: succeeded.
- `git diff --check`: clean.
- Unauthenticated `GET /v1/replay/status` → `401` (auth enforced).

Follow-up items:

- Deploy via `make deploy-web` and run the authenticated browser validation checklist (Dashboard replay context, System Replay Operations block, manual refresh, no mobile overflow).
- Backend may add worker restart/retry controls as later gates; the UI intentionally omits them.

## Gate G066: Multi-App Use Case Segmentation Metadata

Timestamp: `2026-07-10T06:45:00Z`

Status: `planned — architecture gate documented`

Gate name:

- Introduce app/domain/use-case metadata and define the first specialized app profile.

Criteria:

- `app_id`, `domain`, and `use_case` are treated as first-class routing,
  filtering, authorization, and presentation metadata.
- Existing ingestion, normalization, detection, persistence, replay, and UI
  paths continue to work when the metadata is omitted.
- Metadata can flow from raw ingestion through normalized events, signals,
  alerts, insights, and API filters where relevant.
- Initial app profiles include the existing SignalOps Console and the first
  specialized `marketops` profile.
- The backend remains unified; no separate backend, broker, or database is
  created per app profile in this gate.
- Documentation defines metadata defaults, semantics, and non-goals.

Evidence:

- `docs/design/multi_app_use_case_segmentation.md`.
- `docs/build_journal.md`.
- `docs/gate_audit.md`.

Validation performed:

- Documented the proposed segmentation model and accepted next gate.
- Confirmed the design preserves the current unified engine and current
  SignalOps UI as the platform console.
- Confirmed `marketops` is represented as an app profile over the shared core,
  not as a forked backend.

Follow-up items:

- Implement G066 backend metadata support across relevant contracts, storage
  records, API filters, and broker payload propagation.
- Define static initial app profiles for `console` and `marketops`.
- After backend metadata lands, provide frontend-agent a specification for a
  multi-app shell and MarketOps profile composition.

## Gate G066: Multi-App Use Case Segmentation Metadata Implementation

Timestamp: `2026-07-10T07:28:00Z`

Status: `closed — implemented, deployed, and smoke validated`

Gate name:

- Make `app_id`, `domain`, and `use_case` first-class additive metadata and define initial app profiles.

Criteria:

- Raw, normalized, and signal event contracts accept optional `app_id`, `domain`, and `use_case`.
- Raw, normalized, signal, alert, and insight ledgers persist the metadata.
- List/detail APIs expose the metadata and list APIs accept optional filters.
- Missing metadata remains backward compatible through defaults.
- Static app profiles define `console` and `marketops` without a separate backend.
- Massive scheduled market-data ingestion identifies the first specialized app profile as `marketops`.
- Python signal emission and Go signal persistence tolerate and propagate the metadata.

Evidence:

- `internal/appmeta/appmeta.go` defines defaults and static profiles.
- `migrations/000010_app_use_case_metadata.up.sql` adds ledger columns and indexes.
- `contracts/events/*.schema.json` and `pkg/contracts/events.go` include optional metadata fields.
- `internal/api/router.go` exposes `GET /v1/app-profiles` and metadata filters/DTO fields.
- `internal/normalization/processor.go`, `python/signalops_workers/worker.py`, and `internal/signals/processor.go` propagate metadata across normalized events, signals, alerts, and insights.
- `internal/adapters/marketdata/massive/event_builder.go` sets Massive market-data events to `marketops`.
- `docs/api.md`, `docs/build_journal.md`, and `docs/gate_audit.md` document the gate.

Validation performed:

- Go formatting via Docker `gofmt` succeeded.
- Go test suite via Docker `go test ./...` passed.
- Python worker tests passed: 39 tests, 1 existing pytest config warning.
- JSON schema validation passed for all event schemas.
- Compose plus Traefik config validation passed.
- `git diff --check` passed.

Follow-up items:

- Write a frontend-agent specification for `GET /v1/app-profiles`, multi-app shell routing, and the first MarketOps profile.
- Add authenticated browser/API validation for app profile rendering once frontend consumption exists.


Deployment validation:

- Applied Postgres migration `000010_app_use_case_metadata`; `schema_migrations` contains the version and `signal_ledger` has `app_id`, `domain`, and `use_case` columns.
- Applied Timescale migration `000002_app_use_case_metadata`; `schema_migrations` contains the version and temporal `signal_ledger` has `app_id`, `domain`, and `use_case` columns.
- Rebuilt `gateway`, `normalizer`, `signal-persister`, `raw-worker`, `massive-scheduler`, and `massive-puller` images.
- Recreated `gateway`, `normalizer`, `signal-persister`, `raw-worker`, and `massive-scheduler`; force-recreated `web` after gateway recreation.
- Local `GET /healthz` returned `200`.
- Local and public unauthenticated `GET /v1/app-profiles` returned `401`, confirming the new endpoint is deployed behind auth.
- Public `https://signalops.syncratic.io/` returned `200`.
- Gateway, normalizer, and signal-persister startup logs showed clean service starts.

## Gate G066: Commit and Push Audit

Timestamp: `2026-07-10T07:03:01Z`

Status: `pushed`

Evidence:

- Commit: `82f390c Implement G066 app use-case metadata`.
- Remote: `origin git@github.com:lukebabs/signalops.git`.
- Push result: `main` updated from `c424961` to `82f390c`.

Follow-up items:

- Frontend-agent specification for the multi-app shell and MarketOps profile remains the next gate.

## Gate G067: Frontend App Profiles and MarketOps Shell Specification

Timestamp: `2026-07-10T07:08:00Z`

Status: `closed — specification written and ready for frontend-agent`

Gate name:

- Define frontend implementation contract for consuming G066 app profiles and adding the first MarketOps app profile.

Criteria:

- Specification is placed under `docs/frontend/`.
- Specification documents `GET /v1/app-profiles` request/response behavior.
- Specification preserves SignalOps Console as the default app experience.
- Specification defines MarketOps routes under `/marketops/*` without removing existing routes.
- Specification requires `app_id`, `domain`, and `use_case` filter propagation for supported data APIs.
- Specification includes frontend types, API client method, query hook, app context, shell/nav behavior, tests, browser validation, acceptance criteria, and non-goals.

Evidence:

- `docs/frontend/app_profiles_marketops_shell_spec.md`.
- `docs/design/multi_app_use_case_segmentation.md`.
- `internal/appmeta/appmeta.go`.
- `docs/api.md`.

Validation performed:

- Reviewed against current frontend route/shell/API structure.
- Reviewed against G066 backend app profile shape and metadata filter support.
- Confirmed the spec requires no backend changes and keeps the current Console routes intact.

Follow-up items:

- Frontend-agent should implement G067 from `docs/frontend/app_profiles_marketops_shell_spec.md`.
- After implementation, deploy with the Traefik overlay and complete authenticated browser validation.

## Gate G068: Frontend App Profiles and MarketOps Shell Implementation

Timestamp: `2026-07-10T15:32:00Z`

Status: `implemented — local automated validation passed; authenticated browser validation pending operator`

Gate name:

- Implement the G067 frontend contract: consume `GET /v1/app-profiles`, add an app-aware shell with an app selector, MarketOps route aliases under `/marketops/*`, and G066 metadata-filter propagation — without removing or renaming existing console routes.

Criteria:

- `GET /v1/app-profiles` is consumed through the existing authenticated API client.
- Existing SignalOps Console routes and nav remain available and default.
- MarketOps routes are available and registered under `/marketops/*`.
- App selector can switch between Console and MarketOps.
- MarketOps-supported data routes pass `app_id=marketops` and `domain=market_data` to backend list APIs.
- No backend changes.
- Tests/build/audit pass; browser validation confirmed by operator before final close.

Evidence:

- `web/src/apps/appProfiles.ts` (`CONSOLE_PROFILE`, `MARKETOPS_PROFILE` fallbacks), `web/src/apps/AppProfileContext.tsx` (`AppProfileProvider`/`useAppProfile`), `web/src/apps/appRouting.ts` (`appIdFromPathname`, `metadataFilterForApp`, `defaultRouteForApp`, `navForApp`).
- `web/src/components/DashboardShell.tsx`: app selector navigates to the profile default route; nav is rendered from `useAppProfile().nav`.
- `web/src/router.tsx`: the 10 MarketOps aliases are now registered in `rootRoute.addChildren` (previously declared but unregistered — would have 404'd).
- `web/src/routes/{Dashboard,RawEvents,NormalizedEvents,Signals,Alerts,Insights}Route.tsx`: spread `metadataFilter` into the G066-aware list queries.
- `web/src/routes/{Sources,System}Route.tsx`: `Providers`/`Health` headings under MarketOps.
- `web/src/api/client.ts` (`getAppProfiles` + `app_id`/`domain`/`use_case` on the five list methods), `web/src/api/queries.ts` (`useAppProfiles`, `queryKeys.appProfiles`), `web/src/types.ts` (`AppProfile`, `AppProfilesResponse`, metadata filter fields).
- Tests: `web/src/apps/appRouting.test.ts` (11), `web/src/api/appProfiles.test.ts` (9).

Validation performed:

- `cd web && npm test`: 70 passed (12 files).
- `cd web && npm run build`: `tsc` + `vite build` succeeded.
- `cd web && npm audit --json`: 0 vulnerabilities.
- `docker compose -f compose.yaml -f compose.traefik.yaml config --quiet`: OK.
- `git diff --check`: clean.

Validation NOT yet performed (blocks final close):

- Authenticated browser validation (spec §Browser Validation Required): app selector labels and MarketOps selection, `/marketops/*` routing without overflow, `app_id=marketops`/`domain=market_data` in network requests on supported routes, console parity, and unauthenticated redirect. Requires browser-driven IdP login and a `make deploy-web` Traefik-overlay deploy.

Follow-up items:

- Operator deploys via `make deploy-web` (auth flag + Traefik overlay) and completes authenticated browser validation.
- On pass, mark G068 fully closed; file any nav/overflow/auth follow-ups.

## Gate G069: Traefik Overlay Guard for Public Web Rebuilds

Timestamp: `2026-07-10T15:45:00Z`

Status: `closed — root cause fixed and public route validated`

Gate name:

- Prevent public 404 after web rebuilds by making the Traefik overlay the default Compose render for this deployment.

Root cause:

- The running `web` container had no `traefik.*` labels and was attached only to `signalops_default`.
- Docker inspect showed it was created from `compose.yaml` only, so Traefik had no `signalops` router and returned 404 for the public host.

Fix:

- Added `COMPOSE_FILE=compose.yaml:compose.traefik.yaml` to the live deployment `.env` and `.env.example`.
- Updated deployment documentation and Makefile comments to explain the guard.
- Recreated `web` using plain `docker compose up -d --force-recreate web`; the resulting container included Traefik labels and external network attachment.

Validation performed:

- Plain `docker compose config --quiet` passed.
- Plain `docker compose config` rendered the Traefik labels and `syncratic-core_syncratic_net` network.
- `docker inspect signalops-web-1` confirmed labels and both Compose files in `com.docker.compose.project.config_files`.
- Public root returned `200`.
- Public unauthenticated `/v1/app-profiles` returned `401`.
- Local root returned `200`.

Follow-up items:

- Do not remove `COMPOSE_FILE` from the deployment `.env` unless replacing it with another permanent Traefik overlay mechanism.
- Continue using `make deploy-web` for public web image rebuilds to keep frontend auth enabled.

## Gate G068: Frontend App Profiles Deployment Validation

Timestamp: `2026-07-10T15:50:00Z`

Status: `deployed — automated and public route smoke validation passed; authenticated browser validation pending operator`

Gate name:

- Validate and deploy the frontend app-profile/MarketOps implementation.

Validation performed:

- `npm test`: 70 passed across 12 frontend test files.
- `npm run build`: TypeScript and Vite production build succeeded.
- `npm audit --json`: 0 vulnerabilities.
- `docker compose config --quiet`: passed using default `COMPOSE_FILE=compose.yaml:compose.traefik.yaml`.
- `make deploy-web`: completed successfully.
- Deployed asset grep confirmed `MarketOps`, `/marketops/dashboard`, and `app-profiles` code are present in `signalops-web-1`.
- Public route smoke checks returned `200` for all ten MarketOps aliases.
- Public unauthenticated `/v1/app-profiles` returned `401`.
- Docker inspect confirmed `signalops-web-1` has Traefik labels and both Compose files in `com.docker.compose.project.config_files`.
- `git diff --check`: clean.

Outstanding items:

- Complete authenticated browser validation from `docs/frontend/app_profiles_marketops_shell_spec.md`.
- Commit and push the frontend implementation, Traefik guard, and journal updates after browser validation or operator approval.

## Gate G068: Commit and Push Audit

Timestamp: `2026-07-10T16:04:00Z`

Status: `pushed`

Evidence:

- Commit: `bdcf9a8 Implement G068 frontend app profiles and MarketOps shell`.
- Remote: `origin git@github.com:lukebabs/signalops.git`.
- Push result: `main` updated from `21a941b` to `bdcf9a8`.
- The commit contains the frontend implementation, tests, the G067 spec, and journals; the working-tree G068 deployment-validation and G069 Traefik-guard journal entries were also captured.

Follow-up items:

- Commit the G069 Traefik-overlay guard config files (`.env.example`, `Makefile`, `docs/deployment.md`), which remain uncommitted in the working tree.
- Complete authenticated browser validation to fully close G068.

## Gate G068: Final Browser Validation Closeout

Timestamp: `2026-07-10T16:12:00Z`

Status: `closed — authenticated browser validation passed`

Gate name:

- Close frontend app profiles and MarketOps shell implementation after operator browser validation.

Operator validation confirmed:

- App selector displays SignalOps Console and MarketOps.
- Switching to MarketOps works.
- `/marketops/*` routes render successfully without observed layout overflow.
- Supported data routes include `app_id=marketops` and `domain=market_data`.
- Existing SignalOps Console routes remain functional.
- Unauthenticated/auth redirect behavior remains intact.

Evidence:

- Prior automated validation: 70 frontend tests passed, production build passed, audit returned 0 vulnerabilities, public MarketOps route smoke checks returned `200` for all ten aliases.
- Implementation commit: `bdcf9a8 Implement G068 frontend app profiles and MarketOps shell` pushed to `origin/main`.

Follow-up items:

- None for G068. The next frontend work should build on the app-profile shell rather than reopening this gate.

## Gate G070: MarketOps DSM v0 Detector Pack

Timestamp: `2026-07-10T16:48:03Z`

Status: `validated — local tests, schema validation, worker image build, live replay smoke, and lifecycle derivation passed`

Gate name:

- Add the first deterministic MarketOps DSM detector pack on the existing SignalOps normalized-event worker path.

Implementation:

- Added detector `marketops.dsm.eod_price_v1`.
- Scoped matching to normalized Massive equity EOD events with `app_id=marketops`, `domain=market_data`, `source_adapter=market_data.massive`, `dataset=equity_eod_prices`, and `use_case=daily_market_surveillance`.
- Emitted `marketops.dsm.volatility_expansion` and `marketops.dsm.price_quality_exception` as validated `signal.v1` payloads.
- Added stable signal IDs, ticker entities, supporting metrics, semantic evidence, normalized-event evidence, graph-target candidates, confidence, severity, and recommendations.
- Changed the Python worker default detector to `marketops.dsm.eod_price_v1` while retaining `SIGNALOPS_WORKER_DETECTOR_ID` override behavior.
- Propagated `app_id`, `domain`, and `use_case` into worker-built `signal.v1` records.
- Added `docs/marketops/G070_marketops_dsm_reconciliation.md` to reconcile target MarketOps specs with the immediate G070 gate.

Validation performed:

- `env PYTHONDONTWRITEBYTECODE=1 PYTHONPATH=python pytest python/tests`: 48 passed.
- `python3 scripts/validate_json_schemas.py`: passed for all checked-in event schemas.
- `docker compose config --quiet`: passed after the raw-worker detector default change.
- `docker compose build raw-worker`: passed.
- `docker compose up -d --no-deps --build raw-worker`: recreated the always-on worker with the MarketOps detector.
- Published bounded normalized event `evt-g070-marketops-live` to `signalops.local.normalized.v1`; Redpanda accepted partition `2`, offset `6`.
- Worker logs showed detector evaluation and event processing.
- Persister logs showed signal persisted: `sig_marketops_dsm_eod_price_v1_b85d8522f80cb07abc3f`.
- Postgres `signal_ledger` verified `app_id=marketops`, `domain=market_data`, `use_case=daily_market_surveillance`, detector `marketops.dsm.eod_price_v1`, type `marketops.dsm.volatility_expansion`, severity `high`, confidence `0.81`, and event ID `evt-g070-marketops-live`.
- Postgres lifecycle ledgers verified open alert `alert:sig_marketops_dsm_eod_price_v1_b85d8522f80cb07abc3f` and active insight `insight:sig_marketops_dsm_eod_price_v1_b85d8522f80cb07abc3f`.

Follow-up items:

- Continue G071+ with first-class MarketOps asset universe storage/API and broader replay fixtures.

## Gate G071: MarketOps Asset Universe Storage/API

Timestamp: `2026-07-10T17:16:56Z`

Status: `implemented — storage migration, backend tests, gateway build, and live DB seed validation passed; authenticated API smoke pending operator token`

Gate name:

- Promote the MarketOps Top 50 mega-cap seed into first-class storage and a read API.

Implementation:

- Added `marketops_asset_universe` with tenant, app/domain/use-case metadata, source identity, universe group, rank, ticker/company keys, asset type, exchange, sector, industry, active flag, and metadata.
- Seeded 50 `tenant-local` assets for `top50_megacap` from `internal/adapters/marketdata/massive/top50megacap.normalized.csv`.
- Added repository query support and `GET /v1/tenants/{tenant_id}/marketops/assets`.
- Added route test coverage for MarketOps metadata and `active_only` filter propagation.
- Updated `docs/api.md` with the new endpoint.

Validation performed:

- `go test ./internal/api ./internal/storage/postgres`: passed in the Go Docker image.
- `go test ./...`: passed in the Go Docker image.
- `make compose-storage-migrate`: applied migration `000011_marketops_asset_universe`, created indexes, and inserted 50 seed rows.
- `docker compose up -d --no-deps --build gateway`: rebuilt/restarted the gateway and passed build-time `go test ./...`.
- Postgres query verified 50 rows for `tenant-local/top50_megacap`, with ranks 1 through 50.
- Unauthenticated curl returned `401 missing bearer token`, which is expected because the running gateway has `SIGNALOPS_AUTH_ENABLED=true`.

Follow-up items:

- Use an authenticated operator/browser session to smoke `GET /v1/tenants/tenant-local/marketops/assets`.
- G072 should add Massive options contract daily normalization.

## Gate G071 Frontend Follow-Up: MarketOps Asset Universe UI Spec

Timestamp: `2026-07-10T17:28:00Z`

Status: `specified — ready for frontend-agent implementation`

Scope:

- Specify a read-only MarketOps asset universe page against the G071 backend endpoint.

Evidence:

- Added `docs/frontend/marketops_asset_universe_ui_spec.md`.
- Spec references backend commit `0d651bf`, endpoint `GET /v1/tenants/{tenant_id}/marketops/assets`, and the existing app-profile shell patterns.
- Spec includes implementation steps, required UI behavior, tests, validation commands, and browser validation checklist.

Follow-up items:

- Implement the spec in `web/`.
- Validate with `npm test`, `npm run build`, `npm audit --json`, and authenticated browser checks.

## Gate G071 Frontend Follow-Up: MarketOps Asset Universe UI Implementation

Timestamp: `2026-07-10T18:19:00Z`

Status: `closed — local automated validation, deployed route check, seeded data verification, and operator render check passed`

Gate name:

- Implement the read-only MarketOps Asset Universe UI at `/marketops/assets` against the G071 backend.

Evaluation:

- Verified the G071 backend (`0d651bf`) contract matches the spec: route `GET /v1/tenants/{tenant_id}/marketops/assets`, string-parsed `active_only`, `{"assets":[...]}` envelope, and all 21 `marketOpsAssetDTO` JSON fields.

Criteria:

- `/marketops/assets` registered and reachable; MarketOps-only; Console nav unchanged.
- Authenticated API client fetches real assets; read-only, no mock fallback.
- Page renders metrics, dense table, loading/error/empty states, and metadata JSON.
- Unit tests cover API path/query construction and app-routing changes.

Evidence:

- `web/src/types.ts`, `web/src/api/client.ts` (`listMarketOpsAssets`), `web/src/api/queries.ts` (`useMarketOpsAssets`).
- `web/src/routes/MarketOpsAssetsRoute.tsx` (mirrors SourcesRoute).
- `web/src/router.tsx` (route + registration), `web/src/apps/appRouting.ts` (AppRoutePath + Assets nav), `web/src/components/DashboardShell.tsx` (`symbols` icon).
- Tests: `web/src/api/marketopsAssets.test.ts` (7), `web/src/apps/appRouting.test.ts` (+1).
- Spec annotated for two gaps: `MODULE_ICONS` `symbols` entry and deferred route-render test.

Spec fixes applied during implementation:

- Added `symbols: CircleDollarSign` to `DashboardShell` `MODULE_ICONS` (spec omitted it).
- Deferred the route-render test to browser validation (node-only vitest); covered via API/routing/query-key unit tests (G068 precedent).

Validation performed:

- `npm test`: 78 passed. `npm run build`: succeeded. `npm audit --json`: 0 vulnerabilities. `git diff --check`: clean.
- Operator confirmed `/marketops/assets` renders the 50 seeded assets.
- Deployed web route `GET http://localhost:15173/marketops/assets` returned HTTP `200`.
- Postgres verified `tenant-local` / `top50_megacap` contains 50 assets, 50 active assets, ranks `1..50`, 15 sector keys, 16 industry keys, and one source.
- Rank-order sample verified first ranks `NVDA`, `AAPL`, `GOOGL`, `MSFT`, `AMZN`; final ranks include `NFLX`, `PLTR`, `MRK`, `GS`, `GEV`.
- API path/query construction for `universe_group=top50_megacap`, `active_only=true`, and `limit=50` is covered by `web/src/api/marketopsAssets.test.ts`; route code uses the same query parameters.

Validation NOT yet performed:

- Independent machine browser/mobile-overflow check; no local Playwright dependency is present in `web/`, so this remains a manual/operator visual check.

Follow-up items:

- Optional: MarketOps dashboard widget (spec §7), deferred to avoid layout churn.

## Gate G072: Massive Options Contract Daily Normalization

Timestamp: `2026-07-10T18:26:53Z`

Status: `closed — full Go tests, schema validation, normalizer Docker build, and live normalized-ledger smoke passed`

Gate name:

- Normalize Massive option contract daily records into a canonical MarketOps option payload.

Implementation:

- Added a MarketOps/Massive normalization strategy for raw events with `app_id=marketops`, `source_adapter=market_data.massive`, and `dataset=options_contracts_daily`.
- Canonicalized option ticker, underlying ticker, call/put contract type, expiration date, strike, observation date, optional non-negative OHLC/VWAP, non-negative integer volume/open interest, provider contract ID, asset type, and raw provider metadata.
- Required option ticker, underlying ticker, call/put contract type, YYYY-MM-DD expiration/observation dates, and positive strike price.
- Preserved existing identity normalization for other raw events and existing invalid-event DLQ behavior for malformed option payloads.

Evidence:

- `internal/normalization/processor.go`: normalizer strategy hook plus `marketops_massive_option_contract_daily_v1`.
- `internal/normalization/processor_test.go`: canonical payload, entity/app metadata, and invalid payload tests.
- `docs/api.md`: normalized ledger contract note for Massive options records.

Validation performed:

- Dockerized `gofmt` on the touched Go files.
- `go test ./internal/normalization`: passed in the Go Docker image.
- `go test ./...`: passed in the Go Docker image.
- `docker compose build normalizer`: passed; build step also ran `go test ./...`.
- `python3 scripts/validate_json_schemas.py`: passed.
- `docker compose up -d --no-deps --build normalizer`: recreated the live normalizer with the G072 image.
- Published raw event `evt-g072-option-live` to `signalops.local.raw.v1`, partition `2`, offset `10`.
- Normalizer persisted normalized event `evt-g072-option-live` to normalized partition `2`, offset `7`.
- `signalops.normalizer.v1` was Stable with total lag `0`.
- TimescaleDB verified canonical normalized payload fields and strategy `marketops_massive_option_contract_daily_v1`, plus option-contract and ticker entities.

Follow-up items:

- G073 should add the MarketOps feature-builder layer for option-interest and price-derived features.

## Gate G073: MarketOps Feature Builder Layer

Timestamp: `2026-07-10T18:37:43Z`

Status: `closed — full Go tests, schema validation, normalizer Docker build, and live feature-map smoke passed`

Gate name:

- Add the first deterministic MarketOps feature-builder layer for option-interest and price-derived features.

Implementation:

- Added deterministic `normalized_payload.features` enrichment for MarketOps Massive `equity_eod_prices` and `options_contracts_daily` records in the existing Go normalizer.
- Equity EOD feature enrichment preserves the source payload and adds price-derived metrics when inputs exist: `open_close_move_pct`, `intraday_range_pct`, `vwap_distance_pct`, `daily_return_pct`, and `volume`.
- Option contract daily enrichment adds price-derived metrics plus option-interest metrics: `open_interest`, `volume`, `volume_open_interest_ratio`, and `days_to_expiration`.
- Preserved G072 option contract validation/canonicalization and the existing normalization DLQ path for invalid option payloads.
- Kept dedicated feature/artifact services deferred; G073 uses the normalized event boundary as the first deterministic feature-builder layer.

Evidence:

- `internal/normalization/processor.go`: MarketOps Massive feature enrichment helpers and strategies.
- `internal/normalization/processor_test.go`: equity feature tests plus option price/interest feature tests.
- `docs/api.md`: normalized ledger feature-map contract note.

Validation performed:

- Dockerized `gofmt` on the touched Go files.
- `go test ./internal/normalization`: passed in the Go Docker image.
- `go test ./...`: passed in the Go Docker image.
- `python3 scripts/validate_json_schemas.py`: passed.
- `docker compose build normalizer`: passed; build step also ran `go test ./...`.
- `docker compose up -d --no-deps --build normalizer`: recreated the live normalizer with the G073 image.
- Published raw events `evt-g073-equity-live` and `evt-g073-option-live` to `signalops.local.raw.v1`; Redpanda accepted equity partition `2` offset `11` and option partition `1` offset `5`.
- Normalizer persisted equity normalized partition `2` offset `8` and option normalized partition `1` offset `4`.
- `signalops.normalizer.v1` was Stable with total lag `0`.
- TimescaleDB verified persisted equity feature map with price-derived metrics and option feature map with price-derived plus option-interest metrics.

Follow-up items:

- G074 should add DSM artifact generation and graph proposal payloads.

## Gate G074: DSM Artifact And Graph Proposal Payloads

Timestamp: `2026-07-10T18:51:48Z`

Status: `closed — Python tests, schema validation, raw-worker Docker build, and live signal-ledger smoke passed`

Gate name:

- Add DSM artifact generation and graph proposal payloads for MarketOps signals.

Implementation:

- Added stable DSM artifact proposal IDs to `marketops.dsm.eod_price_v1` signal emissions.
- Added `marketops.dsm.signal_artifact.v1` proposal objects in `semantic_evidence` with source event, ticker subject, computed features, quality issues, confidence, severity, and summary.
- Replaced the minimal graph target with graph node candidates for ticker, DSM signal type, and artifact, plus `EXHIBITS_SIGNAL` and `SUPPORTED_BY_ARTIFACT` relationship candidates.
- Kept dedicated artifact tables/services and graph acceptance deferred; G074 persists deterministic proposals through existing `signal.v1` and signal ledger fields.

Evidence:

- `python/signalops_plugins/detectors/marketops.py`: artifact ID, artifact proposal, graph proposal helpers.
- `python/tests/plugins/test_marketops_detector.py`: artifact ID stability, artifact proposal, and graph relationship tests.
- `docs/python_worker.md`: operational description of persisted proposal payloads.

Validation performed:

- `env PYTHONDONTWRITEBYTECODE=1 PYTHONPATH=python pytest python/tests`: 48 passed.
- `python3 scripts/validate_json_schemas.py`: passed.
- `docker compose build raw-worker`: passed.
- `docker compose up -d --no-deps --build raw-worker`: recreated the live worker with the G074 image.
- Published normalized event `evt-g074-marketops-live` to `signalops.local.normalized.v1`, partition `1`, offset `5`.
- `signalops.normalized-worker.v1` was Stable with total lag `0`.
- Signal persister stored `sig_marketops_dsm_eod_price_v1_fc849d452e685952d763` from signal partition `0`, offset `5`.
- Postgres verified artifact ID `artifact_marketops_dsm_v1_dcaff3d9bec0fcd0063e`, embedded `marketops.dsm.signal_artifact.v1`, graph node candidates, `EXHIBITS_SIGNAL`, and `SUPPORTED_BY_ARTIFACT` relationship candidates.
- Alert and insight lifecycle rows were derived for the critical signal.

Follow-up items:

- G075 should add the broader DSM taxonomy pack including accumulation, hedging pressure, speculative call/put pressure, pinning risk, and divergence.

## Gate G075: MarketOps DSM Taxonomy Pack

Timestamp: `2026-07-10T19:06:55Z`

Status: `closed — Python tests, schema validation, compose validation, raw-worker Docker build, and live taxonomy smoke passed`

Gate name:

- Broaden the DSM taxonomy pack with accumulation, hedging pressure, speculative call/put pressure, pinning risk, and divergence.

Implementation:

- Added detector `marketops.dsm.taxonomy_v1` and registered it in the Python worker loader.
- Changed worker defaults in config, Compose, and `.env.example` to `marketops.dsm.taxonomy_v1`; the legacy `marketops.dsm.eod_price_v1` remains explicitly loadable.
- Added deterministic equity taxonomy rules for accumulation and divergence while preserving volatility expansion and price-quality exception behavior.
- Added deterministic option taxonomy rules for hedging pressure, speculative call pressure, speculative put pressure, and pinning risk using G073 option-interest features.
- Reused the G074 artifact proposal and graph target payload path for all emitted taxonomy signals.

Evidence:

- `python/signalops_plugins/detectors/marketops.py`: taxonomy detector implementation.
- `python/signalops_workers/detectors.py` and `python/signalops_workers/config.py`: detector registration/default.
- `python/tests/plugins/test_marketops_detector.py`: taxonomy coverage for all new signal types.
- `python/tests/test_detectors.py` and `python/tests/test_config.py`: loader/default coverage.

Validation performed:

- `env PYTHONDONTWRITEBYTECODE=1 PYTHONPATH=python pytest python/tests`: 56 passed.
- `python3 scripts/validate_json_schemas.py`: passed.
- `docker compose config --quiet`: passed.
- `docker compose build raw-worker`: passed.
- Recreated live `raw-worker` with explicit `SIGNALOPS_WORKER_DETECTOR_ID=marketops.dsm.taxonomy_v1` because the local untracked `.env` still overrides the Compose fallback.
- Published normalized events `evt-g075-taxonomy-accumulation-live` and `evt-g075-taxonomy-pinning-live`; `signalops.normalized-worker.v1` returned to Stable with lag `0`.
- Postgres verified persisted `marketops.dsm.accumulation` and `marketops.dsm.pinning_risk` signals with detector `marketops.dsm.taxonomy_v1`, taxonomy signal IDs, artifact IDs, and lifecycle alert/insight rows.
- Rebuilt/recreated after adding option-interest supporting metrics and published `evt-g075-taxonomy-metrics-live`; Postgres verified `open_interest`, `volume_open_interest_ratio`, `days_to_expiration`, `moneyness_pct`, and `contract_type` in supporting metrics.

Follow-up items:

- Clear or update the local untracked `.env` override during deployment so Compose uses `marketops.dsm.taxonomy_v1` without an explicit shell override.

## Gate G075 Option Taxonomy Live Coverage Addendum

Timestamp: `2026-07-10T19:32:05Z`

Status: `closed — remaining option taxonomy signals live-validated`

Scope:

- Complete live pipeline coverage for the G075 option taxonomy signals not exercised in the initial G075 smoke.

Validation performed:

- Verified live `raw-worker` uses `SIGNALOPS_WORKER_DETECTOR_ID=marketops.dsm.taxonomy_v1`.
- Published bounded daily option contract normalized events `evt-g075-hedging-live`, `evt-g075-spec-call-live`, and `evt-g075-spec-put-live` to `signalops.local.normalized.v1`.
- `signalops.normalized-worker.v1` returned to Stable with total lag `0`.
- Postgres `signal_ledger` verified:
  - `marketops.dsm.hedging_pressure` with severity `high`.
  - `marketops.dsm.speculative_call_pressure` with severity `medium`.
  - `marketops.dsm.speculative_put_pressure` with severity `medium`.
- Verified option-interest supporting metrics persisted for all three signals.
- Verified each signal includes an artifact ID, five graph targets, embedded `marketops.dsm.signal_artifact.v1`, open alert, and active insight.

Follow-up items:

- None for G075 option taxonomy live coverage.


## Gate G076: MarketOps DSM Workbench UI Specification

Timestamp: `2026-07-10T20:05:00Z`

Status: `specified — ready for frontend-agent implementation`

Scope:

- Specify a MarketOps-only DSM workbench at `/marketops/dsm` for inspecting G075 `marketops.dsm.taxonomy_v1` signal output.

Specification:

- Added `docs/frontend/marketops_dsm_workbench_ui_spec.md`.
- Uses existing authenticated APIs: `/v1/signals`, `/v1/signals/{signal_id}`, `/v1/alerts`, and `/v1/insights`.
- Requires MarketOps daily surveillance filters: `app_id=marketops`, `domain=market_data`, `use_case=daily_market_surveillance`, and `detector_id=marketops.dsm.taxonomy_v1`.
- Requires DSM-specific parsing of ticker entities, supporting metrics, artifact proposals, graph candidates, and linked lifecycle records while preserving raw JSON inspection.
- Requires MarketOps nav entry `DSM` at `/marketops/dsm`; Console nav remains unchanged.

Validation requested from frontend-agent:

- `npm test`
- `npm run build`
- `npm audit --audit-level=low`
- `git diff --check`
- Authenticated browser validation of route, query filters, taxonomy filtering, artifact/graph detail rendering, lifecycle links, and mobile/desktop layout.

Follow-up items:

- Frontend-agent implements the spec and records implementation evidence.

## Gate G076: MarketOps DSM Workbench UI Implementation

Timestamp: `2026-07-10T20:08:00Z`

Status: `implemented — local automated validation passed; authenticated browser validation pending operator`

Gate name:

- Implement the MarketOps DSM Workbench UI at `/marketops/dsm` against existing G075 DSM signal output.

Evaluation:

- Verified the G075 `marketops.dsm.taxonomy_v1` payload shape (Python detector + Go signalDTO) matches the spec: `entities[0].external_id`, nested `semantic_evidence[0].artifact` (`subject.symbol`, `artifact_type`), `graph_targets` typed `node_candidate`/`relationship_candidate`, 12 metric keys, 8 signal types. All CONFIRMED; Go persists the fields faithfully as `json.RawMessage`.

Criteria:

- `/marketops/dsm` registered and reachable; MarketOps-only; Console nav unchanged.
- Reuses existing authenticated `/v1/signals`, `/v1/signals/{id}`, `/v1/alerts`, `/v1/insights` with MarketOps daily-surveillance filters.
- Parses DSM nested data defensively (no throws on malformed/missing); taxonomy type filtered client-side.
- Renders metrics, dense table, detail panel (artifact/graph/lifecycle/metrics), and raw JSON sections.

Evidence:

- `web/src/lib/marketopsDsm.ts` + `web/src/lib/marketopsDsm.test.ts`: type-guard helpers (no `JSON.parse`) + 16 unit tests.
- `web/src/routes/MarketOpsDsmRoute.tsx`: filters, metrics, table, detail panel.
- `web/src/router.tsx`, `web/src/apps/appRouting.ts`, `web/src/components/DashboardShell.tsx`: route + DSM nav + `Network` icon.
- `web/src/apps/appRouting.test.ts`: DSM nav + `/marketops/dsm` path detection.

Spec fix applied during implementation:

- Helpers narrow already-deserialized `unknown` values with type guards (never `JSON.parse`, which would throw on parsed arrays); added `graphTargetCounts(signal): { nodes, relationships }` for the detail panel alongside `countGraphTargets` (total).

Validation performed:

- `npm test`: 95 passed. `npm run build`: succeeded. `npm audit --audit-level=low`: 0 vulnerabilities. `git diff --check`: clean.

Validation NOT yet performed (blocks final close):

- Authenticated browser validation from the spec checklist. Requires browser IdP login + `make deploy-web`.

Follow-up items:

- Operator deploys via `make deploy-web` and completes authenticated browser validation.
- Separate in-tree backend artifacts work (`internal/storage/...`, `migrations/000012_*`) is unrelated to this frontend gate and was excluded from this commit.


## Gate G077: MarketOps DSM Artifact Ledger Backend

Timestamp: `2026-07-10T20:24:00Z`

Status: `closed — tests, no-cache Docker build, migration, and live artifact materialization smoke passed`

Gate name:

- Materialize MarketOps DSM artifact proposals as first-class backend ledger records.

Scope:

- Persist deterministic DSM artifact proposals emitted by G074/G075 into `marketops_dsm_artifacts` without changing the signal contract or frontend-agent G076 UI work.

Implementation:

- Added migration `000012_marketops_dsm_artifacts` for a first-class artifact ledger keyed by stable `artifact_marketops_dsm_v1_*` IDs.
- Extracts `semantic_evidence` entries with `type=dsm_artifact_proposal` and embedded `marketops.dsm.signal_artifact.v1` artifacts from MarketOps signal records.
- Materializes artifacts inside `PersistSignalLifecycle` after the signal row is upserted and before transaction commit, preserving retry/idempotency behavior.
- Added repository list/detail methods and gateway endpoints:
  - `GET /v1/marketops/dsm/artifacts`
  - `GET /v1/marketops/dsm/artifacts/{artifact_id}`
- Added API DTOs for artifact payload, semantic evidence, graph targets, supporting metrics, quality issues, and linked signal metadata.

Validation performed:

- Repository unit tests cover artifact extraction, validation, and non-MarketOps ignore behavior.
- Router unit tests cover list/detail responses and query filter propagation.
- Dockerized `gofmt`: passed.
- `go test ./internal/storage/postgres ./internal/api`: passed in Docker.
- `go test ./...`: passed in Docker.

Additional validation performed:

- `python3 scripts/validate_json_schemas.py`: passed.
- `git diff --check`: passed.
- `docker compose build --no-cache gateway signal-persister`: passed and ran `go test ./...` inside the image build.
- `make compose-storage-migrate`: applied `000012_marketops_dsm_artifacts` to live Postgres.
- Recreated `gateway` and `signal-persister` with the new images.
- Published bounded signal `sig_marketops_dsm_taxonomy_v1_g077_artifact_live` to `signalops.local.signal.v1`, partition `1`, offset `4`.
- Postgres verified signal row, materialized artifact `artifact_marketops_dsm_v1_g077_live`, open alert, and active insight.
- Verified artifact metadata: `subject_symbol=AAPL`, artifact type `marketops.dsm.signal_artifact.v1`, signal type `marketops.dsm.pinning_risk`, severity `high`, confidence `0.84`, one event id, and five graph targets.
- Unauthenticated gateway request to `/v1/marketops/dsm/artifacts` returned `401 unauthorized`; authenticated route smoke remains operator-token dependent, while router unit tests cover the handler contract.

Follow-up items:

- G078 can expose first-class artifacts in the frontend DSM workbench or add graph proposal acceptance/storage.


## Gate G078: MarketOps DSM Artifact API Frontend Integration

Timestamp: `2026-07-10T20:48:00Z`

Status: `implemented — frontend tests/build/audit and deployed route smoke passed; authenticated browser validation pending operator`

Gate name:

- Surface first-class MarketOps DSM artifact ledger records in the DSM Workbench.

Scope:

- Wire the G077 artifact API into the existing `/marketops/dsm` workbench without changing backend contracts.

Implementation:

- Added frontend types for `MarketOpsDSMArtifact`, list/detail envelopes, and artifact filters.
- Added API client methods:
  - `listMarketOpsDSMArtifacts`
  - `getMarketOpsDSMArtifact`
- Added React Query keys/hooks for artifact list and detail records.
- Updated `/marketops/dsm` to query artifact ledger records with MarketOps daily-surveillance filters.
- Added `DSM Artifacts` metric tile and table ledger status (`persisted` vs `signal-only`).
- Added a first-class artifact ledger detail panel with artifact id, subject symbol, artifact type, timestamps, event count, and quality issues.
- Preserved signal-derived artifact proposal rendering as fallback when the ledger record is missing or unavailable.

Validation performed:

- API client tests cover artifact list path/query construction, default limit, bearer attachment, artifact-id encoding, envelope parsing, and query keys.
- `npm test`: 100 passed.
- `npm run build`: succeeded.
- `npm audit --audit-level=low`: found 0 vulnerabilities.
- `git diff --check`: passed.

Additional validation performed:

- `make deploy-web`: rebuilt and recreated `signalops-web-1` with the G078 bundle.
- `GET http://localhost:15173/marketops/dsm`: returned HTTP `200`.
- Verified `signalops-web-1` and `signalops-gateway-1` were running after deploy.
- Unauthenticated `/v1/marketops/dsm/artifacts` probe returned expected `401 unauthorized`, preserving auth enforcement.

Validation pending:

- Authenticated browser validation of artifact API calls and rendered ledger panel remains operator-token dependent.


## Gate G078 Closeout Note: Outstanding Operator Validation

Timestamp: `2026-07-10T21:02:00Z`

Status: `documentation-only — code committed and pushed; authenticated browser validation remains operator-token dependent`

Current state:

- G078 implementation is committed, pushed, and deployed locally via `make deploy-web`.
- Automated validation passed: `npm test`, `npm run build`, `npm audit --audit-level=low`, and `git diff --check`.
- Deployed route smoke passed: `/marketops/dsm` returned HTTP `200`.
- Unauthenticated artifact API probe returned expected `401 unauthorized`.

Outstanding:

- Authenticated browser validation with a real operator token:
  - `/marketops/dsm` loads after sign-in.
  - `/v1/marketops/dsm/artifacts` succeeds with bearer auth.
  - DSM Artifacts tile, persisted/signal-only table state, and first-class artifact ledger panel render live data.

Recommended next gate:

- G079: graph proposal acceptance/storage.


## Documentation Organization: Use-Case Folder Structure

Timestamp: `2026-07-11T00:00:00Z`

Status: `documentation-only — structure added`

Scope:

- Add a canonical folder pattern for use-case-specific documentation.

Implementation:

- Added `docs/use_cases/` as the root for documentation scoped by `app_id`, `domain`, and `use_case` metadata.
- Added `docs/use_cases/console/general/` for the default Console use case.
- Added `docs/use_cases/marketops/daily_market_surveillance/` with category folders:
  - `architecture/`
  - `api/`
  - `frontend/`
  - `operations/`
  - `gates/`
- Updated documentation standards to make this layout canonical for future use-case docs.
- Documented the DSM Workbench Ledger semantics: `persisted` means a first-class artifact ledger record exists; signal persistence remains in `signal_ledger`.

Verification performed:

- Reviewed existing metadata/profile references for active use cases.
- Added README files for each new folder so the structure is tracked and discoverable.

Follow-up items:

- Future gates should add use-case-specific docs under `docs/use_cases/{app_id}/{use_case}/` instead of adding loose files at the docs root.
- Existing historical MarketOps specs can remain in `docs/marketops/`; summarize or cross-link into the use-case tree as needed.


## Documentation Organization: MarketOps DSM Persistence Notes

Timestamp: `2026-07-11T00:08:00Z`

Status: `documentation-only — use-case notes added`

Scope:

- Add concrete MarketOps Daily Market Surveillance documentation under the new use-case folder structure.

Implementation:

- Added `architecture/signal_artifact_persistence.md` to explain the relationship between `signal_ledger`, `marketops_dsm_artifacts`, `alert_ledger`, and `insight_ledger`.
- Added `frontend/dsm_workbench_operator_validation.md` to document authenticated DSM Workbench validation and the meaning of `persisted` versus `signal-only`.
- Updated the MarketOps daily-surveillance README and category READMEs to link to the new notes.

Key clarification:

- `persisted` in the DSM Workbench Ledger column means an artifact ledger record exists in `marketops_dsm_artifacts`; the signal itself is already persisted separately in `signal_ledger`.

Verification performed:

- Documentation readback and cross-link review.

Follow-up items:

- Add G079 graph proposal documentation under `architecture/` and `api/` when that gate starts.


## Gate G079 Planning: Graph Proposal Acceptance

Timestamp: `2026-07-11T00:22:00Z`

Status: `documentation-only — proposed gate contract added`

Scope:

- Define the next MarketOps Daily Market Surveillance backend boundary after first-class DSM artifact persistence and frontend artifact visibility.

Implementation:

- Added `architecture/graph_proposal_acceptance.md` to describe the proposed graph proposal ledger, status lifecycle, identity rules, extraction rules, non-goals, and validation expectations.
- Added `api/graph_proposal_api.md` to document proposed graph proposal list/detail/decision endpoints under existing `/v1/*` gateway conventions.
- Added `gates/G079_graph_proposal_acceptance.md` to capture gate goal, inputs, deliverables, acceptance criteria, and deferred work.
- Updated use-case README files to cross-link the new G079 notes.

Key clarification:

- G079 should store reviewable graph proposals derived from persisted DSM artifacts. It should not mutate an external graph database.

Verification performed:

- Documentation readback completed.
- `git diff --check`: passed.

Follow-up items:

- Implement the G079 migration, storage extraction/upsert, APIs, and tests.


## Gate G079: Graph Proposal Acceptance Backend

Timestamp: `2026-07-11T00:46:00Z`

Status: `implemented — backend storage/API complete`

Scope:

- Materialize MarketOps DSM graph target candidates as first-class reviewable proposal records derived from persisted DSM artifacts.

Implementation:

- Added migration `000013_marketops_dsm_graph_proposals` with proposal identity, candidate identity fields, JSON properties/raw candidate payloads, review status, reviewer metadata, and indexes for tenant/time, artifact, signal, status, and symbol lookups.
- Added deterministic proposal id generation from artifact id, signal id, candidate type, and node/relationship identity.
- Added graph proposal extraction from `marketops_dsm_artifacts.graph_targets`, ignoring malformed candidates without failing signal persistence.
- Wired graph proposal upserts into `PersistSignalLifecycle` after DSM artifact upserts, preserving existing proposal review status on replay.
- Added list/detail/decision storage APIs and gateway routes under `/v1/marketops/dsm/graph-proposals`.
- Added tests for extraction, stable ids, malformed candidates, route filters, detail responses, and decision status validation.

Validation performed:

- Containerized `gofmt` with `golang:1.24`: passed.
- Containerized targeted Go tests for `./internal/storage/postgres ./internal/api`: passed.
- Containerized full Go test suite `go test ./...`: passed.
- `git diff --check`: passed.
- Docker build validation for `signal-persister`: passed.

Non-goals preserved:

- No production graph database mutation.
- No frontend graph editing or graph visualization.
- No replacement of `signal_ledger` or `marketops_dsm_artifacts`.

Follow-up items:

- Apply migration and run live persistence validation against the G077 smoke artifact path.
- Hand frontend-agent a read-only graph proposal visibility spec only after the backend endpoint shape is accepted.


## Gate G079 Live Closeout: Graph Proposal Persistence

Timestamp: `2026-07-11T17:50:00Z`

Status: `live validation passed with authenticated API smoke pending operator token`

Scope:

- Apply and validate the G079 graph proposal backend in the running local stack.

Implementation follow-up:

- Fixed live nil-label handling for graph proposal candidates. Relationship candidates often omit `labels`; extraction now normalizes missing labels to an empty slice so Postgres receives an empty array instead of null.
- Added a regression assertion that relationship graph proposals without labels do not produce nil labels.

Validation performed:

- Applied migration `000013_marketops_dsm_graph_proposals`.
- Rebuilt/recreated `gateway` and `signal-persister`; force-recreated `web` after gateway replacement.
- Containerized `gofmt`: passed.
- Containerized targeted Go tests for `./internal/storage/postgres ./internal/api`: passed.
- Containerized full Go test suite `go test ./...`: passed.
- Compose image build for `gateway signal-persister`: passed and ran Dockerfile `go test ./...`.
- Direct and web-proxied gateway health returned `200`.
- Published bounded smoke signal `sig_marketops_dsm_taxonomy_v1_g079_graph_live` to `signalops.local.signal.v1`, partition `2`, offset `4`.
- `signal-persister` persisted the smoke signal.
- Postgres verified the signal, artifact, alert, insight, and five graph proposal rows.
- Graph proposals materialized as three node candidates and two relationship candidates, all status `proposed`.
- Unauthenticated graph proposal API probe returned expected `401 unauthorized`.

Residual risk:

- Positive authenticated API route validation requires a real operator bearer token.
- Historical signal-persister lag remains on older partitions, but the G079 smoke partition committed through the published offset with lag `0`.

Follow-up items:

- Run authenticated graph proposal API list/detail/decision smoke with an operator token.
- Decide whether to drain or separately audit the historical signal-persister lag on partitions `0` and `1`.


## Frontend-Agent Spec: G079 Graph Proposal Visibility

Timestamp: `2026-07-11T18:02:00Z`

Status: `documentation-only — frontend handoff ready`

Scope:

- Define a no-scope-creep frontend-agent task for read-only graph proposal visibility in the existing MarketOps DSM Workbench.

Implementation:

- Added `docs/frontend/marketops_graph_proposals_readonly_spec.md`.
- Cross-linked the spec from the MarketOps Daily Market Surveillance frontend use-case README.

Key guardrails:

- No graph editing.
- No graph canvas or visualization.
- No decision mutation UI.
- No graph database writes.
- Preserve existing `persisted` versus `signal-only` artifact semantics.

Validation performed:

- Documentation readback completed.
- `git diff --check`: passed.


## Gate G079 Frontend Live Validation: Graph Proposal Read-Only UI

Timestamp: `2026-07-11T23:14:00Z`

Status: `live validation passed — read-only UI verified against persisted G079 proposals`

Scope:

- Implement and validate the G079 read-only graph proposal ledger in the existing MarketOps DSM Workbench per `docs/frontend/marketops_graph_proposals_readonly_spec.md`.

Implementation:

- Added `MarketOpsDSMGraphProposal` types, API client methods, React Query hooks, defensive summary helpers, and a read-only Graph Proposal Ledger section in `MarketOpsDsmRoute.tsx`.
- Relabeled the raw signal `graph_targets` box as "Graph Targets (raw evidence)" and preserved it as source evidence alongside the persisted ledger.
- Documented the component-test de-scope in the spec (project ships no render harness; testable concerns covered by API-client tests + pure helper unit tests).

Key guardrails preserved:

- No decision/mutation UI. No graph canvas/editing. No graph database writes.
- No new top-level route. Existing `persisted` versus `signal-only` artifact semantics unchanged.

Validation performed:

- `cd web && npm test`: 118 passed (6 new graph-proposal API client tests + extended `marketopsDsm` helper tests).
- `cd web && npm run build`: TypeScript + Vite production build passed.
- `cd web && npm audit --audit-level=low`: 0 vulnerabilities.
- Static no-mutation check: `grep` of `web/src` confirms zero references to the decision endpoint; the client issues list + detail `GET`s only.
- Live UI: selecting `sig_marketops_dsm_taxonomy_v1_g079_graph_live` renders `Graph Proposal Ledger — Total 5 · Nodes 3 · Relationships 2 · Status: proposed 5`.
- Live UI: expanding node candidate `graphprop_marketops_dsm_v1_ba33f478a58719c78203d6e0` (`ticker:AAPL`, conf `0.84`) shows the read-only detail block with proposal/artifact/signal ids; no accept/reject controls present.
- Confirmed G078 `persisted`/`signal-only` ledger labels and the raw graph-targets evidence view remain intact.

Residual risk:

- Positive authenticated API/UI validation still requires an operator bearer token; the running local gateway currently has auth disabled (curl returns `200`, not the backend closeout's `401`), so the auth-enabled path remains operator-token dependent.
- IdP token endpoint is Imperva-gated for non-browser clients, so browser login is required for the authenticated smoke.

Follow-up items:

- Run the authenticated graph-proposal API/UI smoke with an operator token via browser login.
- Re-enable gateway auth on the local stack to reproduce the `401` unauthenticated probe documented in the backend closeout.


## Gate G079 Authenticated API Smoke

Timestamp: `2026-07-11T23:29:00Z`

Status: `closed — authenticated graph proposal API validation passed`

Scope:

- Validate the G079 graph proposal API with a real operator bearer token.

Validation performed:

- Authenticated graph proposal list returned all five proposals for `sig_marketops_dsm_taxonomy_v1_g079_graph_live`.
- Authenticated graph proposal detail returned relationship proposal `graphprop_marketops_dsm_v1_ebb85656b5b3c82105eb8fe8`.
- Authenticated decision mutation to `accepted` succeeded and derived actor `lukeb` from the token.
- Authenticated restore mutation to `proposed` succeeded and derived actor `lukeb` from the token.
- Postgres confirmed all five smoke proposals are currently status `proposed`.

Result:

- G079 backend, live persistence, and authenticated API validation are complete.

Residual state:

- Historical signal-persister lag on older partitions was closed in the follow-up operational cleanup documented below.


## Historical Signal-Persister Lag Closure

Timestamp: `2026-07-12T00:53:00Z`

Status: `closed — historical partition lag backfilled and offsets reconciled`

Scope:

- Audit and close the historical `signalops.signal-persister.v1` lag on older `signalops.local.signal.v1` partitions `0` and `1` without losing MarketOps DSM ledger data.

Implementation:

- Added `cmd/signal-backfill` for operator-controlled JSONL replay through the existing signal lifecycle processor and Postgres repository.
- Captured and audited the lagged records from partition `0` offsets `5-12` and partition `1` offsets `2-4`.
- Backfilled 11 bounded MarketOps DSM smoke/replay records from G072-G077.
- Materialized missing first-class DSM artifacts and graph proposals for the historical records.

Validation performed:

- Dry-run decoded all 11 payloads before persistence.
- Live backfill completed for partition `0` offsets `5-12` and partition `1` offsets `2-4`.
- Postgres verified `11` distinct signal rows, `11` distinct `marketops_dsm_artifacts` rows, and `55` `marketops_dsm_graph_proposals` rows for the audited set.
- Consumer offsets were advanced after backfill: partition `0` `5 -> 13`, partition `1` `2 -> 5`, partition `2` unchanged at `5`.
- Final Redpanda group state: `signalops.signal-persister.v1` Stable with one member and total lag `0`.

Result:

- The historical persister lag is closed. No audited MarketOps DSM signal, artifact, or graph proposal data was skipped.


## Gate G079 Closeout

Timestamp: `2026-07-12T01:05:00Z`

Status: `closed`

Scope:

- Reconcile G079 after all backend, frontend, authentication, and operational cleanup work completed.

Closeout summary:

- Backend graph proposal acceptance/storage was implemented and validated.
- Read-only frontend graph proposal visibility was implemented and live-validated in the DSM Workbench.
- Authenticated graph proposal API list/detail/decision smoke passed with an operator bearer token.
- Historical signal-persister lag on older partitions `0` and `1` was audited, backfilled, and reconciled to total lag `0`.
- G079 documentation now reflects closed status and the gate index no longer lists G079 as the recommended next gate.

Result:

- G079 is fully closed. The next recommended MarketOps boundary is G080: operator graph proposal review workflow, without production graph database writes.


## Gate G080: Operator Graph Proposal Review Workflow

Timestamp: `2026-07-12T01:12:00Z`

Status: `implemented and locally validated`

Scope:

- Add bounded operator review actions for first-class MarketOps DSM graph proposals in the existing DSM Workbench.

Implementation:

- Added graph proposal decision request/options types.
- Added frontend API client support for `POST /v1/marketops/dsm/graph-proposals/{proposal_id}/decision`.
- Added a React Query mutation hook that refreshes graph proposal list/detail caches after decisions.
- Added inline review controls for accepted, rejected, superseded, and proposed statuses with optional review notes.
- Kept production graph writes, graph visualization, and graph editing out of scope.

Validation performed:

- `cd web && npm test`: 120 passed.
- `cd web && npm test -- src/api/marketopsGraphProposals.test.ts`: 8 passed.
- `cd web && npm run build`: passed.
- `cd web && npm audit --audit-level=low`: 0 vulnerabilities.

Result:

- G080 is implemented as a review-state workflow only. Accepted proposals remain persisted review records and do not mutate a production graph database.


## Gate G081: Back-Test Substrate Specification

Timestamp: `2026-07-12T01:25:00Z`

Status: `specification proposed — awaiting review`

Scope:

- Define a MarketOps back-test substrate before implementation.
- Focus the first version on policy calibration from historical normalized events.

Implementation:

- Added a G081 gate note for the back-test substrate.
- Added architecture documentation distinguishing back-tests from operational replay.
- Added operations documentation with first smoke scenario, safety controls, and go/no-go checklist.
- Updated the MarketOps gate index to point to G081 review and G082 thin MVP implementation after approval.

Key decisions captured:

- First objective: policy calibration.
- First source: historical normalized MarketOps events.
- First storage model: isolated back-test ledgers.
- First policy model: deterministic policy pack, not ML.
- Production graph writes remain deferred.

Validation performed:

- Documentation readback completed.
- `git diff --check`: passed.

Result:

- G081 is ready for operator/architecture review. No implementation has started.

## Gate G081 Spec Refinement

Timestamp: `2026-07-12T03:05:00Z`

Status: `specification proposed — refinement pending review`

Scope:

- Improve reviewability and traceability of the G081 back-test substrate specification before G082.

Implementation:

- Added Documentation Links sections to the G081 gate, architecture, and operations notes so the three review documents cross-reference each other.
- Indexed the new notes in the use-case, architecture, and operations READMEs.
- Grounded the replay contrast with the replay API route and tables instead of an abstract subsystem reference.
- Reframed detector id/version as a run parameter rather than a `normalized_event_ledger` input filter, and noted that symbol/universe filters resolve payload-level via `marketops_asset_universe`.
- Canonicalized back-test policy recommendation tokens to snake_case and aligned the conceptual outputs with the isolated storage model.

Validation performed:

- Documentation readback completed across the seven modified files.
- Cross-link targets verified to exist.
- `git diff --check`: passed.

Result:

- G081 remains ready for operator/architecture review. No implementation has started.


## Gate G081: Back-Test Substrate MVP Implementation

Timestamp: `2026-07-12T04:20:00Z`

Status: `implemented`

Scope:

- Implement isolated MarketOps back-test storage and read APIs.
- Run historical normalized MarketOps events through the existing Python detector path.
- Persist generated signals, artifacts, graph proposals, and deterministic policy recommendations under a back-test run id only.

Implementation:

- Added migration `000014_marketops_backtest_substrate`.
- Added `cmd/marketops-backtest` and `python/signalops_workers/backtest_detector.py`.
- Added `internal/marketops/dsm` extraction and policy helpers.
- Added read-only `/v1/marketops/backtests` routes and tests.

Validation performed:

- Full Go suite passed in Docker.
- Full Python suite passed with `PYTHONPATH=python`.
- JSON schema validation passed.
- Docker builds passed for `marketops-backtest` and `python-worker` targets.

Result:

- G081 is implemented as a bounded MVP substrate. Operational replay and production graph materialization remain separate/deferred.


## Gate G081: Live Smoke Validation

Timestamp: `2026-07-12T03:45:00Z`

Status: `validated — live smoke passed`

Scope:

- Apply the G081 relational migration to the running local stack.
- Run a bounded back-test from historical normalized MarketOps data.
- Verify isolated persistence and authenticated read APIs.

Implementation follow-up:

- Fixed `marketops_backtest_runs.error_message` persistence to use an empty string for non-error runs, matching the NOT NULL schema.

Validation performed:

- Applied migration `000014_marketops_backtest_substrate` with `make compose-storage-migrate`.
- Ran back-test `bt-g081-smoke-20260712` over `tenant-local/src-massive`, dataset `equity_eod_prices`, symbol `SPY`, window `2026-07-09T00:00:00Z` to `2026-07-10T00:00:00Z`.
- Persisted isolated rows: runs `1`, signals `1`, artifacts `1`, graph proposals `5`, policy results `5`.
- Verified production ledger counts did not increase: `signal_ledger=19`, `alert_ledger=18`, `insight_ledger=19`, `marketops_dsm_artifacts=12`, `marketops_dsm_graph_proposals=60`.
- Rebuilt `gateway` and validated authenticated `/v1/marketops/backtests` list/detail/signals/graph-proposals reads.

Result:

- G081 is live-smoke validated. The next gate can focus on operator ergonomics or broader calibration metrics.


## Gate G081: API-Created Back-Test Runs

Timestamp: `2026-07-12T04:25:00Z`

Status: `validated — API creation smoke passed`

Scope:

- Add an operator API for creating bounded MarketOps back-test runs without using the CLI.
- Reuse the same isolated runner semantics as `cmd/marketops-backtest`.
- Preserve the no-production-mutation boundary.

Implementation:

- Added `POST /v1/marketops/backtests` with request validation, bounded `max_records`, symbol filters, detector parameters, and `201 Created` responses containing the run row and metrics.
- Refactored the CLI execution path into `internal/marketops/backtest` for shared gateway and CLI execution.
- Updated the gateway image to include Python detector adapter assets required by API-created runs.
- Added API tests for successful run creation and invalid window rejection.

Validation performed:

- Targeted Go tests passed for `./internal/marketops/backtest ./cmd/marketops-backtest ./internal/api`.
- Full Go suite passed in Docker.
- Full Python suite passed with `PYTHONPATH=python`.
- JSON schema validation passed.
- Local API smoke run `bt-g081-api-smoke-20260712` succeeded with scanned `1`, signals `1`, artifacts `1`, graph proposals `5`, policy results `5`, and `auto_accept_candidate=5`.
- Production ledger counts remained unchanged after the API smoke: `signal_ledger=19`, `alert_ledger=18`, `insight_ledger=19`, `marketops_dsm_artifacts=12`, `marketops_dsm_graph_proposals=60`.
- The supplied bearer could not validate the authenticated POST path because it was expired; endpoint execution was smoke-tested with auth temporarily disabled locally, and the gateway was restored to `SIGNALOPS_AUTH_ENABLED=true` afterward.

Result:

- G081 now supports both CLI-created and API-created isolated back-test runs. A fresh operator bearer is still needed for a final authenticated POST smoke in the running local stack.


## Gate G081: Authenticated API Creation Smoke

Timestamp: `2026-07-12T04:35:00Z`

Status: `validated — authenticated POST smoke passed`

Scope:

- Close the remaining authenticated validation caveat for `POST /v1/marketops/backtests`.
- Confirm the auth-enabled gateway accepts an operator bearer and creates an isolated MarketOps back-test run.

Validation performed:

- Verified the pasted bearer was missing the leading JWT header character; after restoring the expected `eyJ...` prefix, authenticated reads succeeded.
- Ran authenticated API-created back-test `bt-g081-auth-api-smoke-20260712` for `tenant-local/src-massive`, dataset `equity_eod_prices`, symbol `SPY`, window `2026-07-09T00:00:00Z` to `2026-07-10T00:00:00Z`.
- Result: status `succeeded`, scanned `1`, signals `1`, artifacts `1`, graph proposals `5`, policy results `5`, recommendation counts `auto_accept_candidate=5`.
- Isolated back-test totals after smoke: runs `3`, signals `3`, artifacts `3`, graph proposals `15`, policy results `15`.
- Production ledger counts remained unchanged: `signal_ledger=19`, `alert_ledger=18`, `insight_ledger=19`, `marketops_dsm_artifacts=12`, `marketops_dsm_graph_proposals=60`.

Result:

- G081 API creation is validated under the normal auth-enabled gateway.


## Gate G081: Frontend Back-Test UI Specification

Timestamp: `2026-07-12T04:45:00Z`

Status: `specification ready for frontend-agent`

Scope:

- Define the frontend follow-up for the implemented G081 MarketOps back-test substrate.
- Keep the scope limited to isolated back-test run creation and review.
- Avoid production graph mutation, replay semantics, policy promotion, ML training, and async job orchestration.

Result:

- Frontend-agent handoff saved at `docs/frontend/marketops_backtests_ui_spec.md` and indexed from the MarketOps frontend README.


## Gate G081: Back-Test UI Zero-Input State

Timestamp: `2026-07-12T05:32:00Z`

Status: `validated — UI empty-success state added`

Scope:

- Clarify the operator experience for successful back-test runs that scan zero normalized events.
- Distinguish empty input filters from execution failures.

Validation performed:

- Added a tested `isZeroInputBacktest` helper.
- Rendered a MarketOps back-test detail notice when `status=succeeded` and `metrics.scanned=0`.
- Full frontend tests and production build passed.
- Rebuilt the local web container and verified the served bundle contains the new operator text.

Result:

- Operators now get actionable guidance when a run completes successfully but no normalized events matched the selected filters/window.


## Gate G081: UI Known-Good Back-Test Closeout

Timestamp: `2026-07-12T05:36:00Z`

Status: `validated — known-good UI path passed via web proxy`

Scope:

- Validate that the MarketOps back-tests UI backend path can create and inspect a non-empty run after adding the zero-input state.
- Use the same-origin web proxy route that the browser uses.

Validation performed:

- Created `bt-g081-ui-closeout-spy-20260712` through `http://localhost:15173/v1/marketops/backtests` with `SPY`, source `src-massive`, dataset `equity_eod_prices`, and window `2026-07-09T00:00:00Z` to `2026-07-10T00:00:00Z`.
- Result: status `succeeded`, scanned `1`, signals `1`, artifacts `1`, graph proposals `5`, policy results `5`, recommendation counts `auto_accept_candidate=5`.
- Verified run detail, generated back-test signals, and generated back-test graph proposals through the web proxy.
- Confirmed production ledger rows newest at `2026-07-12T05:06:19Z`, before the `05:34` closeout run, so this validation did not mutate production ledgers.
- Restored gateway/web to auth-enabled local configuration and verified health.

Result:

- G081 UI closeout has both zero-input and known-good non-empty run validation.


## Gate G082: Back-Test Calibration Summary MVP

Timestamp: `2026-07-12T05:50:00Z`

Status: `validated — frontend run-list comparison added`

Scope:

- Start the next MarketOps back-test gate with an operator-facing calibration summary over the currently listed runs.
- Keep the first slice frontend-only and filter-scoped so it does not commit to backend aggregate storage or calibration policy persistence.
- Surface zero-input rate, signal yield, policy-results-per-signal, dominant recommendation, and recommendation mix.

Validation performed:

- Added tested run comparison aggregation in `web/src/lib/marketopsBacktests.ts`.
- Rendered the `Calibration Summary` panel in `web/src/routes/MarketOpsBacktestsRoute.tsx`.
- Targeted frontend tests, full frontend tests, production build, and `git diff --check` passed.
- Rebuilt the local web container and verified the served bundle contains `Calibration Summary`.

Result:

- G082 has a first validated calibration/comparison surface for back-test runs without changing backend APIs or production ledgers.


## Gate G082: Persisted Back-Test Calibration Summaries

Timestamp: `2026-07-12T06:02:00Z`

Status: `validated — backend summary substrate implemented`

Scope:

- Add first-class persisted calibration summary snapshots over isolated MarketOps back-test runs.
- Create summaries from existing back-test run metrics rather than production ledgers.
- Expose create/list/detail APIs at `/v1/marketops/backtest-calibration-summaries`.
- Avoid policy promotion, named baseline management, ML training, and frontend workflow expansion in this slice.

Validation performed:

- Added migration `000015_marketops_backtest_calibration_summaries` and applied it locally with `make compose-storage-migrate`.
- Added storage records, repository methods, API DTOs, summary aggregation logic, and router tests.
- Targeted Go tests, full Go tests, JSON schema validation, and gateway Docker build all passed.
- Live unauthenticated probe returned `401 unauthorized`, and Postgres verified the new summary table exists.

Result:

- G082 now has a durable backend substrate for comparing/calibrating back-test run sets without mutating production ledgers.


## Gate G082: Back-Test Calibration Summary UI Wiring

Timestamp: `2026-07-12T06:08:00Z`

Status: `validated — persisted summary UI wired`

Scope:

- Connect the MarketOps Back-Tests UI to the G082 persisted calibration summary API.
- Let operators create a stored snapshot from the current run filters and review recent persisted snapshots.
- Keep the route read-oriented and avoid baseline promotion, threshold editing, policy deployment, model training, or graph writeback.

Validation performed:

- Added frontend API types, client methods, React Query hooks, and tests for calibration summary list/detail/create behavior.
- Rendered `Persisted Calibration Snapshots` in `/marketops/backtests`.
- Targeted frontend tests, full frontend tests, production build, and `git diff --check` passed.
- Rebuilt the local web container and verified the served bundle contains the new panel text.

Result:

- G082 has end-to-end backend plus frontend access to persisted back-test calibration snapshots, still isolated from production ledgers and policy promotion flows.


## Gate G082: Authenticated Calibration Summary Smoke

Timestamp: `2026-07-12T06:28:00Z`

Status: `validated — authenticated create/list/detail passed`

Scope:

- Close the remaining authenticated live validation item for persisted MarketOps back-test calibration summaries.
- Use the dedicated CLI OIDC client environment variables to generate a short-lived bearer without browser interaction.

Validation performed:

- Generated a bearer in-memory from the configured CLI OIDC client.
- Created `btcal-g082-auth-smoke-20260712062745` through `POST /v1/marketops/backtest-calibration-summaries` with `tenant_id=tenant-local`, detector `marketops.dsm.taxonomy_v1`, status `succeeded`, and limit `50`.
- Listed calibration summaries and fetched the created summary by id through authenticated GET requests.
- Result metrics: run count `8`, zero-input count `3`, scanned `5`, signals `5`, policy results `25`, dominant recommendation `auto_accept_candidate` at share `1`.
- Token material was not printed or committed; temporary auth/API response files were removed.

Result:

- G082 persisted calibration summary backend and UI now have authenticated API closeout coverage.


## Gate G083: Back-Test Baselines And Evaluation Specification

Timestamp: `2026-07-12T06:36:00Z`

Status: `specification proposed`

Scope:

- Define the next MarketOps back-test gate after G082 closeout.
- Specify named calibration baselines pointing to immutable G082 summaries.
- Specify stored comparison records with deterministic summary-level deltas and advisory recommendations.
- Specify how G080 operator graph-proposal decisions should become evaluation labels only after normalization through a separate label substrate.

Result:

- Specification saved at `docs/use_cases/marketops/daily_market_surveillance/gates/G083_backtest_baselines_and_evaluation.md`.
- Gate index and back-test architecture docs now reference the G083 direction.
- Implementation remains pending operator review/approval.


## Gate G083: Back-Test Baselines And Stored Comparisons

Timestamp: `2026-07-12T07:05:00Z`

Status: `validated — backend/API baseline comparison substrate implemented`

Scope:

- Add durable MarketOps back-test calibration baselines over immutable G082 calibration summaries.
- Add stored baseline-to-candidate comparison records with deterministic aggregate deltas and conservative advisory recommendations.
- Expose create/list/detail APIs for baselines and comparisons without mutating production ledgers or deploying policies.

Validation performed:

- Added migration `000016_marketops_backtest_calibration_baselines`.
- Added storage records, repository methods, API DTOs, comparison logic, and router tests.
- Targeted API/Postgres tests and full Go tests passed.
- JSON schema validation and `git diff --check` passed.
- Applied migration `000016_marketops_backtest_calibration_baselines`.
- Rebuilt the gateway container; Docker build ran `go test ./...`.
- Authenticated smoke generated a bearer in-memory and validated baseline create/list/detail as HTTP `201/200/200`.
- Authenticated smoke validated comparison create/list/detail as HTTP `201/200/200`; same-summary comparison returned `neutral_candidate`.
- Token material was not printed or committed; temporary auth/API response files were removed.

Result:

- G083 now has a validated backend baseline/comparison substrate ready for frontend-agent wiring and later label-aware scoring.


## Gate G083: Frontend-Agent Baseline Comparison Spec

Timestamp: `2026-07-12T07:22:00Z`

Status: `specification ready for frontend-agent`

Scope:

- Specify frontend wiring for the G083 baseline and stored comparison APIs.
- Define API client methods, query hooks, UI placement, required user flow, tests, and acceptance criteria.
- Keep the implementation limited to baseline/comparison visibility and creation on `/marketops/backtests`.

Result:

- Specification saved at `docs/frontend/marketops_backtest_baselines_ui_spec.md`.
- Existing MarketOps back-tests UI spec now links to the G083 addendum.


## Gate G083: Frontend Baseline Comparison Implementation Closeout

Timestamp: `2026-07-12T19:50:00Z`

Status: `validated — frontend baseline/comparison UI implemented`

Scope:

- Close the loop on the frontend-agent implementation for G083 baseline and stored comparison APIs.
- Validate API client/query hook tests, full frontend tests, production build, local web rebuild, and route availability.
- Keep scope limited to baseline/comparison visibility and creation; label-aware scoring and promotion remain future gates.

Validation performed:

- Targeted MarketOps back-test frontend tests passed: 41 tests.
- Full frontend suite passed: 162 tests.
- Production web build passed.
- Local web container rebuild passed.
- `/marketops/backtests` served HTTP `200` and the built bundle contains the G083 baseline/comparison UI text.

Result:

- G083 backend plus frontend wiring is validated locally. Remaining work is follow-on gate planning for label extraction/label-aware scoring, not G083 closure.


## Gate G084: Evaluation Label Sync

Timestamp: `2026-07-12T20:20:00Z`

Status: `validated — backend/API label sync implemented`

Scope:

- Normalize reviewed G080 MarketOps DSM graph proposal decisions into evaluation labels.
- Store labels in `marketops_backtest_evaluation_labels` with idempotent source proposal/version semantics.
- Expose sync/list/detail APIs without changing the graph proposal decision endpoint.

Validation performed:

- Added migration `000017_marketops_backtest_evaluation_labels`.
- Added storage records, Postgres methods, API DTOs, sync helper logic, router handlers, and route tests.
- Targeted API/Postgres tests and full Go tests passed.
- JSON schema validation and `git diff --check` passed.
- Applied the migration and rebuilt the gateway; Docker build ran `go test ./...`.
- Authenticated smoke synced two accepted graph proposals into `positive` labels, then validated list/detail.
- Token material was not printed or committed; temporary files were removed.

Result:

- G084 provides the durable label substrate needed for a later label-aware scoring/evaluation gate.


## Gate G085: Label-Aware Back-Test Evaluation

Timestamp: `2026-07-12T20:35:00Z`

Status: `validated — backend/API scoring substrate implemented`

Scope:

- Score MarketOps back-test policy recommendations against G084 evaluation labels.
- Persist aggregate scoring snapshots in `marketops_backtest_evaluations`.
- Expose create/list/detail APIs without changing back-test execution, graph proposal decisions, or policy behavior.

Validation performed:

- Added migration `000018_marketops_backtest_evaluations`.
- Added storage records, Postgres methods, API DTOs, scoring helper logic, router handlers, and route tests.
- Targeted API/Postgres tests and full Go tests passed.
- JSON schema validation and `git diff --check` passed.
- Applied the migration and rebuilt the gateway; Docker build ran `go test ./...`.
- Authenticated smoke validated evaluation create/list/detail as HTTP `201/200/200`.
- The smoke run had no matched labels, yielding `needs_more_data`, which is the expected conservative outcome for zero label coverage.

Result:

- G085 provides the durable label-aware scoring substrate. The next useful work is frontend exposure or generating/replaying runs with matched label coverage for meaningful precision/recall review.


## Gate G085: Matched-Label Validation And Frontend Spec

Timestamp: `2026-07-12T20:55:00Z`

Status: `validated — matched-label smoke passed; frontend spec ready`

Scope:

- Create a local matched-label validation scenario so G085 returns meaningful precision/recall rather than `needs_more_data`.
- Document frontend-agent requirements for rendering G085 label-aware back-test evaluations.

Validation performed:

- Accepted five production graph proposals whose graph fact keys match back-test run `bt-g081-ui-closeout-spy-20260712`.
- Synced labels from accepted graph proposal decisions.
- Re-ran G085 evaluation and received precision `1`, recall `1`, label coverage `1`, and recommendation `improvement_candidate`.
- Wrote frontend-agent specification at `docs/frontend/marketops_backtest_evaluations_ui_spec.md`.

Result:

- G085 has both zero-coverage and matched-label validation coverage. Frontend-agent can now wire evaluation visibility without backend changes.


## Gate G085: Frontend Evaluation UI Closeout

Timestamp: `2026-07-12T21:05:00Z`

Status: `validated — frontend label-aware evaluation UI implemented`

Scope:

- Close the loop on frontend-agent implementation for G085 label-aware back-test evaluations.
- Validate API client/query hook tests, full frontend tests, production build, local web rebuild, route availability, and built UI text.
- Keep scope limited to evaluation visibility and creation; label sync controls, graph proposal decisions, policy promotion, graph writeback, and threshold editing remain out of scope.

Validation performed:

- Targeted MarketOps back-test frontend tests passed: 54 tests.
- Full frontend suite passed: 175 tests.
- Production web build passed.
- Local web container rebuild passed.
- `/marketops/backtests` served HTTP `200` and the built bundle contains the G085 evaluation UI text.

Result:

- G085 backend plus frontend visibility is validated locally. Remaining work is broader operator workflow and product decisions around promotion, not G085 closure.


## Gate G086: Calibration Promotion Review Specification

Timestamp: `2026-07-12T21:18:00Z`

Status: `specification proposed`

Scope:

- Define the next gate after G085 label-aware evaluation.
- Specify promotion candidates that package G083/G085 evidence into auditable operator-reviewed records.
- Define readiness rules and decision states without runtime side effects.

Result:

- Specification saved at `docs/use_cases/marketops/daily_market_surveillance/gates/G086_calibration_promotion_review.md`.
- Gate README and back-test architecture docs now reference the G086 direction.


## Gate G086: Calibration Promotion Review Backend

Timestamp: `2026-07-12T21:42:00Z`

Status: `validated — backend/API promotion review implemented`

Scope:

- Add durable promotion candidate records over G083/G085 evidence.
- Add deterministic readiness checks and compact evidence snapshots.
- Expose create/list/detail and decision APIs without runtime side effects.

Validation performed:

- Added migration `000019_marketops_backtest_promotion_candidates`.
- Added storage records, repository methods, API DTOs, readiness logic, and router tests.
- Targeted API/Postgres tests, full Go tests, and JSON schema validation passed.
- Applied migration `000019_marketops_backtest_promotion_candidates`.
- Rebuilt the gateway container; Docker build ran `go test ./...`.
- Authenticated smoke validated create/list/detail/decision as HTTP `201/200/200/200`.
- Smoke candidate `btpromo-g086-auth-smoke-20260712214200` returned readiness `ready_for_review`; decision changed candidate status to `deferred`.
- Token material and temporary API response files were removed.

Result:

- G086 now has a validated backend promotion review substrate. Next work is frontend-agent specification and UI wiring.


## Gate G086: Calibration Promotion Review Backend

Timestamp: `2026-07-12T21:42:00Z`

Status: `validated — backend/API promotion review implemented`

Scope:

- Add durable promotion candidate records over G083/G085 evidence.
- Add deterministic readiness checks and compact evidence snapshots.
- Expose create/list/detail and decision APIs without runtime side effects.

Validation performed:

- Added migration `000019_marketops_backtest_promotion_candidates`.
- Added storage records, repository methods, API DTOs, readiness logic, and router tests.
- Targeted API/Postgres tests, full Go tests, and JSON schema validation passed.
- Applied migration `000019_marketops_backtest_promotion_candidates`.
- Rebuilt the gateway container; Docker build ran `go test ./...`.
- Authenticated smoke validated create/list/detail/decision as HTTP `201/200/200/200`.
- Smoke candidate `btpromo-g086-auth-smoke-20260712214200` returned readiness `ready_for_review`; decision changed candidate status to `deferred`.
- Token material and temporary API response files were removed.

Result:

- G086 now has a validated backend promotion review substrate. Next work is frontend-agent specification and UI wiring.
