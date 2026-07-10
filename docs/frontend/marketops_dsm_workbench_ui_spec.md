# MarketOps DSM Workbench UI Specification

Status: ready for frontend-agent implementation  
Gate: G076 frontend follow-up  
Author: Codex  
Date: 2026-07-10  
Backend baseline: G075 `b803f62 MarketOps DSM taxonomy pack` plus option live coverage `c41bf25`

## Purpose

Add a MarketOps-only DSM workbench page that lets operators inspect the deterministic MarketOps taxonomy signals produced by `marketops.dsm.taxonomy_v1`. The page should make the G070-G075 output usable without asking operators to open raw JSON first: signal taxonomy, ticker, option/equity metrics, artifact proposal, graph candidates, and linked alert/insight lifecycle should be visible in one focused workflow.

This is an operational surveillance view, not a trading terminal, charting package, graph editor, or alert action redesign. Use existing backend APIs and existing frontend conventions.

## Current Backend Contract

No backend API changes are required for this gate. Use the existing authenticated `/v1/*` APIs.

Primary signal list:

```http
GET /v1/signals?tenant_id=tenant-local&app_id=marketops&domain=market_data&use_case=daily_market_surveillance&detector_id=marketops.dsm.taxonomy_v1&limit=50
Authorization: Bearer <access_token>
```

Signal detail:

```http
GET /v1/signals/{signal_id}
```

Lifecycle context:

```http
GET /v1/alerts?tenant_id=tenant-local&app_id=marketops&domain=market_data&use_case=daily_market_surveillance&status=open&limit=50
GET /v1/insights?tenant_id=tenant-local&app_id=marketops&domain=market_data&use_case=daily_market_surveillance&status=active&limit=50
```

The generic MarketOps aliases `/marketops/signals`, `/marketops/alerts`, and `/marketops/insights` already exist. G076 adds a dedicated summary/detail workbench at:

```text
/marketops/dsm
```

## Signal Shape To Use

The backend persists DSM output in `SignalRecord` fields already defined in `web/src/types.ts`:

- `signal_id`
- `signal_type`
- `severity`
- `confidence`
- `detector_id`
- `detector_version`
- `model_version`
- `source_id`
- `source_adapter`
- `dataset`
- `event_ids`
- `entities`
- `supporting_metrics`
- `semantic_evidence`
- `graph_targets`
- `evidence`
- `recommendation`
- `event`
- broker coordinates and timestamps

G075 DSM signals currently use:

```text
marketops.dsm.taxonomy_v1
```

Expected `signal_type` values:

- `marketops.dsm.volatility_expansion`
- `marketops.dsm.price_quality_exception`
- `marketops.dsm.accumulation`
- `marketops.dsm.divergence`
- `marketops.dsm.hedging_pressure`
- `marketops.dsm.speculative_call_pressure`
- `marketops.dsm.speculative_put_pressure`
- `marketops.dsm.pinning_risk`

Useful metric keys in `supporting_metrics`:

- Equity/price: `open_close_move_pct`, `intraday_range_pct`, `vwap_distance_pct`, `daily_return_pct`, `volume`
- Option-interest: `open_interest`, `volume_open_interest_ratio`, `days_to_expiration`, `moneyness_pct`, `contract_type`
- Quality/scoring: `quality_issue_count`, `detector_score`

Useful artifact fields:

- `artifact_ids` may be present on the full signal event/payload/recommendation depending on view path.
- `semantic_evidence[0].artifact` contains a DSM artifact proposal with `artifact_id`, `artifact_type=marketops.dsm.signal_artifact.v1`, `subject.symbol`, `features`, `quality_issues`, `confidence`, `severity`, and summary data.
- `graph_targets` contains node and relationship candidates. Do not mutate or accept graph proposals in this gate; display them as proposal evidence only.

Because these are JSON fields typed as `unknown`, the frontend must parse defensively. Missing or malformed nested values should render as `-` or an empty state, never throw.

## Required Outcome

Create a MarketOps DSM page at `/marketops/dsm` with:

- A dense list of recent DSM taxonomy signals scoped to MarketOps daily surveillance.
- Metrics summarizing signal count, open alert coverage, active insight coverage, high/critical count, and taxonomy type count.
- Filters for taxonomy type, severity, dataset, and limit.
- A selected-signal detail panel that extracts DSM-specific fields before showing raw JSON.
- Links to existing detail workflows where available: `/marketops/signals`, `/marketops/alerts`, `/marketops/insights`, and `/marketops/normalized`.
- A MarketOps nav entry labeled `DSM`.

Keep existing Console routes unchanged. Keep `/marketops/signals` available as the generic signal ledger view.

## Required Implementation

### 1. Route And Navigation

Add a route:

```text
web/src/routes/MarketOpsDsmRoute.tsx
```

Register it in `web/src/router.tsx`:

```text
/marketops/dsm
```

Update `web/src/apps/appRouting.ts`:

- Add `/marketops/dsm` to `AppRoutePath`.
- Add a MarketOps nav item near Signals:

```ts
{ module: 'dsm', to: '/marketops/dsm', label: 'DSM' }
```

Update `web/src/components/DashboardShell.tsx`:

- Add a lucide icon for module `dsm`. Use a real lucide icon such as `Network`, `GitBranch`, or `Radar`; prefer `Network` if available.

Do not add `DSM` to Console nav.

### 2. DSM Parsing Helpers

Add a small helper module, for example:

```text
web/src/lib/marketopsDsm.ts
```

Recommended exports:

```ts
export const MARKETOPS_DSM_DETECTOR_ID = 'marketops.dsm.taxonomy_v1';
export const MARKETOPS_DSM_USE_CASE = 'daily_market_surveillance';

export function dsmShortType(signalType: string): string;
export function dsmFamily(signalType: string): 'equity' | 'option' | 'quality' | 'unknown';
export function getTicker(signal: SignalRecord): string;
export function getMetric(signal: SignalRecord, key: string): string | number | null;
export function getArtifactProposal(signal: SignalRecord): DsmArtifactProposal | null;
export function getArtifactId(signal: SignalRecord): string | null;
export function countGraphTargets(signal: SignalRecord): number;
export function hasLifecycleMatch(signal: SignalRecord, ids: Set<string>): boolean;
```

Use type guards over `unknown`; avoid unsafe casts that can throw at runtime. Keep formatting functions deterministic and unit-testable.

Suggested family mapping:

- `quality`: `price_quality_exception`
- `option`: `hedging_pressure`, `speculative_call_pressure`, `speculative_put_pressure`, `pinning_risk`
- `equity`: `volatility_expansion`, `accumulation`, `divergence`
- `unknown`: anything else

### 3. Data Queries

Use existing hooks/API client unless a tiny wrapper is cleaner.

Default signal query:

```ts
useSignals({
  tenant_id: TENANT_ID,
  app_id: 'marketops',
  domain: 'market_data',
  use_case: 'daily_market_surveillance',
  detector_id: 'marketops.dsm.taxonomy_v1',
  limit,
  severity: selectedSeverity || undefined,
  dataset: selectedDataset || undefined,
})
```

Default lifecycle queries:

```ts
useAlerts({
  tenant_id: TENANT_ID,
  app_id: 'marketops',
  domain: 'market_data',
  use_case: 'daily_market_surveillance',
  status: 'open',
  limit: 100,
})

useInsights({
  tenant_id: TENANT_ID,
  app_id: 'marketops',
  domain: 'market_data',
  use_case: 'daily_market_surveillance',
  status: 'active',
  limit: 100,
})
```

When a signal is selected, use `useSignal(selectedId)` so the detail panel gets the complete persisted signal record.

Do not add polling unless the frontend-agent determines this route needs to mirror existing dashboard behavior. If polling is added, keep it modest and document it.

### 4. Page Layout

Use a compact operational layout, consistent with `SignalsRoute`, `AlertsRoute`, `InsightsRoute`, and `MarketOpsAssetsRoute`.

Top row:

- Title: `DSM Workbench`
- Small scope text: `marketops / market_data / daily_market_surveillance`
- Filters: taxonomy type, severity, dataset, limit
- Refresh button if consistent with existing route patterns

Metric row:

- DSM Signals: number returned by current query
- High/Critical: count by severity
- Open Alerts: count of queried open alerts linked to returned signal ids
- Active Insights: count of queried active insights linked to returned signal ids
- Taxonomy Types: distinct `signal_type` count

Main area:

- Left/main table: recent DSM signals
- Right/detail panel: selected signal DSM details

Table columns:

- Ticker
- Taxonomy
- Family
- Severity
- Confidence
- Dataset
- Key Metrics
- Artifact
- Graph Targets
- Alert/Insight
- Created

Key metrics should be concise:

- Equity rows: show available move/range/return/VWAP metrics.
- Option rows: show open interest, volume/OI, DTE, moneyness, contract type.
- Quality rows: show `quality_issue_count` and quality issue labels if available.

Do not let the table grow horizontally without an overflow container. Text must wrap or truncate cleanly on narrow screens.

### 5. Detail Panel

When no row is selected, show an empty state.

When a row is selected, show:

- Header: ticker, short taxonomy type, severity, confidence, created timestamp
- Identity: signal id with copy button, detector id/version, model version, source/dataset, broker coordinates
- Lifecycle links/status:
  - If matching open alert exists, show alert id and link to `/marketops/alerts`.
  - If matching active insight exists, show insight id and link to `/marketops/insights`.
