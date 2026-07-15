# Syncratic SignalOps Extended Capabilities Specification

## 1. Purpose

This specification extends the base Syncratic SignalOps Infrastructure Specification with the higher-order capabilities required to turn SignalOps from a streaming ingestion subsystem into a reusable enterprise signal intelligence platform.

The base SignalOps infrastructure handles source registration, event ingestion, broker publishing, replayable event history, Event Artifact creation, graph mutation proposals, and Insight Candidate publication.

This extended specification adds the following capabilities:

- Signal Definition Layer
- Rule Engine
- Window Processing
- Stream State Store
- Processing Pipeline Framework
- Signal Registry and Signal Families
- Enrichment Framework
- Feature Store
- Algorithm Plugin Framework
- Confidence Aggregation
- Signal Lifecycle
- Insight Lifecycle
- Expanded Correlation Model
- Signal Graph Model
- Policy Engine
- Backpressure Control
- Prioritized Streams
- Signal Lineage
- Stream Catalog
- Event and Artifact Versioning

These capabilities MUST follow Syncratic's non-interference principle. They MUST NOT modify existing document ingestion, retrieval, graph construction, or Ask workflows. They MUST operate inside SignalOps and communicate with the Syncratic Engine only through approved extension contracts.

---

## 2. Design Goals

SignalOps MUST support the following long-term Syncratic use cases:

- MarketOps: market data ingestion, anomaly detection, asset movement signals, predictive insight candidates, trading research workflows.
- CRM intelligence: account movement, stalled opportunities, churn risk, customer expansion signals, activity pattern detection.
- Security analytics: telemetry correlation, threat signals, identity behavior anomalies, suspicious event clustering.
- Operational intelligence: system telemetry, infrastructure health, incident prediction, SLA breach detection.
- IoT and manufacturing: sensor state, device anomalies, process drift, maintenance signals.
- RFP and procurement intelligence: public opportunity stream monitoring, opportunity classification, vendor or buyer relationship signals.

The extended architecture MUST allow different event domains to share common infrastructure while keeping domain-specific logic isolated through registries, policies, pipelines, enrichers, feature definitions, and algorithm plugins.

---

## 3. Non-Interference Requirements

The implementation MUST preserve the following boundaries:

1. SignalOps MUST remain a separate subsystem.
2. SignalOps MUST NOT directly modify existing Syncratic document ingestion code paths.
3. SignalOps MUST NOT directly write canonical graph state unless the Syncratic Engine exposes an approved adapter for that purpose.
4. SignalOps MUST propose Event Artifacts, Graph Mutation Proposals, Signals, and Insight Candidates through Engine extension contracts.
5. The Syncratic Engine remains the authority for accepting, rejecting, evaluating, and applying knowledge graph mutations.
6. Algorithm execution MUST remain pluggable and outside the Engine core.
7. SignalOps failures MUST NOT degrade existing Ask, retrieval, document ingestion, or graph construction workflows.

---

## 4. Extended Architecture

```text
External Sources
  - market feeds
  - CRM events
  - IoT telemetry
  - security logs
  - operational metrics
  - procurement/RFP feeds

        |
        v

Signal Gateway
  - authentication
  - authorization
  - source validation
  - schema validation
  - rate limits
  - normalization

        |
        v

Broker Abstraction
  - Kafka / Redpanda initially
  - retry topics
  - DLQ topics
  - priority topics

        |
        v

Pipeline Manager
  - pipeline selection
  - stage orchestration
  - family-specific routing
  - replay-aware execution

        |
        +------------------------+
        |                        |
        v                        v

Rule Engine              Enrichment Engine
  - deterministic rules    - source enrichers
  - rule versions          - domain enrichers
  - rule results           - external lookups
  - rule artifacts         - tenant policies

        |                        |
        +-----------+------------+
                    |
                    v

Window and State Layer
  - rolling windows
  - session windows
  - tumbling windows
  - state store
  - aggregate state

                    |
                    v

Feature Engine
  - reusable features
  - feature definitions
  - feature materialization
  - feature history

                    |
                    v

Signal Detection Layer
  - signal definitions
  - signal instances
  - signal lifecycle
  - confidence aggregation

                    |
                    v

Algorithm Plugin Framework
  - anomaly detection
  - clustering
  - forecasting
  - classification
  - trend detection

                    |
                    v

Event Artifact Builder
  - Event Artifacts
  - Rule Artifacts
  - Signal Artifacts
  - Feature Artifacts

                    |
                    v

Engine Extension Adapters
  - Artifact API
  - Entity Resolution API
  - Graph Mutation API
  - Temporal API
  - Evidence API
  - Insight API
  - Evaluation API

                    |
                    v

Syncratic Engine
  - validates artifacts
  - evaluates graph proposals
  - maintains canonical graph
  - evaluates Insight Candidates
```

