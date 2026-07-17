# MarketOps Asset Options Distribution UI Specification

Status: proposed for frontend-agent implementation
Gate: G128 frontend follow-up
Author: Codex
Date: 2026-07-17
Backend baseline: G125-G127, latest backend commit `9d30aa8`

## Purpose

Add an asset-level options intelligence panel to `/marketops/assets` so an analyst can inspect persisted options-chain coverage and call/put distribution evidence for a selected stock such as NVDA.

The UI should make options divergence interpretable without turning the frontend into an ingestion console or algorithm-tuning surface.

## Scope

In scope:

- Extend the existing asset detail experience for `/marketops/assets`.
- Fetch persisted options coverage, distribution snapshots, and chain rows for the selected asset.
- Render call/put open-interest and volume ratios over the available rolling window.
- Render moneyness and expiration bucket distributions from the latest distribution snapshot.
- Let an analyst filter persisted chain rows by trade date and contract type.
- Show enough provider/run metadata to build trust: provider, source id, ingestion run id, trade date, payload hash, last updated.
- Handle empty state cleanly when an asset has no persisted options rows yet.

Out of scope:

- No provider ingestion trigger button.
- No Massive live-preview button. The backend route remains `501 live_preview_not_configured`.
- No Top 50 batch controls.
- No algorithm execution controls.
- No signal proposal/materialization controls.
- No raw provider payload expansion by default; raw JSON may be available behind an explicit row detail disclosure only if existing UI patterns support that safely.

## Backend Contract

Use existing authenticated gateway conventions. Auth is required when enabled.

Coverage:

```http
GET /v1/tenants/{tenant_id}/marketops/assets/{symbol}/options/coverage
Authorization: Bearer <access_token>
```

Distribution snapshots:

```http
GET /v1/tenants/{tenant_id}/marketops/assets/{symbol}/options/distribution?window=10_trade_days&limit=10
Authorization: Bearer <access_token>
```

Chain rows:

```http
GET /v1/tenants/{tenant_id}/marketops/assets/{symbol}/options/chain?trade_date=YYYY-MM-DD&contract_type=call&limit=500
Authorization: Bearer <access_token>
```

Do not call in this UI task:

```http
POST /v1/tenants/{tenant_id}/marketops/assets/{symbol}/options/live-preview
```

## Response Shapes

Coverage response:

```json
{
  "options_coverage": {
    "tenant_id": "tenant-local",
    "symbol": "NVDA",
    "trade_day_count": 27,
    "contract_count": 250,
    "first_trade_date": "2025-12-02T00:00:00Z",
    "last_trade_date": "2026-07-17T00:00:00Z",
    "last_updated_at": "2026-07-17T03:33:03Z"
  }
}
```

Distribution response:

```json
{
  "options_distributions": [
    {
      "tenant_id": "tenant-local",
      "symbol": "NVDA",
      "trade_date": "2026-07-17T00:00:00Z",
      "window_name": "10_trade_days",
      "provider": "massive",
      "trade_days": 5,
      "contract_count": 33,
      "call_contract_count": 3,
      "put_contract_count": 30,
      "total_call_open_interest": 0,
      "total_put_open_interest": 0,
      "total_call_volume": 0,
      "total_put_volume": 0,
      "missing_open_interest_count": 33,
      "call_put_open_interest_ratio": 0,
      "call_put_volume_ratio": 0,
      "ratio_delta": 0,
      "ratio_change_pct": 0,
      "ratio_zscore": 0,
      "change_point_score": 0,
      "confidence": 0,
      "moneyness_distribution": { "95-100%": { "call_open_interest": 0, "put_open_interest": 0 } },
      "expiration_distribution": { "8-30d": { "call_open_interest": 0, "put_open_interest": 0 } },
      "source_trade_dates": ["2026-07-13T00:00:00Z", "2026-07-17T00:00:00Z"],
      "updated_at": "2026-07-17T03:33:03Z"
    }
  ]
}
```

Chain response:

```json
{
  "options_chain": [
    {
      "tenant_id": "tenant-local",
      "symbol": "NVDA",
      "trade_date": "2026-07-17T00:00:00Z",
      "option_ticker": "O:NVDA260717C00170000",
      "provider": "massive",
      "source_id": "src-massive",
      "ingestion_run_id": "optchain_...",
      "contract_type": "call",
      "expiration_date": "2026-07-17T00:00:00Z",
      "strike_price": 170,
      "underlying_close": 172.5,
      "moneyness": 0.9855,
      "volume": 123,
      "open_interest": 1543,
      "implied_volatility": 0.45,
      "delta": 0.51,
      "payload_hash": "...",
      "updated_at": "2026-07-17T03:33:03Z"
    }
  ]
}
```

Use type guards over `unknown`. JSON object fields such as `moneyness_distribution`, `expiration_distribution`, `metrics`, and `raw_payload` are already parsed by the gateway; do not call `JSON.parse` on them.

## Required Implementation

### 1. Types

