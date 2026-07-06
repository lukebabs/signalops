# Syncratic SignalOps Infrastructure Specification

## Purpose

SignalOps is a new subsystem that enables Syncratic to ingest, process,
and correlate continuous data streams such as market data, telemetry, CRM
events, security events, IoT signals, and operational metrics without
modifying the existing document ingestion pipeline.

SignalOps MUST follow Syncratic's non-interference principle. Existing
document ingestion, retrieval, graph construction, and Ask workflows must
continue to operate unchanged.

## Objectives

- Add streaming/event ingestion.
- Build temporal knowledge from continuously arriving events.
- Produce Event Artifacts consumable by the Syncratic Engine.
- Continuously propose knowledge graph updates.
- Publish Insight Candidates for downstream evaluation.
- Remain independently deployable and independently scalable.

## Design Principles

- Separate subsystem.
- API-first.
- Event-driven.
- Asynchronous.
- Idempotent processing.
- Replayable.
- Multi-tenant.
- Observable.
- Horizontally scalable.

## System Boundaries

SignalOps is deployed as a separate subsystem with its own service,
workers, broker configuration, storage connections, Helm chart, namespace,
and scaling rules.

SignalOps MAY ingest and process streaming data independently, but it MUST
communicate with the existing Syncratic Engine only through extension
contracts. SignalOps MUST NOT embed stream processing inside the Engine and
MUST NOT directly modify existing document ingestion, retrieval, graph
construction, or Ask code paths.

The Syncratic Engine remains the authority for validating and applying
knowledge graph mutations. SignalOps proposes graph changes; the Engine
accepts, rejects, or evaluates them.

## High-Level Architecture

```text
External Sources
  - market feeds
  - IoT
  - CRM
  - security telemetry
  - operational systems

        |
        v

Signal Gateway
  - source registration
  - source authentication
  - schema validation
  - rate limiting
  - event normalization

        |
        v

Broker Abstraction
  - Kafka or Redpanda initially
  - NATS-compatible abstraction reserved for future use

        |
        v

Signal Workers
  - consume normalized events
  - enrich metadata
  - write replayable event history
  - create Event Artifacts
  - propose graph mutations
  - publish Insight Candidates

        |
        v

Storage and Engine Adapters
  - PostgreSQL for registry, schemas, policies, jobs, audit, and config
  - TimescaleDB for raw events, normalized events, windows, rollups, replay
  - Qdrant for derived summaries, embeddings, and semantic clusters
  - Syncratic Engine APIs for artifact, entity, graph, temporal, evidence,
    insight, and evaluation contracts
```

## Major Components

### Signal Gateway

The Signal Gateway is the only public ingestion entry point for SignalOps.

Responsibilities:

- Register and manage sources.
- Authenticate sources.
- Authorize source access to tenant-scoped resources.
- Validate event payloads against registered schemas.
- Enforce tenant and source rate limits.
- Normalize accepted events into the broker event envelope.
- Return deterministic idempotency responses for duplicate submissions.
- Emit audit records for source registration, schema changes, accepted
  events, rejected events, and rate-limit decisions.

The Signal Gateway MUST NOT contain domain-specific business logic beyond
validation, normalization, routing, and policy enforcement.

### Stream Broker

SignalOps provides an internal broker abstraction over Kafka and Redpanda.
NATS support is reserved for future implementation. Gateway and worker code
MUST depend on SignalOps broker interfaces rather than vendor-specific
client APIs.

Broker requirements:

- Topics MUST be tenant-aware and environment-aware.
- Messages MUST include schema version, tenant id, source id, event id, and
  correlation id.
- Partition keys SHOULD use `tenant_id:source_id:entity_hint` when an entity
  hint is available, otherwise `tenant_id:source_id`.
- Ordering is guaranteed only within a partition.
- Workers own consumer group membership and offset commits.
- Workers MUST commit offsets only after required durable writes or accepted
  downstream handoff.
- Retry and dead-letter topics MUST preserve the original payload and append
  processing metadata.

Initial topic convention:

- `signalops.<env>.raw.v1`
- `signalops.<env>.normalized.v1`
- `signalops.<env>.artifact.v1`
- `signalops.<env>.graph_mutation.v1`
- `signalops.<env>.insight_candidate.v1`
- `signalops.<env>.retry.<stage>.v1`
- `signalops.<env>.dlq.<stage>.v1`