---

## 5. Signal Definition Layer

### 5.1 Purpose

Events are raw observations. Signals are meaningful derived interpretations.

Example:

- Event: `opportunity.updated`
- Signal: `large_opportunity_stalled`

A signal may be derived from one event, multiple events, elapsed time, windows, rules, features, or algorithm outputs.

### 5.2 Requirements

SignalOps MUST implement a Signal Definition Layer that allows administrators and domain pipelines to define named signals.

A Signal Definition MUST include:

- `signal_id`
- `tenant_id`
- `name`
- `description`
- `signal_family`
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

### 5.3 Signal Instance

A Signal Instance is an observed occurrence of a Signal Definition.

Required fields:

- `signal_instance_id`
- `tenant_id`
- `signal_id`
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

### 5.4 Signal Output

Signal Instances MAY produce:

- Event Artifacts
- Signal Artifacts
- Graph Mutation Proposals
- Insight Candidates
- Evaluation requests

---

## 6. Rule Engine

### 6.1 Purpose

SignalOps MUST support deterministic rules before and alongside machine learning algorithms.

Rules are used for:

- threshold detection
- business policy checks
- escalation conditions
- filtering
- suppression
- routing
- signal creation

### 6.2 Rule Definition

Required fields:

- `rule_id`
- `tenant_id`
- `name`
- `description`
- `rule_language`
- `rule_body`
- `input_event_types`
- `input_features`
- `output_type`
- `severity`
- `version`
- `status`
- `created_at`
- `updated_at`

Supported initial `rule_language` values:

- `json_logic`
- `cel`
- `python_guarded` only for internal trusted deployments

The first implementation SHOULD prioritize JSON Logic or CEL-style expressions to avoid unsafe arbitrary execution.

### 6.3 Rule Result

Required fields:

- `rule_result_id`
- `tenant_id`
- `rule_id`
- `rule_version`
- `event_id`
- `signal_instance_id`
- `matched`
- `score`
- `severity`
- `result_payload`
- `evaluated_at`
- `evidence_refs`
- `correlation_id`

### 6.4 Rule Artifacts

When a rule produces a durable observation, SignalOps MUST create a Rule Artifact.

Required fields:

- `artifact_id`
- `artifact_type = rule_result`
- `tenant_id`
- `rule_id`
- `rule_result_id`
- `source_event_ids`
- `summary`
- `confidence`
- `evidence_refs`
- `created_at`

---

## 7. Window Processing

### 7.1 Purpose

SignalOps MUST support time windows for stream processing.

Required initial window types:

- tumbling windows
- sliding windows
- rolling windows
- session windows
- calendar windows

### 7.2 Window Definition

Required fields:

- `window_id`
- `tenant_id`
- `name`
- `window_type`
- `duration_seconds`
- `slide_seconds`
- `session_gap_seconds`
- `timezone`
- `event_time_field`
- `allowed_lateness_seconds`
- `watermark_policy`
- `status`

### 7.3 Window Result

Required fields:

- `window_result_id`
- `tenant_id`
- `window_id`
- `source_id`
- `entity_key`
- `event_type`
- `window_start`
- `window_end`
- `event_count`
- `aggregate_values`
- `late_event_count`
- `created_at`
- `correlation_id`

### 7.4 Late and Out-of-Order Events

SignalOps MUST distinguish:

- source event time: `occurred_at`
- ingestion time: `observed_at`
- processing time: `processed_at`

Window calculations MUST be based on event time where possible.

Late events MUST be handled according to the configured watermark policy.

---

## 8. Stream State Store

### 8.1 Purpose

Many signal calculations require state across events.

Examples:

- current customer status
- last event by entity
- rolling counters
- moving averages
- active incident state
- open signal state

### 8.2 State Store Requirements

SignalOps MUST provide a state store abstraction.

Initial supported backends SHOULD include:

- Redis for fast operational state
- TimescaleDB for durable aggregate state
- PostgreSQL for low-volume workflow state

The abstraction MUST allow backend replacement without changing pipeline code.

### 8.3 State Record

Required fields:

- `state_key`
- `tenant_id`
- `source_id`
- `entity_key`
- `state_type`
- `state_payload`
- `version`
- `expires_at`
- `updated_at`
- `correlation_id`

### 8.4 State Consistency

State updates MUST be idempotent. State writes SHOULD use optimistic concurrency where possible.

---

## 9. Processing Pipeline Framework

### 9.1 Purpose

SignalOps MUST support named processing pipelines instead of hardcoded worker behavior.

Pipelines determine how events are normalized, enriched, windowed, converted into features, evaluated by rules, processed by algorithms, transformed into artifacts, and submitted to the Engine.

### 9.2 Pipeline Definition

Required fields:

- `pipeline_id`
- `tenant_id`
- `name`
- `description`
- `signal_family`
- `input_event_types`
- `stages`
- `output_targets`
- `version`
- `status`
- `created_at`
- `updated_at`

### 9.3 Pipeline Stage

Required fields:

- `stage_id`
- `stage_type`
- `stage_name`
- `order`
- `config`
- `retry_policy_ref`
- `timeout_seconds`
- `required`

Initial supported stage types:

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

### 9.4 Pipeline Execution Record

Required fields:

- `execution_id`
- `tenant_id`
- `pipeline_id`
- `pipeline_version`
- `event_id`
- `status`
- `started_at`
- `completed_at`
- `stage_results`
- `error_metadata`
- `correlation_id`

---

## 10. Signal Registry and Signal Families

### 10.1 Purpose

Signal Families are analogous to Syncratic document families. They allow SignalOps to route events into domain-aware processing paths.

### 10.2 Initial Signal Families

SignalOps SHOULD support the following initial families:

- `market_tick`
- `trade`
- `crm_change`
- `security_event`
- `telemetry_metric`
- `iot_sensor`
- `finance_event`
- `manufacturing_event`
- `erp_event`
- `network_event`
- `procurement_event`
- `rfp_opportunity`

### 10.3 Signal Family Definition

Required fields:

- `family_id`
- `name`
- `description`
- `schema_refs`
- `pipeline_refs`
- `enrichment_refs`
- `rule_refs`
- `feature_refs`
- `algorithm_refs`
- `graph_mapper_ref`
- `default_retention_policy_ref`
- `status`

---

## 11. Enrichment Framework

### 11.1 Purpose

SignalOps MUST support pluggable enrichment without hardcoding domain logic into workers.

Examples:

- GeoIP lookup
- company lookup
- customer registry lookup
- asset registry lookup
- ticker metadata lookup
- threat intelligence lookup
- device inventory lookup
- calendar and holiday context
- maintenance window context

### 11.2 Enricher Definition

Required fields:

- `enricher_id`
- `tenant_id`
- `name`
- `description`
- `input_fields`
- `output_fields`
- `enricher_type`
- `config`
- `timeout_seconds`
- `cache_policy`
- `version`
- `status`

Supported initial `enricher_type` values:

- `static_lookup`
- `database_lookup`
- `http_lookup`
- `engine_lookup`
- `calendar_context`
- `tenant_registry_lookup`

### 11.3 Enrichment Result

Required fields:

- `enrichment_result_id`
- `tenant_id`
- `enricher_id`
- `event_id`
- `input_hash`
- `output_payload`
- `confidence`
- `cache_hit`
- `created_at`
- `evidence_refs`
- `correlation_id`

---

## 12. Feature Store

### 12.1 Purpose

SignalOps MUST support reusable feature definitions and materialized feature values for algorithms and signal detection.

Features are derived values such as:

- rolling mean
- rolling count
- velocity
- rate of change
- volatility
- trend slope
- frequency
- entropy
- moving average
- percentile rank
- session duration

### 12.2 Feature Definition

Required fields:

- `feature_id`
- `tenant_id`
- `name`
- `description`
- `feature_family`
- `input_event_types`
- `entity_key_fields`
- `window_ref`
- `calculation_type`
- `calculation_config`
- `value_type`
- `version`
- `status`

### 12.3 Feature Value

Required fields:

- `feature_value_id`
- `tenant_id`
- `feature_id`
- `feature_version`
- `entity_key`
- `window_start`
- `window_end`
- `value`
- `value_type`
- `computed_at`
- `source_event_ids`
- `correlation_id`

### 12.4 Storage

Feature definitions SHOULD be stored in PostgreSQL.

Feature values MAY be stored in TimescaleDB, PostgreSQL, or a dedicated feature store backend depending on scale.

The first implementation SHOULD use TimescaleDB for time-indexed feature values.

---

## 13. Algorithm Plugin Framework

### 13.1 Purpose

SignalOps MUST support algorithm plugins without coupling infrastructure to specific libraries.

Algorithm plugins may support:

- anomaly detection
- clustering
- classification
- forecasting
- trend detection
- change-point detection
- sequence detection
- correlation detection

### 13.2 Algorithm Definition

Required fields:

- `algorithm_id`
- `tenant_id`
- `name`
- `description`
- `algorithm_type`
- `runtime_type`
- `input_features`
- `input_event_types`
- `output_schema`
- `config_schema`
- `default_config`
- `version`
- `status`

Supported initial `algorithm_type` values:

- `anomaly_detection`
- `clustering`
- `forecasting`
- `classification`
- `change_point_detection`
- `trend_detection`

Supported initial `runtime_type` values:

- `python_plugin`
- `container_plugin`
- `http_plugin`

