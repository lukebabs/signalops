# MarketOps Asset Universe UI Specification

Status: ready for frontend-agent implementation  
Gate: G071 frontend follow-up  
Author: Codex  
Date: 2026-07-10  
Backend baseline: G071 `0d651bf Implement G071 MarketOps asset universe API`

## Purpose

Add a read-only MarketOps asset universe page to the existing `web/` frontend. Operators should be able to inspect the first-class MarketOps Top 50 mega-cap universe introduced in G071 without leaving the current app-profile shell.

This is not a trading screen, asset editor, or market-data terminal. It is an operational universe view backed by the new backend API and should use the same quiet, dense, table-oriented design as Sources, Pipelines, Rules, and Replay.

## Current Backend Contract

G071 added durable storage and a read API for the MarketOps asset universe.

Endpoint:

```http
GET /v1/tenants/{tenant_id}/marketops/assets?universe_group=top50_megacap&active_only=true&limit=50
Authorization: Bearer <access_token>
```

Authentication is enabled in the deployed/local stack. Unauthenticated requests return:

```json
{"error":"unauthorized","message":"missing bearer token"}
```

Local tenant for the current UI:

```text
tenant-local
```

Query parameters:

- `universe_group`: optional. Defaults backend-side to `top50_megacap`.
- `active_only`: optional. Defaults frontend-side to `true`; set `false` only if an operator control is added.
- `limit`: optional. Use `50` for this page.

Expected response envelope:

```json
{
  "assets": [
    {
      "tenant_id": "tenant-local",
      "app_id": "marketops",
      "domain": "market_data",
      "use_case": "daily_market_surveillance",
      "source_id": "src-massive",
      "universe_group": "top50_megacap",
      "rank": 1,
      "ticker": "NVDA",
      "ticker_key": "nvda",
      "company": "NVIDIA",
      "company_key": "nvidia",
      "asset_type": "equity",
      "exchange": "",
      "sector": "Technology",
      "sector_key": "technology",
      "industry": "Semiconductors",
      "industry_key": "semiconductors",
      "is_active": true,
      "metadata": {"seed":"top50megacap.normalized.csv","provider":"massive"},
      "created_at": "2026-07-10T00:00:00Z",
      "updated_at": "2026-07-10T00:00:00Z"
    }
  ]
}
```

Backend references:

- `docs/api.md`, "MarketOps Asset Universe API"
- `internal/api/router.go`, `GET /v1/tenants/{tenant_id}/marketops/assets`
- `internal/storage/storage.go`, `MarketOpsAssetRecord`
- `migrations/000011_marketops_asset_universe.up.sql`
- `internal/adapters/marketdata/massive/top50megacap.normalized.csv`

## Existing Frontend Context

Use the current `web/` app and conventions. Do not create a new frontend package or redesign the shell.

Relevant files and patterns:

- Types: `web/src/types.ts`
- API client: `web/src/api/client.ts`
- Query hooks: `web/src/api/queries.ts`
- Router: `web/src/router.tsx`
- App routing/nav: `web/src/apps/appRouting.ts`
- App context: `web/src/apps/AppProfileContext.tsx`
- Existing read-only pages: `SourcesRoute`, `PipelinesRoute`, `RulesRoute`
- Shared components: `MetricTile`, `StatusBadge`, `JsonViewer`, `LoadingState`, `ErrorState`, `EmptyState`
- Formatting helpers: `formatUtc`

The app already has MarketOps aliases under `/marketops/*`, but no dedicated asset universe route yet. Current MarketOps nav includes `providers` but does not include `symbols` even though the backend app profile advertises `symbols`.

## Required Outcome

Add a dedicated MarketOps asset universe view:

```text
/marketops/assets
```

Recommended nav label:

```text
Assets
```

Acceptable alternate label if the team wants to mirror the backend profile module name:

```text
Symbols
```

Use one label consistently in nav, route title, tests, and documentation. Prefer `Assets` because the backend resource is named `marketops/assets` and includes company/sector metadata beyond ticker symbols.

