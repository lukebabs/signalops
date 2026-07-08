# SignalOps Frontend Alerts and Active Insights UI Implementation Specification

Status: ready for frontend-agent implementation  
Gate: G048  
Author: Codex  
Date: 2026-07-08  
Backend baseline: G047 alert and insight lifecycle foundation

## Purpose

Expose the durable alert and insight lifecycle rows added in G047 in the existing SignalOps `web/`
frontend. Operators must be able to see open alerts, active insights, inspect their lineage back to
signals and normalized events, and understand which detector/source/dataset produced them.

This gate is read-only. G047 does not expose mutation endpoints for acknowledgement, resolution,
review, dismissal, or suppression, so the frontend must not render working lifecycle action controls.
Do not fabricate alert counts, insight summaries, correlation results, timeline aggregates, or rule
execution state beyond what the backend returns.

## Backend Contracts

Use same-origin requests through the existing web proxy. The gateway endpoints are:

- `GET /v1/alerts?tenant_id={tenant_id}&source_id={source_id}&dataset={dataset}&severity={severity}&status={status}&limit=50`
- `GET /v1/alerts/{alert_id}`
- `GET /v1/insights?tenant_id={tenant_id}&source_id={source_id}&dataset={dataset}&insight_type={insight_type}&status={status}&limit=50`
- `GET /v1/insights/{insight_id}`

All filters are optional. The current UI should use `tenant-local` by default until tenant selection
and authentication exist. Pagination remains limit-only. A non-numeric, empty, or non-positive
`limit` is clamped to the default (`50`) and values above `200` are capped at `200`; the gateway does
not return `400 invalid_limit`.

### G047 Derivation Semantics

The frontend must present these semantics accurately:

- Every valid signal creates or updates one active insight with id `insight:{signal_id}`.
- `medium`, `high`, and `critical` signals create or update one open alert with id
  `alert:{signal_id}`.
- `info` and `low` signals do not create alerts.
- Reprocessing the same signal is idempotent and does not reset existing lifecycle status fields.
- Lifecycle mutation endpoints are deferred until operator identity/authentication exists.

Do not call a low-severity insight a missing alert; that is expected behavior.

### Alert List Envelope

The alert list endpoint returns:

```json
{
  "alerts": [
    {
      "alert_id": "alert:signal-g047-high",
      "tenant_id": "tenant-local",
      "source_id": "src-g047",
      "source_domain": "iot",
      "source_adapter": "iot.generic.sensor",
      "dataset": "sensor_observations",
      "signal_id": "signal-g047-high",
      "detector_id": "signalops.g047.validation",
      "alert_type": "temperature.anomaly",
      "severity": "high",
      "status": "open",
      "title": "High temperature.anomaly alert",
      "summary": "Detector signalops.g047.validation emitted a high signal for sensor_observations.",
      "confidence": 0.91,
      "event_ids": ["g044-live-event"],
      "entities": [{"id": "sensor:g047", "type": "sensor"}],
      "evidence": [{"ref": "g044-live-event", "type": "normalized_event"}],
      "recommendation": {"action": "inspect_sensor"},
      "correlation_id": "corr-g047",
      "first_observed_at": "2026-07-08T22:40:00Z",
      "last_observed_at": "2026-07-08T22:40:00Z",
      "metadata": {
        "derived_from": "signal_ledger",
        "detector_id": "signalops.g047.validation",
        "signal_type": "temperature.anomaly"
      },
      "created_at": "2026-07-08T22:53:15.357758Z",
      "updated_at": "2026-07-08T22:53:15.357758Z"
    }
  ]
}
```

Optional alert fields may be absent until lifecycle mutation APIs exist:

- `acknowledged_at`
- `acknowledged_by`
- `resolved_at`
- `resolved_by`

Current alert status values: `open`, `acknowledged`, `resolved`, `suppressed`. In G048 most local
records will be `open`; the type should still accept all values.

### Alert Detail Envelope

The detail endpoint returns:

```json
{
  "alert": {
    "alert_id": "alert:signal-g047-high"
  }
}
```

The detail object has the same fields as list rows. `entities`, `evidence`, `recommendation`, and
`metadata` are JSON payloads and must be rendered with `JsonViewer`.

### Insight List Envelope

The insight list endpoint returns:

```json
{
  "insights": [
    {
      "insight_id": "insight:signal-g047-high",
      "tenant_id": "tenant-local",
      "source_id": "src-g047",
      "source_domain": "iot",
      "source_adapter": "iot.generic.sensor",
      "dataset": "sensor_observations",
      "signal_id": "signal-g047-high",
      "detector_id": "signalops.g047.validation",
      "insight_type": "temperature.anomaly",
      "status": "active",
      "title": "high signal from signalops.g047.validation",
      "summary": "Detector signalops.g047.validation emitted a high temperature.anomaly signal for sensor_observations.",
      "confidence": 0.91,
      "severity": "high",
      "event_ids": ["g044-live-event"],
      "entities": [{"id": "sensor:g047", "type": "sensor"}],
      "supporting_metrics": {"temperature_c": 47.1},
      "semantic_evidence": [{"summary": "validation signal for G047 lifecycle"}],
      "recommendation": {"action": "inspect_sensor"},
      "correlation_id": "corr-g047",
      "observed_at": "2026-07-08T22:40:00Z",
      "metadata": {
        "derived_from": "signal_ledger",
        "detector_id": "signalops.g047.validation",
        "signal_type": "temperature.anomaly"
      },
      "created_at": "2026-07-08T22:53:15.357758Z",
      "updated_at": "2026-07-08T22:53:15.357758Z"
    }
  ]
}
```

Optional insight fields may be absent until lifecycle mutation APIs exist:

- `reviewed_at`
- `reviewed_by`

Current insight status values: `active`, `reviewed`, `dismissed`, `archived`. In G048 most local
records will be `active`; the type should still accept all values.

### Insight Detail Envelope

The detail endpoint returns:

```json
{
  "insight": {
    "insight_id": "insight:signal-g047-high"
  }
}
```

The detail object has the same fields as list rows. `entities`, `supporting_metrics`,
`semantic_evidence`, `recommendation`, and `metadata` are JSON payloads and must be rendered with
`JsonViewer`.

### Error Handling

Expected gateway errors:

- `404 alert_not_found`
- `404 insight_not_found`
- `500 query_failed`
- `503 storage_unavailable`

Render endpoint-specific errors using the existing error state pattern. One failing widget or detail
query must not blank the whole page.

## Existing Frontend Architecture

Extend the current `web/` app. Reuse:

- API contracts: `web/src/types.ts`
- REST client: `web/src/api/client.ts`
- TanStack Query hooks: `web/src/api/queries.ts`
- Router: `web/src/router.tsx`
- Shell/navigation: `web/src/components/DashboardShell.tsx`
- Shared states and primitives: `States`, `StatusBadge`, `MetricTile`, `JsonViewer`,
  `RefreshButton`, `CopyButton`
- Existing route patterns: `SignalsRoute`, `NormalizedEventsRoute`, `RawEventsRoute`,
  `RulesRoute`, `SourcesRoute`, `PipelinesRoute`

Do not add a new frontend package, new state library, new chart library, new component framework, or
mock data layer.

## Required Implementation

### 1. Types

Add alert and insight contracts to `web/src/types.ts`.

Recommended names:

```ts
export type AlertStatus = 'open' | 'acknowledged' | 'resolved' | 'suppressed';
export type InsightStatus = 'active' | 'reviewed' | 'dismissed' | 'archived';

export interface AlertFilter {
  tenant_id?: string;
  source_id?: string;
  dataset?: string;
  severity?: string;
  status?: string;
  limit?: number;
}

export interface AlertRecord {
  alert_id: string;
  tenant_id: string;
  source_id: string;
  source_domain: string;
  source_adapter: string;
  dataset: string;
  signal_id: string;
  detector_id: string;
  alert_type: string;
  severity: string;
  status: string;
  title: string;
  summary: string;
  confidence: number;
  event_ids: string[];
  entities: unknown;
  evidence: unknown;
  recommendation: unknown;
  correlation_id: string;
  first_observed_at: string;
  last_observed_at: string;
  acknowledged_at?: string;
  acknowledged_by?: string;
  resolved_at?: string;
  resolved_by?: string;
  metadata: unknown;
  created_at: string;
  updated_at: string;
}

export interface AlertsResponse {
  alerts: AlertRecord[];
}

export interface AlertResponse {
  alert: AlertRecord;
}

export interface InsightFilter {
  tenant_id?: string;
  source_id?: string;
  dataset?: string;
  insight_type?: string;
  status?: string;
  limit?: number;
}

export interface InsightRecord {
  insight_id: string;
  tenant_id: string;
  source_id: string;
  source_domain: string;
  source_adapter: string;
  dataset: string;
  signal_id: string;
  detector_id: string;
  insight_type: string;
  status: string;
  title: string;
  summary: string;
  confidence: number;
  severity: string;
  event_ids: string[];
  entities: unknown;
  supporting_metrics: unknown;
  semantic_evidence: unknown;
  recommendation: unknown;
  correlation_id: string;
  observed_at: string;
  reviewed_at?: string;
  reviewed_by?: string;
  metadata: unknown;
  created_at: string;
  updated_at: string;
}

export interface InsightsResponse {
  insights: InsightRecord[];
}

export interface InsightResponse {
  insight: InsightRecord;
}
```