### 13.3 Algorithm Execution Request

Required fields:

- `execution_request_id`
- `tenant_id`
- `algorithm_id`
- `algorithm_version`
- `event_ids`
- `feature_refs`
- `entity_refs`
- `window_ref`
- `config`
- `correlation_id`

### 13.4 Algorithm Result

Required fields:

- `algorithm_result_id`
- `tenant_id`
- `algorithm_id`
- `algorithm_version`
- `execution_request_id`
- `result_type`
- `score`
- `confidence`
- `severity`
- `result_payload`
- `source_event_ids`
- `feature_value_ids`
- `evidence_refs`
- `created_at`
- `correlation_id`

### 13.5 Initial Candidate Libraries

The algorithm processing specification MAY evaluate the following open-source libraries:

- PyOD
- River
- scikit-learn
- HDBSCAN
- DBSCAN via scikit-learn
- STUMPY / matrix profile
- Prophet
- Merlion
- Kats
- ruptures
- statsmodels

This infrastructure specification MUST NOT hardwire any of these libraries.

---

## 14. Confidence Aggregation

### 14.1 Purpose

Multiple rules, enrichers, algorithms, and human evaluations may produce different confidence values for the same signal.

SignalOps MUST support confidence aggregation.

### 14.2 Confidence Policy

Required fields:

- `confidence_policy_id`
- `tenant_id`
- `name`
- `description`
- `aggregation_method`
- `weights`
- `minimum_required_sources`
- `conflict_resolution_policy`
- `version`
- `status`

Supported initial aggregation methods:

- `max`
- `min`
- `mean`
- `weighted_mean`
- `rule_priority`
- `algorithm_priority`
- `human_override`

### 14.3 Aggregated Confidence Result

Required fields:

- `confidence_result_id`
- `tenant_id`
- `policy_id`
- `signal_instance_id`
- `input_scores`
- `aggregated_score`
- `method`
- `created_at`
- `correlation_id`

---

## 15. Signal Lifecycle

### 15.1 Purpose

Signals are not static. SignalOps MUST track the lifecycle of each Signal Instance.

### 15.2 Required States

Signal Instance states:

- `observed`
- `open`
- `growing`
- `stable`
- `suppressed`
- `resolved`
- `archived`

### 15.3 Lifecycle Transition Record

Required fields:

- `transition_id`
- `tenant_id`
- `signal_instance_id`
- `from_state`
- `to_state`
- `transition_reason`
- `trigger_event_id`
- `trigger_rule_id`
- `trigger_algorithm_result_id`
- `transitioned_at`
- `correlation_id`

---

## 16. Insight Lifecycle

### 16.1 Purpose

Insight Candidates require evaluation before they become trusted insights.

### 16.2 Required States

Insight states:

- `candidate`
- `evaluating`
- `accepted`
- `dismissed`
- `learned`
- `archived`

### 16.3 Insight Lifecycle Integration

SignalOps MUST publish Insight Candidates with evidence references and correlation IDs.

The Syncratic Evaluation Agent or Engine evaluation extension decides whether the Insight Candidate is accepted, dismissed, pending, or requires human review.

SignalOps MUST store the returned evaluation status for audit and downstream learning.

---

## 17. Expanded Correlation Model

### 17.1 Purpose

The base infrastructure requires `correlation_id`. Extended SignalOps MUST support richer causality and lineage tracking.

### 17.2 Required Correlation Fields

SignalOps event envelopes SHOULD support:

- `trace_id`
- `correlation_id`
- `causation_id`
- `parent_event_id`
- `child_event_ids`
- `session_id`
- `workflow_id`
- `replay_job_id`

### 17.3 Semantics

- `trace_id` groups an end-to-end processing trace.
- `correlation_id` groups related operations across services.
- `causation_id` identifies the event or process that caused this event.
- `parent_event_id` links derived events to source events.
- `session_id` groups user, device, market, or process sessions.
- `workflow_id` groups a business workflow.
- `replay_job_id` identifies events republished during replay.

---

## 18. Signal Graph Model

### 18.1 Purpose

SignalOps MUST support graph-ready semantics that allow the Syncratic Engine to connect events, signals, insights, decisions, and outcomes.

### 18.2 Conceptual Model

```text
Source
  -> Event
    -> Artifact
      -> Signal
        -> Insight Candidate
          -> Evaluated Insight
            -> Decision
              -> Outcome
```

### 18.3 Graph Proposal Types

In addition to the base mutation types, SignalOps SHOULD support proposal semantics for:

- `event_observed_entity`
- `event_generated_artifact`
- `artifact_generated_signal`
- `signal_supported_insight_candidate`
- `insight_supported_decision`
- `decision_produced_outcome`
- `signal_correlated_with_signal`
- `signal_suppressed_by_policy`
- `signal_resolved_by_event`

