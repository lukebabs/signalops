# SignalOps Frontend Implementation Specification

## Purpose

Build the first SignalOps operational UI against the current backend state. The UI is not a marketing site. It is an operator dashboard for inspecting scheduled market-data ingestion, provider usage, raw event publication, and idempotency state.

This spec is intended for another code agent to implement in this repository.

## Current Backend Status

Completed backend gates available to the frontend:

- G027: Massive scheduler run audit and provider usage are persisted to PostgreSQL.
- G028: Broker-acknowledged Massive raw event publications are persisted to `raw_event_ledger` and `idempotency_records`.
- G029: Gateway exposes storage-backed query APIs for scheduler runs, provider usage, raw events, and idempotency lookup.

Local gateway base URL:

```text
http://localhost:18000
```

Gateway runs in Docker Compose as service `gateway`. The local stack also includes Redpanda, PostgreSQL, topic bootstrap, raw worker, and optional Massive scheduler/puller services.

## Product Intent

The frontend should let an operator answer these questions quickly:

- Is SignalOps running and queryable?
- What ingestion runs happened recently?
- Which runs succeeded, failed, or produced zero events?
- How many provider requests/events did a run consume?
- Which raw events were published, and where did they land in Kafka/Redpanda?
- Can a specific idempotency key be found, and what event does it map to?
- What payload was emitted for a raw event?

## Recommended Stack

This is an operational, data-heavy dashboard with time-series evaluation needs — not a
marketing site. Use a client-side TypeScript SPA; no SSR / meta-framework is required for an
internal, authenticated tool. See `docs/frontend/frontend_evaluation.md` for the full stack
rationale.

Adopted stack:

| Concern | Choice |
|---|---|
| Build / dev server | Vite |
| Framework | React + TypeScript |
| Routing | TanStack Router |
| Server state | TanStack Query |
| Local UI state | Zustand |
| Charts | Apache ECharts (Apache-2.0) |
| Data grid | AG Grid Community (MIT) |
| Styling | Tailwind CSS + a small reusable design system |
| Icons | `lucide-react` |

Per-library guidance:

- **TanStack Query** owns server state — caching, refetch, polling, and per-query loading/error
  states. It directly satisfies this spec's loading/error/empty-state and refresh requirements
  and the `400`/`404`/`500`/`503` error mapping.
- **Zustand** holds local UI state only (selected run/event, active view, filters). Do not
  duplicate server state into it.
- **AG Grid Community (MIT)** covers sorting, filtering, and basic grouping for the Runs and
  Raw Events tables. Enterprise features (pivoting, Excel export, range selection, advanced
  grouping, integrated charting) require a commercial license — treat that as a future
  decision, not a G030 assumption. If license or bundle size becomes a concern, TanStack Table
  (headless, MIT) is the fallback.
- **Apache ECharts** for any time-series or metric visualizations.
- **Code-splitting:** ECharts and AG Grid are heavy. Lazy-load them per route via Vite dynamic
  `import()` so the initial dashboard shell stays light.

Do not build a landing page. The first screen must be the operational dashboard.

### Relationship to the existing `/frontend`

A mature React 18 + Vite 6 + TypeScript 5.7 + Tailwind app already exists at the repository
root (`/frontend/`), using react-router-dom and a `requestJson<T>` API client. This gate
intentionally creates a standalone `web/` package under `signalops/` to keep the subsystem
self-contained and avoid cross-subsystem coupling for the first UI gate. Reuse `/frontend`'s
Tailwind brand token palette where it reduces divergence, but do not import from or extend
`/frontend/` directly.

### Package location and structure

Place the new frontend package under:

```text
web/
```

Recommended structure:

```text
web/
  package.json
  index.html
  vite.config.ts
  src/
    main.tsx
    App.tsx
    router.tsx
    api/
      client.ts
      queries.ts
    store/
      ui.ts
    types.ts
    styles/
      index.css
    components/
      StatusBadge.tsx
      MetricTile.tsx
      JsonViewer.tsx
      RunTable.tsx
      RunDetail.tsx
      RawEventTable.tsx
      RawEventDetail.tsx
      IdempotencyLookup.tsx
```

## Runtime Configuration

The frontend must support a configurable API base URL.

Use:

```text
VITE_SIGNALOPS_API_BASE_URL=http://localhost:18000
```

Default to `http://localhost:18000` when the variable is not set.

### CORS and the dev proxy (required)

The gateway is a bare HTTP mux with **no CORS middleware**. A Vite dev server on `:5173`
calling `http://localhost:18000` directly is cross-origin and will be blocked by the browser.
The supported dev path is a Vite proxy: configure `server.proxy` to forward `/healthz`,
`/readyz`, and `/v1` to `http://localhost:18000`, and have the app call **same-origin** in dev
(leave `VITE_SIGNALOPS_API_BASE_URL` unset/empty so requests go to the proxy).

The absolute `http://localhost:18000` form of `VITE_SIGNALOPS_API_BASE_URL` is for production
or gateway-served deployments, where the gateway must emit CORS headers (not implemented
today). Until then, the proxy — or serving the frontend from the gateway origin — is the only
working path. The Compose `web` service (`:15173` → `:18000`) is also cross-origin and needs
the same proxy.

Example `vite.config.ts` proxy:

```ts
server: {
  host: '0.0.0.0',
  proxy: {
    '/healthz': 'http://localhost:18000',
    '/readyz': 'http://localhost:18000',
    '/v1': 'http://localhost:18000',
  },
}
```

Add npm scripts:

```json
{
  "scripts": {
    "dev": "vite --host 0.0.0.0",
    "build": "tsc && vite build",
    "preview": "vite preview --host 0.0.0.0",
    "test": "vitest run"
  }
}
```

If tests are not introduced in the first frontend gate, `npm run build` must still pass.

## API Contracts

Use the gateway endpoints documented in `docs/api.md`.

### Health

```http
GET /healthz
GET /readyz
```

`/healthz` response:

```json
{
  "status": "ok",
  "service": "signalops-gateway",
  "time": "2026-07-08T00:28:34Z"
}
```

`/readyz` response (note `status` is `"ready"`, not `"ok"`):

```json
{
  "status": "ready",
  "service": "signalops-gateway",
  "time": "2026-07-08T00:28:34Z"
}
```

Do not assume `status == "ok"` for readiness — `/readyz` returns `"ready"`.

### Scheduler Runs

```http
GET /v1/scheduler/runs?limit=50
GET /v1/scheduler/runs/{run_id}
```

List response shape:

```json
{
  "runs": [
    {
      "run_id": "massive:src-massive:20260708T001716.692425267Z",
      "tenant_id": "tenant-local",
      "source_id": "src-massive",
      "source_adapter": "market_data.massive",
      "datasets": ["equity_eod_prices"],
      "observation_date": "2026-07-07T00:00:00Z",
      "dry_run": false,
      "status": "succeeded",
      "started_at": "2026-07-08T00:17:16.692425Z",
      "completed_at": "2026-07-08T00:17:17.133235Z",
      "events_built": 1,
      "events_published": 1,
      "provider_requests": 1,
      "provider_retries": 0,
      "failures": 0,
      "config": {},
      "report": {},
      "created_at": "2026-07-08T00:17:17.133776Z",
      "updated_at": "2026-07-08T00:17:17.133776Z"
    }
  ]
}
```

Runs also expose an optional `error_message` string (`omitempty`, present when a run failed).
Include it in `types.ts` and surface it in the Run detail panel.

### Provider Usage

```http
GET /v1/provider-usage?run_id={run_id}&limit=50
```

`run_id` is optional. When omitted, recent provider usage across runs is returned. The Run
detail panel passes `run_id` to scope usage to a single run.

