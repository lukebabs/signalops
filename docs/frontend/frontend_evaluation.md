# SignalOps Frontend — Spec Evaluation & Stack Architecture

> Status: evaluation of `docs/frontend_implementation_spec.md` against the current backend,
> plus a stack recommendation for a data-heavy, time-series-oriented operational UI.
> Audience: the implementing code agent and reviewers. Date: 2026-07-08.
>
> Every backend claim below was verified against source: `internal/api/router.go`,
> `cmd/gateway/main.go`, `docs/api.md`, `compose.yaml`, `Makefile`,
> `internal/storage/storage.go`, and `docs/gate_audit.md`.

---

## 1. What the current spec gets right

The spec reflects genuine understanding of the backend. These claims are accurate:

- **Gateway URL** — `http://localhost:18000`. `compose.yaml` maps `"18000:8080"` on the
  `gateway` service (`SIGNALOPS_HTTP_ADDR: ":8080"` inside the container).
- **Health shape** — `/healthz` returns `{"status":"ok","service":"signalops-gateway","time":"…"}`
  (`router.go:42-48`; `service` is `"signalops-gateway"` because `main.go:46` passes
  `ServiceName: "signalops-gateway"`).
- **All six query endpoints exist** — `/v1/scheduler/runs`, `/v1/scheduler/runs/{run_id}`,
  `/v1/provider-usage`, `/v1/raw-events`, `/v1/raw-events/{event_id}`, `/v1/idempotency`
  (`router.go:58-146`).
- **Error codes** — `400 missing_query` (`router.go:137`), `404 *_not_found`
  (`scheduler_run_not_found` / `raw_event_not_found` / `idempotency_not_found`, via
  `writeQueryError` `router.go:386-392`), `500 query_failed`, `503 storage_unavailable`
  (`requireQueryRepository`, `router.go:378-383`). Body shape is `{"error":<code>,"message":<text>}`
  (`writeError`, `router.go:415-420`).
- **503 semantics** — returned when `QueryRepository == nil` (storage not wired). This is
  exactly what the spec's System-view probe (`/v1/scheduler/runs?limit=1` → 200 vs 503) relies on.
- **Make targets** — `make compose-up` and `make compose-storage-migrate` both exist; the latter
  runs `--profile storage run --rm postgres-migrate` over `migrations/*.up.sql`.
- **Gates G027 / G028 / G029** — all `passed` in `docs/gate_audit.md`; scopes match the spec's
  "Current Backend Status" section.
- **Status vocabulary** — `storage.go:12-15` defines `started` / `succeeded` / `failed` /
  `canceled` (American spelling). The spec's badge mapping matches the real constants.
- **JSON defaults** — `config`/`report`/`budget`/`metadata` default to `{}` and `entity_hints`
  to `[]` (`jsonRawOrEmptyObject/Array`, `router.go:364-376`), so JSON viewers always receive
  valid JSON.
- **`web/` does not yet exist**; query endpoints are read-only; pagination is limit-only and
  capped at 200 (`queryLimit`, `router.go:394-407`).

## 2. Gaps in the current spec (ranked by impact)

### HIGH — CORS is unaddressed (day-one blocker)
The gateway is a bare `net/http` mux with **no CORS middleware** (confirmed: no `Access-Control-*`
/ `cors` anywhere in `internal/` or `cmd/`). The spec tells the implementer to run a Vite dev
server on `:5173` and point it at `VITE_SIGNALOPS_API_BASE_URL=http://localhost:18000`. A browser
on `:5173` calling `:18000` is cross-origin; without `Access-Control-Allow-Origin` the browser
blocks every response (even simple GETs). This also bites the Compose `web` service
(`:15173` → `:18000`).

**Fix:** add a Vite dev proxy (`server.proxy` for `/healthz`, `/readyz`, `/v1` →
`http://localhost:18000`). In dev the app calls **same-origin** (`VITE_SIGNALOPS_API_BASE_URL`
empty/unset); the absolute `http://localhost:18000` form is for production/gateway-served use
and requires gateway CORS, which is not implemented — so the proxy is the supported dev path.