These SHOULD be translated into the Engine's approved Graph Mutation API format.

---

## 19. Policy Engine

### 19.1 Purpose

Policies control runtime behavior. Rules detect conditions; policies govern how SignalOps behaves.

Examples:

- suppress during maintenance windows
- ignore weekends
- reduce severity during holidays
- escalate after hours
- suppress duplicate alerts
- route critical events to priority topics
- disable algorithms for specific tenants

### 19.2 Policy Definition

Required fields:

- `policy_id`
- `tenant_id`
- `name`
- `description`
- `policy_type`
- `policy_body`
- `scope`
- `version`
- `status`
- `created_at`
- `updated_at`

Supported initial `policy_type` values:

- `suppression`
- `routing`
- `retention`
- `escalation`
- `confidence`
- `replay`
- `rate_limit`
- `algorithm_execution`

### 19.3 Policy Decision Record

Required fields:

- `policy_decision_id`
- `tenant_id`
- `policy_id`
- `event_id`
- `signal_instance_id`
- `decision`
- `reason`
- `created_at`
- `correlation_id`

---

## 20. Backpressure Control

### 20.1 Purpose

SignalOps MUST prevent overload from degrading the subsystem or the Syncratic Engine.

### 20.2 Requirements

SignalOps MUST monitor:

- broker lag
- gateway request rate
- worker queue depth
- worker processing latency
- storage write latency
- Engine extension latency
- DLQ growth
- retry growth

### 20.3 Backpressure Actions

Supported actions SHOULD include:

- throttle source ingestion
- reject low-priority events with retry guidance
- pause non-critical pipelines
- reduce algorithm execution frequency
- route low-priority events to delayed processing
- increase worker replicas through autoscaling
- temporarily disable expensive enrichment

### 20.4 Backpressure Decision Record

Required fields:

- `decision_id`
- `tenant_id`
- `source_id`
- `action`
- `reason`
- `metrics_snapshot`
- `started_at`
- `ended_at`
- `correlation_id`

---

## 21. Prioritized Streams

### 21.1 Purpose

Not all events have the same operational value. SignalOps MUST support priority-aware ingestion and processing.

### 21.2 Priority Levels

Required priority levels:

- `critical`
- `high`
- `medium`
- `low`
- `background`

### 21.3 Priority Assignment

Priority MAY be assigned by:

- source configuration
- event type
- tenant policy
- rule result
- algorithm result
- manual override

### 21.4 Priority-Aware Topics

The broker abstraction SHOULD support either priority topics or priority-aware routing metadata.

Example topic convention:

- `signalops.<env>.normalized.critical.v1`
- `signalops.<env>.normalized.high.v1`
- `signalops.<env>.normalized.medium.v1`
- `signalops.<env>.normalized.low.v1`
- `signalops.<env>.normalized.background.v1`

---

## 22. Signal Lineage

### 22.1 Purpose

SignalOps MUST preserve explainability and auditability by recording lineage across the full signal processing chain.

### 22.2 Required Lineage Chain

Signal lineage SHOULD capture:

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

### 22.3 Lineage Record

Required fields:

- `lineage_id`
- `tenant_id`
- `root_event_id`
- `current_object_type`
- `current_object_id`
- `parent_object_type`
- `parent_object_id`
- `transformation_type`
- `transformation_version`
- `created_at`
- `correlation_id`

---

## 23. Stream Catalog

### 23.1 Purpose

Enterprise administrators need visibility into registered streams, schemas, pipelines, rules, signals, features, algorithms, and retention settings.

SignalOps MUST provide a Stream Catalog.

### 23.2 Catalog Objects

The Stream Catalog MUST expose:

- sources
- source types
- schemas
- topics
- signal families
- pipelines
- enrichers
- rules
- policies
- features
- algorithms
- retention policies
- replay policies
- DLQ status
- lifecycle states

### 23.3 Catalog API Examples

- `GET /v1/tenants/{tenant_id}/catalog/sources`
- `GET /v1/tenants/{tenant_id}/catalog/schemas`
- `GET /v1/tenants/{tenant_id}/catalog/pipelines`
- `GET /v1/tenants/{tenant_id}/catalog/signals`
- `GET /v1/tenants/{tenant_id}/catalog/features`
- `GET /v1/tenants/{tenant_id}/catalog/algorithms`
- `GET /v1/tenants/{tenant_id}/catalog/policies`
- `GET /v1/tenants/{tenant_id}/catalog/dlq`

---

## 24. Event and Artifact Versioning

### 24.1 Purpose

SignalOps MUST support deterministic replay and processing evolution.

Versioning MUST apply to:

- raw event schema
- normalized event schema
- pipeline version
- enrichment version
- rule version
- feature version
- algorithm version
- artifact builder version
- graph mapper version

