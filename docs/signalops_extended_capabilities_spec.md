# Syncratic SignalOps Extended Capabilities Specification

## 1. Purpose

This specification extends the base Syncratic SignalOps Infrastructure and Processing specifications with a phased platform roadmap for higher-order time/stream intelligence capabilities.

SignalOps is a standalone, domain-neutral subsystem for ingesting, processing, correlating, replaying, and evaluating time-based and stream-based observations. Market data, CRM intelligence, security analytics, operational intelligence, IoT/manufacturing, procurement/RFP monitoring, and custom tenant use cases are implemented as source domains and use-case packs on top of the same core platform.

This document is not an all-at-once implementation contract. Capabilities are separated into implementation phases so the first deliverables establish stable contracts, versioning, lineage, and non-interference before advanced algorithm execution and enterprise operations are added.

These capabilities MUST follow Syncratic's non-interference principle. They MUST NOT modify existing document ingestion, retrieval, graph construction, or Ask workflows. They MUST operate inside SignalOps and communicate with the Syncratic Engine only through approved extension contracts.

## 2. Platform Model

SignalOps is split into three layers.

### 2.1 Core Platform

Core platform services are implemented in Go by default.

Go owns:

- source registration and source-domain configuration
- source adapter orchestration
- public REST APIs and catalog APIs
- broker abstraction and topic routing
- idempotency, retries, DLQ, replay, and backpressure coordination
- PostgreSQL and TimescaleDB persistence
- pipeline orchestration and execution records
- safe deterministic rule execution where supported
- state coordination and feature materialization orchestration
- Engine extension adapters
- observability, audit, and deployment lifecycle

Go services MUST NOT import, embed, or directly execute Python libraries.

### 2.2 Processing Runtime

Algorithmic processing is Python-first.

Python owns:

- detector plugins
- ML/NLP libraries
- model scoring
- explainability outputs
- forecasting, clustering, anomaly detection, and classification
- benchmark datasets and algorithm evaluation support

Python algorithm workers communicate with the Go platform through broker events or explicitly versioned internal service APIs. Container or HTTP plugins MAY be added after the Python plugin runner is stable.

### 2.3 Use-Case Packs

Use-case packs provide domain-specific behavior without changing core infrastructure.

Initial source domains:

- `market_data`
- `crm`
- `security`
- `operations`
- `iot`
- `procurement`
- `custom`

Domain model:

- `source_domain`: broad use-case domain, such as `market_data` or `security`.
- `source_adapter`: concrete connector, such as `market_data.massive`.
- `signal_family`: routing and processing family inside a source domain, such as `options_daily_prices`, `crm_opportunity_activity`, or `security_identity_behavior`.

Use-case packs may define schemas, normalizers, enrichers, rules, features, detector policies, graph mapping policies, retention defaults, replay defaults, and benchmark datasets. They MUST NOT bypass core idempotency, replay, temporal storage, tenant isolation, or Engine extension contracts.

## 3. Non-Interference Requirements

The implementation MUST preserve the following boundaries:

1. SignalOps remains a separate subsystem.
2. SignalOps does not directly modify Syncratic document ingestion code paths.
3. SignalOps does not directly write canonical graph state unless the Syncratic Engine exposes an approved adapter for that purpose.
4. SignalOps proposes Event Artifacts, Graph Mutation Proposals, Signals, and Insight Candidates through Engine extension contracts.
5. The Syncratic Engine remains the authority for accepting, rejecting, evaluating, and applying knowledge graph mutations.
6. Algorithm execution remains pluggable and outside the Engine core.
7. SignalOps failures do not degrade existing Ask, retrieval, document ingestion, or graph construction workflows.

## 4. Extended Architecture

```text
External Sources
  - push events and webhooks
  - scheduled REST APIs
  - bulk files and historical datasets
  - broker-fed event sources
  - future WebSocket/live streams

        |
        v

Source Adapter Layer
  - market_data.massive
  - crm.*
  - security.*
  - operations.*
  - iot.*
  - procurement.*
  - custom.*

        |
        v

Go Core Platform
  - gateway and connector framework
  - schema validation and normalization
  - broker routing
  - pipeline manager
  - rule engine
  - catalog APIs
  - persistence, replay, lineage
  - Engine extension adapters

        |
        v

Temporal and Processing Layer
  - windows
  - state store abstraction
  - feature definitions and values
  - signal definitions and instances
  - confidence and lifecycle policies

        |
        v

Python Algorithm Runtime
  - anomaly detection
  - clustering
  - forecasting
  - classification
  - trend and drift detection
  - explainability

        |
        v

SignalOps Outputs
  - Event Artifacts
  - Rule Artifacts
  - Signal Artifacts
  - Graph Mutation Proposals
  - Insight Candidates
  - Evaluation Requests

        |
        v

Syncratic Engine Extension APIs
```

