# G111 Algorithm Result To Signal Proposal Design

Status: proposed design
Timestamp: 2026-07-15T20:54:51Z

## Purpose

G111 defines how persisted `algorithm_results` should later become candidate signal/artifact proposals without mutating production signal, alert, insight, or graph state prematurely.

This is a design gate only. It must be reviewed before implementation.

## Design Principles

- Algorithm results are evidence, not production signals by default.
- Conversion must be explicit, auditable, and reversible.
- Conversion must preserve the original `algorithm_result_id`, execution request id, normalized event lineage, feature refs, result payload, score, confidence, and severity.
- Conversion must avoid duplicating existing deterministic DSM signals unless the algorithm result adds distinct evidence.
- Conversion must not create Alerts and Insights with identical semantics. Alerts should represent individual qualifying events; Insights should aggregate repeated or multi-event patterns.

## Proposed Substrate

Add a proposal ledger, not direct signal writes:

`algorithm_signal_proposals`

Suggested fields:

- `proposal_id`
- `tenant_id`
- `algorithm_result_id`
- `execution_request_id`
- `algorithm_id`
- `algorithm_version`
- `candidate_signal_type`
- `candidate_severity`
- `candidate_confidence`
- `source_event_ids`
- `feature_value_ids`
- `evidence_refs`
- `proposal_payload`
- `decision_status`: `proposed`, `accepted`, `rejected`, `superseded`, `restored`
- `decision_by`
- `decision_note`
- `created_at`
- `updated_at`

## Candidate Signal Mapping

Initial mappings should be conservative:

- `z_score`, `online_anomaly_score`, and `isolation_score` can propose `signalops.algorithm.anomaly_candidate`.
- `change_point_score` can propose `signalops.algorithm.change_point_candidate`.
- `forecast_residual` can propose `signalops.algorithm.forecast_deviation_candidate`.
- `classifier_label` can propose `signalops.algorithm.classification_candidate` only when the label is not `baseline`.

These signal types should remain distinct from MarketOps DSM taxonomy signals until a reviewed mapping proves semantic equivalence.

## Proposal Rules

A result may become a proposal only when all are true:

- result severity is `medium`, `high`, or `critical`;
- result confidence is greater than or equal to the configured proposal threshold;
- source event lineage is present;
- result payload is valid JSON;
- a matching proposal does not already exist for the same `algorithm_result_id` and candidate signal type.

## Operator Review

Operators should be able to:

- accept a proposal for later signal materialization;
- reject it as noise;
- supersede it with a better proposal;
- restore a previously rejected/superseded proposal.

Review decisions should train calibration and threshold decisions later, but should not immediately mutate detector policies.

## Alert And Insight Boundary

If proposals are later materialized:

- Alert creation should happen only for accepted/materialized candidate signals with event-level semantics.
- Insight generation should require grouped evidence across multiple events, symbols, windows, or repeated algorithm outputs.
- A single algorithm result should not generate both an alert and an insight with the same explanation.

## Explainability Requirements

Every materialized candidate must expose:

- source algorithm id/version;
- execution request id;
- result payload;
- score/confidence/severity;
- source event ids;
- feature refs;
- evidence refs;
- conversion policy version;
- operator decision, if any.

## Explicitly Out Of Scope

- Implementing the proposal table/API.
- Writing production `signal.v1` rows.
- Creating alerts or insights.
- Graph proposals.
- Frontend implementation.
- Automatic policy deployment.

## Recommended Next Implementation Gate

G112 should implement the proposal ledger and read-only APIs only. Signal materialization should wait until operators can inspect proposal quality over broader historical windows.