### 24.2 Versioned Processing Record

Required fields:

- `processing_record_id`
- `tenant_id`
- `event_id`
- `raw_schema_version`
- `normalized_schema_version`
- `pipeline_id`
- `pipeline_version`
- `rule_versions`
- `feature_versions`
- `algorithm_versions`
- `artifact_builder_version`
- `graph_mapper_version`
- `processed_at`
- `correlation_id`

### 24.3 Replay Requirement

Replay jobs MUST be able to specify whether to use:

- original processing versions
- latest compatible processing versions
- explicitly selected processing versions

Replay request extension:

```json
{
  "source_id": "src_01J00000000000000000000000",
  "from": "2026-07-01T00:00:00Z",
  "to": "2026-07-06T00:00:00Z",
  "event_types": ["contact.updated"],
  "processing_mode": "latest_compatible",
  "pipeline_version": "2.1.0",
  "reason": "reprocess after schema mapper fix"
}
```

Allowed `processing_mode` values:

- `original_versions`
- `latest_compatible`
- `explicit_versions`

---

## 25. Additional REST API Contracts

### 25.1 Register Signal Definition

`POST /v1/tenants/{tenant_id}/signals`

Request:

```json
{
  "name": "large_opportunity_stalled",
  "description": "Detects high-value opportunities with declining activity or stalled progress.",
  "signal_family": "crm_change",
  "input_event_types": ["opportunity.updated", "activity.created"],
  "required_entities": ["account", "opportunity"],
  "rule_refs": ["rule_large_opp_stalled_v1"],
  "feature_refs": ["feature_days_since_last_activity_v1"],
  "algorithm_refs": [],
  "confidence_policy_ref": "confidence_weighted_crm_v1",
  "lifecycle_policy_ref": "crm_signal_lifecycle_v1"
}
```

Response `201 Created`:

```json
{
  "signal_id": "sig_01J00000000000000000000000",
  "tenant_id": "tenant_123",
  "status": "active"
}
```

### 25.2 Register Pipeline

`POST /v1/tenants/{tenant_id}/pipelines`

Request:

```json
{
  "name": "crm-opportunity-pipeline",
  "description": "Processes CRM opportunity and activity events into signals and insight candidates.",
  "signal_family": "crm_change",
  "input_event_types": ["opportunity.updated", "activity.created"],
  "stages": [
    {
      "stage_type": "normalize",
      "stage_name": "crm_normalizer",
      "order": 1,
      "required": true
    },
    {
      "stage_type": "entity_resolve",
      "stage_name": "account_opportunity_resolver",
      "order": 2,
      "required": true
    },
    {
      "stage_type": "feature_compute",
      "stage_name": "crm_activity_features",
      "order": 3,
      "required": true
    },
    {
      "stage_type": "rule_evaluate",
      "stage_name": "crm_stall_rules",
      "order": 4,
      "required": true
    },
    {
      "stage_type": "signal_detect",
      "stage_name": "crm_signal_detector",
      "order": 5,
      "required": true
    }
  ],
  "output_targets": ["artifact", "graph_mutation", "insight_candidate"]
}
```

Response `201 Created`:

```json
{
  "pipeline_id": "pipe_01J00000000000000000000000",
  "version": "1.0.0",
  "status": "active"
}
```

### 25.3 Register Rule

`POST /v1/tenants/{tenant_id}/rules`

Request:

```json
{
  "name": "large_opportunity_stalled_rule",
  "description": "Flags high-value opportunities with no activity for more than 14 days.",
  "rule_language": "json_logic",
  "rule_body": {
    "and": [
      { ">": [{ "var": "opportunity.amount" }, 1000000] },
      { ">": [{ "var": "features.days_since_last_activity" }, 14] }
    ]
  },
  "input_event_types": ["opportunity.updated", "activity.created"],
  "input_features": ["days_since_last_activity"],
  "output_type": "signal_trigger",
  "severity": "high"
}
```

Response `201 Created`:

```json
{
  "rule_id": "rule_01J00000000000000000000000",
  "version": "1.0.0",
  "status": "active"
}
```

### 25.4 Register Feature

`POST /v1/tenants/{tenant_id}/features`

Request:

```json
{
  "name": "days_since_last_activity",
  "description": "Number of days since the most recent activity for an opportunity.",
  "feature_family": "crm_change",
  "input_event_types": ["activity.created", "opportunity.updated"],
  "entity_key_fields": ["opportunity_id"],
  "window_ref": "rolling_90_days",
  "calculation_type": "time_since_last_event",
  "calculation_config": {
    "event_type": "activity.created"
  },
  "value_type": "number"
}
```

Response `201 Created`:

```json
{
  "feature_id": "feat_01J00000000000000000000000",
  "version": "1.0.0",
  "status": "active"
}
```