Response shape:

```json
{
  "provider_usage": [
    {
      "usage_id": "...",
      "run_id": "...",
      "provider": "massive",
      "dataset": "equity_eod_prices",
      "request_count": 1,
      "retry_count": 0,
      "event_count": 1,
      "budget": {},
      "created_at": "2026-07-08T00:17:17.135605Z"
    }
  ]
}
```

### Raw Events

```http
GET /v1/raw-events?tenant_id={tenant_id}&source_id={source_id}&dataset={dataset}&limit=50
GET /v1/raw-events/{event_id}
```

List filters are optional and can be combined. List response is wrapped as `raw_events`
(plural):

```json
{
  "raw_events": [
    {
      "event_id": "evt_...",
      "tenant_id": "tenant-local",
      "source_id": "src-massive",
      "dataset": "equity_eod_prices",
      "idempotency_key": "idem_...",
      "observation_time": "2026-07-07T00:00:00Z",
      "broker_topic": "signalops.local.raw.v1",
      "broker_partition": 2,
      "broker_offset": 3
    }
  ]
}
```

Detail response is wrapped as `raw_event` (singular):

```json
{
  "raw_event": {
    "event_id": "evt_...",
    "tenant_id": "tenant-local",
    "source_id": "src-massive",
    "source_adapter": "market_data.massive",
    "dataset": "equity_eod_prices",
    "idempotency_key": "idem_...",
    "observation_time": "2026-07-07T00:00:00Z",
    "processing_time": "2026-07-08T00:17:16.692427Z",
    "broker_topic": "signalops.local.raw.v1",
    "broker_partition": 2,
    "broker_offset": 3,
    "payload": {},
    "entity_hints": [],
    "created_at": "2026-07-08T00:17:17.127515Z"
  }
}
```

### Idempotency Lookup

```http
GET /v1/idempotency?tenant_id={tenant_id}&source_id={source_id}&idempotency_key={key}
```

All three query parameters are required.

Response shape:

```json
{
  "idempotency": {
    "tenant_id": "tenant-local",
    "source_id": "src-massive",
    "idempotency_key": "idem_...",
    "event_id": "evt_...",
    "source_adapter": "market_data.massive",
    "dataset": "equity_eod_prices",
    "topic": "signalops.local.raw.v1",
    "partition": 2,
    "offset": 3,
    "payload_hash": "sha256:...",
    "status": "published",
    "metadata": {},
    "first_seen_at": "2026-07-08T00:17:17.131755Z",
    "last_seen_at": "2026-07-08T00:17:17.131755Z"
  }
}
```

## Required Screens

### 1. Dashboard Shell

The first viewport should be the operational dashboard.

Required layout:

- Top app bar with product name `SignalOps` and gateway health indicator.
- Left navigation or top tab navigation with:
  - Runs
  - Raw Events
  - Idempotency
  - System
- Main work area with dense, scannable operational content.

Avoid hero sections, marketing copy, oversized cards, decorative blobs, and one-note color palettes.

### 2. Runs View

Primary view for scheduler audit.

Required elements:

- Refresh button.
- Limit selector: 25, 50, 100, 200.
- Run table sorted by most recent first.
- Columns:
  - status
  - started time
  - source
  - datasets
  - dry run
  - built
  - published
  - provider requests
  - failures
  - duration when `completed_at` exists
- Selecting a run opens a detail panel, not a separate browser page unless routing is already introduced.

Run detail panel must show:

- full run id with copy button
- status badge
- observation date
- start/completion timestamps
- counters
- datasets
- config JSON viewer
- report JSON viewer
- error message if present
- provider usage for this run, fetched from `/v1/provider-usage?run_id=...`

### 3. Raw Events View

Required elements:

- Filter controls for tenant id, source id, dataset, and limit.
- Raw events table with:
  - event id
  - dataset
  - observation time
  - processing time
  - topic
  - partition
  - offset
  - idempotency key
