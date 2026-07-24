# Syncratic SignalOps Foundational Platform Primitives Specification

**Version:** 1.0  
**Status:** Code-agent implementation specification  
**Audience:** Platform engineers, backend engineers, algorithm engineers, data engineers, QA engineers, and operators  
**Primary subsystem:** Syncratic SignalOps  
**Initial validating application:** MarketOps  
**Future application domains:** CyberOps, FraudOps, IoTOps, CRM Intelligence, Supply Chain Intelligence, Infrastructure Operations, and other evidence-driven operational applications

---

## 1. Purpose

This specification defines the foundational platform primitives that MUST become the stable building blocks of Syncratic SignalOps.

SignalOps is a multi-use-case evidence-processing platform. It accepts source data, preserves immutable origin evidence, normalizes source-specific records into stable contracts, derives features and state, executes deterministic and algorithmic analysis, produces reviewable signals and insights, and proposes governed knowledge updates to the Syncratic Engine.

The primitives in this document MUST allow new operational intelligence applications to be implemented by supplying domain-specific schemas, feature definitions, state models, detectors, algorithms, policies, and presentation logic without rebuilding ingestion, lineage, replay, quality control, review, evidence, or materialization infrastructure.

MarketOps is the first validating use case. MarketOps MUST remain a consumer of these primitives rather than their owner.

---

## 2. Architectural Position

SignalOps MUST expose a stable platform model based on the following processing chain:

```text
Source
  |
  v
Dataset
  |
  v
Raw Event
  |
  v
Normalized Event
  |
  +--------------------------+
  |                          |
  v                          v
Feature                  Deterministic Detector
  |
  v
State
  |
  v
State Transition
  |
  +--------------------------+
  |                          |
  v                          v
Detector                 Algorithm
  |                          |
  +------------+-------------+
               |
               v
            Evidence
               |
               v
             Signal
               |
               v
            Artifact
               |
               v
            Proposal
               |
               v
        Review / Evaluation
               |
        +------+------+
        |             |
        v             v
     Insight       Materialization
        |
        v
      Outcome
```

These are logical contracts. A deployment MAY combine selected low-volume services, but it MUST preserve primitive ownership, durable boundaries, lineage, versioning, idempotency, and review controls.

---

## 3. Mandatory Platform Rules

### 3.1 Raw before derived

External or provider payloads MUST be retained in the Raw Event ledger before downstream processing treats them as durable evidence.

### 3.2 Normalized is the processing boundary

Feature builders, state builders, detectors, algorithms, signal evaluators, and application read models MUST consume normalized contracts or explicitly approved derived records. They MUST NOT depend directly on provider payload structure.

### 3.3 Derived data is immutable

Features, states, transitions, detector observations, algorithm results, evidence records, signals, artifacts, proposals, insights, and outcomes MUST be append-only logical records.

Corrections MUST create superseding versions rather than rewriting historical meaning.

### 3.4 Complete lineage

Every derived record MUST identify:

- tenant;
- application/domain;
- source dataset;
- source record identifiers;
- contract or definition version;
- processing run;
- observation time;
- processing time;
- correlation and causation identifiers;
- quality state;
- code or algorithm version where applicable.

### 3.5 Algorithms do not own the platform

Algorithms MUST NOT:

- ingest directly from external providers;
- bypass raw or normalized ledgers;
- own broker orchestration;
- alter canonical source evidence;
- directly create production signals;
- directly mutate the Syncratic knowledge graph;
- directly initiate external operational actions.

### 3.6 Evidence before claims

A Signal, Insight, or Proposal MUST reference sufficient evidence to reconstruct why it exists.

### 3.7 Quality fails closed

Missing, stale, partial, invalid, sparse, contradictory, or otherwise unusable input MUST NOT be silently converted into zero, false, normal, or low risk.

Unsupported claims MUST be blocked while underlying audit and result records remain retained.

### 3.8 Research is not automation

A state, transition, algorithm result, hypothesis, signal candidate, or insight is evidence for evaluation. It MUST NOT become an external action unless a separately approved action policy and integration explicitly authorize it.

### 3.9 Governed materialization

SignalOps proposes canonical changes. The Syncratic Engine or another explicitly approved authority validates and applies graph, artifact, or knowledge mutations.

### 3.10 Point-in-time correctness

Replay, historical evaluation, and backtesting MUST use only data and definitions available at the evaluated timestamp unless the run is explicitly identified as a retrospective reconstruction.

---

## 4. Cross-Cutting Identity and Scope

Every platform primitive MUST carry a common scope envelope.

```json
{
  "tenant_id": "tenant_123",
  "app_id": "marketops",
  "domain": "markets",
  "use_case": "daily_market_surveillance",
  "environment": "production",
  "correlation_id": "corr_01J...",
  "causation_id": "evt_01J...",
  "trace_id": "trace_01J...",
  "processing_run_id": "run_01J..."
}
```

### 4.1 Required scope fields

- `tenant_id`: security and data-isolation boundary.
- `app_id`: bounded SignalOps application.
- `domain`: broad operational domain.
- `use_case`: specific workflow or intelligence objective.
- `environment`: deployment environment.
- `correlation_id`: groups work across a business or processing flow.
- `causation_id`: identifies the direct triggering record.
- `trace_id`: distributed tracing identifier.
- `processing_run_id`: immutable worker, replay, backtest, or scheduled run identifier.

### 4.2 Identifier requirements

Primitive identifiers MUST be:

- globally unique within their primitive type;
- immutable;
- opaque to clients;
- sortable by creation time where practical;
- deterministic when idempotent reconstruction is required.

Preferred format: UUIDv7 or ULID with a primitive prefix.

Examples:

```text
src_01J...
dset_01J...
raw_01J...
norm_01J...
featdef_01J...
featobs_01J...
state_01J...
transition_01J...
detector_01J...
alg_01J...
algresult_01J...
evidence_01J...
signal_01J...
artifact_01J...
proposal_01J...
insight_01J...
outcome_01J...
```

---

## 5. Common Temporal Model

SignalOps MUST distinguish source time from system time.

Every temporal primitive MUST support the applicable fields:

- `occurred_at`: when the source-domain event occurred.
- `observed_at`: when the source or provider observed it.
- `received_at`: when SignalOps received it.
- `processed_at`: when a stage completed.
- `valid_from`: beginning of domain validity.
- `valid_to`: end of domain validity, nullable.
- `as_of_time`: point-in-time cutoff represented by a derived record.
- `session_date`: optional domain business date.
- `superseded_at`: time a semantic version was superseded.

Out-of-order arrival MUST NOT overwrite source chronology. State builders and transition engines MUST explicitly process lateness policy.

---

## 6. Common Quality Model

Every derived primitive MUST contain a quality envelope.

```json
{
  "quality_state": "usable",
  "quality_score": 0.97,
  "quality_dimensions": {
    "completeness": 1.0,
    "freshness": 0.98,
    "validity": 1.0,
    "coverage": 0.92,
    "consistency": 0.96
  },
  "quality_reasons": [],
  "quality_policy_version": "1.0.0"
}
```

### 6.1 Standard quality states