### MEDIUM-HIGH — raw-events list envelope is omitted
The Raw Events section shows only the single-event shape `{"raw_event":{…}}`. The list endpoint
actually returns `{"raw_events":[…]}` (plural, `router.go:112`). Runs (`{"runs":[…]}`) and
provider-usage (`{"provider_usage":[…]}`) list envelopes *are* shown; this one isn't, so an
implementer deriving `types.ts` will guess wrong.

**Fix:** add a list example `{"raw_events":[ {…} ]}` and label list vs detail.

### MEDIUM — `/readyz` shape is undocumented
`/readyz` returns `{"status":"ready",…}` — not `"ok"` (`router.go:50-56`). The System view calls
both endpoints and must not assume `status == "ok"` for readiness.

**Fix:** add the `/readyz` example; note `status` differs (`ok` vs `ready`).

### MEDIUM — `omitempty` optional fields are not flagged
Spec examples show every field populated, but several are `omitempty` and may be **absent**; the
detail panels require displaying them:
- Raw event: `broker_topic`, `broker_partition`, `broker_offset` (`router.go:242-244`).
- Idempotency: `topic`, `partition`, `offset`, `payload_hash` (`router.go:257-260`).
- Run: `error_message` (`router.go:216`) and `completed_at` (`router.go:208`, which the
  "duration when completed_at exists" rule already depends on).

**Fix:** add a data-handling rule that these may be absent and must render as `—`/n/a.

### MEDIUM — no rationale vs the existing repo-root `/frontend`
A mature React 18 + Vite 6 + TS 5.7 + Tailwind 3.4 app already lives at
`/home/adminalien/docker/syncratic-core/frontend/` (react-router-dom 6.30, `requestJson<T>`
API client, `getClientConfig`, `PortalLayout`, brand Tailwind palette). The spec proposes a
brand-new `web/` dir and never mentions `/frontend` or justifies the divergence.

**Fix:** one paragraph acknowledging `/frontend` and stating the standalone `web/` rationale
(subsystem independence, no cross-subsystem coupling for this first gate).

### LOW — status badge "started/running"
The real constant is `"started"` (`storage.go:12`); there is no `"running"`. (Note: the Massive
scheduler currently emits only `succeeded`/`failed`, `cmd/massive-scheduler/main.go:289-292`,
but `started`/`canceled` are valid constants the schema supports, so defensive badges are fine.)

**Fix:** `started/running: blue` → `started: blue`.

### LOW — `provider-usage` `run_id` is optional (unstated)
`api.md` says optional; `router.go:89` allows empty. The Run Detail panel correctly passes it.

**Fix:** note `run_id` is optional (omitting returns recent usage across runs).

### LOW — `error_message` absent from the run list example
The field exists (`error_message,omitempty`, `router.go:216`) and the detail panel requires it,
but the list JSON example omits it.

**Fix:** add/annotate it so `types.ts` is complete.

### LOW — idempotency status enum undocumented
`storage.go:19-23` defines `accepted` / `published` / `processed` / `failed` / `duplicate`. The
spec shows only `"published"`.

**Fix:** list the enum values.

## 3. Strengths to preserve

- Evidence-backed backend contracts — rare and valuable.
- An honest "Known Backend Limitations" section (no fake auth, no cursor pagination, generic
  `POST /v1/events/raw` not yet persisted) that correctly constrains the UI.
- Strong operational UX guidance: dense tables, restrained color, no marketing, copy buttons,
  explicit empty/error/loading states, UTC timestamps, no secret rendering.
- A clever, correct System-view storage probe (`?limit=1` → 200 vs 503) that reuses the 503
  semantics rather than inventing a new health signal.
- Acceptance criteria and the `G030` gate name align with the repo's gate/journal conventions
  (`docs/documentation_standards.md`).

---

## 4. Stack architecture

### Vite is sufficient — and it is the build tool, not the framework
Vite is the **build tool / dev server**; React is the framework. The real question is whether
the app needs SSR (Next.js / Remix) or a client-side SPA is enough. For an **internal,
authenticated, data-heavy operational dashboard**, CSR via Vite + React + TS is the right call:
no SSR/SEO is needed, and server-rendering 100%-dynamic API data adds complexity without value.
The heaviness of data and time-series is handled by the **libraries on top**, below — not by the
build tool or by a meta-framework.

### Adopted stack

