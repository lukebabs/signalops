# SignalOps Frontend Dashboard Implementation Specification

Status: ready for frontend-agent implementation  
Gate: G043  
Author: Codex  
Date: 2026-07-08  
Backend baseline: G042 generic raw-ingest persistence

## Purpose

Promote `/` into the first-class SignalOps operational Dashboard. Compose the current live health,
scheduler runs, raw events, provider usage, sources, pipelines, and rules into one consumable view.
The Dashboard is an operational summary and navigation surface; existing detail routes remain the
places for investigation.

Do not invent alerts, insights, correlations, knowledge evolution, time-series aggregates, or rule
execution results. Those target widgets remain pending backend APIs.

## Current Data Contracts

Use only these existing gateway endpoints through the same-origin web proxy:

- `GET /healthz`
- `GET /readyz`
- `GET /v1/scheduler/runs?limit=10`
- `GET /v1/provider-usage?limit=50`
- `GET /v1/raw-events?tenant_id=tenant-local&limit=10`
- `GET /v1/tenants/tenant-local/catalog/sources?limit=50`
- `GET /v1/tenants/tenant-local/catalog/pipelines?limit=50`
- `GET /v1/tenants/tenant-local/catalog/rules?limit=50`
- `GET /v1/streams/dashboard?channels=health,runs,raw_events,provider_usage,heartbeat`

G042 guarantees that generic raw events returning `202 Accepted` are query-visible in the raw ledger
and idempotency store. Pagination remains limit-only. Catalog APIs remain read-only. Rules are
catalog metadata, not execution state.

## Existing Frontend Architecture

Extend the current `web/` application. Reuse:

- API contracts: `web/src/types.ts`
- REST client: `web/src/api/client.ts`
- TanStack Query hooks: `web/src/api/queries.ts`
- SSE client: `web/src/api/stream.ts`
- Router: `web/src/router.tsx`
- Shell/navigation: `web/src/components/DashboardShell.tsx`
- Shared states and primitives under `web/src/components/`
- Detail routes: Runs, Raw Events, Sources, Pipelines, Rules, and System

Do not add a second state library, chart library, component framework, or mock-data layer.

## Required Implementation

### 1. Route And Navigation

Create `web/src/routes/DashboardRoute.tsx` and lazy-load it from the router.

- `/` renders `DashboardRoute`.
- `/runs` continues to render `RunsRoute`.
- Add Dashboard as the first navigation item using the Lucide `LayoutDashboard` icon.
- Runs remains a distinct navigation item.
- Preserve every existing route and deep link.

### 2. Data Ownership

The Dashboard route composes existing query hooks. The only hook change needed is an unfiltered
provider-usage hook: `useProviderUsage` is gated on a `run_id`, so add an always-enabled variant
(e.g. `useRecentProviderUsage(limit)`) that calls the existing `listProviderUsage(undefined, limit)`
client method — the client already supports omitting `run_id`.

- Fetch health and readiness independently.
- Fetch 10 recent runs and 10 recent raw events.
- Fetch up to 50 provider-usage rows and all three catalogs.
- Each widget receives only its relevant query result and owns its loading, error, empty, and refresh
  behavior. One failed endpoint must not blank the whole Dashboard.
- Use `tenant-local` consistently until tenant selection/authentication exists.
- Do not duplicate server data into a new global store.

### 3. Stream Integration

The SSE subscription is already mounted globally in `web/src/App.tsx` via `web/src/components/DashboardStreamBridge.tsx`, which calls the existing `subscribeDashboardStream` SSE client and applies exactly this invalidation map:

- `health` writes the health snapshot into the `/healthz` query cache (readiness is refetched by its own poll).
- `scheduler_run` invalidates the runs queries.
- `raw_event` invalidates the raw-events queries.
- `provider_usage` invalidates the provider-usage queries.
- `heartbeat` only updates stream freshness/connection state in the UI store.

The Dashboard consumes the resulting query cache and the stream-connection state from `useUi` (`streamConnected`, `lastStreamEventAt`, `streamError`) — do **not** mount a second `DashboardStreamBridge` or open another EventSource. (The `channels` request param accepts aliases like `runs`/`raw_events`, but the backend emits the canonical event names `scheduler_run`/`raw_event`; the existing client binds the canonical names.) REST remains the initial snapshot and reconnection fallback. Do not connect directly to Redpanda, add WebSockets, or claim replay/resume support — the SSE endpoint has no `Last-Event-ID` resume contract.

### 4. Layout

Build a dense operational layout, not a marketing page and not a grid of oversized decorative cards.
Use unframed page bands with compact bordered widgets where framing is necessary. Cards must not be
nested and should use the existing radius (8px maximum).

Desktop order:

1. Page header: `Dashboard`, UTC last-updated time, stream connection indicator, refresh icon button.
2. Metrics strip: Gateway, Readiness, Recent Runs, Failed Runs, Raw Events, Provider Requests, Active
   Sources, Active Pipelines, Active Rules.