- `usable`
- `degraded`
- `partial`
- `stale`
- `missing`
- `invalid`
- `contradictory`
- `not_applicable`
- `suppressed`

### 6.2 Gating semantics

- `usable`: eligible for all configured downstream paths.
- `degraded`: eligible only when the consuming policy permits degraded input.
- `partial`: retained but blocked from claims requiring complete coverage.
- `stale`: retained but blocked from freshness-dependent claims.
- `missing`: no numeric or boolean substitute may be emitted.
- `invalid`: excluded from semantic processing.
- `contradictory`: requires reconciliation or explicit multi-source treatment.
- `not_applicable`: valid absence, distinct from missing.
- `suppressed`: intentionally blocked by policy while retained for audit.

---

## 7. Primitive Registry Architecture

SignalOps MUST provide registries for definitions whose semantics affect processing.

Required registries:

- Source Registry
- Dataset Registry
- Schema Registry
- Pipeline Registry
- Feature Registry
- State Model Registry
- Detector Registry
- Algorithm Registry
- Signal Definition Registry
- Policy Registry
- Artifact Type Registry
- Proposal Type Registry
- Insight Type Registry
- Outcome Definition Registry

All definitions MUST be versioned.

A published semantic version MUST be immutable. Changes MUST create a new version.

Definition lifecycle:

```text
draft -> validating -> active -> deprecated -> retired
```

Only `active` versions may be selected for normal production processing. Replay and historical evaluation MAY select deprecated versions explicitly.

---

# PART I — INGESTION AND EVIDENCE PRIMITIVES

## 8. Source Primitive

### 8.1 Purpose

A Source identifies an external producer, provider, system, operator, device group, data feed, or approved internal producer.

Examples:

- Massive market-data API;
- Salesforce production tenant;
- firewall event stream;
- industrial sensor gateway;
- weather provider;
- operator-uploaded incident feed.

### 8.2 Ownership

The Signal Gateway and Source Registry own Source records.

Domain applications MAY define source-specific adapters but MUST NOT implement independent authentication, idempotency, rate limiting, or tenant ownership models.

### 8.3 Source contract

```json
{
  "source_id": "src_01J...",
  "tenant_id": "tenant_123",
  "name": "massive-market-data-prod",
  "source_type": "market_data_provider",
  "provider": "massive",
  "auth_type": "secret_reference",
  "credential_ref": "vault://signalops/tenant_123/massive",
  "status": "active",
  "allowed_dataset_ids": ["dset_equity_daily", "dset_options_chain_eod"],
  "rate_limit_policy_id": "policy_01J...",
  "retention_policy_id": "policy_01K...",
  "metadata": {},
  "created_at": "2026-07-24T00:00:00Z",
  "updated_at": "2026-07-24T00:00:00Z"
}
```

### 8.4 Requirements

- Secret values MUST NOT be stored in the Source record.
- Source ownership MUST be verified against route tenant context.
- A source MUST list or inherit allowed datasets.
- Source status MUST be enforced before ingestion.
- Source disablement MUST stop new ingestion without deleting historical evidence.
- Source changes MUST create audit events.

### 8.5 APIs

```text
POST   /v1/tenants/{tenant_id}/sources
GET    /v1/tenants/{tenant_id}/sources
GET    /v1/tenants/{tenant_id}/sources/{source_id}
PATCH  /v1/tenants/{tenant_id}/sources/{source_id}
POST   /v1/tenants/{tenant_id}/sources/{source_id}/disable
POST   /v1/tenants/{tenant_id}/sources/{source_id}/enable
```

---

## 9. Dataset Primitive

### 9.1 Purpose

A Dataset defines a stable logical class of observations independent of provider representation.

Examples:

- `market.equity.daily_bar`
- `market.options.chain_eod`
- `cyber.network.connection`
- `crm.opportunity.change`
- `iot.device.telemetry`
- `operations.service_metric`

A Dataset is the contract boundary between source-specific normalization and reusable platform processing.

### 9.2 Dataset contract

```json
{
  "dataset_id": "dset_01J...",
  "dataset_key": "market.options.chain_eod",
  "version": "1.0.0",
  "domain": "markets",
  "record_type": "observation",
  "schema_id": "schema_01J...",
  "entity_types": ["asset", "option_contract"],
  "temporal_semantics": {
    "occurred_at_required": true,
    "session_date_supported": true
  },
  "normalizer_ref": "normalizers/market/options_chain_eod:v1",
  "default_pipeline_id": "pipeline_01J...",
  "quality_policy_id": "policy_01J...",
  "retention_policy_id": "policy_01J...",
  "status": "active"
}
```

### 9.3 Requirements

- Dataset keys MUST be globally unique within a SignalOps deployment.
- Provider-specific names MUST NOT leak into the canonical Dataset key.
- Breaking schema or semantic changes MUST create a new major version.
- A Dataset MUST declare entity, temporal, quality, retention, and normalization semantics.
- Domain workers MUST select processing by Dataset identity and version, not by provider name.

---

## 10. Raw Event Primitive

### 10.1 Purpose

A Raw Event is the immutable provider-shaped evidence accepted by SignalOps before domain normalization.

The Raw Event ledger is the authoritative replay and source-debug record.

### 10.2 Contract

```json
{
  "raw_event_id": "raw_01J...",
  "tenant_id": "tenant_123",
  "app_id": "marketops",
  "domain": "markets",
  "use_case": "daily_market_surveillance",
  "source_id": "src_01J...",
  "dataset_id": "dset_01J...",
  "external_event_id": "provider-event-123",
  "idempotency_key": "idem-123",
  "occurred_at": "2026-07-23T20:00:00Z",
  "observed_at": "2026-07-23T20:01:00Z",
  "received_at": "2026-07-23T20:01:02Z",
  "payload": {},
  "entity_hints": [],
  "provider_metadata": {},
  "schema_hint": "provider-v3",
  "payload_hash": "sha256:...",
  "correlation_id": "corr_01J...",
  "trace_id": "trace_01J...",
  "ingest_status": "accepted"
}
```

### 10.3 Requirements

- Raw Events MUST be immutable.
- Payload hash MUST be calculated before durable acceptance.
- Duplicate submission MUST return the original acceptance result.
- Idempotency scope MUST include tenant, source, dataset, external event identity, and idempotency key.
- Original provider values MUST be preserved.
- Redaction MAY create a protected storage representation, but redaction metadata MUST identify what policy was applied.
- Raw payloads MUST NOT be used as a general algorithm contract.
- Durable storage MUST complete before dependent worker offset acknowledgement.

### 10.4 Broker topic

```text
signalops.<env>.raw.v1
```

---

## 11. Normalized Event Primitive

### 11.1 Purpose

A Normalized Event is the immutable stable processing contract produced from a Raw Event.

### 11.2 Contract