| Concern | Choice | Why |
|---|---|---|
| Build / dev server | **Vite** | Fast HMR; native ESM; trivial code-splitting. |
| Framework | **React + TypeScript** | Matches the existing repo-root `/frontend`; large ecosystem. |
| Routing | **TanStack Router** | Type-safe, code-based or file-based; better type ergonomics than react-router. |
| Server state | **TanStack Query** | Caching, refetch, polling, per-query loading/error states. Directly satisfies the spec's "loading/error/empty for every view," "refresh re-fetches active view," and the 400/404/500/503 error mapping. |
| Local UI state | **Zustand** | Selected run/event, active view, filter state. Lightweight; leaves server state to Query. |
| Charts | **Apache ECharts** (Apache-2.0) | Performant dense time-series; strong for financial/operational visuals. |
| Data grid | **AG Grid Community (MIT)** | Polished sorting/filtering/basic grouping for the Runs and Raw-Events tables. |
| Styling | **Tailwind CSS** + reusable design system | Matches `/frontend`; reuse its brand token palette where sensible to reduce divergence. |
| Icons | **lucide-react** | As the spec already suggests. |

**AG Grid — Community, deliberately.** Community (MIT) covers sorting, filtering, and basic
grouping, which is all the G030 tables need. Enterprise (commercial license) is required for
pivoting, Excel export, range selection, advanced row grouping, and integrated charting — call
that out as a future decision point rather than baking it in. If license entanglement or bundle
size ever matters, **TanStack Table** (headless, MIT, smaller) is the fallback — more rendering
work, zero license risk.

**Code-splitting.** ECharts, AG Grid, and (later) React Flow are heavy. Lazy-load them per route
via Vite dynamic `import()` so the initial dashboard shell stays light.

### Deferred to Future Gates (do not build in G030)
These have **no backing endpoint in the current backend** and would be idle infrastructure:

- **React Flow** (workflow / graph visualization) — the gateway exposes no topology / DAG
  endpoint. Add when a signal-flow or pipeline model exists.
- **SSE + selective WebSocket** (streaming) — the gateway is **REST-only** (`NewRouter`
  registers only `/healthz`, `/readyz`, the `/v1/*` query endpoints, and `POST /v1/events/raw`;
  no streaming handler). TanStack Query's polling/refetch covers "live-ish" until streaming
  endpoints exist.
- **Client-side time-series evaluation** — ECharts renders series; it does not compute them. If
  real client-side evaluation is needed (resampling, rolling windows, indicators) over large
  series, reach for **Arquero** (Arrow DataFrame, MIT) or **DuckDB-WASM** — or, more scalably,
  push aggregation to the backend (TimescaleDB). **Note:** the current API returns *discrete
  records* (runs / events / usage rows, capped at 200), not time-bucketed series, so serious
  time-series charting implies **new backend aggregation endpoints** — a backend gate, not a
  frontend one.

---

## 5. CORS / dev proxy (the HIGH gap, restated for the implementer)

Add to the spec's Runtime Configuration:

- Vite `server.proxy`: forward `/healthz`, `/readyz`, `/v1` → `http://localhost:18000`.
- Dev: app calls **same-origin** (leave `VITE_SIGNALOPS_API_BASE_URL` unset/empty); the proxy
  targets `:18000`.
- Production / gateway-served: set `VITE_SIGNALOPS_API_BASE_URL` to the absolute base — but this
  requires the gateway to emit CORS headers, which it does not today. Until then, the proxy (or
  serving the frontend from the gateway origin) is the only working path.
- The Compose `web` service (`:15173` → `:18000`) is also cross-origin and needs the same proxy.

---

## 6. Recommendation & open decisions

**Recommendation:** adopt the stack above; fix the HIGH + MEDIUM gaps (CORS/proxy, raw-events
list envelope, `/readyz` shape, `omitempty` handling, `/frontend` rationale); fold in the LOW
items opportunistically; defer React Flow, streaming, and client-side time-series evaluation to
later gates backed by real backend endpoints. With these, the spec is ready for **G030:
Operational Dashboard UI Foundation**.

**Open decisions:**
- Grid: AG Grid Community now; TanStack Table as fallback if license or bundle becomes an issue.
- Time-series evaluation: client-side (Arquero / DuckDB-WASM) vs. backend (TimescaleDB)
  aggregation — decide when the product need is concrete.
- React Flow / streaming timing: gated on backend topology and streaming endpoints.