## 5. Capability Model

### 5.1 Signal Definition Layer

Events are raw observations. Signals are meaningful derived interpretations.

A Signal Definition describes a named signal and its detection contract. A Signal Instance is an observed occurrence of a Signal Definition.

Signal Definition fields:

- `signal_id`
- `tenant_id`
- `source_domain`
- `signal_family`
- `name`
- `description`
- `input_event_types`
- `required_entities`
- `rule_refs`
- `feature_refs`
- `algorithm_refs`
- `confidence_policy_ref`
- `lifecycle_policy_ref`
- `schema_version`
- `status`
- `created_at`
- `updated_at`

Signal Instance fields:

- `signal_instance_id`
- `tenant_id`
- `signal_id`
- `source_domain`
- `signal_family`
- `source_event_ids`
- `artifact_ids`
- `entity_refs`
- `feature_values`
- `rule_results`
- `algorithm_results`
- `confidence`
- `state`
- `started_at`
- `last_observed_at`
- `resolved_at`
- `evidence_refs`
- `correlation_id`

### 5.2 Rule Engine

Rules provide deterministic processing before and alongside machine learning algorithms.

Initial rule languages:

- `json_logic`
- `cel`

`python_guarded` MAY be used only for internal trusted deployments after sandboxing and operational controls are approved.

Rules may support threshold detection, policy checks, escalation conditions, filtering, suppression, routing, and signal creation. The first implementation MUST prioritize JSON Logic or CEL-style expressions to avoid unsafe arbitrary execution.

### 5.3 Window Processing

SignalOps supports time windows for stream and scheduled observations.

Window types:

- tumbling
- sliding
- rolling
- session
- calendar

Window calculations SHOULD use event time where possible. SignalOps MUST distinguish source event time, ingestion time, and processing time. Late events are handled through configured watermark policy.

### 5.4 Stream State Store

Signal calculations often require state across events, including current entity status, last event by entity, rolling counters, moving averages, active incident state, and open signal state.

Initial state backends:

- Redis or equivalent for fast operational state
- TimescaleDB for durable aggregate state
- PostgreSQL for low-volume workflow state

Redis MUST NOT become the only canonical source of replayable event truth.

### 5.5 Processing Pipeline Framework

Pipelines determine how events are normalized, enriched, windowed, converted into features, evaluated by rules, processed by algorithms, transformed into artifacts, and submitted to the Engine.

Pipeline definitions include tenant, source domain, signal family, input event types, ordered stages, output targets, version, and status.

Initial stage types:

- `normalize`
- `entity_resolve`
- `enrich`
- `window`
- `feature_compute`
- `rule_evaluate`
- `algorithm_execute`
- `signal_detect`
- `artifact_build`
- `graph_propose`
- `insight_publish`
- `evaluation_request`

### 5.6 Signal Registry and Signal Families

Signal families route events into domain-aware processing paths. Signal families are subordinate to source domains.

Examples:

- `market_data.options_daily_prices`
- `market_data.options_contracts`
- `crm.opportunity_activity`
- `security.identity_behavior`
- `operations.service_health`
- `iot.sensor_telemetry`
- `procurement.rfp_opportunity`

### 5.7 Enrichment Framework

Enrichment is pluggable and must not hardcode domain behavior into core workers.

Initial enricher types:

- `static_lookup`
- `database_lookup`
- `http_lookup`
- `engine_lookup`
- `calendar_context`
- `tenant_registry_lookup`

Examples include GeoIP lookup, company lookup, customer registry lookup, asset registry lookup, ticker metadata lookup, threat intelligence lookup, device inventory lookup, calendar context, and maintenance window context.

### 5.8 Feature Store

SignalOps supports reusable feature definitions and materialized feature values for algorithms and signal detection.

Feature definitions SHOULD be stored in PostgreSQL. The first implementation SHOULD use TimescaleDB for time-indexed feature values. A dedicated feature store backend remains future scope until scale requires it.

### 5.9 Algorithm Plugin Framework

Algorithm plugins may support anomaly detection, clustering, classification, forecasting, trend detection, change-point detection, sequence detection, and correlation detection.

Initial runtime types:

- `python_plugin`
- `container_plugin`
- `http_plugin`

The first plugin implementation SHOULD provide one safe reference Python runner and one deterministic test algorithm before supporting arbitrary tenant-provided plugins.

