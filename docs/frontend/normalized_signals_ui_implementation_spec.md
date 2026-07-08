# SignalOps Frontend Normalized Events and Signals UI Implementation Specification

Status: ready for frontend-agent implementation  
Gate: G046  
Author: Codex  
Date: 2026-07-08  
Backend baseline: G044 durable normalized event pipeline and G045 durable signal persistence

## Purpose

Expose the new durable normalized-event and signal ledgers in the existing SignalOps `web/`
frontend. Operators must be able to inspect normalized events emitted by the Go normalizer and
signals emitted by Python algorithm workers then persisted by the Go signal persister.

This gate makes G044/G045 visible in the product. It is read-only. Do not invent alerts, insights,
correlation results, timeline aggregates, rule execution state, or remediation workflow.

## Backend Contracts

Use same-origin requests through the existing web proxy. The gateway endpoints are:

- `GET /v1/normalized-events?tenant_id={tenant_id}&source_id={source_id}&dataset={dataset}&limit=50`
- `GET /v1/normalized-events/{event_id}`
- `GET /v1/signals?tenant_id={tenant_id}&source_id={source_id}&dataset={dataset}&detector_id={detector_id}&severity={severity}&limit=50`
- `GET /v1/signals/{signal_id}`

All query filters are optional. The current UI should use `tenant-local` by default until tenant
selection and authentication exist. Pagination remains limit-only.

### Normalized Event List Envelope

The normalized list endpoint returns:

```json
{
  "normalized_events": [
    {
      "event_id": "evt_5d5a94a0e8ea5d149ec19947",
      "tenant_id": "tenant-local",
      "source_id": "src-massive",
      "source_adapter": "market_data.massive",
      "dataset": "equity_eod_prices",
      "schema_id": "signalops.normalized_signal_event.v1",
      "schema_version": "1.0.0",
      "observation_time": "2026-07-08T00:00:00Z",
      "processing_time": "2026-07-08T00:00:01Z",
      "confidence": 1,
      "entities": [{"type": "ticker", "id": "AAPL"}],
      "evidence": [{"type": "raw_event", "event_id": "raw-1"}],
      "metadata": {"normalizer": "signalops.normalizer"},
      "event": {"schema": "signalops.normalized_signal_event.v1"},
      "raw_topic": "signalops.local.raw.v1",
      "raw_partition": 2,
      "raw_offset": 6,
      "normalized_topic": "signalops.local.normalized.v1",
      "normalized_partition": 2,
      "normalized_offset": 2,
      "created_at": "2026-07-08T00:00:02Z"
    }
  ]
}
```

### Normalized Event Detail Envelope

The detail endpoint returns:

```json
{
  "normalized_event": {
    "event_id": "evt_5d5a94a0e8ea5d149ec19947"
  }
}
```

The detail object has the same fields as list rows. The `event`, `entities`, `evidence`, and
`metadata` fields are JSON payloads and must be rendered with the existing `JsonViewer`.

### Signal List Envelope

The signal list endpoint returns:

```json
{
  "signals": [
    {
      "signal_id": "signalops.static_test.low",
      "tenant_id": "tenant-local",
      "source_id": "src-massive",
      "source_adapter": "market_data.massive",
      "dataset": "equity_eod_prices",
      "detector_id": "signalops.static_test",
      "detector_version": "0.1.0",
      "model_version": "none",
      "signal_type": "quality",
      "severity": "low",
      "confidence": 0.25,
      "event_ids": ["evt_5d5a94a0e8ea5d149ec19947"],
      "window_start": "2026-07-08T00:00:00Z",
      "window_end": "2026-07-08T00:00:00Z",
      "entities": [{"type": "ticker", "id": "AAPL"}],
      "supporting_metrics": {"score": 0.25},
      "graph_targets": [],
      "semantic_evidence": [],
      "evidence": [{"type": "normalized_event", "event_id": "evt_5d5a94a0e8ea5d149ec19947"}],
      "recommendation": {"action": "observe"},
      "event": {"schema": "signalops.signal.v1"},
      "broker_topic": "signalops.local.signal.v1",
      "broker_partition": 0,
      "broker_offset": 3,
      "created_at": "2026-07-08T00:00:03Z"
    }
  ]
}
```

### Signal Detail Envelope

The detail endpoint returns:

```json
{
  "signal": {
    "signal_id": "signalops.static_test.low"
  }
}
```

The detail object has the same fields as list rows. The `event`, `event_ids`, `entities`, `metrics`,
`graph_targets`, `semantic_evidence`, `evidence`, and `recommendation` fields are JSON payloads or
arrays and must be rendered with the existing `JsonViewer`.