3. Two-column operational band: Processing Health (larger) and Catalog Inventory (smaller).
4. Two-column activity band: Recent Runs (larger) and Provider Usage (smaller).
5. Full-width Recent Event Stream.

Responsive behavior:

- Collapse bands to one column below the existing application breakpoint.
- Metrics wrap into stable equal-width cells without horizontal page scrolling.
- Tables may use their own horizontal overflow container.
- No text, status, or controls may overlap at 375px mobile width.

### 5. Widget Contracts

**Processing Health**

Show gateway/readiness status, SSE connected/reconnecting state, latest heartbeat time, latest run
status/time, and failed-run count from the fetched sample. Link to `/system` and `/runs`. Do not infer
worker, Redpanda, or PostgreSQL health beyond exposed endpoints.

**Catalog Inventory**

Show total and active counts for Sources, Pipelines, and Rules. Include compact links to `/sources`,
`/pipelines`, and `/rules`. Rule severity is catalog configuration, not an active alert count.

**Recent Runs**

Show at most five newest runs with source adapter, datasets, status, started time, event count, and
failure count. Reuse existing status rendering. Link the widget heading or trailing action to `/runs`.
Do not recreate the full Runs AG Grid.

**Provider Usage**

Aggregate the fetched rows by provider for display only: request count, retry count, and event count.
Label the sample clearly by its existing UI context without implying lifetime totals. Use a compact
plain table or restrained bars using existing CSS; do not introduce a chart dependency.

**Recent Event Stream**

Show at most eight newest raw ledger records with observation time, source adapter, dataset, event ID,
and broker partition/offset. The current `RawEventsRoute` selects events via UI state, not a URL query
param, so link to `/raw-events` (plain) — do not invent a `?event_id=` deep-link contract here.

### 6. States And Accessibility

- Use existing loading, error, and empty-state components per widget.
- Refresh is an icon button with an accessible label and tooltip.
- Status is never color-only; include text.
- Tables use semantic headings and accessible labels.
- Keyboard focus remains visible.
- Reduced-motion users receive no decorative animation.

## Explicit Non-Goals

- Alerts widget or alert counts
- Timeline/time-bucket charts
- Correlation graph
- Active insights
- Knowledge evolution
- Rule execution history
- Pipeline execution controls
- Source/rule/pipeline mutation
- Authentication or tenant selector
- Cursor pagination
- New backend endpoints
- Mock records or fabricated zero-valued operational claims

These target Dashboard areas should be added only when their backend subsystem contracts exist. Do
not render disabled teaser cards for them in G043.

## Expected Files

At minimum:

- `web/src/routes/DashboardRoute.tsx` (new)
- `web/src/router.tsx`
- `web/src/components/DashboardShell.tsx`
- `web/src/api/queries.ts` and/or `web/src/api/client.ts` if unfiltered provider usage needs adjustment
- Dashboard-specific component/CSS files only where decomposition materially improves readability
- focused frontend tests
- `docs/build_journal.md`
- `docs/gate_audit.md`

## Validation Requirements

Run and record:

```bash
cd web && npm test
cd web && npm run build
cd web && npm audit --json
docker compose build web
docker compose up -d web
curl -fsS http://localhost:15173/
curl -fsS http://localhost:15173/healthz
curl -fsS 'http://localhost:15173/v1/scheduler/runs?limit=10'
curl -fsS 'http://localhost:15173/v1/raw-events?tenant_id=tenant-local&limit=10'
curl -fsS 'http://localhost:15173/v1/provider-usage?limit=50'
curl -fsS 'http://localhost:15173/v1/tenants/tenant-local/catalog/sources?limit=50'
curl -fsS 'http://localhost:15173/v1/tenants/tenant-local/catalog/pipelines?limit=50'
curl -fsS 'http://localhost:15173/v1/tenants/tenant-local/catalog/rules?limit=50'
```

Also perform Playwright validation at desktop and 375px mobile widths. Capture screenshots and verify:

- Dashboard is the `/` route and navigation state is correct.
- No overlap or horizontal page overflow.
- Every widget independently renders live data or a truthful state.
- SSE connection does not multiply after navigation/unmount/remount.
- Browser console has no errors.

If Playwright is unavailable, document that exact residual gap rather than claiming visual validation.

## G043 Acceptance Criteria

- `/` is a first-class Dashboard; Runs remains at `/runs`.
- All seven current backend data areas are represented: health, runs, raw events, provider usage,
  sources, pipelines, and rules.
- Data is real, tenant-scoped where applicable, and independently failure-isolated by widget.
- SSE events invalidate only relevant REST query state, with one subscription cleaned up on unmount.
- Unsupported target capabilities are neither fabricated nor presented as active.
- Desktop/mobile layout is consumable and accessible.
- Tests, production build, audit, Compose build, proxy checks, and browser validation pass or any
  unavailable browser validation is explicitly recorded.
- `docs/build_journal.md` and `docs/gate_audit.md` receive UTC timestamped G043 evidence.