### 25.5 Register Algorithm

`POST /v1/tenants/{tenant_id}/algorithms`

Request:

```json
{
  "name": "crm_activity_anomaly_detector",
  "description": "Detects unusual drops in CRM activity patterns.",
  "algorithm_type": "anomaly_detection",
  "runtime_type": "python_plugin",
  "input_features": ["activity_count_7d", "days_since_last_activity"],
  "input_event_types": ["activity.created", "opportunity.updated"],
  "output_schema": {
    "score": "number",
    "confidence": "number",
    "severity": "string"
  },
  "config_schema": {},
  "default_config": {}
}
```

Response `201 Created`:

```json
{
  "algorithm_id": "algo_01J00000000000000000000000",
  "version": "1.0.0",
  "status": "active"
}
```

---

## 26. Database Ownership

### 26.1 PostgreSQL

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

### 26.2 TimescaleDB

TimescaleDB remains canonical for replayable time-series history.

Additional hypertables SHOULD include:

- `window_results`
- `feature_values`
- `signal_observations`
- `algorithm_timeseries_results`
- `state_history`

### 26.3 Redis or Equivalent State Store

Redis MAY be used for:

- hot state
- active sessions
- counters
- rate-limiting state
- short-lived pipeline state
- deduplication caches

Redis MUST NOT become the only canonical source of replayable event truth.

### 26.4 Qdrant

Qdrant MAY store:

- signal summaries
- insight summaries
- incident summaries
- semantic clusters
- embeddings of signal narratives

Qdrant entries MUST be rebuildable from TimescaleDB, PostgreSQL, and Engine-approved artifacts.

---

## 27. Observability Extensions

SignalOps MUST add observability for the extended capabilities.

### 27.1 Required Metrics

- rule evaluation count by rule, status, tenant, and source
- rule match rate
- pipeline execution latency by stage
- pipeline failure rate by stage
- enrichment latency by enricher
- enrichment failure rate
- feature computation latency
- algorithm execution latency
- algorithm failure rate
- signal creation count by family and state
- signal lifecycle transition count
- confidence aggregation count and score distribution
- policy decision count by action
- backpressure activation count
- priority topic lag
- catalog object count

### 27.2 Required Logs

- pipeline selected
- pipeline stage started/completed/failed
- rule evaluated
- signal created or updated
- signal lifecycle changed
- policy decision applied
- confidence aggregation performed
- algorithm plugin invoked
- algorithm plugin failed
- backpressure decision applied
- lineage record created

### 27.3 Required Traces

Traces MUST connect:

```text
Gateway request
  -> broker publish
  -> pipeline execution
  -> enrichment
  -> feature computation
  -> rule evaluation
  -> algorithm execution
  -> signal detection
  -> artifact creation
  -> Engine extension request
```

---

## 28. Security and Tenant Isolation Extensions

The extended capabilities MUST enforce tenant isolation across:

- signal definitions
- rule definitions
- pipeline definitions
- feature definitions
- algorithm definitions
- enrichment configurations
- policy definitions
- feature values
- algorithm results
- signal instances
- lineage records
- catalog APIs

Rules and algorithms MUST NOT access another tenant's data.

Algorithm plugins MUST run with constrained permissions. Container-based plugins SHOULD run with:

- no privilege escalation
- read-only root filesystem where possible
- resource limits
- network restrictions unless explicitly required
- tenant-scoped credentials only

Logs and metrics MUST redact PII, secrets, and sensitive payload values.

---

## 29. Deployment Extensions

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

Helm values SHOULD include configuration for:

- enabled signal families
- enabled worker pools
- algorithm plugin mode
- Redis/state store settings
- priority topic settings
- backpressure thresholds
- feature retention policies
- signal retention policies
- algorithm resource limits

---

## 30. Acceptance Criteria

The extended implementation is acceptable when the following scenarios pass.

### 30.1 Signal Definition

- A tenant can register a Signal Definition.
- Signal Definitions are versioned.
- Inactive Signal Definitions are not used for new detections.

### 30.2 Rule Engine

- A tenant can register a rule.
- A matching event produces a Rule Result.
- Rule Results include evidence, version, score, and correlation ID.
- Invalid rule bodies are rejected before activation.

### 30.3 Pipeline Framework

- A pipeline can be registered with ordered stages.
- Events are routed to the correct pipeline based on source, event type, and signal family.
- Stage failure follows retry policy.
- Pipeline execution records include all stage results.

### 30.4 Enrichment Framework

- An event can be enriched through a configured enricher.
- Enrichment results are persisted with input hash, output payload, confidence, and evidence.
- Enrichment failure does not corrupt canonical event history.

### 30.5 Window and Feature Processing