Keep optional lifecycle fields optional. Keep status/severity as `string` inside the records for
forward compatibility, even if helper union types are exported.

### 2. API Client

Update `web/src/api/client.ts`:

- Import `AlertFilter`, `AlertsResponse`, `AlertResponse`, `InsightFilter`, `InsightsResponse`, and
  `InsightResponse`.
- Add:

```ts
listAlerts: (filter: AlertFilter = {}) =>
  get<AlertsResponse>('/v1/alerts', {
    tenant_id: filter.tenant_id,
    source_id: filter.source_id,
    dataset: filter.dataset,
    severity: filter.severity,
    status: filter.status,
    limit: filter.limit ?? 50,
  }),
getAlert: (alertId: string) =>
  get<AlertResponse>(`/v1/alerts/${encodeURIComponent(alertId)}`),
listInsights: (filter: InsightFilter = {}) =>
  get<InsightsResponse>('/v1/insights', {
    tenant_id: filter.tenant_id,
    source_id: filter.source_id,
    dataset: filter.dataset,
    insight_type: filter.insight_type,
    status: filter.status,
    limit: filter.limit ?? 50,
  }),
getInsight: (insightId: string) =>
  get<InsightResponse>(`/v1/insights/${encodeURIComponent(insightId)}`),
```

### 3. Query Hooks

Update `web/src/api/queries.ts`:

```ts
alerts: (filter: AlertFilter) => ['alerts', filter] as const,
alert: (alertId: string) => ['alert', alertId] as const,
insights: (filter: InsightFilter) => ['insights', filter] as const,
insight: (insightId: string) => ['insight', insightId] as const,
```

Add hooks:

```ts
export function useAlerts(filter: AlertFilter) {
  return useQuery({
    queryKey: queryKeys.alerts(filter),
    queryFn: () => api.listAlerts(filter),
  });
}

export function useAlert(alertId: string | null) {
  return useQuery({
    queryKey: queryKeys.alert(alertId ?? ''),
    queryFn: () => api.getAlert(alertId!),
    enabled: !!alertId,
  });
}

export function useInsights(filter: InsightFilter) {
  return useQuery({
    queryKey: queryKeys.insights(filter),
    queryFn: () => api.listInsights(filter),
  });
}

export function useInsight(insightId: string | null) {
  return useQuery({
    queryKey: queryKeys.insight(insightId ?? ''),
    queryFn: () => api.getInsight(insightId!),
    enabled: !!insightId,
  });
}
```

### 4. Routes And Navigation

Create two read-only routes:

- `web/src/routes/AlertsRoute.tsx`
- `web/src/routes/InsightsRoute.tsx`

Update `web/src/router.tsx`:

- Lazy-load `AlertsRoute`.
- Lazy-load `InsightsRoute`.
- Add `/alerts`.
- Add `/insights`.

Update `web/src/components/DashboardShell.tsx`:

- Add `Alerts` near `Signals`.
- Add `Insights` near `Alerts` or immediately after `Signals`.
- Use lucide icons. Recommended: `BellRing` or `TriangleAlert` for Alerts and `Lightbulb` for
  Insights.
- Preserve the existing wrapping navigation behavior from the G046 validation fix. Do not reintroduce
  horizontal nav overflow.

Preserve every existing route and deep link.

### 5. Alerts Page

The Alerts route must:

- Use `tenant-local` by default.
- Fetch `GET /v1/alerts?tenant_id=tenant-local&status=open&limit=50` by default.
- Provide local filters for `source_id`, `dataset`, `severity`, `status`, and `limit`.
- Render loading, error, and empty states using existing shared components.
- Render only returned backend data.
- Allow selecting a row to load detail via `GET /v1/alerts/{alert_id}`.

Required page metrics:

- Alerts: `alerts.length`
- Open: count where `status === 'open'`
- High/Critical: count where severity is `high` or `critical`
- Distinct Sources
- Average Confidence over the fetched sample

Required table columns:

- Alert: title, alert id, summary, copy button
- Severity
- Status
- Type: `alert_type`
- Detector: `detector_id`
- Source/Dataset
- Confidence
- Event Count
- First Observed
- Last Observed
- Updated

Required detail panel sections:

- Identity: alert id, signal id, detector id, alert type
- Lifecycle: status, severity, first/last observed, optional acknowledged/resolved fields
- Source: tenant, source domain, source id, source adapter, dataset
- Confidence and correlation id
- Event IDs, with plain text links where useful:
  - `signal_id` links to `/signals`
  - each `event_id` links to `/normalized-events`
- Entities JSON
- Evidence JSON
- Recommendation JSON
- Metadata JSON

Do not render enabled acknowledgement/resolution buttons. A small non-actionable status display is
acceptable; avoid disabled teaser controls unless they materially clarify unavailable functionality.

### 6. Insights Page

The Insights route must:

- Use `tenant-local` by default.
- Fetch `GET /v1/insights?tenant_id=tenant-local&status=active&limit=50` by default.
- Provide local filters for `source_id`, `dataset`, `insight_type`, `status`, and `limit`.
- Render loading, error, and empty states using existing shared components.
- Render only returned backend data.
- Allow selecting a row to load detail via `GET /v1/insights/{insight_id}`.

Required page metrics:

- Insights: `insights.length`
- Active: count where `status === 'active'`
- Insight Types: distinct `insight_type`
- High/Critical: count where severity is `high` or `critical`
- Average Confidence over the fetched sample

Required table columns:

- Insight: title, insight id, summary, copy button
- Status
- Severity
- Type: `insight_type`
- Detector: `detector_id`
- Source/Dataset
- Confidence
- Event Count
- Observed
- Updated

Required detail panel sections:

- Identity: insight id, signal id, detector id, insight type
- Lifecycle: status, observed at, optional reviewed fields
- Source: tenant, source domain, source id, source adapter, dataset
- Severity, confidence, correlation id
- Event IDs, with plain text links where useful:
  - `signal_id` links to `/signals`
  - each `event_id` links to `/normalized-events`
- Entities JSON
- Supporting metrics JSON
- Semantic evidence JSON
- Recommendation JSON
- Metadata JSON

Do not render enabled review/dismiss/archive buttons. Those require backend mutation endpoints.

### 7. Dashboard Integration

Enhance `web/src/routes/DashboardRoute.tsx` without changing its operational purpose:

- Replace any placeholder or derived-from-signals alert/insight content with real G047 API data.
- Add/confirm metric tiles for:
  - Open Alerts from `GET /v1/alerts?tenant_id=tenant-local&status=open&limit=50`
  - Active Insights from `GET /v1/insights?tenant_id=tenant-local&status=active&limit=50`
- Add or update compact widgets for:
  - Alerts: at most five open alerts, sorted by backend response order.
  - Active Insights: at most five active insights, sorted by backend response order.
- Link headings or trailing actions to `/alerts` and `/insights`.
- Each widget owns loading, error, empty, and refresh behavior.
- Do not remove existing health, runs, raw events, normalized events, signals, provider usage,
  sources, pipelines, or rules sections.

Dashboard labels must distinguish signals, alerts, and insights:

- Signal: detector output.
- Alert: operator-facing lifecycle item derived from medium/high/critical signals.
- Insight: operator-facing analysis item derived from every signal.

### 8. Stream Integration

Do not open a second EventSource and do not connect to Redpanda from the browser.

Current SSE channels still do not include alerts or insights. For G048:

- REST queries are the source of truth.
- Dashboard widgets and route pages may refresh through local refresh controls and ordinary query
  refetching.