- Artifact proposal summary:
  - artifact id
  - artifact type
  - subject symbol
  - quality issues
  - summary
- Metric sections:
  - Price metrics
  - Option-interest metrics
  - Quality/scoring metrics
- Graph proposal summary:
  - node candidate count
  - relationship candidate count
  - raw graph target JSON available below
- Evidence links:
  - event ids link to `/marketops/normalized`
  - signal id link to `/marketops/signals`

At the bottom, keep collapsible or existing `JsonViewer` sections for:

- `Supporting Metrics`
- `DSM Artifact Proposal`
- `Graph Targets`
- `Semantic Evidence`
- `Evidence`
- `Recommendation`
- `Full Signal Event`

### 6. Filters

Required controls:

- Taxonomy type select:
  - `all`
  - all eight known DSM signal types listed above
- Severity select:
  - `all`, `info`, `low`, `medium`, `high`, `critical`
- Dataset select:
  - `all`, `equity_eod_prices`, `options_contracts_daily`
- Limit select:
  - `25`, `50`, `100`, `200`

The backend does not currently expose `signal_type` as a query filter. Apply taxonomy type filtering client-side after fetching the backend list. Severity, dataset, detector, app, domain, use_case, and limit should be passed to the backend.

### 7. Links And Routing Behavior

Use MarketOps-prefixed links from this route:

- Signal ledger: `/marketops/signals`
- Normalized events: `/marketops/normalized`
- Alerts: `/marketops/alerts`
- Insights: `/marketops/insights`
- Assets: `/marketops/assets`

Do not link to Console routes from this MarketOps-specific page unless the app shell later adds cross-app link semantics.

### 8. Tests

Add focused tests. Do not rely only on visual inspection.

Required unit tests:

- `web/src/lib/marketopsDsm.test.ts`
  - extracts ticker from `entities` when present
  - falls back to artifact `subject.symbol` when entity is absent
  - extracts artifact id/proposal from `semantic_evidence`
  - handles malformed `unknown` JSON without throwing
  - classifies equity/option/quality families
  - counts node and relationship graph targets
- `web/src/apps/appRouting.test.ts`
  - MarketOps nav includes `DSM` at `/marketops/dsm`
  - Console nav does not include `/marketops/dsm`
  - `appIdFromPathname('/marketops/dsm')` returns `marketops`
- If route rendering tests are practical in the current Vitest setup, add one smoke test for empty/loading/error states. If not practical, document the gap like prior frontend gates did.

Existing API client tests do not need a new endpoint test if the route reuses `listSignals`, `listAlerts`, `listInsights`, and `getSignal`. Add query/filter assertions only if new wrapper hooks are introduced.

### 9. Validation Commands

Run from `web/`:

```bash
npm test
npm run build
npm audit --audit-level=low
```

Then from repo root:

```bash
git diff --check
```

If the local stack is running, validate route reachability:

```bash
curl -i http://localhost:15173/marketops/dsm
```

Authenticated browser validation checklist:

- Select MarketOps in the app selector.
- Confirm nav includes `DSM` and does not remove `Assets`, `Signals`, `Alerts`, or `Insights`.
- Open `/marketops/dsm`.
- Confirm the network request for signals includes:
  - `tenant_id=tenant-local`
  - `app_id=marketops`
  - `domain=market_data`
  - `use_case=daily_market_surveillance`
  - `detector_id=marketops.dsm.taxonomy_v1`
- Confirm recent live G075 taxonomy signals appear if the local ledger contains them:
  - `marketops.dsm.hedging_pressure`
  - `marketops.dsm.speculative_call_pressure`
  - `marketops.dsm.speculative_put_pressure`
  - `marketops.dsm.pinning_risk`
  - `marketops.dsm.accumulation`
- Select a row and confirm artifact id, graph target counts, supporting metrics, and lifecycle links render.
- Verify the taxonomy type filter works client-side.
- Check desktop and mobile widths for no incoherent overlap or horizontal page overflow outside intentional table scroll.

## Non-Goals

- Do not create new backend DSM endpoints.
- Do not create artifact tables, graph acceptance APIs, or graph editing UI.
- Do not change signal, alert, or insight lifecycle semantics.
- Do not remove generic `/marketops/signals`, `/marketops/alerts`, or `/marketops/insights`.
- Do not make this page a trading execution interface.

## Documentation Updates Required By Frontend-Agent

After implementation, update:

- `docs/build_journal.md` with files changed, validation commands, route/browser evidence, and any deferred checks.
- `docs/gate_audit.md` with a new G076 implementation section.

Commit and push the implementation after validation.
