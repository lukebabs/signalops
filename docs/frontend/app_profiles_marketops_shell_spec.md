# G067 Frontend App Profiles and MarketOps Shell Specification

Timestamp: `2026-07-10T07:08:00Z`

## Purpose

Implement the frontend side of G066. The backend now exposes first-class
`app_id`, `domain`, and `use_case` metadata across raw events, normalized
events, signals, alerts, and insights, and provides static app profiles through
`GET /v1/app-profiles`.

The frontend must consume this backend surface and introduce a scalable
multi-app representation without replacing the existing SignalOps Console.

## Backend Status

Backend G066 is closed, deployed, and pushed.

Relevant backend commit:

- `82f390c Implement G066 app use-case metadata`

Relevant backend endpoint:

```http
GET /v1/app-profiles
Authorization: Bearer <access_token>
```

Unauthenticated requests return `401` when auth is enabled. Authenticated
requests return:

```json
{
  "app_profiles": [
    {
      "app_id": "console",
      "label": "SignalOps Console",
      "default_route": "/dashboard",
      "domains": ["market_data", "crm", "security", "operations", "iot", "procurement", "custom"],
      "enabled_modules": ["dashboard", "event_explorer", "timeline", "correlation", "insights", "pipelines", "rules", "sources", "health", "replay", "administration", "settings"],
      "dashboard_profile": "console.default"
    },
    {
      "app_id": "marketops",
      "label": "MarketOps",
      "default_route": "/marketops/dashboard",
      "domains": ["market_data"],
      "enabled_modules": ["dashboard", "symbols", "option_contracts", "signals", "alerts", "replay", "providers", "pipelines", "health"],
      "dashboard_profile": "marketdata.default"
    }
  ]
}
```

The exact order should not be assumed except that both `console` and
`marketops` must be supported when present.

## Current Frontend Context

The current frontend has:

- `web/src/router.tsx` with one route tree under `DashboardShell`.
- `web/src/components/DashboardShell.tsx` with hardcoded SignalOps nav.
- API client and hooks in `web/src/api/client.ts` and `web/src/api/queries.ts`.
- Route components for Dashboard, raw events, normalized events, signals,
  alerts, insights, sources, pipelines, rules, replay, and system health.
- Existing filters for source/dataset/detector/status/severity, but not yet
  `app_id`, `domain`, or `use_case`.

Do not rewrite the application. Extend the current shell and route system with
configuration-driven app profile support.

## Required Outcome

The frontend must support two app experiences over the same backend:

```text
SignalOps Console
  Dashboard
  Runs
  Raw Events
  Normalized
  Idempotency
  Sources
  Pipelines
  Rules
  Replay
  Signals
  Alerts
  Insights
  System
```

```text
MarketOps
  Dashboard
  Providers
  Raw Events
  Normalized
  Signals
  Alerts
  Insights
  Replay
  Pipelines
  Health
```

The current SignalOps Console must remain the default app. MarketOps must be an
additional profile, not a replacement.

## API Client Work

Add frontend types in `web/src/types.ts`:

```ts
export interface AppProfile {
  app_id: string;
  label: string;
  default_route: string;
  domains: string[];
  enabled_modules: string[];
  dashboard_profile: string;
}

export interface AppProfilesResponse {
  app_profiles: AppProfile[];
}
```

Add API method in `web/src/api/client.ts`:

```ts
getAppProfiles: () => get<AppProfilesResponse>('/v1/app-profiles')
```

Add query key and hook in `web/src/api/queries.ts`:

```ts
appProfiles: ['app-profiles'] as const

export function useAppProfiles() {
  return useQuery({
    queryKey: queryKeys.appProfiles,
    queryFn: api.getAppProfiles,
    staleTime: 5 * 60 * 1000,
  });
}
```

The hook must use the existing authenticated API client. Do not create a new
auth mechanism.

## Metadata Filter Work

Extend frontend filter types to include:

```ts
app_id?: string;
domain?: string;
use_case?: string;
```

Apply to at least:

- `RawEventFilter`
- `NormalizedEventFilter`
- `SignalFilter`
- `AlertFilter`
- `InsightFilter`

Update API client methods to send these query parameters:

- `listRawEvents`
- `listNormalizedEvents`
- `listSignals`
- `listAlerts`
- `listInsights`

Do not add these filters to replay jobs yet; G066 did not add replay-job
metadata filtering.

## App Context

Create a small app profile/context layer.

Recommended files:

```text
web/src/apps/appProfiles.ts
web/src/apps/AppProfileContext.tsx
web/src/apps/appRouting.ts
```

Responsibilities:

- Load profiles using `useAppProfiles`.
- Select current app from the route prefix:
  - `/marketops/*` means `marketops`.
  - all existing routes mean `console`.
- Provide `currentApp`, `currentAppId`, `metadataFilter`, and helper labels to
  child components.
- Fall back to a local static console profile if the app profiles request fails,
  so the existing UI remains usable.

Suggested metadata filter behavior:

```ts
console:
  {}

marketops:
  {
    app_id: 'marketops',
    domain: 'market_data'
  }
```

Do not force `use_case` globally for MarketOps yet. The backend uses
`daily_market_surveillance` for Massive scheduled events, but future MarketOps
use cases may share the same app/domain.

## Routing Work

Keep existing routes intact.

Add MarketOps aliases that reuse existing route components with app context:

```text
/marketops/dashboard      -> DashboardRoute
/marketops/providers      -> SourcesRoute or a thin MarketOpsProvidersRoute wrapper
/marketops/raw-events     -> RawEventsRoute
/marketops/normalized     -> NormalizedEventsRoute
/marketops/signals        -> SignalsRoute
/marketops/alerts         -> AlertsRoute
/marketops/insights       -> InsightsRoute
/marketops/replay         -> ReplayJobsRoute
/marketops/pipelines      -> PipelinesRoute
/marketops/health         -> SystemRoute
```

Do not remove existing routes:

```text
/
/runs
/raw-events
/normalized-events
/idempotency
/sources
/pipelines
/rules
/replay
/signals
/alerts
/insights
/system
```

Route aliases should not duplicate business logic. Reuse existing components
and let app context/filtering change the data and labels.

## Shell and Navigation

Update `DashboardShell` so it is app-aware.

Required behavior:

- Header should still show the authenticated user and health indicator.
- Header should show the active app label:
  - `SignalOps Console`
  - `MarketOps`
- Add an app selector using the profiles from `GET /v1/app-profiles`.
- App selector should navigate to the selected profile's `default_route`.
- Navigation items should come from the active app profile/module map.
- Console keeps the existing nav labels where possible.
- MarketOps uses market-facing labels:
  - `Providers` instead of `Sources`
  - `Health` instead of `System`
  - `Dashboard`, `Raw Events`, `Normalized`, `Signals`, `Alerts`, `Insights`,
    `Replay`, `Pipelines`

Use existing visual language. This is not a redesign.

## Route Data Behavior

Existing route components should become app-aware by reading the app context and
passing `metadataFilter` into queries.

Minimum required app-aware routes:

- Dashboard
- Raw Events
- Normalized Events
- Signals
- Alerts
- Insights
- Sources/Providers
- Pipelines
- Replay Jobs
- System/Health

For routes whose backend APIs do not yet support app/domain/use-case filters
(sources, pipelines, replay status/jobs, provider usage), do not fake isolation.
Instead:

- Apply filters only to endpoints that support them.
- Use app-aware labels and concise hints where useful.
- Keep shared operational views available.

For routes whose backend APIs do support G066 filters, pass:

```ts
...metadataFilter
```

into query filters.

## Dashboard Behavior

For `console`:

- Preserve the current Dashboard layout and data behavior.

For `marketops`:

- Reuse the current Dashboard route initially.
- Apply `app_id=marketops` and `domain=market_data` to raw events,
  normalized events, signals, alerts, and insights queries where used.
- Adjust visible labels only where low-risk:
  - `Sources` -> `Providers`
  - `Events` may be presented as `Market Events` in local headings if already
    present.
- Do not create a separate large MarketOps dashboard redesign in this gate.

## Tests Required

Add or update tests for:

- API client `getAppProfiles` attaches auth and parses response.
- Query hook/key for app profiles.
- App profile route detection:
  - `/` -> `console`
  - `/dashboard` or existing routes -> `console`
  - `/marketops/dashboard` -> `marketops`
- Metadata filter helper:
  - console returns `{}`
  - marketops returns `{ app_id: 'marketops', domain: 'market_data' }`
- API methods include `app_id`, `domain`, and `use_case` when present for raw,
  normalized, signal, alert, and insight list calls.
- Shell renders available app profile labels and can select MarketOps.

Prefer focused unit tests around pure helpers and API methods. Add route/shell
component tests only if the existing test setup already supports them cleanly.

## Browser Validation Required

After implementation, validate with auth enabled:

1. Load `https://signalops.syncratic.io/`.
2. Confirm SignalOps Console remains the default experience.
3. Confirm the app selector shows `SignalOps Console` and `MarketOps`.
4. Select `MarketOps`; confirm navigation to `/marketops/dashboard`.
5. Confirm MarketOps nav labels render without overflow on desktop and mobile.
6. Confirm MarketOps raw/normalized/signals/alerts/insights routes load without
   errors.
7. Confirm browser network requests for supported data routes include
   `app_id=marketops` and `domain=market_data`.
8. Confirm console routes still load without app/domain filters unless the user
   explicitly sets them.
9. Confirm unauthenticated access still redirects through the existing auth
   flow.

Also run:

```bash
cd web && npm test
cd web && npm run build
cd web && npm audit --json
```

Run Compose validation from repo root:

```bash
docker compose -f compose.yaml -f compose.traefik.yaml config --quiet
```

## Acceptance Criteria

- `GET /v1/app-profiles` is consumed by the frontend through the existing auth
  API client.
- Existing SignalOps Console routes and nav remain available.
- MarketOps routes are available under `/marketops/*`.
- App selector can switch between Console and MarketOps.
- MarketOps-supported data routes pass `app_id=marketops` and
  `domain=market_data` to backend list APIs.
- No backend changes are required for this frontend gate.
- Tests/build/audit pass and are journaled.
- Browser validation confirms no broken routing, auth regression, or layout
  overflow.

## Non-Goals

- Do not implement a database-backed app registry.
- Do not add new backend endpoints.
- Do not build new market algorithms.
- Do not create a completely separate frontend app bundle.
- Do not remove or rename existing console routes.
- Do not add tenant-custom app creation UI.
- Do not add role-based app visibility in this gate; consume all profiles the
  backend returns.
