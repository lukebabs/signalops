# G130 Options Distribution Quality Metrics

Status: implemented backend/data-quality substrate
Timestamp: 2026-07-17T00:00:00Z

## Purpose

G130 closes the immediate data-quality question for NVDA options distribution features.

The investigation confirmed that persisted `open_interest=0` values were present in Massive raw payloads, not caused by parser loss. The missing capability was explicit quality metadata to distinguish usable call/put ratios from all-zero or denominator-zero artifacts.

## Implemented Scope

- Added open-interest quality metrics to `marketops_options_distribution_daily.metrics`:
  - `open_interest_zero_count`;
  - `open_interest_positive_count`;
  - `open_interest_zero_rate`;
  - `open_interest_quality`;
  - `call_put_oi_denominator_is_zero`;
  - `call_put_oi_ratio_quality`.
- Added ratio quality categories:
  - `usable`;
  - `partial_zero`;
  - `all_zero`;
  - `denominator_zero`;
  - `missing`;
  - `empty`.
- Propagated the quality fields into normalized `options_distribution_daily` feature payloads.
- Added unit tests for all-zero and denominator-zero OI cases.

## Live NVDA Findings

After rerunning G129 backfill and G126 feature materialization:

- 27 distribution snapshots were regenerated.
- Ratio-quality breakdown:
  - `usable`: 9;
  - `all_zero`: 10;
  - `denominator_zero`: 6;
  - `partial_zero`: 2.
- The latest 2026-07-17 snapshot is `all_zero` because Massive returned `open_interest=0` for all 33 persisted contracts on that trade date.
- Large ratio outliers such as 2026-06-26 are now marked `denominator_zero`, preventing them from being treated as trustworthy call/put divergence without review.

## Validation

- Focused Go tests passed for `./internal/marketops/options`, `./cmd/marketops-options-distribution-backfill`, and `./cmd/marketops-options-feature-materializer`.
- Docker target builds passed for `marketops-options-distribution-backfill` and `marketops-options-feature-materializer`; the builds ran the full Go suite.
- Live NVDA backfill regenerated 27 distribution snapshots.
- Live feature materialization wrote 27 normalized feature rows with quality metadata into the temporal store.
- Algorithm runner still scanned 27 usable numeric samples and wrote 27 results, but proposal generation should be quality-filtered before use.

## Decision

Do not generate review proposals from the current `call_put_open_interest_ratio` results until the proposal path can filter or downgrade `call_put_oi_ratio_quality != usable`.

## Follow-On

- Add quality-aware filtering to algorithm proposal generation for options-distribution features.
- Surface `open_interest_quality` and `call_put_oi_ratio_quality` in the asset options UI so analysts understand whether a ratio is trustworthy.
