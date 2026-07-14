# G101 Bounded Options Campaign Expansion

Status: live-validated operational expansion.

## Purpose

G101 extends the calibrated back-test input path from equity EOD rows to a bounded options daily sample, proving that options coverage can be ingested, discovered, consumed by a G095 campaign, and summarized without widening into broad historical ingestion.

## Scope

- Use the existing Massive puller, raw topic, normalizer, and MarketOps back-test APIs.
- Ingest a bounded `options_contracts_daily` sample for `2026-07-13`.
- Confirm G096 coverage returns a data-bearing options row.
- Run a matching G095 campaign over that options row.
- Persist a G082 calibration summary over succeeded MarketOps taxonomy options runs.

## Live Validation

Credential preflight:

- `scripts/marketops_massive_credential_preflight.sh` passed with Massive HTTP `200`.

Ingestion smoke:

- Command used bounded overrides: `MARKETOPS_INGEST_SMOKE_DATASETS=options`, `MARKETOPS_INGEST_SMOKE_MAX_COMPANIES=1`, `MARKETOPS_INGEST_SMOKE_OPTIONS_LIMIT=3`, `MARKETOPS_INGEST_SMOKE_MAX_PROVIDER_REQUESTS=1`, `MARKETOPS_INGEST_SMOKE_MAX_EVENTS_BUILT=3`, `MARKETOPS_INGEST_SMOKE_MAX_EVENTS_PUBLISHED=3`.
- Result: `provider_requests=1`, `events_built=3`, `events_published=3`, `events_by_dataset={"options_contracts_daily":3}`, `failures=0`.

Coverage:

- `GET /v1/marketops/backtest-coverage` returned one `NVDA` `options_contracts_daily` row for `2026-07-13` with `event_count=3`.

Campaign:

- Campaign id: `btcamp-g101-options3-20260714180221`.
- Status: `succeeded`.
- Planned child runs: `1`.
- Completed child runs: `1`.
- Failed child runs: `0`.
- Scanned: `3`.
- Signals: `0`.
- Child run id: `btcamp-g101-options3-20260714180221-options-contracts-daily-nvda-20260713`.

Calibration summary:

- Summary id: `btsum-g101-options3-20260714180416`.
- Filtered summary scope: succeeded MarketOps taxonomy runs for `tenant-local`, `src-massive`, `options_contracts_daily`.
- Run count: `1`.
- Succeeded count: `1`.
- Zero-input count: `0`.
- Scanned: `3`.
- Signals: `0`.
- Artifacts: `0`.
- Policy results: `0`.

## Result

The back-test/calibration substrate is no longer equity-only at the input layer. Options daily rows can flow through Massive ingestion, normalized coverage, campaign execution, and calibration summary persistence.

## Boundary

No runtime policy deployment, detector mutation, direct ledger insert, synthetic data generation, graph writeback, or broad options-history campaign was performed.
