# G126 Options Distribution Algorithm Features

Status: implemented backend/CLI substrate
Timestamp: 2026-07-17T00:00:00Z

## Purpose

G126 makes G125 options distribution snapshots consumable by the existing SignalOps algorithm runner.

The key design choice is to feed algorithms from derived per-symbol `options_distribution_daily` normalized feature events rather than raw option-chain rows.

## Implemented Scope

- Added `NormalizedEventFromDistribution` to convert one persisted `marketops_options_distribution_daily` snapshot into a canonical `normalized_event_ledger` row.
- Added dataset `options_distribution_daily` with MarketOps metadata: `app_id=marketops`, `domain=market_data`, `use_case=daily_market_surveillance`, and `source_adapter=market_data.massive`.
- Added feature payload fields for:
  - `call_put_open_interest_ratio`;
  - `call_put_volume_ratio`;
  - `call_put_oi_ratio_delta`;
  - `call_put_oi_ratio_change_pct`;
  - `call_put_oi_zscore`;
  - `call_put_oi_change_point_score`;
  - total call/put open interest and volume.
- Added `signalops-marketops-options-feature-materializer` CLI to upsert feature events from persisted distribution snapshots.
- Added a Docker build target for the materializer.

## Usage

Example dry run:

```sh
signalops-marketops-options-feature-materializer   --tenant-id tenant-local   --symbol NVDA   --window 10_trade_days   --limit 10   --dry-run
```

When a separate Timescale temporal store is enabled, pass `SIGNALOPS_TEMPORAL_DATABASE_URL` to the materializer. The algorithm runner reads normalized events through the temporal connection.

After feature rows exist, existing algorithms can run over `options_distribution_daily`, for example:

```sh
signalops-algorithm-runner   --tenant-id tenant-local   --algorithm-id signalops.algorithms.zscore_anomaly_v1   --dataset options_distribution_daily   --symbols NVDA   --feature call_put_open_interest_ratio   --window-start 2026-07-01T00:00:00Z   --window-end 2026-07-17T00:00:00Z
```

## Boundaries

- No provider ingestion or Massive live calls in this gate.
- No automatic signal proposal generation in this gate.
- No production signal materialization.
- No frontend changes.
- No Top 50 expansion.

## Validation

- Feature-event unit test validates canonical MarketOps metadata, normalized payload, feature map, and derived-feature ingestion mode.
- CLI tests validate write and dry-run behavior.
- Focused Go tests passed for `./internal/marketops/options`, `./cmd/marketops-options-feature-materializer`, and `./internal/algorithms`.
- Full Go suite, JSON schema validation, and Docker target build passed.
- Local Compose dry-run returned `scanned=0`, `upserted=0`, and `dry_run=true`, which is expected before G127 populates distribution rows.

## Follow-On

- G127 should populate persisted options chain/distribution data for NVDA last 10 trade days using Massive option-chain snapshots with strict provider budgets.
- G128 should hand frontend-agent the asset-detail options UX spec after backend data is available.
