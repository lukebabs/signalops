# MarketOps Back-Test Evaluations UI Specification

Status: ready for frontend-agent implementation
Gate: G085 frontend follow-up
Author: Codex
Date: 2026-07-12
Backend baseline: `01b00b0 Implement G085 label-aware evaluations`
Validation baseline: matched-label smoke `bteval-g085-matched-smoke-20260712205000`

## Purpose

Wire `/marketops/backtests` to the G085 label-aware evaluation APIs so operators can score a selected back-test run against synchronized G084 labels and inspect precision/recall style results.

This is an evaluation visibility workflow only. It must not add policy promotion, detector threshold editing, graph writeback, model training, or automatic acceptance controls.

## Existing Context

The MarketOps back-test page already supports:

- isolated back-test run creation and inspection;
- persisted calibration snapshots;
- calibration baselines and stored comparisons;
- run-scoped generated graph proposals and policy results.

G084 added evaluation labels from graph proposal decisions. G085 added stored label-aware back-test evaluations.

## Backend Contract

Use the existing authenticated same-origin `/v1/*` API client pattern.

### Create Evaluation

```http
POST /v1/marketops/backtest-evaluations
Content-Type: application/json
```

Request:

```json
{
  "evaluation_id": "optional-client-id",
  "tenant_id": "tenant-local",
  "run_id": "bt-g081-ui-closeout-spy-20260712",
  "label_source": "g080_graph_proposal_decision"
}
```

Required fields:

- `tenant_id`
- `run_id`

Optional fields:

- `evaluation_id`
- `label_source`
- `requested_by`

Response envelope:

```json
{
  "backtest_evaluation": {
    "evaluation_id": "bteval-...",
    "tenant_id": "tenant-local",
    "app_id": "marketops",
    "domain": "market_data",
    "use_case": "daily_market_surveillance",
    "run_id": "bt-...",
    "detector_id": "marketops.dsm.taxonomy_v1",
    "dataset": "equity_eod_prices",
    "label_source": "g080_graph_proposal_decision",
    "label_version": "marketops.eval_label.v1",
    "scoring_version": "marketops.eval_scoring.v1",
    "requested_by": "operator-local",
    "candidate_count": 5,
    "labeled_count": 5,
    "positive_count": 5,
    "negative_count": 0,
    "superseded_count": 0,
    "unresolved_count": 0,
    "true_positive": 5,
    "false_positive": 0,
    "true_negative": 0,
    "false_negative": 0,
    "manual_review_count": 0,
    "unscored_count": 0,
    "precision": 1,
    "recall": 1,
    "specificity": 0,
    "accuracy": 1,
    "label_coverage": 1,
    "recommendation": "improvement_candidate",
    "recommendation_note": "automatic recommendations align with available labels",
    "metrics": {
      "matched_samples": []
    },
    "created_at": "2026-07-12T20:50:00Z"
  }
}
```

### List Evaluations

```http
GET /v1/marketops/backtest-evaluations?tenant_id=tenant-local&run_id=bt-...&limit=50
```

Response envelope:

```json
{
  "backtest_evaluations": []
}
```

Supported filters:

- `tenant_id`
- `app_id`
- `domain`
- `use_case`
- `run_id`
- `detector_id`
- `dataset`
- `recommendation`
- `limit`

### Get Evaluation

```http
GET /v1/marketops/backtest-evaluations/{evaluation_id}
```

Response envelope:

```json
{
  "backtest_evaluation": {}
}
```

## Frontend Work

Add TypeScript API types for:

- `MarketOpsBacktestEvaluation`
- `MarketOpsBacktestEvaluationCreateRequest`
- list/detail/create response envelopes
- metrics shape with optional `matched_samples` and `scoring_notes`

Add API client methods near existing MarketOps back-test client methods:

```ts
listMarketOpsBacktestEvaluations(filter)
getMarketOpsBacktestEvaluation(evaluationId)
createMarketOpsBacktestEvaluation(request)
```

Add React Query hooks and query keys following existing back-test patterns:

```ts
useMarketOpsBacktestEvaluations(filter)
useMarketOpsBacktestEvaluation(evaluationId)
useCreateMarketOpsBacktestEvaluation()
```

Invalidate evaluation list/detail queries after create. Do not invalidate production DSM graph proposal or signal queries.

## UI Placement

Add a compact `Label-Aware Evaluations` panel on `/marketops/backtests`, near the existing calibration baseline/comparison panels.

The panel should:

- list recent evaluations for the current tenant and selected run when available;
- allow creating an evaluation for the selected run;
- show a clear empty state when no run is selected;
- show a clear empty state when no labels match the selected run;
- avoid any controls that imply deployment, promotion, or automatic decisioning.

Keep the page operational and dense. Do not add a hero section or explanatory marketing copy.

## Display Requirements

For each evaluation, display at least:

- evaluation id;
- run id;
- recommendation;
- recommendation note;
- candidate count;
- labeled count;
- label coverage;
- precision;
- recall;
- accuracy;
- true positive, false positive, true negative, false negative;
- manual review count;
- created time.

Use compact percentages for `precision`, `recall`, `specificity`, `accuracy`, and `label_coverage`.

Recommendation tokens:

- `needs_more_data`
- `manual_review_required`
- `improvement_candidate`
- `neutral_candidate`
- `regression_candidate`

Treat recommendation tokens as advisory evaluation outcomes, not commands.

## Required User Flow

1. Operator opens `/marketops/backtests`.
2. Operator selects a back-test run.
3. Operator sees existing label-aware evaluations for that run, if any.
4. Operator clicks create evaluation.
5. The new evaluation appears without a full page reload.
6. Operator can inspect precision, recall, label coverage, recommendation, and confusion counts.

## Empty And Warning States

Handle these states explicitly:

- no selected run: disable create evaluation;
- no evaluation rows: show existing empty-state style;
- evaluation with `labeled_count=0`: show `needs_more_data` and explain through existing subtle helper/error style that no synchronized labels matched the run;
- low `label_coverage`: display coverage prominently;
- API validation error: surface existing form error pattern;
- authenticated `401`: preserve existing auth behavior;
- `404` detail fetch: use existing not-found/error presentation.

## Tests

Add or update tests for:

- API client envelope parsing for list/detail/create;
- query key stability for evaluation filters;
- evaluation list rendering;
- create evaluation mutation invalidates expected queries;
- `needs_more_data` rendering for zero label coverage;
- percentage formatting for precision/recall/coverage;
- no policy deployment, graph writeback, threshold edit, or graph proposal decision controls appear in this panel.

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

- `/marketops/backtests` can list label-aware evaluations from `/v1/marketops/backtest-evaluations`.
- An operator can create an evaluation for a selected run.
- Precision, recall, label coverage, recommendation, and confusion counts render correctly.
- Zero-label evaluations render as `needs_more_data` without implying failure of the API.
- Existing back-test run, persisted summary, baseline/comparison, DSM, asset, replay, alert, and insight views still build and test.

## Non-Goals

Do not implement:

- policy promotion;
- detector threshold editing;
- graph writeback;
- proposal accept/reject/supersede/restore controls;
- label creation or label sync controls;
- ML training or model registry workflows;
- PnL or trading simulation;
- backend API changes.
