# MarketOps Back-Test Promotion Review UI Specification

Status: ready for frontend-agent implementation
Gate: G086 frontend follow-up
Author: Codex
Date: 2026-07-12
Backend baseline: `9858ca7 Implement G086 promotion candidates`
Validation baseline: authenticated smoke candidate `btpromo-g086-auth-smoke-20260712214200`

## Purpose

Wire `/marketops/backtests` to the G086 promotion candidate APIs so operators can package G083 baseline comparison evidence and optional G085 label-aware evaluation evidence into an auditable promotion-review record.

This is a review workflow only. The UI must not deploy policies, edit detector thresholds, write graph state, train models, or imply that an approval changes runtime behavior.

## Existing Context

The MarketOps back-test page already supports:

- isolated back-test run creation and inspection;
- persisted calibration snapshots;
- calibration baselines and stored comparisons;
- label-aware back-test evaluations.

G086 adds backend APIs for promotion candidates and operator review decisions.

## Backend Contract

Use the existing authenticated same-origin `/v1/*` API client pattern. Do not add a new base URL or auth mechanism.

### Create Promotion Candidate

```http
POST /v1/marketops/backtest-promotion-candidates
Content-Type: application/json
```

Request:

```json
{
  "candidate_id": "optional-client-id",
  "tenant_id": "tenant-local",
  "baseline_id": "btbase-g083-auth-smoke-20260712070500",
  "comparison_id": "btcmp-g083-auth-smoke-20260712070500",
  "evaluation_id": "bteval-g085-matched-smoke-20260712205000",
  "candidate_version": "taxonomy-v1-policy-v1-review-20260712"
}
```

Required fields:

- `tenant_id`
- `baseline_id`
- `comparison_id`

Optional fields:

- `candidate_id`
- `evaluation_id`
- `candidate_version`
- `requested_by`

Response envelope:

```json
{
  "promotion_candidate": {
    "candidate_id": "btpromo-...",
    "tenant_id": "tenant-local",
    "app_id": "marketops",
    "domain": "market_data",
    "use_case": "daily_market_surveillance",
    "baseline_id": "btbase-...",
    "comparison_id": "btcmp-...",
    "evaluation_id": "bteval-...",
    "run_id": "bt-...",
    "detector_id": "marketops.dsm.taxonomy_v1",
    "detector_version": "v1",
    "dataset": "equity_eod_prices",
    "policy_version": "marketops.backtest.policy_v1",
    "candidate_version": "taxonomy-v1-policy-v1-review-20260712",
    "readiness_status": "ready_for_review",
    "readiness_reasons": ["comparison and evaluation evidence meet review thresholds"],
    "evidence": {
      "baseline": {},
      "comparison": {},
      "evaluation": {},
      "detector": {},
      "run": {},
      "policy_version": "marketops.backtest.policy_v1",
      "readiness": {}
    },
    "status": "proposed",
    "requested_by": "operator-local",
    "reviewed_by": "",
    "decision_note": "",
    "created_at": "2026-07-12T21:42:00Z",
    "updated_at": "2026-07-12T21:42:00Z"
  }
}
```

Readiness status values:

- `ready_for_review`
- `needs_more_data`
- `manual_review_required`
- `regression_detected`
- `blocked`

### List Promotion Candidates

```http
GET /v1/marketops/backtest-promotion-candidates?tenant_id=tenant-local&baseline_id=btbase-...&status=proposed&limit=50
```

Response envelope:

```json
{
  "promotion_candidates": []
}
```

Supported filters:

- `tenant_id`
- `app_id`
- `domain`
- `use_case`
- `baseline_id`
- `comparison_id`
- `evaluation_id`
- `run_id`
- `detector_id`
- `dataset`
- `readiness_status`
- `status`
- `limit`

### Get Promotion Candidate

```http
GET /v1/marketops/backtest-promotion-candidates/{candidate_id}
```

Response envelope:

```json
{
  "promotion_candidate": {}
}
```

### Decide Promotion Candidate

```http
POST /v1/marketops/backtest-promotion-candidates/{candidate_id}/decision
Content-Type: application/json
```

Request:

```json
{
  "status": "deferred",
  "decision_note": "Reviewed; defer until more label coverage is available."
}
```

Required field:

- `status`

Allowed decision statuses:

- `approved_for_promotion`
- `rejected`
- `deferred`
- `superseded`

Optional fields:

- `reviewed_by`
- `decision_note`

The decision endpoint mutates only the promotion candidate row. It does not deploy anything.

## Frontend Work

Add TypeScript API types for:

- `MarketOpsBacktestPromotionCandidate`
- `MarketOpsBacktestPromotionCandidateCreateRequest`
- `MarketOpsBacktestPromotionCandidateDecisionRequest`
- list/detail/create/decision response envelopes
- evidence snapshot shape with optional `baseline`, `comparison`, `evaluation`, `detector`, `run`, and `readiness` blocks