The page must be available only as a MarketOps route in this gate. Do not add a SignalOps Console route unless explicitly requested later.

## Required Implementation

### 1. Types

Update `web/src/types.ts`:

```ts
export interface MarketOpsAsset {
  tenant_id: string;
  app_id: string;
  domain: string;
  use_case: string;
  source_id: string;
  universe_group: string;
  rank: number;
  ticker: string;
  ticker_key: string;
  company: string;
  company_key: string;
  asset_type: string;
  exchange: string;
  sector: string;
  sector_key: string;
  industry: string;
  industry_key: string;
  is_active: boolean;
  metadata: unknown;
  created_at: string;
  updated_at: string;
}

export interface MarketOpsAssetsResponse {
  assets: MarketOpsAsset[];
}

export interface MarketOpsAssetFilter {
  tenant_id?: string;
  universe_group?: string;
  active_only?: boolean;
  limit?: number;
}
```

Keep backend strings permissive. Do not encode the 50 tickers as a TypeScript union.

### 2. API Client

Update `web/src/api/client.ts`:

- Import `MarketOpsAssetsResponse` and `MarketOpsAssetFilter`.
- Add:

```ts
listMarketOpsAssets: (filter: MarketOpsAssetFilter = {}) =>
  get<MarketOpsAssetsResponse>(
    `/v1/tenants/${encodeURIComponent(filter.tenant_id ?? 'tenant-local')}/marketops/assets`,
    {
      universe_group: filter.universe_group || 'top50_megacap',
      active_only: filter.active_only === false ? 'false' : 'true',
      limit: filter.limit ?? 50,
    },
  ),
```

The existing authenticated API client already attaches bearer tokens. Do not add a new auth mechanism.

### 3. Query Hook

Update `web/src/api/queries.ts`:

```ts
marketOpsAssets: (filter: MarketOpsAssetFilter) => ['marketops-assets', filter] as const,
```

Add:

```ts
export function useMarketOpsAssets(filter: MarketOpsAssetFilter = { tenant_id: 'tenant-local', universe_group: 'top50_megacap', active_only: true, limit: 50 }) {
  return useQuery({
    queryKey: queryKeys.marketOpsAssets(filter),
    queryFn: () => api.listMarketOpsAssets(filter),
    staleTime: 5 * 60 * 1000,
  });
}
```

The universe seed changes slowly, so a 5-minute stale time is acceptable.

### 4. Route And Navigation

Create:

```text
web/src/routes/MarketOpsAssetsRoute.tsx
```

Update `web/src/router.tsx`:

- Lazy-load `MarketOpsAssetsRoute`.
- Register `/marketops/assets`.

Update `web/src/apps/appRouting.ts`:

- Add `'/marketops/assets'` to `AppRoutePath`.
- Add `{ module: 'symbols', to: '/marketops/assets', label: 'Assets' }` to `MARKETOPS_NAV`.
- Keep existing MarketOps routes intact.

Do not remove `/marketops/providers`; providers and assets are separate concepts.

### 5. Page Behavior

`MarketOpsAssetsRoute` must:

- Use `useTenant()` for the tenant ID.
- Call `useMarketOpsAssets({ tenant_id: tenantId, universe_group: 'top50_megacap', active_only: true, limit: 50 })`.
- Render loading, error, and empty states using existing shared components.
- Render only backend data; no mock rows or hardcoded fallback tickers.
- Be read-only; no create/edit/delete/reorder controls.

Required page header:

- `h1`: `Assets`
- small tenant/universe text, for example `Tenant tenant-local · top50_megacap`

Required metrics:

- Universe Assets: `assets.length`
- Active Assets: count where `is_active === true`
- Sectors: distinct non-empty `sector_key` or `sector`
- Industries: distinct non-empty `industry_key` or `industry`

Optional useful metric:

- Source: display `src-massive` if all rows share it; otherwise count distinct sources.

Required table columns:

- Rank
- Asset: ticker, company, asset type
- Sector
- Industry
- Source
- Status
- Updated

