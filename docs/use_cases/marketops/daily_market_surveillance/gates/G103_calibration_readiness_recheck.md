# G103 Calibration Readiness Re-Check

Status: live-validated advisory readiness snapshot.

## Purpose

G103 re-checks G094 calibration readiness after G102 added bounded multi-day equity and options evidence. This gate does not add ingestion breadth, change detectors, tune thresholds, or deploy policy. It only asks the existing readiness API whether the current evidence is sufficient for calibration readiness.

## Scope

- Use the existing G094 `/v1/marketops/backtest-calibration-readiness` API.
- Reuse the existing persisted baseline/comparison/evaluation/promotion evidence from G083-G086.
- Include both `equity_eod_prices` and `options_contracts_daily` in the dataset scope.
- Include the Top 50 universe group so symbol coverage is measured against the real MarketOps readiness target.
- Persist one advisory readiness snapshot.

## Request Evidence

Readiness snapshot:

- Readiness id: `btready-g103-recheck-20260714185649`.
- Baseline id: `btbase-g083-auth-smoke-20260712070500`.
- Comparison id: `btcmp-g083-auth-smoke-20260712070500`.
- Evaluation id: `bteval-g085-matched-smoke-20260712205000`.
- Promotion candidate id: `btpromo-g086-auth-smoke-20260712214200`.
- Dataset scope: `equity_eod_prices`, `options_contracts_daily`.
- Universe group: `top50_megacap`.
- Window: `2026-07-09T00:00:00Z` through `2026-07-14T00:00:00Z`.

## Live Validation Result

Status:

- `needs_more_historical_data`.

Readiness reasons:

- Historical Top 50/window coverage is below readiness thresholds.
- Options daily window coverage is below readiness threshold.
- Reviewed label volume or label coverage is below readiness thresholds.

Coverage metrics:

- Run count: `34`.
- Succeeded run count: `34`.
- Scanned: `30`.
- Covered symbols: `5`.
- Universe symbols: `50`.
- Symbol coverage ratio: `0.1`.
- Distinct historical windows: `8`.
- Options windows: `5`.
- Dataset counts: `equity_eod_prices=28`, `options_contracts_daily=6`.

Label and evaluation metrics:

- Matched labels: `7`.
- Distinct graph facts: `7`.
- Conflicting graph facts: `0`.
- Conflicting label ratio: `0`.
- Evaluation present: `true`.
- Evaluation precision: `1`.
- Evaluation recall: `1`.
- Evaluation label coverage: `1`.

Thresholds:

- Minimum symbol coverage ratio: `0.8`.
- Minimum historical windows: `20`.
- Minimum options windows: `10`.
- Minimum reviewed labels: `100`.
- Minimum label coverage ratio: `0.8`.
- Maximum conflicting label ratio: `0.05`.

## Result

G102 improved the evidence substrate, but calibration readiness is still correctly blocked. The remaining gaps are scale and reviewed labels, not API wiring or input path viability.

## Boundary

No runtime policy deployment, detector mutation, threshold relaxation, synthetic labels, graph writeback, frontend change, or additional ingestion was performed.
