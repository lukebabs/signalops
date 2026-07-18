# G134 Expanded Options Quality Gate Validation

Status: implemented validation gate

Date: 2026-07-18

## Purpose

G134 closes the loop between G133 expanded options coverage and G131 quality-aware proposal filtering.

G133 added persisted AAPL/MSFT options distribution feature rows. Their call/put open-interest ratio evidence is non-usable (`all_zero` and `denominator_zero`). G134 verifies that the generic algorithm layer can still write auditable results over those rows while the proposal layer blocks them from entering analyst review or production signal materialization.

## Scope

In scope:

- Run `signalops.algorithms.zscore_anomaly_v1` over expanded AAPL/MSFT `options_distribution_daily` feature rows.
- Generate proposals for the resulting execution with the G131 proposal gate enabled.
- Verify algorithm results persist.
- Verify low-quality options ratio results produce zero signal proposals.
- Verify no algorithm materializations or production algorithm signals are created.

Out of scope:

- No new ingestion.
- No Top 50 expansion beyond the G133 AAPL/MSFT validation data.
- No UI work.
- No materialization request.
- No policy deployment.

## Commands

Algorithm execution:

```bash
signalops-algorithm-runner   --tenant-id tenant-local   --algorithm-id signalops.algorithms.zscore_anomaly_v1   --dataset options_distribution_daily   --symbols AAPL,MSFT   --feature call_put_open_interest_ratio   --window-start 2026-07-01T00:00:00Z   --window-end 2026-07-19T00:00:00Z   --max-records 20   --batch-size 10   --min-samples 2   --z-threshold 3
```

Proposal generation:

```bash
signalops-algorithm-proposal-generator   --tenant-id tenant-local   --execution-request-id algexec_acbb37f455555b59a2b90fc1   --algorithm-id signalops.algorithms.zscore_anomaly_v1   --result-type z_score   --min-confidence 0   --limit 20   --created-by operator-local
```

## Validation Results

Execution request:

- `algexec_acbb37f455555b59a2b90fc1`

Algorithm runner metrics:

- scanned: 5
- usable numeric samples: 5
- results written: 5
- mean: 1.6
- stddev: 1.496663

Result quality breakdown:

- `AAPL all_zero`: 1
- `AAPL denominator_zero`: 2
- `MSFT all_zero`: 1
- `MSFT denominator_zero`: 1

Proposal generator metrics:

- scanned: 5
- proposed: 0
- skipped: 5

Ledger checks:

- `algorithm_signal_proposals`: 0 rows for the execution.
- `algorithm_signal_materializations`: 0 rows for the execution.
- `signal_ledger`: 0 production algorithm signal rows linked to the execution.

## Result

The algorithm substrate can score expanded options feature rows for audit, but G131 correctly prevents non-usable call/put open-interest ratio evidence from becoming reviewable proposals or production signals.

## Deferred

- Run the same validation on symbols with enough positive open-interest data to produce `usable` ratio quality.
- Increase per-symbol contract limits where provider budget allows.
- Run algorithm proposal generation across a mixed usable/non-usable symbol set and confirm only usable rows enter the proposal queue.