### Signal Workers

Signal Workers consume broker events and perform asynchronous processing.

Responsibilities:

- Consume normalized events.
- Enforce idempotency before side effects.
- Enrich metadata with tenant, source, schema, correlation, and processing
  context.
- Persist replayable event history to TimescaleDB.
- Create Event Artifacts.
- Call Engine extension APIs for entity resolution, artifact registration,
  graph mutation proposals, temporal data, evidence, insight, and evaluation.
- Publish Insight Candidates.
- Route retryable and non-retryable failures to the correct broker topics.
- Emit logs, metrics, traces, and audit records.

Workers MUST be horizontally scalable and safe to restart. Processing MUST
be idempotent across duplicate messages, worker crashes, and replay jobs.

## REST API Contracts

All public REST APIs MUST require tenant context and source authentication
unless explicitly marked internal. Request bodies and responses MUST be JSON.

### Register Source

`POST /v1/tenants/{tenant_id}/sources`

Registers a stream source for a tenant.

Request:

```json
{
  "name": "crm-prod",
  "source_type": "crm",
  "auth_type": "api_key",
  "schema_id": "crm-contact-event",
  "rate_limit_per_minute": 6000,
  "metadata": {
    "owner": "revops"
  }
}
```

Response `201 Created`:

```json
{
  "source_id": "src_01J00000000000000000000000",
  "tenant_id": "tenant_123",
  "status": "active",
  "created_at": "2026-07-06T00:00:00Z"
}
```

### Register Schema

`POST /v1/tenants/{tenant_id}/schemas`

Registers an event schema version. Schemas MUST be versioned. Backward
compatible additions MAY be accepted as minor versions. Breaking changes
MUST create a new major version.

Request:

```json
{
  "schema_id": "crm-contact-event",
  "version": "1.0.0",
  "format": "json_schema",
  "definition": {}
}
```

Response `201 Created`:

```json
{
  "schema_id": "crm-contact-event",
  "version": "1.0.0",
  "status": "active"
}
```

### Ingest Event

`POST /v1/tenants/{tenant_id}/sources/{source_id}/events`

Accepts a single event from a registered source.

Required headers:

- `Authorization`
- `Idempotency-Key`
- `X-Correlation-Id`

Request:

```json
{
  "event_id": "evt_01J00000000000000000000000",
  "timestamp": "2026-07-06T00:00:00Z",
  "event_type": "contact.updated",
  "payload": {},
  "entity_hints": [
    {
      "type": "contact",
      "external_id": "contact_123"
    }
  ],
  "metadata": {}
}
```

Response `202 Accepted`:

```json
{
  "event_id": "evt_01J00000000000000000000000",
  "status": "accepted",
  "correlation_id": "corr_123"
}
```

Duplicate submissions with the same tenant, source, event id, and
idempotency key MUST return the original result without producing duplicate
side effects.

### Create Replay Job

`POST /v1/tenants/{tenant_id}/replay-jobs`

Creates a replay job from durable event history.

Request:

```json
{
  "source_id": "src_01J00000000000000000000000",
  "from": "2026-07-01T00:00:00Z",
  "to": "2026-07-06T00:00:00Z",
  "event_types": ["contact.updated"],
  "reason": "reprocess after schema mapper fix"
}
```

Response `202 Accepted`:

```json
{
  "job_id": "replay_01J00000000000000000000000",
  "status": "queued"
}
```

### Health and Readiness

- `GET /healthz` returns process health.
- `GET /readyz` returns readiness for broker, PostgreSQL, TimescaleDB, and
  configured Engine extension dependencies.

## Event Contracts

All SignalOps events MUST include:

- `tenant_id`
- `source_id`
- `event_id`
- `schema_id`
- `schema_version`
- `occurred_at`
- `observed_at`
- `correlation_id`
- `idempotency_key`

### RawSignalEvent

The event accepted from the source before full normalization.

Required fields:

- `tenant_id`
- `source_id`
- `event_id`
- `event_type`
- `occurred_at`
- `observed_at`
- `payload`
- `entity_hints`
- `metadata`

### NormalizedSignalEvent

The validated and normalized event published by the Gateway.

Required fields:

- `tenant_id`
- `source_id`
- `event_id`
- `event_type`
- `occurred_at`
- `observed_at`
- `normalized_payload`
- `entities`
- `confidence`
- `metadata`
- `evidence`

### EventArtifact

An Event Artifact is the Engine-consumable representation of a normalized
signal.

Required fields:

- `artifact_id`
- `event_id`
- `source_id`
- `tenant_id`
- `timestamp`
- `event_type`
- `entities`
- `confidence`
- `metadata`
- `evidence`
- `schema_version`
- `correlation_id`

Field semantics:

- `artifact_id` MUST be stable for the same tenant, source, and event id.
- `timestamp` MUST represent the source event time when available.
- `observed_at` MUST be retained in metadata when different from source
  event time.
- `confidence` MUST be a number from `0.0` to `1.0`.
- `evidence` MUST include source payload references or retained excerpts
  sufficient for audit without exposing unauthorized tenant data.

### GraphMutationProposal

SignalOps proposes mutations. The Syncratic Engine validates and applies
them.

Required fields:

- `proposal_id`
- `tenant_id`
- `event_id`
- `artifact_id`
- `mutation_type`
- `subject`
- `predicate`
- `object`
- `confidence`
- `valid_from`
- `valid_to`
- `observed_at`
- `evidence`
- `correlation_id`

Supported mutation types:

- `create_entity`
- `update_relationship`
- `update_confidence`
- `create_temporal_edge`
- `retire_relationship`

### InsightCandidate

An Insight Candidate is a derived observation that requires evaluation
before it is presented as an insight.

Required fields:

- `candidate_id`
- `tenant_id`
- `source_event_ids`
- `artifact_ids`
- `summary`
- `entities`
- `confidence`
- `evidence`
- `created_at`
- `correlation_id`

## Storage Ownership

### PostgreSQL

PostgreSQL is canonical for SignalOps operational metadata:

- source registry
- source credentials metadata, excluding secret values
- schema registry
- tenant policies
- replay jobs
- idempotency records
- audit records
- configuration

### TimescaleDB

TimescaleDB stores replayable time-series history:

- raw signal events
- normalized signal events
- windows
- aggregates
- rollups
- replay cursors and replayable history

TimescaleDB is the canonical store for replay input. Retention policies MUST
be configurable by tenant and source.

### Neo4j

Neo4j stores graph state owned by the Syncratic Engine:

- Event nodes
- Entity nodes
- temporal relationships
- applied graph mutations

SignalOps MUST NOT directly write canonical graph changes to Neo4j unless
the Engine exposes that path as an approved Graph Mutation API adapter.

### Qdrant

Qdrant stores derived semantic data:

- event summaries
- incident summaries
- embeddings
- semantic clusters

Qdrant MUST NOT be treated as canonical event truth. Qdrant entries MUST be
rebuildable from TimescaleDB and Engine-approved artifacts.

## Engine Extension Contracts

Engine extensions are contracts only. Stream processing MUST remain inside
SignalOps.

Required extension APIs:

- Artifact API: register or update Event Artifacts.
- Entity Resolution API: resolve source entity hints to canonical entities.
- Graph Mutation API: submit mutation proposals and receive accepted,
  rejected, or pending status.
- Temporal API: submit valid-time and observed-time relationship metadata.
- Insight API: publish Insight Candidates.
- Evidence API: register evidence references for audit and explainability.
- Evaluation API: request evaluation of Insight Candidates and mutation
  confidence.

Every Engine extension request MUST include tenant id, correlation id,
idempotency key, source id, and evidence references. Every response MUST be
auditable and must distinguish accepted, rejected, pending, and retryable
failure states.

## Temporal Knowledge

Relationships support:

- `valid_from`
- `valid_to`
- `observed_at`

SignalOps MUST propose continuous temporal mutations rather than requiring
graph rebuilds. When source events arrive out of order, SignalOps MUST keep
the original event timestamp and observed timestamp distinct so the Engine
can evaluate temporal validity.

## Failure Handling

SignalOps processing MUST be idempotent. The idempotency scope is tenant id,
source id, event id, and idempotency key.

Retryable failures include transient broker, database, and Engine extension
errors. Non-retryable failures include schema validation failure,
authorization failure, malformed payloads, and unsupported event types.

