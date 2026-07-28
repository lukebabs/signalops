# Algorithm Outcome Evaluation Operations

`signalops-marketops-algorithm-evaluator` writes isolated MarketOps evaluation rows. It does not create production algorithm execution requests, production algorithm results, signals, proposals, policies, or model updates.

## Backfill Campaign

Plan the 400-calendar-day equity history campaign for the active Top 50:

```bash
signalops-marketops-algorithm-evaluation-backfill \
  --tenant-id tenant-local \
  --universe-group top50_megacap \
  --window-start 2025-06-21 \
  --window-end 2026-07-26 \
  --campaign-id algeval_equity_top50_20260726
```

The command records 10-symbol, 20-calendar-day children with strict per-child provider and event budgets. It makes no provider call unless both flags below are supplied:

```bash
--execute-pull --acknowledge-writes
```

The child process is the existing `signalops-massive-puller`, so every equity bar follows the raw-topic and normalizer path. Each completed child is checkpointed in the campaign manifest only after its normalized equity rows are visible in TimescaleDB; rerunning the same campaign ID resumes only unfinished children. Wait for normalization coverage before starting evaluation. Historical Massive options contract data is not treated as historical IV, Greeks, open interest, or quote history.

## Evaluation

Run both modes against provenance-complete persisted data. Add the registry-enforcement flag (or set SIGNALOPS_PLATFORM_REGISTRY_ENFORCEMENT=true) to reject input events or feature observations whose active platform-definition provenance cannot be verified:

```bash
signalops-marketops-algorithm-evaluator \
  --tenant-id tenant-local \
  --universe-group top50_megacap \
  --window-start 2025-06-21 \
  --window-end 2026-07-26 \
  --as-of 2026-07-25 \
  --modes retrospective,walk_forward \
  --run-id algeval_top50_20260726
```

Walk-forward mode scores a session with only earlier sessions. Retrospective mode is a diagnostic comparison and is not evidence of predictive performance. The evaluator records 1, 5, 10, and 20-session outcomes, score/event-study distributions, directional hit intervals, and forecast error metrics. Results are research evidence only.

Historical `risk_reward_temporal_v1` uses its technical/equity factors and records options corroboration as unavailable. Complete-input risk/reward cohorts become available only after enough prospective analytics-ready option captures accumulate.

## Read APIs

- `GET /v1/marketops/algorithm-evaluations`
- `GET /v1/marketops/algorithm-evaluations/{run_id}?tenant_id=...`
- `GET /v1/marketops/algorithm-evaluations/{run_id}/results?tenant_id=...`
- `GET /v1/marketops/algorithm-evaluations/{run_id}/outcomes?tenant_id=...`
- `GET /v1/marketops/algorithm-evaluation-backfills`
- `GET /v1/marketops/algorithm-evaluation-backfills/{campaign_id}?tenant_id=...`

A confidence interval reports uncertainty; it is not a promotion, approval, or deployment decision.