Add frontend types near the existing MarketOps asset types:

- `MarketOpsOptionsCoverage`
- `MarketOpsOptionsCoverageResponse`
- `MarketOpsOptionsDistribution`
- `MarketOpsOptionsDistributionsResponse`
- `MarketOpsOptionsChainRow`
- `MarketOpsOptionsChainResponse`
- `MarketOpsOptionsChainFilter`

Recommended unions:

```ts
export type MarketOpsOptionContractType = 'call' | 'put';
export type MarketOpsOptionsWindowName = '10_trade_days';
```

Bucket JSON can be represented as:

```ts
export interface MarketOpsOptionsBucketTotals {
  call_open_interest?: number;
  put_open_interest?: number;
  call_volume?: number;
  put_volume?: number;
  contract_count?: number;
}
```

### 2. API Client

Add client methods:

```ts
getMarketOpsOptionsCoverage(tenantId: string, symbol: string): Promise<MarketOpsOptionsCoverageResponse>
listMarketOpsOptionsDistributions(tenantId: string, symbol: string, filter?: { window?: string; limit?: number }): Promise<MarketOpsOptionsDistributionsResponse>
listMarketOpsOptionsChain(tenantId: string, symbol: string, filter?: MarketOpsOptionsChainFilter): Promise<MarketOpsOptionsChainResponse>
```

List query params should include only defined values:

- distribution: `window`, `limit`; default `window=10_trade_days`, `limit=10`.
- chain: `trade_date`, `contract_type`, `limit`; default `limit=500`.

### 3. React Query Hooks

Add query keys and hooks:

```ts
marketOpsOptionsCoverage(tenantId, symbol)
marketOpsOptionsDistributions(tenantId, symbol, filter)
marketOpsOptionsChain(tenantId, symbol, filter)
```

Hooks:

```ts
useMarketOpsOptionsCoverage(tenantId, symbol)
useMarketOpsOptionsDistributions(tenantId, symbol, filter)
useMarketOpsOptionsChain(tenantId, symbol, filter)
```

Enable queries only when `tenantId` and `symbol` are present. Chain query should wait until a selected trade date is available from the latest distribution or user selection.

### 4. Asset Route Integration

Update the asset detail area for `/marketops/assets`. When an asset row/card is selected:

- Fetch coverage.
- Fetch distributions with `window=10_trade_days&limit=10`.
- Select the latest distribution by `trade_date` as the default chain trade date.
- Fetch chain rows for that date with default `limit=500`.

Do not create a new top-level route unless the existing asset page already uses nested detail routes.

### 5. UI Requirements

Add an `Options` panel or tab in the selected asset detail. Recommended layout:

- Coverage strip: trade days, contracts, first/last trade date, last updated.
- Latest distribution summary: call/put OI ratio, call/put volume ratio, ratio delta, z-score, confidence.
- Rolling ratio view: compact line or bar chart over returned distribution snapshots.
- Bucket views: side-by-side moneyness and expiration buckets, each showing call vs put open interest and volume.
- Chain table: option ticker, type, expiration, strike, moneyness, open interest, volume, IV, delta, updated time.
- Controls: trade-date selector, segmented call/put/all filter, row limit selector.

Use existing charting/table primitives if present. If no chart library exists, use accessible HTML/CSS bars rather than adding a dependency. Keep the view dense and analytical; avoid marketing-style cards.

### 6. Empty And Error States

- `404 options_coverage_not_found`: show `No persisted options coverage for {symbol}` and do not show an ingestion button.
- Empty distributions: show `No options distribution snapshots yet`.
- Empty chain rows for selected date/filter: show `No chain rows match this filter`.
- `401/403`: follow existing auth handling.
- `501 live_preview_not_configured`: should never be triggered by this UI task.

### 7. Analyst Interpretation

The UI should make these distinctions clear through labels and grouping:

- Coverage means persisted data availability, not market signal strength.
- Distribution means derived evidence from persisted chain rows.
- Algorithm feature readiness means the distribution can be materialized into `options_distribution_daily`; do not imply an algorithm result exists unless the existing algorithm-result APIs say so.
- Missing open interest should be visible because it directly weakens call/put ratio interpretation.

### 8. Tests

Add frontend tests consistent with the current stack:

- API client builds correct options URLs and query params.
- Query hooks are disabled without symbol.
- Asset detail renders coverage and latest distribution values.
- Empty coverage `404` renders the no-coverage state.
- Chain filter changes request `trade_date` and `contract_type`.
- Bucket rendering handles missing/partial bucket JSON.

## Acceptance Criteria

- Selecting NVDA on `/marketops/assets` shows persisted options coverage from the G127 live run.
- The latest distribution panel shows `10_trade_days`, latest trade date, contract counts, call/put ratios, and missing-open-interest count.
- Chain rows render for the selected trade date and can be filtered by call/put without page reload.
- No UI control triggers provider ingestion or `live-preview`.
- Tests pass and no existing MarketOps asset UI behavior regresses.