Table guidance:

- Use a plain HTML table matching Sources/Pipelines/Rules. Do not use AG Grid.
- Keep rows dense and scannable.
- Use `StatusBadge` with `active`/`inactive` derived from `is_active`.
- Use monospace text for ticker and source IDs.
- Use `formatUtc` for `updated_at`.
- Do not introduce a new color-heavy visual system.

Recommended icon: `Landmark`, `BadgeDollarSign`, or `CircleDollarSign` from `lucide-react`. Use an existing lucide icon; do not draw custom SVGs.

### 6. JSON Sections

Below the table, render one JSON section using `JsonViewer`:

```ts
assets.map((asset) => ({
  ticker: asset.ticker,
  metadata: asset.metadata,
}))
```

Title:

```text
Asset Metadata
```

Do not display the full response JSON as the primary UI. The table is the primary experience.

### 7. Dashboard Integration

Enhance `DashboardRoute` only if it can be done without layout churn.

Recommended minimal addition under MarketOps only:

- Add a compact widget or metric link for Asset Universe count using `useMarketOpsAssets`.
- Link to `/marketops/assets`.

Do not add a global Console dashboard tile for MarketOps assets.

If adding this causes layout instability or broad dashboard changes, skip dashboard integration and keep this gate to the route/nav/API client.

### 8. Tests

Add focused tests consistent with existing frontend tests.

Minimum expected coverage:

- API client builds `/v1/tenants/tenant-local/marketops/assets` with `universe_group=top50_megacap`, `active_only=true`, and `limit=50`.
- API client sends `active_only=false` when explicitly requested.
- Query key/hook wiring is stable, if query hooks are tested in the current style.
- App routing includes `/marketops/assets` in MarketOps nav and keeps Console nav unchanged.
- Route renders returned assets, metrics, and empty state.

Use existing test patterns from:

- `web/src/api/appProfiles.test.ts`
- `web/src/apps/appRouting.test.ts`
- route tests already present in `web/src/routes` or API test files

Do not add Playwright unless the repo already has browser validation wiring available for this page in the same workflow.

## Validation Commands

Run from repo root unless noted:

```bash
cd web && npm test
cd web && npm run build
cd web && npm audit --json
```

If deploying the frontend after implementation:

```bash
make deploy-web
```

Then perform authenticated browser validation because the deployed gateway has auth enabled.

## Browser Validation Checklist

With an authenticated operator session:

1. Navigate to `/marketops/assets`.
2. Confirm the route renders under the MarketOps shell and nav highlights/contains `Assets`.
3. Confirm the page shows 50 seeded assets for `tenant-local/top50_megacap`.
4. Confirm top-ranked rows include `NVDA`, `AAPL`, `GOOGL`, `MSFT`, and `AMZN` in rank order.
5. Confirm metric counts are coherent: 50 universe assets, 50 active assets, non-zero sectors, non-zero industries.
6. Confirm network request includes `/v1/tenants/tenant-local/marketops/assets`, `universe_group=top50_megacap`, `active_only=true`, and `limit=50`.
7. Confirm unauthenticated access still redirects through the auth flow or receives the existing auth gate behavior.
8. Confirm mobile width has no horizontal page overflow; table overflow may be horizontally scrollable inside its container.

## Non-Goals

- No asset editing, import, delete, activation, or reorder controls.
- No market prices, charts, quote lookup, or live market-data polling.
- No options contracts UI; that belongs after G072.
- No graph proposal UI; that belongs after G074.
- No replacement of Providers/Sources route.
- No hardcoded row fallback in the UI.

## Acceptance Criteria

- `/marketops/assets` is registered and reachable in the SPA.
- MarketOps nav includes `Assets`; Console nav is unchanged.
- The page fetches real backend assets with the authenticated API client.
- The page renders metrics, a dense table, loading/error/empty states, and metadata JSON.
- Unit tests cover API path/query construction and app routing changes.
- `npm test`, `npm run build`, and `npm audit --json` pass.