```json
{
  "normalized_event_id": "norm_01J...",
  "raw_event_id": "raw_01J...",
  "tenant_id": "tenant_123",
  "app_id": "marketops",
  "domain": "markets",
  "use_case": "daily_market_surveillance",
  "source_id": "src_01J...",
  "dataset_id": "dset_01J...",
  "dataset_version": "1.0.0",
  "schema_version": "1.2.0",
  "event_type": "options.chain.observed",
  "occurred_at": "2026-07-23T20:00:00Z",
  "observed_at": "2026-07-23T20:01:00Z",
  "processed_at": "2026-07-23T20:01:04Z",
  "as_of_time": "2026-07-23T20:00:00Z",
  "entities": [],
  "normalized_payload": {},
  "validation": {
    "status": "valid",
    "errors": []
  },
  "quality": {},
  "normalizer_version": "1.3.0",
  "source_lineage": {
    "raw_event_ids": ["raw_01J..."]
  },
  "correlation_id": "corr_01J...",
  "causation_id": "raw_01J..."
}
```

### 11.3 Requirements

- Normalized Events MUST be derived only from durable Raw Events.
- Normalization failure MUST be persisted and observable.
- Invalid events MUST NOT enter feature, detector, or algorithm pipelines.
- Numeric zero, missing, and not applicable MUST remain distinguishable.
- The normalizer version MUST be recorded.
- Re-normalization MUST create a new Normalized Event version or superseding record.
- Multiple Raw Events MAY contribute to one normalized record only when the dataset contract explicitly permits aggregation.

### 11.4 Broker topic

```text
signalops.<env>.normalized.v1
```

---

# PART II — DERIVED KNOWLEDGE PRIMITIVES

## 12. Feature Primitive

### 12.1 Purpose

A Feature is a versioned, reusable measurement derived from normalized or approved derived evidence at a point in time.

A Feature consists of:

1. `FeatureDefinition` — semantic and computational contract.
2. `FeatureObservation` — immutable calculated value.

### 12.2 Feature Definition

```json
{
  "feature_definition_id": "featdef_01J...",
  "feature_key": "market.iv.atm.30d",
  "version": "1.0.0",
  "domain": "markets",
  "value_type": "decimal",
  "unit": "ratio",
  "description": "Interpolated at-the-money implied volatility near 30 DTE.",
  "required_dataset_versions": [
    {
      "dataset_key": "market.options.chain_eod",
      "version_range": "^1.0.0"
    }
  ],
  "computation_ref": "features/market/iv_atm_30d:v1",
  "window_definition": {},
  "missing_value_policy": "no_observation",
  "quality_policy_id": "policy_01J...",
  "status": "active"
}
```

### 12.3 Feature Observation

```json
{
  "feature_observation_id": "featobs_01J...",
  "feature_definition_id": "featdef_01J...",
  "feature_version": "1.0.0",
  "subject": {
    "entity_type": "asset",
    "entity_id": "AAPL"
  },
  "as_of_time": "2026-07-23T20:00:00Z",
  "session_date": "2026-07-23",
  "value": 0.284,
  "value_status": "present",
  "quality": {},
  "source_lineage": {
    "normalized_event_ids": ["norm_01J..."]
  },
  "computation": {
    "implementation_version": "git:abc123",
    "parameters": {}
  },
  "processing_run_id": "run_01J..."
}
```

### 12.4 Requirements

- Feature semantics MUST be registry-driven.
- Feature values MUST be deterministic for identical inputs, definition version, parameters, and point-in-time cutoff unless explicitly classified as stochastic.
- Features MUST distinguish present, missing, not applicable, invalid, and suppressed values.
- Feature definitions MUST specify windows, units, data types, and missing-value behavior.
- Features MUST be reusable across detectors and algorithms.
- Application-specific features MAY reside in application packages, but they MUST use the shared Feature contracts and registry.
- Feature observations MUST never be updated in place.

### 12.5 APIs

```text
GET  /v1/feature-definitions
GET  /v1/feature-definitions/{feature_key}/versions
GET  /v1/features?subject_id=&feature_key=&from=&to=
POST /v1/internal/feature-materialization-jobs
```

---

## 13. State Primitive

### 13.1 Purpose

A State is a canonical, versioned representation of the condition of an entity, system, relationship, or bounded context at a point in time.

Examples:

- market state for an asset;
- security posture for a host;
- operational health for a service;
- engagement state for a CRM opportunity;
- health state for an IoT device.

### 13.2 State Model Definition

```json
{
  "state_model_id": "statemodel_01J...",
  "state_key": "market.asset.daily_state",
  "version": "1.0.0",
  "domain": "markets",
  "subject_entity_type": "asset",
  "required_features": [
    "market.price.return_1d",
    "market.iv.atm.30d"
  ],
  "builder_ref": "states/market/asset_daily:v1",
  "schema": {},
  "lateness_policy": "rebuild_and_supersede",
  "status": "active"
}
```

### 13.3 State Record

```json
{
  "state_id": "state_01J...",
  "state_model_id": "statemodel_01J...",
  "state_model_version": "1.0.0",
  "subject": {
    "entity_type": "asset",
    "entity_id": "AAPL"
  },
  "as_of_time": "2026-07-23T20:00:00Z",
  "valid_from": "2026-07-23T20:00:00Z",
  "valid_to": null,
  "state_payload": {},
  "state_labels": ["high_iv", "positive_momentum"],
  "quality": {},
  "feature_observation_ids": ["featobs_01J..."],
  "source_lineage": {},
  "processing_run_id": "run_01J..."
}
```

### 13.4 Requirements

- A State MUST declare its model and model version.
- State MUST be built from point-in-time eligible inputs.
- State builders MUST not infer unavailable source evidence.
- Late data behavior MUST be defined per model.
- Recalculated historical State MUST supersede rather than overwrite.
- State labels MUST be derived, explainable, and versioned.
- State is not automatically a Signal.

---

## 14. State Transition Primitive

### 14.1 Purpose

A State Transition represents a meaningful change between states.

It enables SignalOps to prioritize change, acceleration, persistence, regime shifts, migration, concentration changes, and divergence rather than relying only on static values.

### 14.2 Contract

```json
{
  "transition_id": "transition_01J...",
  "state_model_id": "statemodel_01J...",
  "subject": {
    "entity_type": "asset",
    "entity_id": "AAPL"
  },
  "from_state_id": "state_01H...",
  "to_state_id": "state_01J...",
  "transition_type": "regime_change",
  "as_of_time": "2026-07-23T20:00:00Z",
  "change_payload": {
    "level_change": 0.04,
    "percent_change": 0.16,
    "z_score": 2.3,
    "percentile": 0.97,
    "acceleration": 0.8,
    "persistence_periods": 3
  },
  "quality": {},
  "definition_version": "1.0.0",
  "source_lineage": {}
}
```

### 14.3 Supported transition classes

- level change;
- percentage change;
- threshold crossing;
- acceleration or deceleration;
- persistence;
- reversal;
- regime change;
- cross-bucket migration;
- concentration shift;
- divergence;
- convergence;
- corroboration;
- resolution.

### 14.4 Requirements

- Transition definitions MUST be versioned.
- Comparators and baseline windows MUST be explicit.
- A missing prior state MUST produce an explicit non-comparable condition.
- State Transition records MUST be immutable.
- Transitions MAY feed detectors and algorithms but MUST NOT bypass Signal evaluation.

---

# PART III — ANALYTICAL PRIMITIVES

## 15. Detector Primitive

### 15.1 Purpose