Candidate Python libraries MAY include PyOD, River, scikit-learn, HDBSCAN, STUMPY, Prophet, Merlion, Kats, ruptures, and statsmodels. SignalOps infrastructure MUST NOT hardwire any of these libraries.

### 5.10 Confidence Aggregation

SignalOps supports confidence aggregation from rules, enrichers, algorithms, and human evaluations.

Initial aggregation methods:

- `max`
- `min`
- `mean`
- `weighted_mean`
- `rule_priority`
- `algorithm_priority`
- `human_override`

### 5.11 Signal and Insight Lifecycle

Signal Instance states:

- `observed`
- `open`
- `growing`
- `stable`
- `suppressed`
- `resolved`
- `archived`

Insight states:

- `candidate`
- `evaluating`
- `accepted`
- `dismissed`
- `learned`
- `archived`

The Syncratic Evaluation Agent or Engine evaluation extension decides whether an Insight Candidate is accepted, dismissed, pending, or requires human review. SignalOps stores the returned evaluation status for audit and downstream learning.

### 5.12 Expanded Correlation and Lineage

SignalOps event envelopes SHOULD support:

- `trace_id`
- `correlation_id`
- `causation_id`
- `parent_event_id`
- `child_event_ids`
- `session_id`
- `workflow_id`
- `replay_job_id`

Lineage SHOULD capture:

```text
Source
  -> Raw Event
  -> Normalized Event
  -> Enrichment Result
  -> Window Result
  -> Feature Value
  -> Rule Result
  -> Algorithm Result
  -> Signal Instance
  -> Artifact
  -> Graph Mutation Proposal
  -> Insight Candidate
  -> Evaluation Result
```

### 5.13 Signal Graph Model

SignalOps supports graph-ready semantics so the Syncratic Engine can connect events, signals, insights, decisions, and outcomes.

Graph proposal semantics may include:

- `event_observed_entity`
- `event_generated_artifact`
- `artifact_generated_signal`
- `signal_supported_insight_candidate`
- `insight_supported_decision`
- `decision_produced_outcome`
- `signal_correlated_with_signal`
- `signal_suppressed_by_policy`
- `signal_resolved_by_event`

These semantics MUST be translated into the Engine's approved Graph Mutation API format.

### 5.14 Policy Engine

Policies govern runtime behavior. Rules detect conditions; policies control how SignalOps behaves.

Initial policy types:

- `suppression`
- `routing`
- `retention`
- `escalation`
- `confidence`
- `replay`
- `rate_limit`
- `algorithm_execution`

### 5.15 Backpressure and Prioritized Streams

SignalOps prevents overload from degrading the subsystem or the Syncratic Engine.

Backpressure monitors broker lag, gateway request rate, worker queue depth, worker latency, storage write latency, Engine extension latency, DLQ growth, and retry growth.

Priority levels:

- `critical`
- `high`
- `medium`
- `low`
- `background`

The broker abstraction SHOULD support either priority topics or priority-aware routing metadata.

### 5.16 Stream Catalog

The Stream Catalog gives tenant administrators visibility into registered sources, schemas, topics, signal families, pipelines, enrichers, rules, policies, features, algorithms, retention policies, replay policies, DLQ status, and lifecycle states.

Example APIs:

- `GET /v1/tenants/{tenant_id}/catalog/sources`
- `GET /v1/tenants/{tenant_id}/catalog/schemas`
- `GET /v1/tenants/{tenant_id}/catalog/pipelines`
- `GET /v1/tenants/{tenant_id}/catalog/signals`
- `GET /v1/tenants/{tenant_id}/catalog/features`
- `GET /v1/tenants/{tenant_id}/catalog/algorithms`
- `GET /v1/tenants/{tenant_id}/catalog/policies`
- `GET /v1/tenants/{tenant_id}/catalog/dlq`

### 5.17 Event and Artifact Versioning

Versioning applies to raw event schema, normalized event schema, pipeline version, enrichment version, rule version, feature version, algorithm version, artifact builder version, and graph mapper version.

Replay jobs MUST support:

- `original_versions`
- `latest_compatible`
- `explicit_versions`

## 6. REST API Extensions

### 6.1 Register Signal Definition

`POST /v1/tenants/{tenant_id}/signals`

Registers a versioned signal definition for a source domain and signal family.

### 6.2 Register Pipeline

`POST /v1/tenants/{tenant_id}/pipelines`

Registers an ordered, versioned processing pipeline.

### 6.3 Register Rule

`POST /v1/tenants/{tenant_id}/rules`

Registers a deterministic rule. Invalid rule bodies MUST be rejected before activation.

### 6.4 Register Feature