- Selecting an event opens a detail panel.

Raw event detail panel must show:

- event id with copy button
- idempotency key with copy button
- broker topic/partition/offset
- timestamps
- entity hints JSON viewer
- payload JSON viewer

### 4. Idempotency View

Required form fields:

- tenant id
- source id
- idempotency key

Required behavior:

- Disable lookup until all three fields have content.
- On success, show event id, status, topic, partition, offset, payload hash, first/last seen timestamps, and metadata JSON.
- Provide a button/link to fetch the matching raw event by event id.
- On 404, show a clear empty state: `No idempotency record found`.

### 5. System View

Required elements:

- Gateway health from `/healthz`.
- Gateway readiness from `/readyz`.
- API base URL currently in use.
- Last refresh timestamp.
- Storage query availability indication:
  - call `/v1/scheduler/runs?limit=1`
  - if response is `503 storage_unavailable`, show storage unavailable.
  - if response is `200`, show storage available.

## Interaction Requirements

- All network requests must show loading states.
- Error states must display the endpoint and short error message.
- Empty states must be explicit and non-alarming.
- Provide copy-to-clipboard controls for run id, event id, idempotency key, payload hash, and broker coordinates.
- Refresh should re-fetch only the active view unless the System view is active.
- Keep selected run/event details stable until a new row is selected or refresh removes it.
- Use accessible table semantics and visible focus states.

## Visual Design Requirements

This is an operational tool. Use a quiet, utilitarian visual style:

- Dense but readable tables.
- Restrained color usage.
- Status badges:
  - succeeded: green
  - failed: red
  - started: blue
  - canceled: gray
  - dry-run: neutral/amber tag
- Avoid heavy gradients, hero imagery, large empty cards, decorative orbs, and marketing sections.
- Cards may be used for metric tiles or detail panels, but do not nest cards inside cards.
- Buttons should use icons where natural, preferably from `lucide-react`.
- JSON viewers should use monospace formatting and allow long payloads to scroll inside a constrained pane.

## Data Handling Rules

- Treat all timestamps as UTC and display them consistently.
- Show absolute timestamps. Relative time may be secondary.
- Do not mutate backend state from the query UI.
- Do not display secrets. No Massive API key or `.env` values should ever be read or rendered.
- Preserve raw JSON structure when displaying `config`, `report`, `payload`, `entity_hints`, and `metadata`.
- Several fields are `omitempty` and may be **absent** in real responses; render them as `—`
  (or "n/a") rather than assuming presence:
  - Run: `completed_at`, `error_message`.
  - Raw event: `broker_topic`, `broker_partition`, `broker_offset`.
  - Idempotency: `topic`, `partition`, `offset`, `payload_hash`.
- Idempotency `status` is one of: `accepted`, `published`, `processed`, `failed`, `duplicate`.

## Error Handling

Map backend errors to UI states:

- `400 missing_query`: show validation guidance in the Idempotency view.
- `404 *_not_found`: show a not-found empty state.
- `500 query_failed`: show backend query failure.
- `503 storage_unavailable`: show storage unavailable and direct the operator to check `SIGNALOPS_DATABASE_URL` and Postgres health.
- Network failure: show gateway unreachable and the configured API base URL.

## Local Development

Expected backend startup:

```bash
make compose-up
make compose-storage-migrate
```

Gateway should be available at:

```text
http://localhost:18000
```

If the frontend has a dev server, use a separate port, for example:

```text
http://localhost:5173
```

The frontend code agent should start the dev server after implementation and report the local URL.

## Docker Expectations

Add a frontend Dockerfile only if the implementation gate includes Compose integration. For the first UI gate, a Vite dev server is acceptable.

If Compose integration is included, add a `web` service that:

- depends on `gateway`
- exposes host port `15173` or another unused port
- sets `VITE_SIGNALOPS_API_BASE_URL=http://localhost:18000` for browser access — note this is cross-origin (`:15173` → `:18000`); the Vite dev proxy does not apply to a built/preview image, so the Compose path requires gateway CORS (not implemented today) or serving the frontend from the gateway origin

Do not break existing Compose services.

## Testing And Validation

Minimum validation before marking the frontend gate complete:

```bash
npm install
npm run build
```

If tests are added:

```bash
npm test
```

Manual/API validation:

```bash
curl -fsS http://localhost:18000/healthz
curl -fsS 'http://localhost:18000/v1/scheduler/runs?limit=2'
curl -fsS 'http://localhost:18000/v1/raw-events?limit=2'
```

Browser validation:

- Dashboard loads without console errors.
- Runs view renders live scheduler rows when available.
- Selecting a run fetches provider usage.
- Raw Events view renders live raw event rows when available.
- Selecting a raw event shows JSON payload.
- Idempotency lookup works with a known persisted idempotency key.
- Storage unavailable state can be simulated by pointing the frontend at a gateway without storage wiring or by mocking a `503` response.

## Acceptance Criteria

The frontend implementation is complete when:

- A user can open the app and immediately inspect SignalOps operational state.
- Health/readiness is visible.
- Recent scheduler runs render from the live gateway API.
- Run detail and provider usage are viewable.
- Raw event list/detail is viewable with payload JSON.
- Idempotency lookup works and handles missing records cleanly.
- Loading, error, and empty states are implemented for every view.
- `npm run build` passes.
- Documentation is updated with frontend run instructions.
- The gate audit and build journal are updated with timestamped validation evidence.

## Known Backend Limitations To Respect

- Query pagination is currently limit-based only. Do not invent cursor parameters in the frontend yet.
- Current query endpoints are read-only.
- The gateway now exposes `GET /v1/streams/dashboard` as an SSE endpoint for dashboard updates. REST query endpoints remain the snapshot/detail fallback; WebSockets remain future scope for bidirectional controls.
- There are no topology, DAG, or signal-flow endpoints. Do not build graph/workflow visualization (e.g. React Flow) yet.
- Query endpoints return discrete records (runs, raw events, usage rows), not time-bucketed series. Do not assume time-series aggregation is available from the backend yet.
- Generic `POST /v1/events/raw` publish persistence is not yet implemented; the UI should not assume all gateway-ingested raw events exist in `raw_event_ledger`.
- Authentication/authorization is not implemented yet. Do not add a fake login flow.
- Normalized market-data persistence and signal query APIs are future scope.

## Future Gates (out of scope for G030)

The following are intentionally **not** part of the first UI gate because the backend does not
yet support them. Design the stack so they can be added later without rework:

- **React Flow** (workflow / graph visualization) — pending backend topology / DAG endpoints.
- **Frontend SSE adoption** — wire dashboard widgets to `GET /v1/streams/dashboard` through the frontend subscription bridge while keeping REST fallback. WebSockets remain pending bidirectional operator-control needs.
- **Client-side time-series evaluation** — ECharts renders series but does not compute them.
  If real client-side evaluation (resampling, rolling windows, indicators) is needed over
  large series, reach for Arquero (Arrow DataFrame, MIT) or DuckDB-WASM, or push aggregation
  to the backend (TimescaleDB). Either path implies new backend aggregation endpoints.

## Suggested First Implementation Gate

Recommended next gate name:

```text
G030: Operational Dashboard UI Foundation
```

Scope:

- Scaffold `web/` with Vite + React + TypeScript, using the adopted stack (TanStack Router/Query, Zustand, ECharts, AG Grid Community, Tailwind).
- Implement dashboard shell, health status, Runs view, Raw Events view, Idempotency view, and System view.
- Use live gateway API by default.
- Add frontend run instructions to docs.
- Validate with `npm run build` and browser/manual checks.