A Detector is deterministic or bounded declarative logic that identifies a candidate condition from normalized events, features, states, or transitions.

A Detector produces a `DetectorObservation`, not a production Signal.

### 15.2 Detector Definition

```json
{
  "detector_id": "detector_01J...",
  "detector_key": "market.iv_expansion",
  "version": "1.0.0",
  "domain": "markets",
  "input_contract": {
    "features": ["market.iv.atm.30d"],
    "transitions": ["market.iv.atm.30d.change"]
  },
  "evaluation_ref": "detectors/market/iv_expansion:v1",
  "parameters": {
    "minimum_z_score": 2.0,
    "minimum_persistence": 2
  },
  "required_quality_states": ["usable", "degraded"],
  "status": "active"
}
```

### 15.3 Detector Observation

```json
{
  "detector_observation_id": "detobs_01J...",
  "detector_id": "detector_01J...",
  "detector_version": "1.0.0",
  "subject": {},
  "as_of_time": "2026-07-23T20:00:00Z",
  "matched": true,
  "severity": "medium",
  "score": 0.82,
  "confidence": 0.91,
  "result_payload": {},
  "quality": {},
  "evidence_ids": [],
  "source_lineage": {},
  "processing_run_id": "run_01J..."
}
```

### 15.4 Requirements

- Detector execution MUST be reproducible.
- Detector thresholds and policies MUST be versioned.
- Detector output MUST retain matched and non-matched results when configured for research or calibration.
- Detectors MUST not write Signal, Artifact, Proposal, Insight, graph, or action state directly.
- A detector MUST declare accepted quality states.
- Rules and detectors MAY share a common runtime, but their contracts MUST remain explicit.

---

## 16. Algorithm Primitive

### 16.1 Purpose

An Algorithm is a versioned statistical, temporal, clustering, machine-learning, scoring, or analytical component operating behind a platform contract.

Algorithms produce immutable Algorithm Results.

### 16.2 Algorithm Definition

```json
{
  "algorithm_id": "alg_01J...",
  "algorithm_key": "platform.change_point_detection",
  "version": "1.0.0",
  "runtime": "python",
  "container_image": "registry/signalops-algorithms:1.0.0",
  "entrypoint": "signalops.algorithms.change_point",
  "input_contract_version": "1.0.0",
  "output_contract_version": "1.0.0",
  "required_features": [],
  "parameters_schema": {},
  "determinism": "deterministic",
  "resource_profile": "cpu-medium",
  "timeout_seconds": 120,
  "quality_policy_id": "policy_01J...",
  "status": "active"
}
```

### 16.3 Execution Request

```json
{
  "algorithm_execution_id": "algexec_01J...",
  "algorithm_id": "alg_01J...",
  "algorithm_version": "1.0.0",
  "subject": {},
  "window": {
    "from": "2026-06-01T00:00:00Z",
    "to": "2026-07-23T20:00:00Z",
    "point_in_time_cutoff": "2026-07-23T20:00:00Z"
  },
  "feature_vector_contract": "market.asset.temporal.v1",
  "feature_observation_ids": [],
  "parameters": {},
  "mode": "production_evaluation",
  "processing_run_id": "run_01J..."
}
```

### 16.4 Algorithm Result

```json
{
  "algorithm_result_id": "algresult_01J...",
  "algorithm_execution_id": "algexec_01J...",
  "algorithm_id": "alg_01J...",
  "algorithm_version": "1.0.0",
  "subject": {},
  "as_of_time": "2026-07-23T20:00:00Z",
  "score": 0.78,
  "confidence": 0.84,
  "severity": "medium",
  "result_payload": {},
  "quality": {},
  "runtime_metadata": {
    "image_digest": "sha256:...",
    "duration_ms": 843,
    "seed": null
  },
  "source_lineage": {},
  "created_at": "2026-07-23T20:02:00Z"
}
```

### 16.5 Requirements

- Algorithm execution MUST use persisted inputs.
- Runtime package and image digest MUST be retained.
- Stochastic algorithms MUST record random seed or reproducibility limitations.
- Results MUST be retained even when a later quality gate blocks production use.
- Algorithms MUST be isolated from ingestion credentials.
- Algorithm timeouts and resource limits MUST be enforced.
- Algorithm definitions MUST support Python initially and bounded Go implementations where justified.
- Research, backtest, shadow, and production evaluation modes MUST be isolated.
- Algorithm output MUST not directly create production Signal state.

### 16.6 Broker topics

```text
signalops.<env>.algorithm.request.v1
signalops.<env>.algorithm.result.v1
signalops.<env>.retry.algorithm.v1
signalops.<env>.dlq.algorithm.v1
```

---

## 17. Evidence Primitive

### 17.1 Purpose

Evidence is the auditable bridge between source records and claims.

An Evidence record references facts, observations, excerpts, feature values, state changes, detector outcomes, algorithm results, or approved external context.

### 17.2 Contract

```json
{
  "evidence_id": "evidence_01J...",
  "evidence_type": "state_transition",
  "subject": {},
  "summary": "ATM 30D implied volatility increased to the 97th trailing percentile.",
  "claim_scope": "market_volatility_expansion",
  "as_of_time": "2026-07-23T20:00:00Z",
  "source_refs": [
    {
      "primitive_type": "transition",
      "primitive_id": "transition_01J..."
    }
  ],
  "payload_ref": "timescale://...",
  "excerpt": {},
  "quality": {},
  "access_policy_id": "policy_01J...",
  "lineage_hash": "sha256:..."
}
```

### 17.3 Requirements

- Evidence MUST be independently addressable.
- Evidence MUST retain source references rather than copying uncontrolled payloads into every downstream record.
- Evidence access MUST enforce tenant and data classification policy.
- Evidence summaries MUST not overstate underlying records.
- Evidence must remain reconstructable when semantic indexes are rebuilt.
- Qdrant MAY index Evidence summaries but MUST NOT become canonical evidence storage.
- Every Signal and Insight MUST reference Evidence IDs.

---

# PART IV — CLAIM AND GOVERNANCE PRIMITIVES

## 18. Signal Primitive

### 18.1 Purpose

A Signal is a versioned, evidence-backed observation that a defined condition exists and is eligible for governed downstream evaluation.

A Signal is not necessarily an alert, recommendation, graph mutation, or external action.

### 18.2 Signal Definition

```json
{
  "signal_definition_id": "signaldef_01J...",
  "signal_key": "market.downside_hedging_expansion",
  "version": "1.0.0",
  "domain": "markets",
  "description": "Material expansion in downside hedging evidence.",
  "required_evidence_types": [
    "feature_observation",
    "state_transition"
  ],
  "optional_corroboration": [
    "algorithm_result"
  ],
  "evaluation_ref": "signals/market/downside_hedging_expansion:v1",
  "quality_policy_id": "policy_01J...",
  "confidence_policy_id": "policy_01J...",
  "lifecycle_policy_id": "policy_01J...",
  "status": "active"
}
```

### 18.3 Signal Record

