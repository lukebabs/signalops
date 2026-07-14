# G100 Bounded Equity Campaign Expansion

Status: live-validated operational expansion.

## Purpose

G100 expands from the single NVDA input smoke to a small bounded Top 50 equity sample so G095 campaigns can be validated against multiple data-bearing normalized events without starting broad historical ingestion.

## Scope

- Use the existing Massive puller, raw topic, normalizer, and back-test campaign APIs.
- Ingest a bounded equity EOD sample for `2026-07-13`.
- Confirm G096 coverage returns multiple symbols.
- Run a matching G095 campaign over those symbols.
- Persist a G082 calibration summary over succeeded MarketOps taxonomy runs.

## Live Validation

Ingestion smoke:

- Command used bounded overrides: `MARKETOPS_INGEST_SMOKE_DATASETS=equity`, `MARKETOPS_INGEST_SMOKE_MAX_COMPANIES=3`, `MARKETOPS_INGEST_SMOKE_MAX_PROVIDER_REQUESTS=3`, `MARKETOPS_INGEST_SMOKE_MAX_EVENTS_BUILT=3`, `MARKETOPS_INGEST_SMOKE_MAX_EVENTS_PUBLISHED=3`.
- Result: `provider_requests=3`, `events_built=3`, `events_published=3`, `failures=0`.

Coverage:

- `GET /v1/marketops/backtest-coverage` returned `event_count=1` for each of `AAPL`, `GOOGL`, and `NVDA` on `2026-07-13`.

Campaign:

- Campaign id: `btcamp-g100-equity3-20260714175733`.
- Status: `succeeded`.
- Completed child runs: `3`.
- Scanned: `3`.
- Signals: `0`.

Calibration summary:

- Summary id: `btsum-g100-equity3-20260714175749`.
- Filtered summary scope: succeeded MarketOps taxonomy runs for `tenant-local`, `src-massive`, `equity_eod_prices`.
- Run count: `13`.
- Succeeded count: `13`.
- Zero-input count: `4`.
- Scanned: `9`.
- Signals: `5`.
- Policy results: `25`.

## Result

The back-test/calibration substrate is no longer blocked on empty input for equity EOD. The next narrow calibration step is to repeat this over additional dates and then add a small options sample so G094 readiness can move beyond sparse equity-only evidence.

## Boundary

No runtime policy deployment, detector mutation, direct ledger insert, synthetic data generation, or graph writeback was performed.
