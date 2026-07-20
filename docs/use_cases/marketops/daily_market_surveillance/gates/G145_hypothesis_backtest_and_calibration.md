# G145 Hypothesis Backtest And Calibration

Status: implemented backend adapter and versioned calibration contract

Date: 2026-07-20

## Purpose

G145 adapts research hypothesis evaluations and forward outcomes into the existing isolated MarketOps back-test substrate. It adds reproducible single-version, comparison, and chronological walk-forward reports without creating production signals, changing hypothesis lifecycle state, or treating sparse live evidence as calibration readiness.

## Implemented Scope

- Adds `internal/marketops/hypothesisbacktest`, a deterministic adapter over persisted hypothesis evaluations, forward outcomes, and point-in-time segment observations.
- Adds `cmd/marketops-hypothesis-backtest` and a distroless `marketops-hypothesis-backtest` Docker target.
- Reuses `marketops_backtest_runs` for isolated execution audit and `marketops_backtest_calibration_summaries` for report persistence.
- Stores the full report in the existing calibration summary `parameters` object under schema `marketops.hypothesis_calibration.v1`; existing calibration-summary list/detail APIs expose it without a parallel ledger or migration.
- Supports:
  - `single`: one exact hypothesis version;
  - `comparison`: exactly two distinct versions, ordered baseline then candidate;
  - `walk_forward`: one version with an expanding chronological train window and non-overlapping test folds.
- Requires an explicit symbol cohort capped at 10 and applies a hard per-version, per-symbol query limit of 1,000.
- Requires the exact hypothesis key, hypothesis version, and outcome calculation version. G145 defaults to `marketops.forward_outcome.v1`.
- Produces evaluation, eligible-state, trigger, trigger-rate, matured-sample, directional-hit, mean/median return, favorable/adverse excursion, drawdown-incidence, realized-volatility-change, confidence-band precision, and Brier calibration-error metrics.
- Segments results by horizon, symbol, year, earnings window, and point-in-time volatility term-structure regime.
- Emits explicit minimum-sample warnings overall and for walk-forward train/test folds, counting independent triggered evaluations rather than inflating the threshold across horizons.
- Supports dry-run calculation with no back-test run or calibration-summary writes.

## Point-In-Time And Isolation Semantics

Every ledger query includes an exact hypothesis key and version. Outcome queries additionally include the exact calculation version and hypothesis-evaluation source type. The calculator rejects cross-version or unmatched evaluation/outcome input instead of silently pooling it.

Only evaluations inside the inclusive requested window and known no later than the requested as-of date are admitted. Only matured outcomes whose maturity date is no later than that as-of date contribute to calibration metrics. Pending, missing-price, later-maturing, non-triggered, ineligible, and invalidated sources do not become negative samples or zero returns.

Earnings and term-structure segments use same-symbol, same-session usable feature observations known no later than the evaluation timestamp. If multiple eligible revisions exist, the latest `as_of_time`, then greatest deterministic observation ID, wins. Unavailable segment context remains `unknown`.

Walk-forward folds sort distinct evaluation sessions, use an expanding train prefix, and assign each subsequent test block once. Test observations never enter an earlier train boundary.

## Existing Substrate Mapping

The adapted back-test record uses:

- `source_id=marketops.research_ledgers`;
- `source_adapter=marketops.hypothesis_backtest`;
- `dataset=hypothesis_evaluations`;
- `detector_id=marketops.hypothesis.<key>`;
- exact hypothesis versions in `detector_version`, filters, and report payload.

Generic summary counters map scanned rows to evaluations, signals to triggered research evaluations, and artifacts to matured outcome samples. Rich hypothesis metrics remain in the versioned report payload rather than being mislabeled as graph proposals or policy decisions.

## Validation

Focused tests prove:

- exact hypothesis-version and outcome-calculation-version isolation;
- comparison versions remain independently summarized;
- chronological expanding walk-forward folds have non-overlapping test windows;
- outcome maturity after the requested as-of date cannot leak into a report;
- later event revisions cannot change an earlier evaluation segment;
- event/regime, horizon, asset, year, and confidence-band aggregation;
- minimum-sample warnings and permanent `promotion_allowed=false`;
- dry runs perform no writes;
- persisted runs and summaries use the existing isolated back-test repositories.

Focused storage and API regression tests pass after adding the exact outcome calculation-version query predicate. Full repository validation is recorded in the build journal and gate audit.

## Safety Boundary

G145 does not synthesize evaluations or outcomes, lower the G141 source-coverage threshold, call a provider, train a model, calculate trading PnL or transaction costs, write operational signals/artifacts/proposals, mutate graph state, change hypothesis lifecycle status, create promotion candidates, or approve deployment.

A favorable comparison is advisory metadata only. Sparse or zero live samples remain explicit warnings. Existing G094 readiness and operator review controls remain separate.

## Deferred Architecture Surface

Cross-sectional ranking, evidence-domain ablation, 3/60-session outcomes, bootstrap confidence intervals, historical universe membership, reviewed-label false-positive/missed-opportunity metrics, frontend calibration views, and automatic baseline/comparison record creation remain later bounded work. They are not inferred from unavailable evidence.

## Next Gate

G146 should bridge only eligible hypothesis lifecycle states into the existing proposal workflow, preserve preflight/review/materialization controls, and add opportunity analyst disposition. G145 reports may inform review but must not trigger promotion automatically.

## Links

- Canonical architecture: `../../../../design/syncratic_marketops_market_state_intelligence_architecture_v1.md`
- Reconciliation: `../architecture/market_state_intelligence_evaluation.md`
- Operations: `../operations/hypothesis_backtesting.md`
- G144 feature completion: `G144_market_feature_and_transition_completion.md`
- Adapter: `../../../../../internal/marketops/hypothesisbacktest/calibration.go`
- Runner: `../../../../../internal/marketops/hypothesisbacktest/runner.go`
