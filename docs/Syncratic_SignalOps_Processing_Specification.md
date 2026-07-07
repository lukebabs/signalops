# Syncratic SignalOps Processing Specification

## Purpose

This specification defines the intelligence pipeline used by SignalOps to
turn normalized stream events into algorithmic signals, Event Artifacts,
Graph Mutation Proposals, and Insight Candidates.

Algorithms are isolated plug-ins. SignalOps must never depend on a single
algorithm implementation, and no processing code may be embedded in the
Syncratic Engine.

## Runtime Architecture

SignalOps uses a polyglot runtime model:

- Go owns infrastructure orchestration.
- Python owns algorithmic and ML/NLP execution.
- Kafka or Redpanda topics are the preferred boundary between Go and Python
  processes.
- Go services must not import, embed, or directly execute Python libraries.
- Python algorithm workers must communicate through broker events or
  explicitly versioned internal service APIs.

## Internal Communication Protocols

SignalOps uses two internal communication paths between the Go core platform
and the Python algorithm system.

Durable path:

- Protocol: Kafka or Redpanda broker topics.
- Payload format v1: JSON.
- Schema contract v1: JSON Schema under `contracts/`.
- Use for Go-to-Python algorithm jobs, Python-to-Go algorithm results,
  replayable processing, retries, DLQ routing, batch/windowed processing, and
  any work that must survive process restarts.

Fast sync path:

- Protocol: gRPC with Protobuf contracts.
- Use only for bounded request/response calls where the result is
  non-authoritative until the Go core persists or republishes it.
- Suitable for low-latency model scoring, health/status/control APIs, or
  small internal lookups that are safe to retry from the caller.
- gRPC responses from Python must not become canonical truth by themselves.

Decision rule:

- Use Kafka/Redpanda when work must be durable, replayable, retryable, or
  auditable.
- Use gRPC when work is short-lived, bounded, request/response, and safe to
  retry from the caller.
- Use REST only for public APIs, tenant/admin APIs, and compatibility
  boundaries.
- Do not use direct in-process Python calls from Go.

Go responsibilities:

- Signal Gateway and public REST APIs.
- Broker abstraction and topic publishing.
- Idempotency, retries, DLQ routing, and replay orchestration.
- PostgreSQL and TimescaleDB persistence.
- Engine extension adapters.
- Observability, health checks, and deployment lifecycle.

Python responsibilities:

- Feature extraction.
- Statistical anomaly detection.
- Change detection.
- Drift detection.
- Semantic enrichment with GLiNER, spaCy, embeddings, and related libraries.
- Model scoring.
- Detector explainability.
- Algorithm-specific benchmark and evaluation support.

Rust may be introduced later for narrow hot-path processing components, such
as parsing, redaction, or high-volume feature extraction, but it is not the
default runtime for initial SignalOps processing.

## Processing Pipeline

Processing starts from infrastructure-defined event contracts and produces
Engine-compatible outputs.

```text
RawSignalEvent
  -> NormalizedSignalEvent
  -> Python Feature Extraction Workers
  -> Python Detection Workers
  -> Signal
  -> Graph Context Lookup
  -> Semantic Enrichment
  -> EventArtifact
  -> GraphMutationProposal
  -> InsightCandidate
  -> Syncratic Engine Extension APIs
```

Runtime ownership:

- Go Gateway validates and normalizes source input into `NormalizedSignalEvent`.
- Python workers consume `NormalizedSignalEvent` and emit `Signal` records.
- Go persistence and adapter workers consume `Signal` records and derived
  outputs, persist canonical state, and call Engine extension APIs.
- Python workers may publish enriched outputs when the enrichment is
  algorithmic, but Go remains responsible for durable handoff, idempotency,
  and Engine API calls.

## Contract Relationships

The processing spec uses the event contracts defined in the infrastructure
specification.