```json
{
  "signal_id": "signal_01J...",
  "signal_definition_id": "signaldef_01J...",
  "signal_version": "1.0.0",
  "subject": {},
  "as_of_time": "2026-07-23T20:00:00Z",
  "signal_status": "candidate",
  "direction": "negative",
  "severity": "high",
  "score": 0.86,
  "confidence": 0.83,
  "summary": "Downside hedging expanded while price momentum remained overbought.",
  "evidence_ids": ["evidence_01J..."],
  "detector_observation_ids": [],
  "algorithm_result_ids": [],
  "quality": {},
  "source_event_ids": [],
  "correlation_id": "corr_01J..."
}
```

### 18.4 Signal lifecycle

```text
candidate
  -> evaluating
  -> accepted
  -> active
  -> strengthening
  -> stable
  -> weakening
  -> resolved
  -> archived
```

Alternative terminal states:

```text
rejected
dismissed
suppressed
expired
invalidated
```

### 18.5 Requirements

- Signal creation MUST be controlled by a Signal Definition.
- Signal confidence MUST reflect evidence support, quality, corroboration, and calibration—not a single raw model score.
- Signal lifecycle transitions MUST be policy-controlled and auditable.
- Duplicate evidence for the same subject, definition, and observation window MUST be correlation-controlled.
- Signals MUST not automatically mutate canonical graph state.
- An accepted Signal MAY be converted into an Artifact or Proposal according to policy.

---

## 19. Artifact Primitive

### 19.1 Purpose

An Artifact is the stable Engine-consumable representation of an operational observation or derived Signal.

Artifacts provide a common boundary between SignalOps applications and the Syncratic Engine.

### 19.2 Contract

```json
{
  "artifact_id": "artifact_01J...",
  "artifact_type": "signal_observation",
  "artifact_schema_version": "1.0.0",
  "tenant_id": "tenant_123",
  "app_id": "marketops",
  "domain": "markets",
  "subject": {},
  "title": "AAPL downside hedging expansion",
  "summary": "Evidence indicates expanding downside protection demand.",
  "occurred_at": "2026-07-23T20:00:00Z",
  "observed_at": "2026-07-23T20:01:00Z",
  "entities": [],
  "confidence": 0.83,
  "quality": {},
  "evidence_ids": [],
  "source_primitive_refs": [
    {
      "primitive_type": "signal",
      "primitive_id": "signal_01J..."
    }
  ],
  "correlation_id": "corr_01J..."
}
```

### 19.3 Requirements

- Artifact IDs MUST be stable for the same semantic source identity and artifact schema version.
- Artifact creation MUST be idempotent.
- Artifacts MUST retain Evidence references.
- Artifact registration in the Engine MUST be governed by an adapter contract.
- Artifact content MUST not expose unauthorized raw tenant payloads.
- Artifact updates MUST create new versions or use an Engine-approved revision model.

### 19.4 Broker topic

```text
signalops.<env>.artifact.v1
```

---

## 20. Proposal Primitive

### 20.1 Purpose

A Proposal is a governed request to materialize, relate, classify, publish, or otherwise apply a derived claim.

Proposal types include:

- graph mutation;
- artifact registration;
- insight publication;
- signal materialization;
- entity resolution update;
- temporal relationship update;
- confidence update;
- relationship retirement.

### 20.2 Contract

```json
{
  "proposal_id": "proposal_01J...",
  "proposal_type": "graph_mutation",
  "proposal_schema_version": "1.0.0",
  "tenant_id": "tenant_123",
  "app_id": "marketops",
  "subject": {},
  "operation": {
    "mutation_type": "create_temporal_edge",
    "predicate": "HAS_MARKET_STATE",
    "object": {}
  },
  "valid_from": "2026-07-23T20:00:00Z",
  "valid_to": null,
  "confidence": 0.83,
  "quality": {},
  "evidence_ids": [],
  "artifact_ids": [],
  "source_signal_ids": [],
  "proposal_status": "pending_review",
  "review_policy_id": "policy_01J...",
  "idempotency_key": "proposal:...",
  "correlation_id": "corr_01J..."
}
```

### 20.3 Proposal lifecycle

```text
draft
 -> quality_check
 -> pending_review
 -> accepted
 -> materializing
 -> materialized
```

Alternative states:

```text
rejected
needs_information
retryable_failure
failed
withdrawn
expired
superseded
```

### 20.4 Requirements

- Proposal creation MUST not equal materialization.
- The authoritative receiving system MUST return accepted, rejected, pending, or retryable failure status.
- Proposal decisions MUST include actor, reason, timestamp, and policy version.
- Invalid proposals MUST remain auditable.
- Retry after authoritative acceptance MUST use idempotency to prevent duplicate side effects.
- Application code MUST not bypass Proposal review by directly writing to Neo4j or canonical Engine graph tables.

### 20.5 Broker topic

```text
signalops.<env>.proposal.v1
signalops.<env>.proposal.result.v1
```

---

## 21. Insight Primitive

### 21.1 Purpose

An Insight is a human-consumable, evidence-backed interpretation produced after evaluation.

An Insight is not automatically an external action or recommendation.

### 21.2 Contract

```json
{
  "insight_id": "insight_01J...",
  "insight_type": "operational_observation",
  "insight_schema_version": "1.0.0",
  "tenant_id": "tenant_123",
  "app_id": "marketops",
  "subject": {},
  "title": "Downside hedging expanded despite positive price momentum",
  "summary": "Options evidence shows elevated downside protection demand while the underlying remains overbought.",
  "interpretation": "This combination warrants analyst review for possible downside risk or volatility expansion.",
  "limitations": [
    "Open-interest changes do not identify participant intent."
  ],
  "confidence": 0.81,
  "severity": "high",
  "quality": {},
  "evidence_ids": [],
  "signal_ids": [],
  "artifact_ids": [],
  "proposal_ids": [],
  "insight_status": "published",
  "created_at": "2026-07-23T20:05:00Z"
}
```

### 21.3 Insight lifecycle

```text
candidate -> evaluating -> accepted -> published -> archived
```

Alternative states:

```text
dismissed
suppressed
invalidated
superseded
```

### 21.4 Requirements

- Insight text MUST remain faithful to Evidence.
- Known limitations MUST be preserved.
- Generated interpretation MUST not introduce unsupported claims.
- Insight publication MUST respect application policy.
- Ask context MUST provide evidence separately from interpretation.
- Insights SHOULD be regenerable from accepted evidence and definition versions.

---

## 22. Outcome Primitive

### 22.1 Purpose

An Outcome records what occurred after a Signal, Insight, Proposal, decision, or materialized condition.

Outcome data enables calibration, effectiveness measurement, supervised learning, audit, and hypothesis evaluation.

### 22.2 Outcome Definition

```json
{
  "outcome_definition_id": "outcomedef_01J...",
  "outcome_key": "market.forward_return_5d",
  "version": "1.0.0",
  "subject_type": "asset",
  "anchor_primitive_types": ["signal", "insight"],
  "horizon": "P5D",
  "calculation_ref": "outcomes/market/forward_return:v1",
  "point_in_time_policy": "strict",
  "status": "active"
}
```

### 22.3 Outcome Record