### Error Handling

Expected gateway errors:

- `404 normalized_event_not_found`
- `404 signal_not_found`
- `500 query_failed`
- `503 storage_unavailable`

A non-numeric, empty, or non-positive `limit` is silently clamped to the default (`50`) and values above `200` are capped at `200`; the gateway does not return a `400` for a bad limit.

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
- Existing route patterns: `RawEventsRoute`, `RulesRoute`, `SourcesRoute`, `PipelinesRoute`

Do not add a new frontend package, new state library, new chart library, new component framework, or
mock data layer.

## Required Implementation

### 1. Types

Add normalized-event and signal contracts to `web/src/types.ts`.

Recommended names:

```ts
export interface NormalizedEventFilter {
  tenant_id?: string;
  source_id?: string;
  dataset?: string;
  limit?: number;
}

export interface NormalizedEvent {
  event_id: string;
  tenant_id: string;
  source_id: string;
  source_adapter: string;
  dataset: string;
  schema_id: string;
  schema_version: string;
  observation_time: string;
  processing_time: string;
  confidence: number;
  entities: unknown;
  evidence: unknown;
  metadata: unknown;
  event: unknown;
  raw_topic: string;
  raw_partition: number;
  raw_offset: number;
  normalized_topic: string;
  normalized_partition: number;
  normalized_offset: number;
  created_at: string;
  updated_at: string;
}

export interface NormalizedEventsResponse {
  normalized_events: NormalizedEvent[];
}

export interface NormalizedEventResponse {
  normalized_event: NormalizedEvent;
}

export interface SignalFilter {
  tenant_id?: string;
  source_id?: string;
  dataset?: string;
  detector_id?: string;
  severity?: string;
  limit?: number;
}

export interface SignalRecord {
  signal_id: string;
  tenant_id: string;
  source_id: string;
  source_adapter: string;
  dataset: string;
  detector_id: string;
  detector_version: string;
  model_version: string;
  signal_type: string;
  severity: string;
  confidence: number;
  event_ids: string[];
  window_start: string;
  window_end: string;
  entities: unknown;
  supporting_metrics: unknown;
  graph_targets: unknown;
  semantic_evidence: unknown;
  evidence: unknown;
  recommendation: unknown;
  event: unknown;
  broker_topic: string;
  broker_partition: number;
  broker_offset: number;
  created_at: string;
  updated_at: string;
}

export interface SignalsResponse {
  signals: SignalRecord[];
}

export interface SignalResponse {
  signal: SignalRecord;
}
```

Broker coordinate fields (`raw_*`/`normalized_*` for normalized events and `broker_*` for signals)
and signal `window_start`/`window_end` are required in the gateway DTOs (no `omitempty`, `NOT NULL`)
and are always present. The DTOs also return additional fields not listed above (e.g. normalized
`idempotency_key`, `normalized_payload`; signal `source_domain`, `ingestion_mode`, `artifact_ids`,
`timestamp`, `observation_time`, `effective_time`, `processing_time`, `correlation_id`, `trace_id`,
`causation_id`, `replay_job_id`) which the UI may optionally display. The `unknown` fields
(`entities`, `evidence`, `supporting_metrics`, `event`, etc.) carry arbitrary JSON payloads.

### 2. API Client

Update `web/src/api/client.ts`:

- Import the new response and filter types.
- Add:

```ts
listNormalizedEvents: (filter: NormalizedEventFilter = {}) =>
  get<NormalizedEventsResponse>('/v1/normalized-events', {
    tenant_id: filter.tenant_id,
    source_id: filter.source_id,
    dataset: filter.dataset,
    limit: filter.limit ?? 50,
  }),
getNormalizedEvent: (eventId: string) =>
  get<NormalizedEventResponse>(`/v1/normalized-events/${encodeURIComponent(eventId)}`),
listSignals: (filter: SignalFilter = {}) =>
  get<SignalsResponse>('/v1/signals', {
    tenant_id: filter.tenant_id,
    source_id: filter.source_id,
    dataset: filter.dataset,
    detector_id: filter.detector_id,
    severity: filter.severity,
    limit: filter.limit ?? 50,
  }),
getSignal: (signalId: string) =>
  get<SignalResponse>(`/v1/signals/${encodeURIComponent(signalId)}`),
```

### 3. Query Hooks

Update `web/src/api/queries.ts`:

```ts
normalizedEvents: (filter: NormalizedEventFilter) => ['normalized-events', filter] as const,
normalizedEvent: (eventId: string) => ['normalized-event', eventId] as const,
signals: (filter: SignalFilter) => ['signals', filter] as const,
signal: (signalId: string) => ['signal', signalId] as const,
```

Add hooks:

```ts
export function useNormalizedEvents(filter: NormalizedEventFilter) {
  return useQuery({
    queryKey: queryKeys.normalizedEvents(filter),
    queryFn: () => api.listNormalizedEvents(filter),
  });
}

export function useNormalizedEvent(eventId: string | null) {
  return useQuery({
    queryKey: queryKeys.normalizedEvent(eventId ?? ''),
    queryFn: () => api.getNormalizedEvent(eventId!),
    enabled: !!eventId,
  });
}

export function useSignals(filter: SignalFilter) {
  return useQuery({
    queryKey: queryKeys.signals(filter),
    queryFn: () => api.listSignals(filter),
  });
}

export function useSignal(signalId: string | null) {
  return useQuery({
    queryKey: queryKeys.signal(signalId ?? ''),
    queryFn: () => api.getSignal(signalId!),
    enabled: !!signalId,
  });
}
```

### 4. Routes And Navigation

Create two read-only routes:

- `web/src/routes/NormalizedEventsRoute.tsx`
- `web/src/routes/SignalsRoute.tsx`

Update `web/src/router.tsx`:

- Lazy-load `NormalizedEventsRoute`.
- Lazy-load `SignalsRoute`.
- Add `/normalized-events`.
- Add `/signals`.

Update `web/src/components/DashboardShell.tsx`:

- Add `Normalized` or `Normalized Events` near `Raw Events`.
- Add `Signals` near `Rules`.
- Use lucide icons, for example `FileCheck2` for normalized events and `Radar` or `BellRing` for
  signals. Use icon+text nav items consistent with existing navigation.

Preserve every existing route and deep link.

### 5. Normalized Events Page

The Normalized Events route must:

- Use `tenant-local` by default.
- Fetch `GET /v1/normalized-events?tenant_id=tenant-local&limit=50`.
- Provide local filters for `source_id`, `dataset`, and `limit`.
- Render loading, error, and empty states using existing shared components.
- Render only returned backend data.
- Allow selecting a row to load detail via `GET /v1/normalized-events/{event_id}`.

Required page metrics:

- Normalized Events: `normalized_events.length`
- Distinct Sources
- Distinct Datasets
- Average Confidence over the fetched sample

Required table columns:

- Event: event id, schema id/version, copy button
- Source: source adapter, source id
- Dataset
- Observation Time
- Confidence
- Raw Broker Position: topic, partition, offset
- Normalized Broker Position: topic, partition, offset
- Created

Required detail panel sections:

- Identity and timing
- Source and dataset
- Raw and normalized broker lineage
- Entities JSON
- Evidence JSON
- Metadata JSON
- Full normalized event JSON

If a broker coordinate is absent, render `-` or `n/a`; do not show `0` unless the API actually
returns numeric zero.

### 6. Signals Page

The Signals route must:

- Use `tenant-local` by default.
- Fetch `GET /v1/signals?tenant_id=tenant-local&limit=50`.
- Provide local filters for `source_id`, `dataset`, `detector_id`, `severity`, and `limit`.
- Render loading, error, and empty states using existing shared components.
- Render only returned backend data.
- Allow selecting a row to load detail via `GET /v1/signals/{signal_id}`.

Required page metrics:

- Signals: `signals.length`
- Detector Count: distinct `detector_id`
- High/Critical: count by severity over the fetched sample
- Average Confidence over the fetched sample

Required table columns:

- Signal: signal id, signal type, copy button
- Detector: id/version
- Model: version
- Source/Dataset
- Severity
- Confidence
- Event Count
- Window
- Broker Position
- Created

Required detail panel sections:

- Identity, detector, model, source, dataset
- Severity, confidence, signal type
- Event IDs, with plain text links where the id can navigate to `/normalized-events`
- Window start/end
- Signal broker lineage
- Entities JSON
- Supporting metrics JSON
- Graph targets JSON
- Semantic evidence JSON
- Evidence JSON
- Recommendation JSON
- Full signal event JSON

Severity must not be color-only; include visible text. If there is no existing severity component,
use a small local badge that stays visually restrained and consistent with `StatusBadge`.

### 7. Dashboard Integration

Enhance `web/src/routes/DashboardRoute.tsx` without changing its purpose:

- Add normalized-event and signal counts to the existing metrics strip if space allows.
- Add a compact recent Signals widget or add Signals to the operational/activity band.
- Link the normalized metric/widget to `/normalized-events`.
- Link the signals metric/widget to `/signals`.
- Each new dashboard widget must own its loading, error, empty, and refresh state.
- Do not replace the existing raw events, runs, provider usage, catalog, or health widgets.

Do not label signals as alerts or insights. A signal is detector output; alert/insight lifecycle is
a later backend gate.

### 8. Stream Integration

Do not open a second EventSource and do not connect to Redpanda from the browser.

Current SSE channels do not include normalized events or signals. For this gate:

- REST queries are the source of truth.
- Dashboard widgets may refresh through their local refresh controls and ordinary query refetching.
- Do not claim live normalized/signal streaming.
- Do not add WebSockets.
- Do not implement `Last-Event-ID` resume; the backend stream does not support it yet.

If the frontend-agent chooses to add passive polling for these two query families, keep it modest
and document it in the journal. Prefer manual refresh unless the current app already has a matching
polling pattern.

## UX Requirements

- Dense operational UI, not marketing layout.
- Keep cards restrained and avoid nested cards.
- Tables may use horizontal overflow containers; the page itself must not horizontally overflow at
  375px mobile width.
- Text and controls must not overlap at desktop or mobile widths.
- UTC timestamps should use existing date formatting helpers.
- JSON payloads should be collapsible or contained enough to avoid destroying page layout.
- Copy buttons need accessible labels.
- Keyboard focus must remain visible.
- Status/severity/confidence must not depend on color alone.
- Do not render mock records or placeholder operational claims.

## Explicit Non-Goals

- Alert or insight pages
- Acknowledge/escalate workflows
- Rule execution results
- Pipeline execution state
- Correlation graph
- Timeline aggregation
- Knowledge evolution
- Normalized or signal mutation
- Tenant selector/authentication
- Cursor pagination
- New backend endpoints
- Browser-to-broker communication
- WebSocket streaming

## Expected Files

At minimum:

- `web/src/types.ts`
- `web/src/api/client.ts`
- `web/src/api/queries.ts`
- `web/src/router.tsx`
- `web/src/components/DashboardShell.tsx`
- `web/src/routes/NormalizedEventsRoute.tsx`
- `web/src/routes/SignalsRoute.tsx`
- `web/src/routes/DashboardRoute.tsx`
- focused frontend tests
- `docs/build_journal.md`
- `docs/gate_audit.md`

Shared components may be added only where they reduce real duplication between normalized events and
signals.

## Validation Requirements

Run and record:

```bash
cd web && npm test
cd web && npm run build
cd web && npm audit --json
docker compose build web
docker compose up -d web
curl -fsS http://localhost:15173/
curl -fsS 'http://localhost:15173/v1/normalized-events?tenant_id=tenant-local&limit=10'
curl -fsS 'http://localhost:15173/v1/signals?tenant_id=tenant-local&limit=10'
```

If live normalized events and signals exist, also verify one detail endpoint each through the web
proxy:

```bash
curl -fsS 'http://localhost:15173/v1/normalized-events/{event_id}'
curl -fsS 'http://localhost:15173/v1/signals/{signal_id}'
```

Perform browser validation at desktop and 375px mobile widths. Capture screenshots and verify:

- Navigation contains Dashboard, Raw Events, Normalized Events, Signals, Sources, Pipelines, Rules,
  and System without overlap.
- `/normalized-events` renders live backend data or a truthful empty/error state.
- Selecting a normalized event loads its detail panel.
- `/signals` renders live backend data or a truthful empty/error state.
- Selecting a signal loads its detail panel.
- Dashboard additions link to the new routes.
- Browser console has no errors.
- No extra dashboard SSE connection is opened.

If Playwright or browser automation is unavailable, document the exact residual gap rather than
claiming visual validation.

## G046 Acceptance Criteria

- Normalized Events UI exists at `/normalized-events`.
- Signals UI exists at `/signals`.
- Both pages use real G044/G045 REST APIs with typed client methods and TanStack Query hooks.
- Both pages support truthful loading, error, empty, list, selection, and detail states.
- Dashboard surfaces normalized-event and signal summaries without fabricating alerts or insights.
- Routes and navigation are preserved and accessible.
- No unsupported streaming, mutation, replay, auth, alert, or insight capability is claimed.
- Tests, production build, audit, Compose build/deploy, API proxy checks, and browser validation pass
  or any unavailable validation is explicitly recorded.
- `docs/build_journal.md` and `docs/gate_audit.md` receive UTC timestamped G046 evidence.