- `RawSignalEvent`: source event before full normalization.
- `NormalizedSignalEvent`: validated event envelope published by the Gateway.
- `Signal`: algorithmic detection result produced by processing workers.
- `EventArtifact`: Engine-consumable evidence artifact derived from events
  and signals.
- `GraphMutationProposal`: proposed graph update submitted to the Engine for
  validation.
- `InsightCandidate`: evidence-backed candidate insight created after signal
  agreement or high-severity evidence.

`Signal` is not a replacement for `EventArtifact` or `InsightCandidate`.
It is the intermediate algorithmic output that explains what a detector found.

## Signal Object

Required fields:

- `signal_id`
- `tenant_id`
- `source_id`
- `event_ids`
- `artifact_ids`
- `signal_type`
- `detector_id`
- `detector_version`
- `model_version`
- `timestamp`
- `window_start`
- `window_end`
- `confidence`
- `severity`
- `entities`
- `supporting_metrics`
- `graph_targets`
- `semantic_evidence`
- `evidence`
- `recommendation`
- `correlation_id`

Field semantics:

- `signal_id` must be stable for the same tenant, detector, input window,
  model version, and configuration.
- `confidence` must be a number from `0.0` to `1.0`.
- `severity` must be one of `info`, `low`, `medium`, `high`, or `critical`.
- `event_ids` must reference the source events that caused the signal.
- `artifact_ids` may be empty until Go adapter workers create or register
  Event Artifacts.
- `evidence` must contain enough provenance for audit and evaluation without
  exposing unauthorized tenant data.

## Plugin Architecture

Detector plug-ins are Python-first packages unless a specific detector is
approved for another runtime.

Every detector implements:

- `initialize(config, model_registry, runtime_context)`
- `detect(normalized_events, feature_context)`
- `explain(detection_result)`
- `emit_signal(detection_result, explanation)`

The initial SDK contract is implemented in
`python/signalops_plugins/detectors/base.py`. The reference `signalops.noop`
detector exercises lifecycle wiring and emits no signals.

Plugin requirements:

- Plugins must be selected by tenant, source, event type, and configured
  processing policy.
- Plugin input and output schemas must be versioned.
- Plugins must produce deterministic output for the same input, model
  version, detector version, and configuration.
- Plugins must include detector id, detector version, model version, and
  confidence in emitted signals.
- Plugins must emit structured errors that can be routed through retry and
  DLQ behavior defined by the infrastructure specification.
- Plugins must not write directly to PostgreSQL, TimescaleDB, Neo4j, Qdrant,
  or Engine APIs unless explicitly implemented as an approved adapter.

SignalOps never depends on one algorithm. Multiple detectors may run against
the same event stream, and detectors may be enabled, disabled, or replaced by
configuration.

## Detection Layers

### Statistical Detection

Purpose: detect numerical anomalies.

Examples:

- spikes
- drops
- outliers
- threshold breaches
- unexpected variance

### Change Detection

Purpose: detect structural changes in observed behavior.

Examples:

- trend shifts
- regime changes
- seasonality changes
- baseline changes

### Drift Detection

Purpose: detect degradation or change in data and model behavior.

Examples:

- concept drift
- data drift
- model degradation
- feature distribution changes

### Semantic Enrichment

Semantic enrichment runs in Python algorithm workers.

Supported capabilities:

- GLiNER entity extraction.
- spaCy NLP pipelines.
- embedding generation.
- relationship extraction.
- theme and incident clustering.

Semantic enrichment may write derived vectors or summaries through approved
Qdrant adapters, but Qdrant remains derived storage. Canonical event truth
remains in TimescaleDB and Engine-approved artifacts.

### Graph Correlation

Graph correlation uses Engine-approved graph context APIs.

Given a `Signal`, graph correlation may resolve:

- affected entities
- neighboring entities
- communities
- prior incidents
- related Event Artifacts
- candidate temporal relationships

Graph correlation produces Graph Context. It must not directly mutate
canonical graph state. Any graph change must be emitted as a
`GraphMutationProposal` and validated by the Syncratic Engine.

