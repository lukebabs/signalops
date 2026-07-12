# G086 Calibration Promotion Review

Status: backend/API implemented and authenticated smoke validated
Use case: MarketOps Daily Market Surveillance

## Goal

Add an operator-reviewed calibration promotion workflow that turns G083 baseline comparisons and G085 label-aware evaluations into auditable promotion candidates.

G086 should answer: "Is this detector/policy configuration ready to be proposed for promotion?" It should not deploy the configuration, change detector thresholds, alter runtime policy behavior, write graph state, or train models.

## Starting Point

G083 provides:

- named calibration baselines;
- stored baseline-to-candidate summary comparisons;
- advisory comparison recommendations such as `improvement_candidate`, `neutral_candidate`, `regression_candidate`, `manual_review_required`, and `needs_more_data`.

G084 provides:

- normalized evaluation labels from G080 operator graph proposal decisions.

G085 provides:

- persisted label-aware back-test evaluations;
- scoring metrics such as precision, recall, accuracy, label coverage, and confusion counts;
- advisory evaluation recommendations.

The current UI can surface baselines, comparisons, and label-aware evaluations on `/marketops/backtests`.

## Problem

The system can now measure candidate behavior, but there is no durable operator workflow for deciding what a candidate means operationally.

Without a first-class review layer, promotion discussions remain implicit in UI state or ad hoc documentation. That creates ambiguity around:

- which baseline comparison was used;
- which label-aware evaluation was used;
- whether the candidate had enough label coverage;
- whether the operator accepted, rejected, or deferred the candidate;
- what follow-up action is allowed;
- whether a later policy deployment gate is justified.

## Scope

Implement or specify a review substrate for calibration promotion candidates.

In scope:

- promotion candidate records that reference G083/G085 evidence;
- deterministic readiness checks over existing comparison and evaluation records;
- operator decisions on promotion candidates;
- immutable audit fields for decision actor, decision time, and decision note;
- read/create/list/detail APIs;
- frontend-agent specification for read/review UI if backend is implemented.

Out of scope:

- detector threshold editing;
- policy deployment;
- runtime feature-flag changes;
- graph writeback;
- automatic promotion;
- model training;
- model registry integration;
- PnL/trading simulation;
- production signal or graph proposal mutation.

## Promotion Candidate Model

Implemented table: `marketops_backtest_promotion_candidates`.

Fields:

- `candidate_id`
- `tenant_id`
- `app_id`
- `domain`
- `use_case`
- `baseline_id`
- `comparison_id`
- `evaluation_id`
- `run_id`
- `detector_id`
- `detector_version`
- `dataset`
- `policy_version`
- `candidate_version`
- `readiness_status`
- `readiness_reasons`
- `evidence`
- `status`
- `requested_by`
- `reviewed_by`
- `reviewed_at`
- `decision_note`
- `created_at`
- `updated_at`

Implemented `status` values:

- `proposed`
- `approved_for_promotion`
- `rejected`
- `deferred`
- `superseded`

Implemented `readiness_status` values:

- `ready_for_review`
- `needs_more_data`
- `manual_review_required`
- `regression_detected`
- `blocked`

The candidate should reference immutable evidence rows rather than copying their full raw payload as the source of truth. The `evidence` JSON can cache a compact snapshot for fast UI rendering and audit readability.

## Readiness Rules

G086 readiness rules should be deterministic and conservative.

Initial rule recommendations:

- If no comparison is present, status is `blocked`.
- If no evaluation is present, status is `manual_review_required`.
- If comparison recommendation is `regression_candidate`, status is `regression_detected`.
- If comparison recommendation is `needs_more_data`, status is `needs_more_data`.
- If evaluation recommendation is `needs_more_data`, status is `needs_more_data`.
- If evaluation label coverage is below a configured minimum, status is `needs_more_data`.
- If evaluation false positives are non-zero and precision is below threshold, status is `manual_review_required`.
- If comparison and evaluation recommendations are both favorable and coverage is sufficient, status is `ready_for_review`.

Initial review thresholds:

- minimum label coverage: `0.8`
- minimum precision: `0.9`
- minimum recall: `0.8`

These thresholds are review criteria only. They must not alter detector runtime behavior.

## Evidence Snapshot

The candidate `evidence` JSON should include compact, stable facts for the UI and audit trail:

- baseline id and summary id;
- comparison id, recommendation, reason, and key deltas;
- evaluation id, recommendation, note, precision, recall, accuracy, label coverage, candidate count, labeled count, TP/FP/TN/FN;
- detector id/version;
- run id;
- policy version;
- readiness status and reasons.

Do not embed bearer tokens, operator emails beyond existing actor identifiers, or full production payloads.

## API Shape

Implemented APIs:

- `POST /v1/marketops/backtest-promotion-candidates`
- `GET /v1/marketops/backtest-promotion-candidates`
- `GET /v1/marketops/backtest-promotion-candidates/{candidate_id}`
- `POST /v1/marketops/backtest-promotion-candidates/{candidate_id}/decision`

Create request:

```json
{
  "candidate_id": "optional-client-id",
  "tenant_id": "tenant-local",
  "baseline_id": "btbase-...",
  "comparison_id": "btcmp-...",
  "evaluation_id": "bteval-...",
  "candidate_version": "taxonomy-v1-policy-v1-candidate-20260712",
  "requested_by": "operator-local"
}
```

Decision request:

```json
{
  "status": "approved_for_promotion",
  "reviewed_by": "operator-local",
  "decision_note": "Approved for later deployment planning; no runtime change made by this action."
}
```

Decision actions must update only the promotion candidate row. They must not modify detector settings, policies, graph proposals, signals, alerts, or insights.

## Frontend Shape

If a frontend-agent spec is requested after backend implementation, it should add a compact `Promotion Review` panel to `/marketops/backtests`.

The panel should let an operator:

- select an existing baseline comparison;
- select an existing label-aware evaluation;
- create a promotion candidate;
- inspect readiness status and reasons;
- approve, reject, defer, or supersede the candidate as an audit decision.

The UI must label approvals as review decisions only, not deployed changes.

## Acceptance Criteria

G086 is complete when:

- promotion candidates can be created from existing G083/G085 evidence;
- readiness status is deterministic and stored;
- candidates can be listed and fetched by id;
- operator decisions can approve/reject/defer/supersede a candidate without runtime side effects;
- docs clearly state that policy deployment remains a future gate;
- tests cover readiness rules, tenant/evidence consistency, and decision mutations;
- authenticated API smoke validates create/list/detail/decision.

## Validation Plan

- Add unit tests for readiness rule outcomes.
- Add API route tests for create/list/detail/decision.
- Run targeted Go tests for `internal/api` and `internal/storage/postgres`.
- Run full Go tests.
- Run JSON schema validation.
- Apply migration locally.
- Rebuild gateway.
- Authenticated smoke:
  - create candidate from known G083 comparison and G085 evaluation;
  - list candidates;
  - fetch candidate detail;
  - submit decision;
  - verify no production ledgers changed.

## Follow-On Gate

G087 should handle deployment planning only after G086 review records exist. That gate should decide whether promotion means a stored policy config, a feature flag, a detector version pointer, or another controlled release mechanism.