```json
{
  "outcome_id": "outcome_01J...",
  "outcome_definition_id": "outcomedef_01J...",
  "outcome_version": "1.0.0",
  "anchor": {
    "primitive_type": "signal",
    "primitive_id": "signal_01J..."
  },
  "subject": {},
  "evaluation_time": "2026-07-30T20:00:00Z",
  "outcome_payload": {},
  "success_label": null,
  "quality": {},
  "source_lineage": {},
  "processing_run_id": "run_01J..."
}
```

### 22.4 Requirements

- Outcome definitions MUST identify horizon and calculation semantics.
- Missing future evidence MUST remain missing.
- Later corrections MUST create superseding Outcome versions.
- Outcome evaluation MUST not alter the original Signal or Insight.
- Calibration jobs MUST operate on immutable Outcome and source records.
- Domain applications MUST define success semantics; the platform provides the common contract and execution framework.

---

# PART V — SUPPORTING PLATFORM SERVICES

## 23. Pipeline Primitive

A Pipeline defines an ordered, versioned set of stages and routing policies.

```json
{
  "pipeline_id": "pipeline_01J...",
  "pipeline_key": "market.options_eod",
  "version": "1.0.0",
  "input_dataset_id": "dset_01J...",
  "stages": [
    {"type": "normalize", "ref": "normalizers/market/options:v1"},
    {"type": "feature", "ref": "feature-pack/market/options:v1"},
    {"type": "state", "ref": "states/market/daily:v1"},
    {"type": "detector", "ref": "detectors/market/options:v1"},
    {"type": "algorithm", "ref": "algorithms/platform/change-point:v1"},
    {"type": "signal", "ref": "signals/market/hypothesis-pack:v1"}
  ],
  "error_policy_id": "policy_01J...",
  "replay_policy_id": "policy_01J...",
  "status": "active"
}
```

Requirements:

- Stages MUST reference versioned contracts.
- Pipeline versions MUST be immutable once active.
- Replay MUST identify the selected Pipeline version.
- Application pipelines MUST use shared stage interfaces.
- A Pipeline MUST not hide direct graph or action side effects.

---

## 24. Policy Primitive

Policies separate runtime governance from domain logic.

Required policy categories:

- quality gates;
- confidence aggregation;
- retention;
- rate limits;
- retry and DLQ;
- lateness and out-of-order handling;
- signal lifecycle;
- suppression and deduplication;
- review;
- materialization;
- data access;
- privacy and redaction;
- replay;
- backtest isolation;
- escalation;
- maintenance windows;
- holidays and domain calendars.

Policy evaluation MUST be versioned, deterministic where practical, and auditable.

---

## 25. Confidence Aggregation

SignalOps MUST treat confidence as a structured result rather than an arbitrary scalar.

A confidence calculation SHOULD consider:

- evidence quality;
- evidence independence;
- detector confidence;
- algorithm confidence;
- agreement or disagreement;
- historical calibration;
- source reliability;
- temporal freshness;
- coverage;
- persistence;
- contradiction penalties.

```json
{
  "confidence": 0.83,
  "confidence_breakdown": {
    "evidence_quality": 0.91,
    "detector_support": 0.86,
    "algorithm_support": 0.78,
    "source_reliability": 0.95,
    "correlation_penalty": 0.08,
    "contradiction_penalty": 0.0
  },
  "confidence_policy_version": "1.0.0"
}
```

Correlated evidence MUST NOT be counted as fully independent support.

---

## 26. Lineage Service

The Lineage Service MUST provide traversal across:

```text
Source
 -> Dataset
 -> Raw Event
 -> Normalized Event
 -> Feature
 -> State
 -> Transition
 -> Detector Observation
 -> Algorithm Result
 -> Evidence
 -> Signal
 -> Artifact
 -> Proposal
 -> Insight
 -> Outcome
```

Required operations:

```text
GET /v1/lineage/{primitive_type}/{primitive_id}/ancestors
GET /v1/lineage/{primitive_type}/{primitive_id}/descendants
GET /v1/lineage/{primitive_type}/{primitive_id}/graph
GET /v1/lineage/{primitive_type}/{primitive_id}/explain
```

Lineage graphs MUST be tenant-filtered and access-policy-filtered.

---

## 27. Replay and Backtest Service

Replay MUST use persisted records rather than hidden provider re-fetch.

Replay modes:

- re-normalize;
- re-materialize features;
- rebuild states;
- re-run detectors;
- re-run algorithms;
- re-evaluate signals;
- reconstruct artifacts or proposals;
- full pipeline replay.

Backtest requirements:

- isolated run identity;
- isolated result namespace or storage scope;
- strict point-in-time cutoff;
- selected definition versions;
- no mutation of production ledgers;
- no production proposal materialization;
- complete run manifest;
- reproducible parameters.

---

## 28. Materialization Service

The Materialization Service submits accepted proposals to authoritative systems.

It MUST:

- enforce idempotency;
- persist request and response;
- classify accepted, rejected, pending, and retryable states;
- preserve authoritative external identifiers;
- retry only retryable failures;
- prevent application workers from directly mutating canonical graph state;
- emit audit, metrics, and trace data.

---

## 29. Semantic Index Service

Qdrant or another vector store MAY contain:

- Evidence summaries;
- Signal summaries;
- Insight summaries;
- Artifact embeddings;
- semantic clusters.

The semantic index MUST:

- be rebuildable from canonical ledgers;
- retain primitive and tenant references;
- enforce tenant partitioning;
- never become authoritative event truth;
- never be the only location of evidence required for audit.

---

# PART VI — STORAGE, BROKER, AND SERVICE OWNERSHIP

## 30. Canonical Storage Ownership

### PostgreSQL

Canonical for:

- registries;
- definitions and versions;
- source metadata;
- tenant policies;
- pipeline definitions;
- idempotency;
- replay jobs;
- review decisions;
- audit metadata;
- operational configuration.

### TimescaleDB

Canonical for:

- Raw Event ledger;
- Normalized Event ledger;
- Feature Observations;
- State history;
- State Transitions;
- Detector Observations;
- Algorithm execution and result history;
- time-based Evidence;
- Signal history;
- Outcome history;
- replay cursors and time-series rollups.

A deployment MAY use PostgreSQL with TimescaleDB extension as one physical cluster while preserving logical ownership.

### Syncratic Engine graph storage

Canonical for Engine-approved:

- entities;
- relationships;
- temporal graph state;
- applied graph mutations;
- Engine-owned artifact references.

SignalOps MUST access this state only through approved extension APIs.

### Qdrant

Non-canonical derived semantic index.

---

## 31. Broker Standard

Redpanda SHOULD be the initial SignalOps broker implementation.

SignalOps MUST use Kafka-compatible interfaces and MUST NOT bind core processing to Redpanda-only APIs.

Required topic families:

```text
signalops.<env>.raw.v1
signalops.<env>.normalized.v1
signalops.<env>.feature.v1
signalops.<env>.state.v1
signalops.<env>.transition.v1
signalops.<env>.detector.result.v1
signalops.<env>.algorithm.request.v1
signalops.<env>.algorithm.result.v1
signalops.<env>.signal.v1
signalops.<env>.artifact.v1
signalops.<env>.proposal.v1
signalops.<env>.proposal.result.v1
signalops.<env>.insight.v1
signalops.<env>.outcome.v1
signalops.<env>.retry.<stage>.v1
signalops.<env>.dlq.<stage>.v1
```