Add API client methods near the existing MarketOps back-test client methods:

```ts
listMarketOpsBacktestPromotionCandidates(filter)
getMarketOpsBacktestPromotionCandidate(candidateId)
createMarketOpsBacktestPromotionCandidate(request)
decideMarketOpsBacktestPromotionCandidate(candidateId, request)
```

Add React Query hooks and query keys following existing back-test patterns:

```ts
useMarketOpsBacktestPromotionCandidates(filter)
useMarketOpsBacktestPromotionCandidate(candidateId)
useCreateMarketOpsBacktestPromotionCandidate()
useDecideMarketOpsBacktestPromotionCandidate()
```

Invalidate promotion candidate list/detail queries after create and decision mutations. Do not invalidate production DSM graph proposal, signal, alert, insight, or policy queries.

## UI Placement

Add a compact `Promotion Review` panel on `/marketops/backtests`, near the existing calibration baseline/comparison and label-aware evaluation panels.

The panel should:

- list recent promotion candidates for the current tenant and selected filters;
- allow creating a candidate from a selected baseline comparison;
- allow optionally attaching a selected label-aware evaluation;
- show readiness status and readiness reasons;
- show compact evidence summary;
- allow review decisions on existing candidates.

Keep the page operational and dense. Do not add a hero section or explanatory marketing copy.

## Required User Flow

1. Operator opens `/marketops/backtests`.
2. Operator creates or selects an existing baseline comparison.
3. Operator creates or selects an existing label-aware evaluation when available.
4. Operator creates a promotion candidate.
5. The candidate appears without a full page reload.
6. Operator reviews readiness status, reasons, comparison evidence, and evaluation evidence.
7. Operator records a decision: approve for promotion planning, reject, defer, or supersede.
8. The candidate row updates without a full page reload.

## Display Requirements

For each promotion candidate, display at least:

- candidate id;
- baseline id;
- comparison id;
- evaluation id, when present;
- run id, when present;
- readiness status;
- readiness reasons;
- candidate status;
- requested by;
- reviewed by, when present;
- reviewed at, when present;
- created time.

For evidence, display at least:

- comparison recommendation and reason;
- evaluation recommendation and note, when present;
- precision, recall, accuracy, and label coverage, when present;
- policy version and detector version, when present.

Use compact percentages for precision, recall, accuracy, and label coverage.

## Decision Controls

Render decision actions only for stored candidates. The controls should map to these statuses:

- Approve for promotion planning -> `approved_for_promotion`
- Reject -> `rejected`
- Defer -> `deferred`
- Supersede -> `superseded`

Use button labels that make the audit-only nature clear. Avoid labels like `Deploy`, `Promote Now`, `Enable`, `Activate`, or `Apply Thresholds`.

Decision notes should be optional but visible. Use the existing textarea/input style from the app.

## Error And Empty States

Handle these states explicitly:

- no baseline comparisons available: show empty state and disable create candidate;
- no label-aware evaluations available: allow candidate creation without evaluation, but expect readiness to require manual review;
- readiness `needs_more_data`: render as insufficient evidence, not a failure;
- readiness `regression_detected`: render as high-risk review state;
- authenticated `401`: preserve existing auth behavior;
- `404` detail fetch: use existing not-found/error presentation;
- invalid decision status or API validation error: surface existing form error pattern.

## Tests

Add or update tests for:

- API client methods and response envelope parsing;
- query key stability for promotion candidate filters;
- list render;
- create candidate mutation success invalidates expected queries;
- decision mutation success invalidates expected queries;
- readiness status and readiness reasons render;
- evidence metrics render as compact percentages;
- no deploy/threshold/graph-write controls appear in the promotion review panel.

Run at minimum:

```bash
cd web && npm test -- src/api/marketopsBacktests.test.ts src/lib/marketopsBacktests.test.ts
cd web && npm test
cd web && npm run build
```

Then rebuild and smoke the local web container:

```bash
docker compose up -d --build web
curl http://localhost:15173/marketops/backtests
```

## Acceptance Criteria

The task is complete when:

- `/marketops/backtests` can list promotion candidates.
- An operator can create a promotion candidate from an existing baseline comparison.
- An operator can optionally attach a label-aware evaluation.
- Readiness status, readiness reasons, and evidence summary are visible.
- An operator can submit approve-for-promotion-planning, reject, defer, or supersede decisions.
- The UI clearly states or implies that these are audit decisions only, not runtime deployment.
- Query invalidation refreshes candidate panels without a full page reload.
- Existing back-test run, persisted summary, baseline, comparison, evaluation, DSM, asset, replay, alert, and insight views still build and test.

## Non-Goals

Do not implement:

- detector threshold editing;
- policy deployment;
- feature flag changes;
- graph writeback;
- automatic promotion;
- model training;
- model registry workflows;
- PnL/trading simulation;
- label sync controls;
- backend API changes.

Deployment planning should remain a later gate after promotion-review records exist.
