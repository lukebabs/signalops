# G102 Bounded Multi-Day Campaign Expansion

Status: live-validated operational expansion.

## Purpose

G102 expands the calibration input substrate from single-day samples to a small multi-day slice across the already-proven equity and options daily datasets. The goal is to increase back-test breadth without introducing new architecture, detector changes, runtime deployment, or broad historical backfill.

## Scope

- Use the existing Massive puller, raw topic, normalizer, G096 coverage preflight, G095 campaign API, and G082 calibration summary API.
- Ingest three recent market days: `2026-07-09`, `2026-07-10`, and `2026-07-13`.
- Ingest bounded equity EOD samples for three Top 50 symbols per day.
- Ingest bounded options daily samples for one symbol per day.
- Run matching bounded campaigns over the data-bearing windows.
- Persist refreshed calibration summaries for `equity_eod_prices` and `options_contracts_daily`.

## Live Validation

Credential preflight:

- `scripts/marketops_massive_credential_preflight.sh` passed with Massive HTTP `200` for the older validation date `2026-07-10`.

Equity ingestion:

- Dates: `2026-07-09`, `2026-07-10`, `2026-07-13`.
- Per-date bounds: `MARKETOPS_INGEST_SMOKE_DATASETS=equity`, `MARKETOPS_INGEST_SMOKE_MAX_COMPANIES=3`, `MARKETOPS_INGEST_SMOKE_MAX_PROVIDER_REQUESTS=3`, `MARKETOPS_INGEST_SMOKE_MAX_EVENTS_BUILT=3`, `MARKETOPS_INGEST_SMOKE_MAX_EVENTS_PUBLISHED=3`.
- Aggregate result: `provider_requests=9`, `events_built=9`, `events_published=9`, `failures=0`.

Options ingestion:

- Dates: `2026-07-09`, `2026-07-10`, `2026-07-13`.
- Per-date bounds: `MARKETOPS_INGEST_SMOKE_DATASETS=options`, `MARKETOPS_INGEST_SMOKE_MAX_COMPANIES=1`, `MARKETOPS_INGEST_SMOKE_OPTIONS_LIMIT=3`, `MARKETOPS_INGEST_SMOKE_MAX_PROVIDER_REQUESTS=1`, `MARKETOPS_INGEST_SMOKE_MAX_EVENTS_BUILT=3`, `MARKETOPS_INGEST_SMOKE_MAX_EVENTS_PUBLISHED=3`.
- Aggregate result: `provider_requests=3`, `events_built=9`, `events_published=9`, `events_by_dataset={"options_contracts_daily":9}`, `failures=0`.

Coverage:

- `equity_eod_prices` coverage over `2026-07-09T00:00:00Z` through `2026-07-14T00:00:00Z` returned `10` events across `AAPL`, `GOOGL`, `NVDA`, and prior `SPY` coverage.
- Scoped G102 equity campaign symbols were `AAPL`, `GOOGL`, and `NVDA`, with `3` events each.
- `options_contracts_daily` coverage over the same window returned `10` events across `NVDA` and prior `SPY` coverage.
- Scoped G102 options campaign symbol was `NVDA`, with `9` events.

Campaigns:

- Equity campaign id: `btcamp-g102-equity3x3-20260714185013`.
- Equity campaign status: `succeeded`.
- Equity planned child runs: `15`.
- Equity completed child runs: `15`.
- Equity failed child runs: `0`.
- Equity scanned: `9`.
- Equity signals: `1`.
- Equity artifacts: `1`.
- Equity policy results: `5`.

- Options campaign id: `btcamp-g102-options1x3-20260714185013`.
- Options campaign status: `succeeded`.
- Options planned child runs: `5`.
- Options completed child runs: `5`.
- Options failed child runs: `0`.
- Options scanned: `9`.
- Options signals: `0`.
- Options artifacts: `0`.
- Options policy results: `0`.

Calibration summaries:

- Equity summary id: `btsum-g102-equity3x3-20260714185032`.
- Equity run count: `28`.
- Equity succeeded count: `28`.
- Equity zero-input count: `10`.
- Equity scanned: `18`.
- Equity signals: `6`.
- Equity artifacts: `6`.
- Equity policy results: `30`.
- Equity recommendation counts: `auto_accept_candidate=25`, `manual_review_required=5`.

- Options summary id: `btsum-g102-options1x3-20260714185032`.
- Options run count: `6`.
- Options succeeded count: `6`.
- Options zero-input count: `2`.
- Options scanned: `12`.
- Options signals: `0`.
- Options artifacts: `0`.
- Options policy results: `0`.

## Result

MarketOps calibration now has a bounded multi-day substrate across both equity EOD and options daily inputs. The campaign and summary layers can consume that substrate without schema changes or runtime policy deployment.

## Boundary

No runtime policy deployment, detector mutation, direct ledger insert, synthetic data generation, graph writeback, frontend change, or broad Top 50/options historical backfill was performed. G094 readiness snapshots were not forced because that API requires baseline/comparison evidence and this gate was limited to data-bearing campaign breadth.
