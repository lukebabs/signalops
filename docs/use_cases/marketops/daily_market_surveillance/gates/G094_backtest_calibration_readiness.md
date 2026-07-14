# G094 Back-Test Calibration Readiness

Status: specification proposed
Use case: MarketOps Daily Market Surveillance

## Goal

Start the Back-Test And Calibration workstream that makes promotion decisions trustworthy before any runtime policy deployment is implemented.

G094 should answer: "Do we have enough historical coverage and reviewed labels to treat a promotion candidate as calibration-ready?" It should not deploy policies, change detector thresholds, switch runtime policy versions, mutate graph state, train models, or automate trading/PnL evaluation.

## Starting Point

Implemented substrate:

- G081: isolated back-test execution and run-scoped generated outputs.
- G082: persisted calibration summary snapshots over back-test runs.
- G083: named baselines and stored baseline comparisons.
- G084: graph proposal decisions normalized into evaluation labels.
- G085: label-aware back-test evaluations.
- G086: promotion candidate review records.
- G087: deployment planning specification only; no runtime execution.

Current risk:

- validation is still heavily smoke/seed oriented;
- label-aware metrics are only meaningful when enough reviewed labels match generated proposals;
- an approved promotion candidate is an audit decision, not evidence that runtime deployment is safe.

## Scope

G094 should define and then implement a calibration readiness layer over existing back-test, baseline, label, evaluation, and promotion records.

In scope:

- historical coverage criteria for Top 50 MarketOps assets;
- equity EOD and options daily dataset coverage accounting;
- minimum run counts and window coverage by symbol/dataset;
- label volume and label-quality thresholds;
- readiness snapshots that explain whether a detector/policy candidate has enough evidence;
- APIs or CLI helpers to create/list/detail readiness snapshots if implementation is approved;
- operator-facing documentation for how to produce broader historical runs without mutating production ledgers.

Out of scope:

- policy deployment or execution;
- runtime detector threshold edits;
- feature flag or configuration pointer mutation;
- graph writeback;
- production signal, alert, insight, artifact, or graph proposal mutation;
- model training;
- PnL or trading simulation;
- automatic acceptance of promotion candidates.

## Recommended Implementation Slice

Recommended first slice: **calibration readiness snapshots**.

Add a read/compute layer that evaluates existing records and stores or returns a compact readiness result. The readiness result should be advisory and auditable. It should be usable by G086/G087 reviewers but must not perform deployment.

A readiness snapshot should assess:

- run coverage: number of succeeded runs, symbols covered, datasets covered, and total scanned records;
- historical window coverage: earliest/latest observation window and number of distinct trading windows;
- Top 50 coverage: covered symbols versus current `top50_megacap` universe;
- options/equity balance: whether both `equity_eod_prices` and `options_contracts_daily` have meaningful coverage when the candidate claims options-aware behavior;
- label coverage: matched labels, unmatched labels, label coverage ratio, and stale-label count if timestamps are available;
- label quality: accepted/rejected/superseded distribution, reviewer count if available, and conflicting labels for the same graph fact key;
- evaluation strength: precision, recall, false positives, false negatives, recommendation, and `needs_more_data` reasons from G085;
- promotion/deployment block: whether the evidence satisfies minimum readiness thresholds.

## Initial Thresholds

Initial thresholds are conservative review gates, not runtime policy knobs.

Suggested MVP thresholds:

- Top 50 symbol coverage: at least `80%` before production deployment planning can be considered.
- Minimum historical windows: at least `20` distinct observation dates per covered symbol for an equity-only candidate.
- Options-aware candidate coverage: at least `10` distinct options daily windows for symbols used in options-derived claims.
- Minimum reviewed labels: at least `100` matched labels overall before label-aware metrics can support promotion.
- Minimum per-recommendation labels: at least `20` matched labels for each recommendation class that may be automated.
- Minimum label coverage ratio: `0.8`.
- Maximum conflicting-label ratio: `0.05`.
- G085 recommendation must not be `needs_more_data` or `regression_candidate`.

These values should be configurable in the snapshot request once implementation begins. Defaults should remain conservative.

## Readiness Status

Suggested status values:

- `calibration_ready`: historical coverage and label thresholds are sufficient for deployment planning review.
- `needs_more_historical_data`: back-test run/window/symbol coverage is insufficient.
- `needs_more_labels`: matched reviewed labels are insufficient.
- `label_quality_blocked`: conflicts or stale labels make the evaluation unreliable.
- `regression_detected`: comparison or evaluation evidence indicates a regression.
- `manual_review_required`: thresholds are mixed or evidence is sufficient but not clean enough for automatic recommendation.
- `blocked`: required baseline/comparison/evaluation evidence is missing.

A readiness snapshot can be favorable only when deployment remains a separate later decision. `calibration_ready` must not mean deployed.

## Historical Run Plan

Use existing isolated back-test APIs or CLI paths. Do not use operational replay for calibration unless the goal is explicitly to republish operational pipeline events.

Recommended run expansion:

1. Equity EOD sweep: Top 50 symbols, rolling historical windows, bounded by `max_records` per run.
2. Options daily sweep: symbols with normalized options coverage, separate runs from equity EOD.
3. Candidate comparison: summarize runs into G082 calibration summaries, compare to named G083 baseline.
4. Label sync: sync reviewed G080 graph proposal decisions into G084 labels.
5. Label-aware evaluation: run G085 over candidate outputs and synced labels.
6. Readiness snapshot: evaluate historical coverage plus label quality before any G086/G087 promotion review is treated as actionable.

