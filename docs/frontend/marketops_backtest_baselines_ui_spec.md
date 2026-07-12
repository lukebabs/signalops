# MarketOps Back-Test Baselines UI Specification

Status: ready for frontend-agent implementation
Gate: G083 frontend follow-up
Author: Codex
Date: 2026-07-12
Backend baseline: `db88f39 Implement G083 backtest baselines`

## Purpose

Wire the `/marketops/backtests` frontend to the G083 baseline and comparison APIs so operators can turn a persisted calibration summary into a named baseline, compare another persisted summary against it, and inspect the stored comparison result.

This is a UI wiring task over the existing backend substrate. It must not introduce policy promotion, graph writeback, label scoring, detector threshold editing, or model training controls.

## Existing Context

The route already exists:

```text
/marketops/backtests
```

The page already supports:

- isolated back-test run creation and inspection;
- run-scoped generated signals, graph proposals, and policy results;
- transient calibration comparison metrics;
- persisted G082 calibration snapshots through `/v1/marketops/backtest-calibration-summaries`.

G083 adds backend APIs for named baselines and stored baseline-to-candidate comparisons.

## Backend Contract

Use the existing authenticated same-origin API client pattern. Do not add a separate base URL or auth mechanism.

### Baselines

Create baseline:

```http
POST /v1/marketops/backtest-calibration-baselines
Content-Type: application/json
```

Request:

```json
{
  "baseline_id": "optional-client-id",
  "tenant_id": "tenant-local",
  "name": "Taxonomy July baseline",
  "description": "Optional operator note",
  "summary_id": "btcal-g082-auth-smoke-20260712062745",
  "scope": {
    "symbols": ["AAPL"]
  },
  "status": "active"
}
```

Required fields:

- `tenant_id`
- `name`
- `summary_id`

Optional fields:

- `baseline_id`
- `description`
- `scope`
- `status`: `active` or `archived`
- `created_by`

Response envelope:

```json
{
  "calibration_baseline": {
    "baseline_id": "btbase-...",
    "tenant_id": "tenant-local",
    "app_id": "marketops",
    "domain": "market_data",
    "use_case": "daily_market_surveillance",
    "name": "Taxonomy July baseline",
    "description": "Optional operator note",
    "summary_id": "btcal-...",
    "detector_id": "marketops.dsm.taxonomy_v1",
    "dataset": "equity_eod_prices",
    "scope": {},
    "status": "active",
    "created_by": "operator-local",
    "created_at": "2026-07-12T07:05:00Z",
    "updated_at": "2026-07-12T07:05:00Z"
  }
}
```

List baselines:

```http
GET /v1/marketops/backtest-calibration-baselines?tenant_id=tenant-local&dataset=equity_eod_prices&detector_id=marketops.dsm.taxonomy_v1&status=active&limit=50
```

Response envelope:

```json
{
  "calibration_baselines": []
}
```

Get baseline:

```http
GET /v1/marketops/backtest-calibration-baselines/{baseline_id}
```

Response envelope:

```json
{
  "calibration_baseline": {}
}
```

### Comparisons

Create comparison:

```http
POST /v1/marketops/backtest-calibration-comparisons
Content-Type: application/json
```

Request:

```json
{
  "comparison_id": "optional-client-id",
  "tenant_id": "tenant-local",
  "baseline_id": "btbase-...",
  "candidate_summary_id": "btcal-..."
}
```

Required fields:

- `tenant_id`
- `baseline_id`
- `candidate_summary_id`

Optional fields:

- `comparison_id`
- `created_by`

Response envelope:

```json
{
  "calibration_comparison": {
    "comparison_id": "btcmp-...",
    "tenant_id": "tenant-local",
    "baseline_id": "btbase-...",
    "baseline_summary_id": "btcal-baseline",
    "candidate_summary_id": "btcal-candidate",
    "detector_id": "marketops.dsm.taxonomy_v1",
    "dataset": "equity_eod_prices",
    "comparison_metrics": {
      "baseline": {},
      "candidate": {},
      "deltas": {}
    },
    "recommendation": "neutral_candidate",
    "recommendation_reason": "candidate is within baseline tolerance bands",
    "created_by": "operator-local",
    "created_at": "2026-07-12T07:05:00Z"
  }
}
```

List comparisons:

```http
GET /v1/marketops/backtest-calibration-comparisons?tenant_id=tenant-local&baseline_id=btbase-...&recommendation=neutral_candidate&limit=50
```

Response envelope:

```json
{
  "calibration_comparisons": []
}
```

Get comparison:

```http
GET /v1/marketops/backtest-calibration-comparisons/{comparison_id}
```

Response envelope:

```json
{
  "calibration_comparison": {}
}
```

Recommendation values:

- `needs_more_data`
- `regression_candidate`
- `improvement_candidate`
- `neutral_candidate`
- `manual_review_required`