Retry behavior:

- Retryable failures SHOULD use bounded exponential backoff.
- Retry attempts MUST include attempt count and last error metadata.
- Exhausted retryable failures MUST be routed to the appropriate DLQ topic.
- DLQ messages MUST retain original payload, normalized payload when
  available, error class, error message, stage, attempt count, and
  correlation id.
- Poison messages MUST NOT block unrelated partitions indefinitely.

Partial failure behavior:

- If a worker writes durable event history but fails before Engine handoff,
  retry MUST resume from the durable state without duplicating the history
  write.
- If the Engine accepts a mutation but the worker fails before committing the
  broker offset, replay MUST use idempotency keys to avoid duplicate Engine
  side effects.
- If Qdrant indexing fails after canonical writes succeed, the event SHOULD
  be retried for semantic indexing without rolling back canonical storage.

## Security and Multi-Tenancy

SignalOps MUST enforce tenant isolation at API, broker, storage, audit, and
observability boundaries.

Security requirements:

- Source authentication is required for ingestion.
- Authorization MUST verify that a source belongs to the tenant in the route.
- Tenant id MUST be carried in every event envelope, storage record, log,
  metric label where safe, audit event, and Engine extension request.
- Source secret values MUST be stored only in an approved secret store, not
  in PostgreSQL plaintext.
- Payloads containing PII or secrets MUST be redacted in logs and metrics.
- Audit records MUST capture source registration, schema changes, event
  acceptance, event rejection, replay job creation, mutation proposal
  submission, Engine rejection, and DLQ routing.

## Observability

SignalOps MUST emit structured logs, metrics, and traces.

Required logs:

- API request accepted/rejected.
- Schema validation failure.
- Rate-limit decision.
- Worker processing start/end.
- Engine extension accepted/rejected/retryable failure.
- Retry and DLQ routing.
- Replay job lifecycle events.

Required metrics:

- ingest request count by tenant, source, status, and event type.
- ingest latency.
- broker publish latency.
- worker processing latency.
- worker success/failure count.
- retry count by stage.
- DLQ count by stage and error class.
- Engine extension latency and status.
- replay job progress.

Required traces:

- Gateway request through broker publish.
- Worker consume through storage writes and Engine extension calls.
- Replay job scan through republished events.

Alert conditions SHOULD include sustained ingest failures, DLQ growth,
worker lag, replay job failure, Engine extension error rate, and storage
connectivity loss.

## Deployment

SignalOps MUST be independently deployable.

Deployment requirements:

- Independent Helm chart.
- Independent Kubernetes namespace.
- Configurable broker, PostgreSQL, TimescaleDB, Qdrant, and Engine endpoint
  settings.
- Separate deployments for Gateway and worker pools.
- Horizontal autoscaling for Gateway and workers.
- Liveness and readiness probes.
- Resource requests and limits.
- Pod disruption budgets for production deployments.
- Tenant-safe configuration and secret handling.

## Deliverables

- SignalOps service.
- Helm deployment.
- REST APIs.
- Worker framework.
- Event contracts.
- Engine adapters.
- Integration tests.
- Operational dashboards and alerts.

## Acceptance Criteria

The implementation is acceptable when the following scenarios pass:

- Source registration succeeds, persists source metadata in PostgreSQL, and
  emits an audit record.
- Valid event ingestion returns `202 Accepted`, publishes a normalized event
  to the broker, persists replayable history, and records metrics.
- Invalid schema submissions are rejected without publishing broker events.
- Duplicate submissions with the same idempotency scope return the original
  outcome and do not duplicate storage or Engine side effects.
- Retryable worker failure is retried and then succeeds without duplicate
  canonical writes.
- Exhausted retryable failure is routed to DLQ with original payload,
  processing stage, attempt count, error metadata, and correlation id.
- Historical replay republishes events from TimescaleDB for the requested
  tenant, source, event types, and time range.
- Tenant isolation prevents one tenant from registering, reading, replaying,
  or mutating another tenant's events.
- Invalid graph mutation proposals are rejected by the Engine and retained
  as auditable rejected proposals.
- Insight Candidates are published with evidence references and correlation
  ids.
- Existing document ingestion, retrieval, graph construction, and Ask
  workflows continue to operate unchanged.