- A rolling window computes expected aggregates from event history.
- A Feature Definition produces Feature Values for entity-scoped events.
- Feature Values are queryable by tenant, feature, entity, and time range.

### 30.6 Algorithm Plugins

- An algorithm plugin can be registered.
- A pipeline can invoke the algorithm plugin.
- Algorithm Results include score, confidence, payload, evidence, and source features.
- Failed algorithm executions route to retry or DLQ without blocking unrelated events.

### 30.7 Confidence Aggregation

- Multiple input confidence values can be aggregated by policy.
- Aggregated confidence is stored and attached to the Signal Instance.
- Human override policy can supersede algorithm confidence where configured.

### 30.8 Signal Lifecycle

- A Signal Instance can move from `observed` to `open`.
- A Signal Instance can move to `suppressed` by policy.
- A Signal Instance can move to `resolved` after a resolving event.
- All transitions are auditable.

### 30.9 Insight Lifecycle

- A Signal Instance can publish an Insight Candidate.
- The Insight Candidate includes evidence and correlation ID.
- Evaluation response is stored and reflected in the insight lifecycle state.

### 30.10 Lineage

- A lineage chain can be reconstructed from raw event to Insight Candidate.
- Lineage records include transformation type and version.
- Replay-generated objects preserve replay job lineage.

### 30.11 Backpressure and Priority

- Critical events continue processing when low-priority streams are throttled.
- Sustained worker lag triggers a backpressure decision.
- Backpressure decisions are auditable.

### 30.12 Catalog

- A tenant can list registered sources, schemas, pipelines, signals, rules, features, algorithms, and policies.
- Catalog APIs do not expose other tenant objects.

### 30.13 Replay and Versioning

- Replay can run using original processing versions.
- Replay can run using latest compatible versions.
- Replay output records processing versions used.

### 30.14 Non-Interference

- Existing Syncratic document ingestion continues unchanged.
- Existing retrieval continues unchanged.
- Existing graph construction continues unchanged.
- Existing Ask workflows continue unchanged.
- SignalOps can be disabled without breaking the core Syncratic Engine.

---

## 31. Recommended Implementation Phases

### Phase 1: Foundation

- Signal Registry
- Pipeline Definitions
- Rule Engine
- Pipeline Execution Records
- Catalog APIs
- Basic lineage records

### Phase 2: Temporal Processing

- Window Definitions
- Window Results
- State Store abstraction
- Feature Definitions
- Feature Values

### Phase 3: Signal Intelligence

- Signal Detection Layer
- Signal Instance lifecycle
- Confidence aggregation
- Policy Engine
- Insight lifecycle integration

### Phase 4: Algorithm Plugins

- Algorithm registry
- Python plugin execution
- Container plugin execution
- Algorithm Results
- Algorithm worker pool

### Phase 5: Enterprise Operations

- Backpressure control
- Priority streams
- Advanced catalog visibility
- Replay version controls
- Operational dashboards and alerts

---

## 32. Code Agent Notes

The code agent should implement this specification as an extension of the existing SignalOps subsystem, not as modifications to the Syncratic Engine core.

Recommended folder structure:

```text
signalops/
  app/
    api/
      routes/
        sources.py
        schemas.py
        events.py
        replay.py
        catalog.py
        signals.py
        rules.py
        pipelines.py
        features.py
        algorithms.py
        policies.py
    core/
      config.py
      security.py
      tenancy.py
      idempotency.py
      errors.py
    broker/
      base.py
      kafka.py
      redpanda.py
      topics.py
    pipelines/
      manager.py
      registry.py
      stages.py
      execution.py
    rules/
      engine.py
      json_logic.py
      cel.py
    enrichment/
      base.py
      registry.py
      runners.py
    windows/
      definitions.py
      processor.py
      watermarks.py
    state/
      base.py
      redis_store.py
      timescale_store.py
    features/
      definitions.py
      compute.py
      store.py
    algorithms/
      registry.py
      runners.py
      python_plugin.py
      container_plugin.py
    signals/
      definitions.py
      detector.py
      lifecycle.py
      confidence.py
    policies/
      engine.py
      decisions.py
    lineage/
      recorder.py
      queries.py
    engine_adapters/
      artifact.py
      entity.py
      graph.py
      temporal.py
      evidence.py
      insight.py
      evaluation.py
    storage/
      postgres.py
      timescale.py
      migrations/
    observability/
      logging.py
      metrics.py
      tracing.py
  workers/
    gateway_worker.py
    pipeline_worker.py
    enrichment_worker.py
    feature_worker.py
    algorithm_worker.py
    replay_worker.py
  helm/
  tests/
```

The first deliverable should avoid over-optimizing algorithm execution. Build the contract, registry, execution record, and one safe reference runner first. The core value is the stable platform seam, not the first algorithm.