## Insight Trigger

An Insight Candidate should be created when evidence indicates a meaningful
condition that may require attention, evaluation, or recommendation.

Default trigger policy:

- Create an Insight Candidate when at least two independent signals agree
  within a configured time window.
- Agreement requires overlapping tenant, entity, source domain, or graph
  context.
- Candidate confidence must meet the configured tenant/source threshold.
- Semantic or metric evidence must be attached.
- Duplicate candidates for the same entity, signal type, and time window
  must be suppressed or merged.

High-severity exception:

- A single `critical` signal may create an Insight Candidate when confidence
  is high and evidence is complete.
- The candidate must include the reason the multi-signal rule was bypassed.

Insight Trigger output must include source signals, evidence references,
confidence, entities, summary, recommendation when available, and correlation
id.

## Recommendation Pipeline

Recommendations are evidence-backed outputs, not direct graph mutations.

```text
Signal
  -> Historical Events
  -> Graph Context
  -> Related Documents
  -> Evidence Bundle
  -> Optional LLM Narrative
  -> InsightCandidate
```

The recommendation pipeline may use LLMs to produce narrative summaries, but
LLM output must be grounded in evidence references. Recommendations must be
stored as part of the Insight Candidate and evaluated before presentation as
accepted insight.

## Evaluation

Processing quality is measured through the existing Syncratic Evaluation
subsystem.

Required measures:

- precision
- recall
- false positives
- false negatives
- insight acceptance
- recommendation usefulness
- detector latency
- detector failure rate
- confidence calibration

Evaluation records must include detector id, detector version, model version,
input data window, tenant-safe evidence references, and final disposition.

## Human Feedback

Human feedback updates detector confidence, configuration, thresholds, and
evaluation data. It must not rewrite historical event facts.

Feedback behavior:

- Accepted insights may increase detector confidence or threshold priority.
- Rejected insights may reduce detector confidence or suppress similar
  candidates.
- Feedback must be tenant-scoped and auditable.
- Feedback integration must use KEE or the approved Syncratic feedback
  mechanism.

## Failure Handling

Processing failures must follow the infrastructure retry and DLQ model.

Failure requirements:

- Retryable plugin failures must include detector id, model version, stage,
  attempt count, and error class.
- Non-retryable plugin failures must produce auditable rejection records.
- Poison events must not block unrelated tenants, sources, or partitions.
- Python worker crashes must be recoverable through broker replay and
  idempotent signal ids.
- Go adapter workers must enforce idempotency before persistence or Engine
  handoff.

## Deliverables

- Processing pipeline.
- Python-first Plugin SDK.
- Signal object contract.
- Detector configuration model.
- Insight Trigger Engine.
- Recommendation pipeline.
- Evaluation integration.
- Human feedback integration.
- Unit tests.
- Integration tests.
- Benchmark datasets.

## Acceptance Criteria

The implementation is acceptable when the following scenarios pass:

- A Python detector worker consumes `NormalizedSignalEvent` and emits a
  valid `Signal`.
- Detector selection works by tenant, source, event type, and processing
  policy.
- A detector produces deterministic output for the same input, model version,
  detector version, and configuration.
- Semantic enrichment uses Python NLP or embedding libraries without
  embedding Python into Go services.
- Graph context lookup returns context without directly mutating canonical
  graph state.
- A multi-signal agreement creates an Insight Candidate with evidence.
- A high-confidence `critical` signal creates an Insight Candidate with an
  explicit bypass reason.
- Duplicate Insight Candidates are suppressed or merged within the configured
  time window.
- Plugin failure routes through retry or DLQ behavior with detector metadata.
- Go persistence and Engine adapter workers consume Python outputs and
  publish Engine-compatible Event Artifacts, Graph Mutation Proposals, and
  Insight Candidates.
- Existing document ingestion, retrieval, graph construction, and Ask
  workflows continue to operate unchanged.