Treat these as advisory labels only. Do not render them as deploy/promote decisions.

## Frontend Work

Add TypeScript API types for:

- `MarketOpsBacktestCalibrationBaseline`
- `MarketOpsBacktestCalibrationComparison`
- create/list/detail request and response shapes for both resources
- comparison metrics shape with `baseline`, `candidate`, and `deltas`

Add API client methods near the existing MarketOps back-test client methods:

```ts
listMarketOpsBacktestCalibrationBaselines(filter)
getMarketOpsBacktestCalibrationBaseline(baselineId)
createMarketOpsBacktestCalibrationBaseline(request)
listMarketOpsBacktestCalibrationComparisons(filter)
getMarketOpsBacktestCalibrationComparison(comparisonId)
createMarketOpsBacktestCalibrationComparison(request)
```

Add React Query hooks and query keys following the existing back-test summary patterns:

```ts
useMarketOpsBacktestCalibrationBaselines(filter)
useMarketOpsBacktestCalibrationBaseline(baselineId)
useCreateMarketOpsBacktestCalibrationBaseline()
useMarketOpsBacktestCalibrationComparisons(filter)
useMarketOpsBacktestCalibrationComparison(comparisonId)
useCreateMarketOpsBacktestCalibrationComparison()
```

Invalidate baseline list/detail queries after baseline creation. Invalidate comparison list/detail queries after comparison creation. Do not invalidate production signal, DSM artifact, or graph proposal queries.

## UI Placement

Add a compact panel on `/marketops/backtests` near the existing `Persisted Calibration Snapshots` panel.

Suggested panel title:

```text
Calibration Baselines
```

The panel should include:

- active baseline list for the current tenant/detector/dataset filters;
- create-baseline control that uses a selected persisted calibration summary;
- baseline name input;
- optional description input;
- candidate summary selector from persisted calibration summaries;
- compare action that creates a stored comparison;
- recent comparisons list for the selected baseline.

Keep the UI dense and operational. Do not add marketing copy, explanatory hero sections, or decorative cards.

## Required User Flow

1. Operator opens `/marketops/backtests`.
2. Operator sees persisted calibration summaries from G082.
3. Operator selects one persisted summary and creates a named baseline.
4. New baseline appears without full page reload.
5. Operator selects a baseline and a candidate persisted summary.
6. Operator creates a stored comparison.
7. New comparison appears without full page reload.
8. Operator can inspect recommendation, recommendation reason, and key deltas.

## Display Requirements

For baselines, display at least:

- name;
- status;
- summary id;
- detector id;
- dataset;
- created time.

For comparisons, display at least:

- comparison id;
- baseline summary id;
- candidate summary id;
- recommendation;
- recommendation reason;
- created time.

For comparison metrics, display the key deltas that exist in `comparison_metrics.deltas`:

- `run_count_delta`
- `zero_input_rate_delta`
- `scanned_delta`
- `signal_yield_delta`
- `policy_results_per_signal_delta`
- `auto_accept_candidate_share_delta`
- `auto_reject_candidate_share_delta`
- `manual_review_required_share_delta`
- `supersede_existing_candidate_share_delta`
- `dominant_recommendation_changed`

Format ratios as compact percentages where that matches the existing calibration summary UI. Keep raw JSON expandable only if the existing page already uses raw expandable blocks for diagnostics.

## Error And Empty States

Handle these states explicitly:

- no persisted summaries available: disable baseline creation and show the existing empty-state style;
- no active baselines available: allow creating one from a persisted summary;
- baseline creation validation errors: surface API error text using existing form error pattern;
- comparison creation validation errors: surface API error text using existing form error pattern;
- authenticated `401`: preserve existing auth behavior;
- `404` detail fetch: show the existing not-found/error presentation.

## Tests

Add or update unit tests for:

- API client methods and response envelope parsing;
- query key stability for baseline and comparison filters;
- baseline list render;
- create baseline mutation success invalidates expected queries;
- comparison list render;
- create comparison mutation success invalidates expected queries;
- no production graph proposal mutation controls appear in the baseline/comparison panel.

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

- `/marketops/backtests` can list active baselines from `/v1/marketops/backtest-calibration-baselines`.
- An operator can create a baseline from a persisted calibration summary.
- An operator can create a stored comparison between a selected baseline and candidate summary.
- The comparison list renders recommendation, reason, and key deltas.
- Query invalidation refreshes baseline/comparison panels without a full page reload.
- Existing back-test run, persisted summary, DSM, asset, replay, alert, and insight views still build and test.

## Non-Goals

Do not implement:

- detector threshold editing;
- policy deployment or promotion;
- graph writeback;
- accept/reject/supersede/restore controls;
- label extraction from G080 operator decisions;
- precision/recall or label-aware scoring;
- ML training or model registry workflows;
- PnL or trading simulation;
- changes to backend APIs.

Label-aware scoring and operator-decision normalization should remain a later gate.
