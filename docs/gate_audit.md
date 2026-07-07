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
