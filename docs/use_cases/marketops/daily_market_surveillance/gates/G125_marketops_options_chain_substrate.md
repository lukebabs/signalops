# G125 MarketOps Options Chain Substrate

Status: implemented backend substrate
Timestamp: 2026-07-16T00:00:00Z

## Purpose

G125 starts the MarketOps options workflow from the asset universe by adding a durable options-chain and derived distribution substrate for a first NVDA-focused slice.

The goal is to support asset-level inspection and later algorithm scoring over call/put divergence without forcing algorithms to consume raw option-contract rows directly.

## Implemented Scope

- Added `marketops_options_chain_daily` for persisted full-chain rows keyed by tenant, symbol, trade date, and option ticker.
- Added `marketops_options_distribution_daily` for derived per-symbol daily feature snapshots over the canonical `10_trade_days` window.
- Added repository records and Postgres read/upsert methods for chain rows, coverage, and distribution snapshots.
- Added a deterministic distribution builder using open interest as the primary basis and volume as the secondary basis.
- Added same-origin gateway read APIs for coverage, distribution, and chain rows under the existing asset route family.
- Added a reserved live-preview endpoint that returns `501 live_preview_not_configured` until a Massive live client is explicitly wired.

## Distribution Features

The first derived snapshot includes:

- call/put open-interest ratio;
- call/put volume ratio;
- total call and put open interest;
- total call and put volume;
- contract counts and missing-open-interest count;
- strike moneyness buckets: `<90%`, `90-95%`, `95-100%`, `100-105%`, `105-110%`, `>110%`;
- expiration buckets: `0-7d`, `8-30d`, `31-60d`, `61d+`;
- 10-trade-day ratio delta, ratio percent change, z-score, change-point score, and confidence.

## API Surface

- `GET /v1/tenants/{tenant_id}/marketops/assets/{symbol}/options/coverage`
- `GET /v1/tenants/{tenant_id}/marketops/assets/{symbol}/options/distribution?window=10_trade_days&limit=10`
- `GET /v1/tenants/{tenant_id}/marketops/assets/{symbol}/options/chain?trade_date=YYYY-MM-DD&contract_type=call&limit=500`
- `POST /v1/tenants/{tenant_id}/marketops/assets/{symbol}/options/live-preview`

## Boundaries

- No Top 50 batch expansion yet.
- No frontend implementation in this gate.
- No algorithm execution or signal proposal generation over options distribution snapshots yet.
- No live Massive preview wiring yet.
- No alert, insight, graph, Syncratic, or materialization fanout.

## Validation

- Focused API tests cover coverage, distribution, chain, and live-preview boundary routes.
- Distribution unit tests cover open-interest ratios, volume ratios, moneyness buckets, expiration buckets, and divergence metrics.
- Focused Go tests passed for `./internal/api`, `./internal/storage/postgres`, and `./internal/marketops/options`.

## Follow-On

- G126 should emit or expose `options_distribution_daily` as algorithm-ready normalized feature evidence and run existing algorithms over `call_put_open_interest_ratio` / divergence metrics.
- G127 should expand bounded ingestion and distribution creation from NVDA to the active Top 50 asset universe with provider request budgets.
- G128 should hand off the asset-detail options UX to frontend-agent.
