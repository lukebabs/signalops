# Hypothesis Backtest And Calibration Operations

Use `signalops-marketops-hypothesis-backtest` to summarize persisted research evaluations and matured outcomes inside the isolated MarketOps back-test ledgers. The command never writes operational signals, proposals, graph records, lifecycle mutations, or promotions.

## Single Version

```bash
signalops-marketops-hypothesis-backtest \
  --tenant-id tenant-local \
  --hypothesis-key H001 \
  --hypothesis-versions v1 \
  --symbols AAPL \
  --mode single \
  --window-start 2026-01-01 \
  --window-end 2026-07-20 \
  --as-of 2026-07-20 \
  --minimum-sample-size 100
```

## Version Comparison

Versions are ordered as baseline, candidate.

```bash
signalops-marketops-hypothesis-backtest \
  --hypothesis-key H001 \
  --hypothesis-versions v1,v2 \
  --symbols AAPL,MSFT \
  --mode comparison \
  --window-start 2026-01-01 \
  --window-end 2026-07-20 \
  --as-of 2026-07-20
```

The comparison is advisory. It does not create a promotion candidate or alter either definition.

## Walk Forward

```bash
signalops-marketops-hypothesis-backtest \
  --hypothesis-key H004 \
  --hypothesis-versions v1 \
  --symbols AAPL \
  --mode walk_forward \
  --window-start 2026-01-01 \
  --window-end 2026-07-20 \
  --as-of 2026-07-20 \
  --train-sessions 60 \
  --test-sessions 20 \
  --max-folds 6
```

Training expands chronologically. Each evaluation session belongs to at most one test fold.

## Dry Run And Bounds

Add `--dry-run` to calculate and print the report without creating a back-test run or calibration summary.

Operational bounds:

- one exact hypothesis key;
- one version for single/walk-forward, exactly two for comparison;
- explicit symbols only, maximum 10;
- hard query limit 1-1,000 per version and symbol;
- exact outcome calculation version, default `marketops.forward_outcome.v1`;
- as-of date on or after the evaluation window end;
- maximum 20 walk-forward folds.

A query that reaches its declared limit fails rather than returning a silently truncated report.

## Reading Results

Use the existing APIs:

- `GET /v1/marketops/backtests/{run_id}`
- `GET /v1/marketops/backtest-calibration-summaries/{summary_id}`
- `GET /v1/marketops/backtest-calibration-summaries?tenant_id=tenant-local&detector_id=marketops.hypothesis.h001`

The versioned hypothesis report is returned in the calibration summary `parameters` field.

## Interpretation

Treat `below_minimum_sample_size=true`, unknown event/regime segments, empty folds, or zero matured samples as evidence limitations. The threshold counts independent triggered evaluations with at least one matured outcome, not repeated horizons. Do not reinterpret sparse results as neutral performance. `promotion_allowed` is always false; deployment and lifecycle decisions remain separate operator-controlled workflows.
