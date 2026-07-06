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