## Label Volume And Quality

G084/G085 labels should be treated as operator-review evidence, not perfect ground truth.

The readiness layer should track:

- source proposal id and graph fact key;
- normalized label (`positive`, `negative`, or review token already supported by G084);
- decision source status from G080;
- decision timestamp and reviewer when present;
- duplicate labels for the same graph fact key;
- conflicting labels for the same graph fact key;
- stale labels whose source proposal semantics no longer match the current detector/policy candidate.

Promotion candidates should stay `needs_more_data` when label volume or quality is weak, even if smoke metrics look favorable.

## API Shape Proposal

If implemented as backend/API in the next slice, use a narrow MarketOps route family:

- `POST /v1/marketops/backtest-calibration-readiness`
- `GET /v1/marketops/backtest-calibration-readiness`
- `GET /v1/marketops/backtest-calibration-readiness/{readiness_id}`

Create request example:

```json
{
  "tenant_id": "tenant-local",
  "baseline_id": "btbase-...",
  "comparison_id": "btcmp-...",
  "evaluation_id": "bteval-...",
  "candidate_id": "btpromo-...",
  "dataset_scope": ["equity_eod_prices", "options_contracts_daily"],
  "universe_group": "top50_megacap",
  "window_start": "2026-06-01T00:00:00Z",
  "window_end": "2026-07-14T00:00:00Z",
  "thresholds": {
    "min_symbol_coverage_ratio": 0.8,
    "min_reviewed_labels": 100,
    "min_label_coverage_ratio": 0.8,
    "max_conflicting_label_ratio": 0.05
  }
}
```

Response should include status, reasons, coverage metrics, label metrics, referenced evidence ids, and threshold values used.

## Acceptance Criteria

G094 is accepted when:

- policy deployment is explicitly blocked until calibration readiness is favorable and separately approved;
- readiness criteria cover historical data breadth and label quality, not only smoke success;
- Top 50, equity EOD, and options daily coverage expectations are documented;
- label volume and conflict/staleness concerns are documented;
- implementation plan defines how readiness will be computed from existing G081-G086 records;
- no production ledgers or runtime policy settings are mutated.

## Validation Plan For Implementation

When implementation is approved:

- add unit tests for readiness status classification;
- add API tests for create/list/detail if routes are implemented;
- seed or identify runs with both sufficient and insufficient coverage;
- seed or identify labels with enough matched coverage and with conflicts;
- run targeted Go tests for the readiness package/API;
- run full Go tests;
- validate authenticated API smoke;
- confirm no production signal/artifact/proposal/graph tables are mutated.

## Follow-On Gates

- G095: historical back-test campaign runner or operator batch API for Top 50 equity/options windows.
- G096: label quality dashboard and conflict-resolution workflow.
- G097: persisted calibration readiness API/UI.
- G098: controlled policy deployment execution, only after readiness and deployment planning gates are accepted.

## Implemented Slice

Implemented on `2026-07-14T15:46:09Z`:

- Added migration `000021_marketops_backtest_calibration_readiness` for persisted readiness snapshots.
- Added storage records and repository methods for create/list/detail readiness snapshots.
- Added same-origin gateway routes:
  - `POST /v1/marketops/backtest-calibration-readiness`
  - `GET /v1/marketops/backtest-calibration-readiness`
  - `GET /v1/marketops/backtest-calibration-readiness/{readiness_id}`
- Implemented conservative readiness classification over G083 baseline/comparison evidence, optional G085 evaluation, optional G086 promotion candidate, succeeded G081 runs, G084 labels, and optional Top 50 universe assets.
- Kept policy deployment blocked: readiness snapshots are advisory evidence and do not mutate runtime detector/policy configuration or production graph/signal/artifact ledgers.

Validation performed:

- `docker run --rm -v ... golang:1.22-bookworm go test ./internal/api ./internal/storage/postgres -count=1`: passed.
- `docker run --rm -v ... golang:1.22-bookworm go test ./... -count=1`: passed.
- `python3 scripts/validate_json_schemas.py`: passed.

## Deploy And Live Validation Closeout

Validated on `2026-07-14T15:46:09Z`:

- Applied relational migration `000021_marketops_backtest_calibration_readiness` with `make compose-storage-migrate`.
- Rebuilt/restarted `signalops-gateway-1` with `docker compose -f compose.yaml -f compose.traefik.yaml up -d --build gateway`.
- Authenticated prerequisite list calls for baselines, comparisons, evaluations, and promotion candidates returned HTTP `200`.
- Authenticated `POST /v1/marketops/backtest-calibration-readiness` returned HTTP `201` for `btready-g094-auth-smoke-20260714154609`.
- Authenticated detail and filtered list reads returned HTTP `200`.
- The readiness snapshot correctly stayed conservative: `needs_more_historical_data` with `4/50` symbol coverage, `4` distinct windows, `0` options windows, and `7` matched labels.
- No runtime policy, detector threshold, graph state, production signal/artifact/proposal ledger, or promotion decision was mutated by readiness creation.