- Do not claim live alert/insight streaming.
- Do not add WebSockets.
- Do not implement `Last-Event-ID` resume; the backend stream does not support it yet.

If the frontend-agent adds modest polling for alerts/insights, document it in the journal. Manual
refresh is preferred unless matching an existing local pattern.

## UX Requirements

- Dense operational UI, not a marketing layout.
- Keep cards restrained and avoid nested cards.
- Tables may use horizontal overflow containers; the page itself must not horizontally overflow at
  375px mobile width.
- Text and controls must not overlap at desktop or mobile widths.
- UTC timestamps should use existing date formatting helpers.
- JSON payloads should be contained enough to avoid destroying page layout.
- Copy buttons need accessible labels.
- Keyboard focus must remain visible.
- Status/severity/confidence must not depend on color alone; include visible text.
- Alerts should be visually noticeable but not alarmist; avoid heavy red-only pages.
- Do not render mock records or placeholder operational claims.

## Explicit Non-Goals

- Acknowledge, resolve, suppress, review, dismiss, or archive actions
- Alert or insight creation/editing/deletion
- Rule execution results
- Pipeline execution state
- Correlation graph
- Timeline aggregation
- Knowledge evolution
- Tenant selector/authentication
- Cursor pagination
- New backend endpoints
- Alert/insight SSE or WebSocket streaming
- Browser-to-broker communication

## Expected Files

At minimum:

- `web/src/types.ts`
- `web/src/api/client.ts`
- `web/src/api/queries.ts`
- `web/src/router.tsx`
- `web/src/components/DashboardShell.tsx`
- `web/src/routes/AlertsRoute.tsx`
- `web/src/routes/InsightsRoute.tsx`
- `web/src/routes/DashboardRoute.tsx`
- focused frontend tests
- `docs/build_journal.md`
- `docs/gate_audit.md`

Shared components may be added only where they reduce meaningful duplication between Alerts,
Insights, Signals, and Normalized Events.

## Validation Requirements

Run and record:

```bash
cd web && npm test
cd web && npm run build
cd web && npm audit --json
docker compose build web
docker compose up -d web
curl -fsS http://localhost:15173/
curl -fsS http://localhost:15173/alerts
curl -fsS http://localhost:15173/insights
curl -fsS 'http://localhost:15173/v1/alerts?tenant_id=tenant-local&status=open&limit=10'
curl -fsS 'http://localhost:15173/v1/insights?tenant_id=tenant-local&status=active&limit=10'
```

If live alert and insight rows exist, also verify one detail endpoint each through the web proxy:

```bash
curl -fsS 'http://localhost:15173/v1/alerts/alert:signal-g047-high'
curl -fsS 'http://localhost:15173/v1/insights/insight:signal-g047-high'
```

Perform browser validation at desktop and 375px mobile widths. Capture screenshots and verify:

- Navigation contains Dashboard, Raw Events, Normalized Events, Signals, Alerts, Insights, Sources,
  Pipelines, Rules, and System without overlap.
- `/alerts` renders live backend data or a truthful empty/error state.
- Selecting an alert loads its detail panel.
- `/insights` renders live backend data or a truthful empty/error state.
- Selecting an insight loads its detail panel.
- Dashboard alert/insight widgets use real G047 APIs and link to the new routes.
- Browser console has no errors.
- No extra dashboard SSE connection is opened.
- No horizontal page overflow at 375px.

If Playwright or browser automation is unavailable, document the exact residual gap rather than
claiming visual validation.

## G048 Acceptance Criteria

- Alerts UI exists at `/alerts`.
- Active Insights UI exists at `/insights`.
- Both pages use real G047 REST APIs with typed client methods and TanStack Query hooks.
- Both pages support truthful loading, error, empty, list, selection, and detail states.
- Dashboard surfaces Open Alerts and Active Insights from real backend APIs.
- UI clearly distinguishes signals, alerts, and insights.
- No unsupported lifecycle mutation, streaming, replay, auth, correlation, timeline, or rule execution
  capability is claimed.
- Routes and navigation are preserved and accessible.
- Tests, production build, audit, Compose build/deploy, API proxy checks, and browser validation pass
  or any unavailable validation is explicitly recorded.
- `docs/build_journal.md` and `docs/gate_audit.md` receive UTC timestamped G048 evidence.
