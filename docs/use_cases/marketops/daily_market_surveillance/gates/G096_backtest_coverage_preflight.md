# G096 Back-Test Coverage Preflight

Status: implemented backend/API slice.

## Purpose

G096 closes the immediate operational gap discovered after G095 live validation: campaigns can be valid and still scan zero rows if the normalized MarketOps event ledger has no data-bearing windows. The gate adds a read-only preflight API so operators can inspect available input before launching calibration campaigns.

## Implemented Scope

- `GET /v1/marketops/backtest-coverage`.
- Normalized event coverage grouped by dataset and subject symbol.
- Same default MarketOps metadata boundary used by campaigns: `app_id=marketops`, `domain=market_data`, `use_case=daily_market_surveillance`.
- Optional dataset, source, symbol, and window filters.

## Out Of Scope

- Data ingestion or replay.
- Synthetic seed generation.
- Runtime policy deployment.
- Detector or graph mutation.
- Campaign creation side effects.

## Validation

The endpoint should return `200` with an empty `coverage` array when no normalized MarketOps rows exist. That is the expected local state before broader historical ingestion/backfill.