Topic payloads MUST use versioned envelopes.

Partition keys SHOULD use:

```text
tenant_id:dataset_id:subject_key
```

or, when subject is unavailable:

```text
tenant_id:source_id
```

---

# PART VII — API CONTRACTS

## 32. Registry APIs

```text
POST /v1/datasets
GET  /v1/datasets
GET  /v1/datasets/{dataset_key}/versions

POST /v1/pipelines
GET  /v1/pipelines
GET  /v1/pipelines/{pipeline_key}/versions

POST /v1/feature-definitions
GET  /v1/feature-definitions

POST /v1/state-models
GET  /v1/state-models

POST /v1/detectors
GET  /v1/detectors

POST /v1/algorithms
GET  /v1/algorithms

POST /v1/signal-definitions
GET  /v1/signal-definitions

POST /v1/outcome-definitions
GET  /v1/outcome-definitions
```

Definition creation and activation SHOULD be restricted to internal administrative roles.

## 33. Operational Read APIs

```text
GET /v1/raw-events
GET /v1/normalized-events
GET /v1/features
GET /v1/states
GET /v1/transitions
GET /v1/detector-observations
GET /v1/algorithm-executions
GET /v1/algorithm-results
GET /v1/evidence
GET /v1/signals
GET /v1/artifacts
GET /v1/proposals
GET /v1/insights
GET /v1/outcomes
```

All collection APIs MUST support tenant enforcement, bounded pagination, temporal filters, subject filters, definition/version filters, quality filters, and correlation filters.

## 34. Job APIs

```text
POST /v1/replay-jobs
GET  /v1/replay-jobs/{job_id}

POST /v1/feature-materialization-jobs
POST /v1/state-build-jobs
POST /v1/detector-execution-jobs
POST /v1/algorithm-execution-jobs
POST /v1/signal-evaluation-jobs
POST /v1/outcome-evaluation-jobs
```

Production job submission MUST be policy-controlled.

---

# PART VIII — SECURITY AND MULTI-TENANCY

## 35. Security Requirements

- Tenant isolation MUST apply at API, broker, storage, cache, semantic index, logs, traces, metrics, and Engine adapter boundaries.
- Every record MUST include `tenant_id`.
- Database row-level security SHOULD be enabled where supported.
- Broker ACLs MUST restrict producer and consumer identities.
- Source credentials MUST reside in an approved secret manager.
- Sensitive payloads MUST be encrypted at rest.
- PII, secrets, tokens, and regulated values MUST be redacted from logs.
- Evidence access MUST honor classification and field-level access policy.
- Replay and backtest jobs MUST not weaken access controls.
- Administrative registry mutations MUST be audited.
- Cross-tenant correlation is prohibited unless a separately approved privacy-preserving contract exists.

---

# PART IX — OBSERVABILITY

## 36. Required Metrics

Per primitive and stage:

- records accepted;
- records rejected;
- processing latency;
- event-time lag;
- broker lag;
- retry count;
- DLQ count;
- quality-state distribution;
- lineage gaps;
- idempotency hits;
- materialization status;
- active definition versions;
- replay progress;
- algorithm duration and resource usage;
- signal acceptance/rejection rates;
- outcome availability and evaluation lag.

High-cardinality identifiers MUST NOT be used as unbounded metric labels.

## 37. Required Traces

End-to-end traces MUST cover:

```text
Gateway
 -> Raw Event write
 -> broker publish
 -> normalization
 -> derived processing
 -> detector/algorithm execution
 -> evidence
 -> signal evaluation
 -> artifact/proposal
 -> authoritative response
```

## 38. Required Audit Events

- Source created, changed, disabled, or enabled.
- Dataset or definition version created or activated.
- Raw Event accepted or rejected.
- Normalization failed.
- Replay or backtest started and completed.
- Quality gate blocked a downstream claim.
- Signal accepted, rejected, suppressed, or resolved.
- Proposal created and reviewed.
- Materialization accepted or rejected.
- Insight published or invalidated.
- Outcome calculated or superseded.

---

# PART X — CODE ORGANIZATION

## 39. Recommended Repository Structure

```text
signalops/
  cmd/
    gateway/
    worker/
    scheduler/
    materializer/

  internal/
    platform/
      identity/
      tenancy/
      idempotency/
      broker/
      lineage/
      quality/
      policy/
      audit/
      replay/
      materialization/

    primitives/
      source/
      dataset/
      raw_event/
      normalized_event/
      feature/
      state/
      transition/
      detector/
      algorithm/
      evidence/
      signal/
      artifact/
      proposal/
      insight/
      outcome/

    registries/
      schema/
      pipeline/
      feature/
      state_model/
      detector/
      algorithm/
      signal/
      policy/
      outcome/

    adapters/
      engine/
      qdrant/
      postgres/
      timescale/
      redpanda/
      kafka/

    applications/
      marketops/
        datasets/
        normalizers/
        features/
        states/
        detectors/
        algorithms/
        signals/
        outcomes/
      cyberops/
      crmops/
      iotops/

  contracts/
    jsonschema/
    protobuf/
    openapi/

  migrations/
  helm/
  dashboards/
  tests/
```

### 39.1 Boundary rule

Code under `applications/<app>` MAY implement domain semantics but MUST depend on shared primitive interfaces.

Shared platform packages MUST NOT import application packages.

---

# PART XI — IMPLEMENTATION PHASES

## 40. Phase 1: Contract Foundation

Implement:

- common scope envelope;
- temporal model;
- quality model;
- identifier conventions;
- Source and Dataset registries;
- Raw and Normalized Event contracts;
- JSON Schema or Protobuf definitions;
- contract validation tests.

Exit criteria:

- MarketOps source and datasets can be represented without provider-specific fields leaking into normalized processing contracts.

## 41. Phase 2: Derived Primitives

Implement:

- Feature Definition and Observation;
- State Model and State;
- State Transition;
- lineage references;
- immutable TimescaleDB storage;
- read APIs.

Exit criteria:

- MarketOps daily feature, market state, and transition records use shared primitives.

## 42. Phase 3: Analytical Runtime

Implement:

- Detector registry and observations;
- Algorithm registry, execution requests, runtime adapter, and result ledger;
- Python runtime container;
- quality gates;
- broker topics and retry/DLQ handling.

Exit criteria:

- A deterministic MarketOps detector and a reusable platform algorithm execute from persisted inputs and produce immutable results.

## 43. Phase 4: Claims and Governance

Implement:

- Evidence;
- Signal Definition and Signal;
- confidence aggregation;
- Artifact;
- Proposal;
- review lifecycle;
- Engine adapters.

Exit criteria:

- MarketOps evidence produces a Signal candidate, accepted Artifact, and auditable graph Proposal without direct graph mutation.

## 44. Phase 5: Insight and Outcome Loop

Implement:

- Insight;
- Outcome Definition and Outcome;
- evaluation jobs;
- Ask context bundle;
- calibration read models.

Exit criteria:

- An accepted MarketOps Signal produces a bounded Insight and later receives a point-in-time-correct Outcome.

## 45. Phase 6: Second-Domain Validation

Implement one non-market vertical slice, preferably:

- security telemetry anomaly;
- CRM opportunity-risk transition; or
- IoT device-health state.

Exit criteria:

- The second domain uses the same registries, ledgers, pipeline stages, lineage, quality, detector/algorithm runtime, Signal, Proposal, Insight, and Outcome contracts without modifying foundational primitive schemas.

---

# PART XII — TESTING REQUIREMENTS

## 46. Contract Tests

Each primitive MUST have:

- schema validation;
- required-field tests;
- enum tests;
- semantic version compatibility tests;
- serialization round-trip tests;
- tenant scope tests;
- temporal-field tests;
- quality-state tests.

## 47. Idempotency Tests

Verify:

- duplicate raw ingestion produces one durable Raw Event;
- replay does not duplicate Feature Observations;
- repeated Algorithm Result delivery does not duplicate Signal effects;
- accepted Proposal retry does not duplicate authoritative mutation;
- duplicate Outcome jobs produce one logical Outcome version.

## 48. Lineage Tests

For a materialized Proposal, the system MUST reconstruct:

```text
Proposal
 <- Artifact
 <- Signal
 <- Evidence
 <- Detector Observation / Algorithm Result
 <- Transition / State / Feature
 <- Normalized Event
 <- Raw Event
 <- Dataset
 <- Source
```

Missing required lineage MUST fail the test.

## 49. Quality Tests

Verify that:

- missing input does not become zero;
- stale data cannot satisfy freshness-required Signal policies;
- partial coverage blocks complete-coverage claims;
- contradictory evidence reduces confidence or triggers review;
- unusable algorithm input retains a result record but blocks Signal materialization.

## 50. Replay and Point-in-Time Tests

Verify that:

- replay uses persisted evidence;
- historical runs cannot access future Feature Observations;
- backtest records are isolated from production;
- selected definition versions are recorded;
- re-normalization creates superseding records.

## 51. Multi-Tenant Tests

Verify that one tenant cannot:

- read another tenant's primitives;
- submit jobs over another tenant's records;
- use another tenant's Source;
- resolve another tenant's Evidence reference;
- access another tenant's semantic index entries;
- submit Proposals into another tenant's Engine scope.

## 52. Failure Tests

Verify:

- broker interruption;
- database interruption;
- algorithm timeout;
- malformed runtime output;
- Engine retryable failure;
- Engine rejection;
- DLQ routing;
- worker restart after durable write but before offset commit;
- Qdrant failure after canonical writes.

---

# PART XIII — ACCEPTANCE CRITERIA

The foundational platform implementation is acceptable when all of the following are true:

1. A Source and Dataset can be registered with immutable, versioned semantics.
2. A provider payload is stored as one idempotent Raw Event before downstream processing.
3. The Raw Event is normalized into a stable Dataset contract with explicit validation and quality state.
4. A Feature Observation can be reproduced from persisted normalized inputs and a Feature Definition version.
5. A point-in-time State can be built from versioned Feature Observations.
6. A State Transition can express a meaningful change without mutating either State.
7. A Detector can produce an immutable Detector Observation without directly creating a production Signal.
8. A Python Algorithm can execute from persisted feature vectors and produce an immutable, reproducible Algorithm Result.
9. Evidence can reference Feature, State, Transition, Detector, and Algorithm records while preserving tenant-safe lineage.
10. A Signal Definition can combine eligible Evidence into a candidate Signal through versioned quality and confidence policies.
11. A Signal cannot bypass quality gates, review policy, or Proposal workflow.
12. An Artifact can be created idempotently and submitted through the Engine Artifact API.
13. A Proposal can be accepted, rejected, pending, or retryable and remains fully auditable.
14. SignalOps does not directly mutate canonical Neo4j graph state.
15. An Insight can be generated without introducing unsupported claims and includes limitations and Evidence references.
16. An Outcome can be calculated at a declared horizon without modifying the originating Signal.
17. Complete lineage can be traversed from an Outcome or Proposal back to Source and Raw Event.
18. Replay rebuilds derived records from persisted ledgers without provider re-fetch.
19. Backtesting is point-in-time correct and isolated from production ledgers.
20. Tenant isolation is enforced across all primitives and services.
21. MarketOps can implement its feature, market-state, transition, hypothesis/signal, opportunity/insight, and outcome workflows using these shared contracts.
22. A second application domain can be added without changing the foundational primitive schemas.
23. Existing Syncratic document ingestion, retrieval, graph construction, Ask workflows, and Engine authority remain unchanged.

---

# PART XIV — CODE-AGENT EXECUTION RULES

The code agent MUST:

1. Treat this specification as an additive SignalOps platform implementation.
2. Preserve existing working MarketOps and SignalOps flows.
3. Inspect the current repository before creating duplicate services, tables, or contracts.
4. Reuse existing idempotency, broker, ledger, proposal-review, graph-review, materialization, quality, and evidence-purity mechanisms where they already exist.
5. Implement database migrations incrementally.
6. Avoid destructive schema changes.
7. Keep shared primitives independent of MarketOps packages.
8. Keep provider-specific structures inside source adapters and Raw Events.
9. Use normalized Dataset contracts for downstream code.
10. Use Redpanda through Kafka-compatible broker interfaces.
11. Keep Python algorithms behind versioned execution contracts.
12. Never permit algorithm code to directly write production Signals, Proposals, graph state, or external actions.
13. Store all definition versions used by every derived record.
14. Implement idempotency before side effects.
15. Commit broker offsets only after required durable writes or accepted downstream handoff.
16. Retain complete failure metadata in retry and DLQ records.
17. Implement tenant authorization in handlers and persistence queries.
18. Add unit, integration, replay, quality, lineage, and multi-tenant tests with each phase.
19. Update OpenAPI and event schemas alongside implementation.
20. Produce a migration and rollout note for every phase.

---

## 53. Final Architectural Position

SignalOps primitives are not domain entities owned by MarketOps, CyberOps, CRM Intelligence, or any future application.

They are the shared language of Syncratic operational intelligence:

```text
Sources produce Datasets.
Datasets produce Raw and Normalized Events.
Events produce Features.
Features compose State.
States produce Transitions.
Detectors and Algorithms evaluate evidence.
Evidence supports Signals.
Signals become Artifacts and Proposals.
Governed evaluation produces Insights and Materialized Knowledge.
Outcomes close the learning loop.
```

New SignalOps applications SHOULD primarily consist of:

- Dataset definitions;
- source adapters;
- normalization mappings;
- feature packs;
- state models;
- transition definitions;
- detectors;
- algorithm configurations or domain algorithms;
- Signal definitions;
- policies;
- Insight templates;
- Outcome definitions;
- domain read models and user interfaces.

They SHOULD NOT rebuild ingestion, immutable ledgers, replay, algorithm orchestration, evidence, quality, lineage, review, materialization, or Engine integration.

This separation is the foundation for expanding SignalOps into multiple operational intelligence products while preserving a single governed, explainable, replayable, and independently scalable platform.

