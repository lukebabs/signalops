# G087 Deployment Planning

Status: specification proposed
Use case: MarketOps Daily Market Surveillance

## Goal

Add a deployment-planning substrate that turns an approved G086 promotion candidate into an auditable release plan.

G087 should answer: "What exactly would be deployed later, through which mechanism, with which evidence and rollback plan?" It should not execute the deployment, change detector thresholds, switch runtime policy versions, write graph state, train models, or alter production signal behavior.

## Starting Point

G086 provides promotion candidates with operator review decisions:

- `marketops_backtest_promotion_candidates`
- `POST /v1/marketops/backtest-promotion-candidates`
- `GET /v1/marketops/backtest-promotion-candidates`
- `GET /v1/marketops/backtest-promotion-candidates/{candidate_id}`
- `POST /v1/marketops/backtest-promotion-candidates/{candidate_id}/decision`

A candidate can be marked `approved_for_promotion`, but that approval is intentionally audit-only. It is permission to plan deployment, not permission to mutate runtime behavior.

## Problem

The system now has evidence and review approval, but it still lacks a formal release object describing what a later deployment would change.

Without a deployment plan record, operators cannot reliably answer:

- which approved candidate is being prepared for release;
- whether the release target is detector version, policy version, feature flag, or configuration pointer;
- what environment is targeted;
- what rollout strategy is intended;
- what preflight checks are required;
- what rollback plan exists;
- who requested, reviewed, and approved the plan;
- whether execution is still blocked pending a later gate.

## Scope

Implement or specify durable deployment plan records for approved MarketOps promotion candidates.

In scope:

- deployment plan records referencing G086 promotion candidates;
- release target type and target payload;
- environment and rollout strategy metadata;
- preflight checklist state;
- rollback plan metadata;
- plan review states;
- create/list/detail APIs;
- status transition API for plan review decisions;
- frontend-agent specification for read/review UI if backend is implemented.

Out of scope:

- actual deployment execution;
- runtime detector threshold changes;
- runtime policy pointer changes;
- feature flag mutation;
- graph writeback;
- production signal/artifact/proposal mutation;
- model training or model registry writes;
- Kubernetes, Helm, or infrastructure rollout mechanics;
- automatic rollback.

## Deployment Plan Model

Suggested table: `marketops_backtest_deployment_plans`.

Fields:

- `plan_id`
- `tenant_id`
- `app_id`
- `domain`
- `use_case`
- `candidate_id`
- `candidate_status_at_creation`
- `detector_id`
- `detector_version`
- `policy_version`
- `candidate_version`
- `target_environment`
- `target_type`
- `target_payload`
- `rollout_strategy`
- `preflight_checks`
- `rollback_plan`
- `evidence`
- `status`
- `requested_by`
- `reviewed_by`
- `reviewed_at`
- `decision_note`
- `created_at`
- `updated_at`

Suggested `target_type` values:

- `detector_version_pointer`
- `policy_version_pointer`
- `configuration_bundle`
- `feature_flag_plan`
- `manual_runbook`

Suggested `target_environment` values:

- `local`
- `staging`
- `production`

Suggested `status` values:

- `draft`
- `ready_for_review`
- `approved_for_execution`
- `rejected`
- `deferred`
- `superseded`

`approved_for_execution` is still not execution. It means a later G088 execution gate may consume this plan.

## Plan Creation Rules

Creation should be conservative:

- Candidate must exist.
- Candidate tenant must match the request tenant.
- Candidate status should be `approved_for_promotion`; otherwise plan creation should either fail or create `draft` with a blocking preflight reason. Recommended MVP: fail with `400` unless candidate status is `approved_for_promotion`.
- Target type must be one of the allowed tokens.
- Target payload must be JSON and must not contain secrets.
- Rollback plan must be present for `staging` or `production` targets.
- Production target plans should default to `draft`; they should require explicit review before `approved_for_execution`.

## Evidence Snapshot

The deployment plan `evidence` JSON should include compact references and facts:

- promotion candidate id;
- promotion candidate readiness status and status;
- baseline/comparison/evaluation ids;
- detector id/version;
- policy version;
- candidate version;
- readiness reasons;
- decision note from G086;
- target type and target environment;
- rollout strategy summary;
- rollback plan summary.

Do not copy bearer tokens, secrets, raw provider payloads, or full production event data.

## API Shape

Suggested APIs:

- `POST /v1/marketops/backtest-deployment-plans`
- `GET /v1/marketops/backtest-deployment-plans`
- `GET /v1/marketops/backtest-deployment-plans/{plan_id}`
- `POST /v1/marketops/backtest-deployment-plans/{plan_id}/decision`

Create request:

```json
{
  "plan_id": "optional-client-id",
  "tenant_id": "tenant-local",
  "candidate_id": "btpromo-...",
  "target_environment": "staging",
  "target_type": "manual_runbook",
  "target_payload": {
    "runbook": "Prepare policy pointer update for review; no automatic execution."
  },
  "rollout_strategy": {
    "mode": "manual_review",
    "steps": ["verify candidate evidence", "prepare release diff", "request execution approval"]
  },
  "preflight_checks": {
    "requires_gateway_health": true,
    "requires_recent_backtest": true,
    "requires_rollback_plan": true
  },
  "rollback_plan": {
    "mode": "manual_revert",
    "summary": "Restore previous policy pointer if execution gate fails post-deploy checks."
  },
  "requested_by": "operator-local"
}
```

Decision request:

```json
{
  "status": "approved_for_execution",
  "reviewed_by": "operator-local",
  "decision_note": "Approved for later execution gate; no runtime change made by this action."
}
```

Decision actions must update only the deployment plan row.

## Frontend Shape

If a frontend-agent spec is requested after backend implementation, add a compact `Deployment Plans` panel to `/marketops/backtests` near `Promotion Review`.

The panel should let an operator:

- select an approved promotion candidate;
- create a deployment plan;
- inspect target type, environment, rollout strategy, preflight checks, rollback plan, and evidence;
- approve, reject, defer, or supersede the plan as an audit/review decision.

The UI must clearly label approval as approval for a later execution gate, not execution.

## Acceptance Criteria

G087 is complete when:

- deployment plans can be created only from approved promotion candidates;
- deployment plans can be listed and fetched by id;
- deployment plan review decisions can be recorded without runtime side effects;
- target type, target environment, rollout strategy, preflight checks, rollback plan, and evidence are persisted;
- docs clearly state that actual execution remains a future gate;
- tests cover candidate status gating, tenant consistency, target validation, and decision transitions;
- authenticated API smoke validates create/list/detail/decision.

## Validation Plan

- Add unit tests for plan creation validation.
- Add API route tests for create/list/detail/decision.
- Add storage tests where repository behavior needs coverage.
- Run targeted Go tests for `internal/api` and `internal/storage/postgres`.
- Run full Go tests.
- Run JSON schema validation.
- Apply migration locally.
- Rebuild gateway.
- Authenticated smoke:
  - create or reuse an `approved_for_promotion` candidate;
  - create a deployment plan;
  - list plans;
  - fetch plan detail;
  - submit review decision;
  - verify no runtime policy, detector, graph, signal, alert, or insight tables changed.

## Follow-On Gate

G088 should define controlled execution mechanics only after deployment plans exist. That gate must decide whether execution is manual-only, feature-flag driven, configuration-pointer based, or a separate release service workflow.