`POST /v1/tenants/{tenant_id}/features`

Registers a reusable feature definition.

### 6.5 Register Algorithm

`POST /v1/tenants/{tenant_id}/algorithms`

Registers an algorithm definition and runtime contract. Registering an algorithm does not grant direct access to tenant data outside the configured pipeline scope.

### 6.6 Catalog APIs

Catalog APIs are read-only tenant-scoped APIs for inspecting sources, schemas, pipelines, rules, features, algorithms, policies, DLQ status, and lifecycle state.

## 7. Storage Ownership

### 7.1 PostgreSQL

PostgreSQL remains canonical for operational metadata and configuration.

Additional tables SHOULD include:

- `signal_definitions`
- `signal_instances`
- `signal_lifecycle_transitions`
- `rule_definitions`
- `rule_results`
- `pipeline_definitions`
- `pipeline_executions`
- `enricher_definitions`
- `enrichment_results`
- `feature_definitions`
- `algorithm_definitions`
- `algorithm_results`
- `confidence_policies`
- `confidence_results`
- `policy_definitions`
- `policy_decisions`
- `lineage_records`
- `processing_version_records`
- `catalog_metadata`
- `backpressure_decisions`

### 7.2 TimescaleDB

TimescaleDB remains canonical for replayable time-series history.

Additional hypertables SHOULD include:

- `window_results`
- `feature_values`
- `signal_observations`
- `algorithm_timeseries_results`
- `state_history`

### 7.3 Redis Or Equivalent State Store

Redis MAY be used for hot state, active sessions, counters, rate-limiting state, short-lived pipeline state, and deduplication caches.

Redis MUST NOT become the only canonical source of replayable event truth.

### 7.4 Qdrant

Qdrant MAY store signal summaries, insight summaries, incident summaries, semantic clusters, and embeddings of signal narratives.

Qdrant entries MUST be rebuildable from TimescaleDB, PostgreSQL, and Engine-approved artifacts.

## 8. Security And Tenant Isolation

Extended capabilities MUST enforce tenant isolation across signal definitions, rules, pipelines, features, algorithms, enrichers, policies, feature values, algorithm results, signal instances, lineage records, and catalog APIs.

Rules and algorithms MUST NOT access another tenant's data.

Algorithm plugins MUST run with constrained permissions. Container-based plugins SHOULD run with no privilege escalation, read-only root filesystem where possible, resource limits, network restrictions unless explicitly required, and tenant-scoped credentials only.

Logs and metrics MUST redact PII, secrets, and sensitive payload values.

## 9. Observability Extensions

Required metrics include rule evaluation count, rule match rate, pipeline execution latency by stage, pipeline failure rate by stage, enrichment latency, feature computation latency, algorithm execution latency, signal creation count, lifecycle transitions, confidence aggregation, policy decisions, backpressure activation, priority topic lag, and catalog object count.

Required logs include pipeline selection, stage lifecycle, rule evaluation, signal lifecycle changes, policy decisions, confidence aggregation, algorithm plugin invocation/failure, backpressure decisions, and lineage creation.

Required traces connect Gateway request, broker publish, pipeline execution, enrichment, feature computation, rule evaluation, algorithm execution, signal detection, artifact creation, and Engine extension request.

## 10. Deployment Extensions

SignalOps deployment SHOULD add optional worker pools for:

- pipeline execution workers
- enrichment workers
- feature computation workers
- algorithm workers
- signal lifecycle workers
- replay workers
- catalog API service

Each worker pool SHOULD be independently scalable.

Algorithm workers SHOULD support resource-specific scheduling, including CPU-only, GPU-enabled, or memory-optimized execution profiles.

Helm values SHOULD include configuration for enabled source domains, enabled worker pools, algorithm plugin mode, Redis/state store settings, priority topic settings, backpressure thresholds, feature retention policies, signal retention policies, and algorithm resource limits.

## 11. Implementation Phases

### Phase 1: Platform Foundation

Goal: establish the domain-neutral platform contract and a minimal safe processing framework.

Deliverables:

- source domains and source adapters in source registry
- schema registry integration with normalized event contracts
- pipeline definitions and ordered stage metadata
- basic rule engine using JSON Logic or CEL
- pipeline execution records
- basic lineage records from raw event to pipeline output
- Stream Catalog read APIs for sources, schemas, pipelines, rules, and DLQ status
- Engine adapter boundaries for artifacts, graph proposals, insights, evidence, and evaluation
- Go project scaffolding for platform services
- shared schema/contracts directory

Acceptance criteria:

- A tenant can register a source with `source_domain`, `source_adapter`, and `signal_family`.
- A pipeline can be registered with ordered stages.
- Events are routed to the correct pipeline by source domain, source adapter, event type, and signal family.
- A matching deterministic rule produces a Rule Result with evidence, version, score, and correlation ID.
- Invalid rule bodies are rejected before activation.
- Pipeline execution records include all stage results.
- Basic lineage can be reconstructed from raw event to pipeline output.
- Catalog APIs do not expose other tenant objects.
- Existing document ingestion, retrieval, graph construction, and Ask workflows continue unchanged.

### Phase 2: Temporal Processing

Goal: add durable time/window processing and reusable feature materialization.

Deliverables:

- window definitions
- window results
- watermark and late-event policy metadata
- state store abstraction
- Redis or equivalent hot-state adapter
- TimescaleDB-backed feature values
- feature definitions
- processing version records
- replay modes for original, latest compatible, and explicit versions

Acceptance criteria:

- Rolling, tumbling, and calendar windows compute expected aggregates from event history.
- Late events are handled according to configured watermark policy.
- Feature Values are queryable by tenant, feature, entity, and time range.
- Replay can run using original processing versions.
- Replay can run using latest compatible versions.
- Replay output records processing versions used.

### Phase 3: Signal Intelligence

Goal: make signals first-class objects with lifecycle, confidence, policy, and insight integration.

Deliverables:

- signal definitions
- signal instances
- signal lifecycle transitions
- confidence policies and confidence results
- policy definitions and decisions
- insight lifecycle integration
- signal graph proposal semantics

Acceptance criteria:

- A tenant can register a Signal Definition.
- Signal Definitions are versioned.
- Inactive Signal Definitions are not used for new detections.
- A Signal Instance can move from `observed` to `open`, `suppressed`, `resolved`, and `archived`.
- Multiple confidence values can be aggregated by policy.
- Human override policy can supersede algorithm confidence where configured.
- A Signal Instance can publish an Insight Candidate with evidence and correlation ID.
- Evaluation response is stored and reflected in insight lifecycle state.
- Signal graph semantics are translated into approved Engine Graph Mutation API requests.

### Phase 4: Algorithm Plugins

Goal: introduce Python-first algorithm execution without coupling Go infrastructure to Python libraries.

Deliverables:

- algorithm registry
- Python plugin execution request/result contracts
- one safe reference Python plugin runner
- one deterministic reference algorithm
- container plugin runner
- HTTP plugin runner
- algorithm worker pool
- benchmark datasets and evaluation records

Acceptance criteria:

- An algorithm plugin can be registered.
- A pipeline can invoke the algorithm plugin through broker or versioned internal API boundaries.
- Algorithm Results include score, confidence, payload, evidence, source features, algorithm version, and model version.
- Failed algorithm executions route to retry or DLQ without blocking unrelated events.
- Go services do not import or embed Python libraries.
- Algorithm workers enforce tenant scope and constrained permissions.

### Phase 5: Enterprise Operations

Goal: add operational controls for production-scale multi-tenant signal intelligence.

Deliverables:

- backpressure control
- priority-aware routing metadata or priority topics
- advanced catalog visibility
- operational dashboards and alerts
- advanced replay/version controls
- independently scalable worker pools
- algorithm resource scheduling profiles

Acceptance criteria:

- Critical events continue processing when low-priority streams are throttled.
- Sustained worker lag triggers a backpressure decision.
- Backpressure decisions are auditable.
- Priority topic lag is observable.
- Tenant administrators can list registered sources, schemas, pipelines, signals, rules, features, algorithms, policies, lifecycle states, and DLQ status.
- SignalOps can be disabled without breaking the core Syncratic Engine.

## 12. Recommended Repository Layout

The repository SHOULD separate Go core services, Python algorithm workers, and shared contracts.

```text
signalops/
  cmd/
    gateway/
    worker/
    catalog/
    replay/
  internal/
    api/
    auth/
    broker/
    catalog/
    config/
    engineadapters/
    idempotency/
    lineage/
    observability/
    pipelines/
    policies/
    rules/
    schemas/
    security/
    sources/
    state/
    storage/
    tenancy/
    windows/
  pkg/
    contracts/
  contracts/
    events/
    artifacts/
    signals/
    pipelines/
    rules/
    features/
    algorithms/
  python/
    signalops_plugins/
      sdk/
      runners/
      reference_algorithms/
      tests/
  deploy/
    helm/
  migrations/
  tests/
```

The first deliverable should avoid over-optimizing algorithm execution. Build the platform contract, registries, execution records, lineage, catalog visibility, and one safe reference runner first. The core value is the stable platform boundary, not the first algorithm.
