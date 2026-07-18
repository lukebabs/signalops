# G135 Live Options Positive Quality Path

Status: implemented validation gate

Date: 2026-07-18

## Purpose

G135 validates the positive side of the G131/G133 options-quality workflow using a real live Massive pull for a non-NVDA symbol.

G134 proved that non-usable AAPL/MSFT options ratio results are blocked from proposal generation. G135 finds and persists a bounded non-NVDA sample with at least one `call_put_oi_ratio_quality=usable` row, then verifies that only the usable row becomes a signal proposal.

## Scope

In scope:

- Run a bounded live Massive dry-run over a small Top 50 slice.
- Identify a non-NVDA symbol with usable options call/put OI ratio evidence.
- Persist that bounded symbol pull.
- Materialize options distribution feature rows.
- Run z-score algorithm scoring over the persisted feature rows.
- Run quality-aware proposal generation.
- Verify only usable evidence creates a proposal.
- Verify no materializations or production algorithm signals are created.

Out of scope:

- No scheduler.
- No broad Top 50 write run.
- No UI work.
- No production materialization.
- No policy deployment.

## Live Pull

Dry-run command:

```bash
signalops-marketops-options-coverage-runner   --tenant-id tenant-local   --universe-group top50_megacap   --max-symbols 5   --limit 50   --max-pages 1   --window-days 10   --distribution-limit 10   --dry-run
```

Dry-run result:

- symbols processed: 5
- fetched contracts: 250
- converted contracts: 250
- distributions built: 51
- writes: 0
- aggregate quality counts: `usable=10`, `partial_zero=2`, `all_zero=15`, `denominator_zero=24`
- non-NVDA usable evidence found in `AMZN`.

Persisted command:

```bash
signalops-marketops-options-coverage-runner   --tenant-id tenant-local   --symbols AMZN   --max-symbols 1   --limit 50   --max-pages 1   --window-days 10   --distribution-limit 10
```

Persisted result:

- run id: `optcov_0db0a5614c70aca430674449`
- fetched: 50
- converted: 50
- skipped: 0
- chain upserted: 50
- distributions upserted: 4
- normalized feature rows upserted: 4
- quality counts: `usable=1`, `all_zero=1`, `denominator_zero=2`

## Algorithm And Proposal Validation

Algorithm execution:

- execution request: `algexec_cb5de5407e4a222ff1a24992`
- algorithm: `signalops.algorithms.zscore_anomaly_v1`
- dataset: `options_distribution_daily`
- feature: `call_put_open_interest_ratio`
- symbol: `AMZN`
- scanned: 4
- usable numeric samples: 4
- results: 4
- mean: 5426.5
- stddev: 9322.278758

Result quality breakdown:

- `AMZN usable`: 1
- `AMZN all_zero`: 1
- `AMZN denominator_zero`: 2

Proposal generation:

- scanned: 4
- proposed: 1
- skipped: 3
- proposal id: `algsigprop_bede162c6a016bc5ecabc8d6`
- proposal quality: `usable`
- non-usable proposals: 0

Ledger checks:

- `algorithm_signal_proposals`: 1 row for the execution, all usable.
- `algorithm_signal_materializations`: 0 rows for the execution.
- `signal_ledger`: 0 production algorithm signal rows linked to the execution.

## Result

G135 confirms the complete quality-aware positive path for non-NVDA options data: live provider pull, persisted chain/distribution/features, algorithm scoring, and proposal generation only for usable call/put OI ratio evidence.

## Deferred

- Run broader capped Top 50 pulls with larger per-symbol limits.
- Find additional symbols with multiple usable ratio dates for stronger algorithm calibration.
- Run mixed-symbol proposal validation once a larger usable/non-usable sample exists.
- Materialization remains a separate explicit operator decision.
