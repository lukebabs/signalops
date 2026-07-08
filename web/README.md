# SignalOps Web (Operational Dashboard)

Client-side operational UI for SignalOps, built against the gateway query APIs
(G029). See `docs/frontend_implementation_spec.md` for the full spec and
`docs/frontend/frontend_evaluation.md` for the stack rationale.

## Stack

Vite + React + TypeScript, TanStack Router, TanStack Query, Zustand,
Apache ECharts, AG Grid Community, Tailwind CSS, `lucide-react`.

## Prerequisites

- Node.js `>=20 <23`.
- The SignalOps gateway running locally on `http://localhost:18000`
  (`make compose-up && make compose-storage-migrate` from the subsystem root).

## Install

```bash
cd web
npm install
```

## Develop

```bash
npm run dev
```

The app runs on `http://localhost:5173/`. The browser calls same-origin paths
(`/healthz`, `/readyz`, `/v1`) and Vite proxies them to the gateway, so no CORS
configuration is needed (the gateway has no CORS middleware).

Override the proxy target (server-side) if the gateway is elsewhere:

```bash
SIGNALOPS_GATEWAY_URL=http://localhost:18000 npm run dev
```

## Build / Preview

```bash
npm run build     # tsc type-check + vite build -> dist/
npm run preview   # serve the production build
```

## Configuration

- `VITE_SIGNALOPS_API_BASE_URL` — browser-facing API base. Leave **unset** in
  dev (uses the proxy / same-origin). Set an absolute URL only for
  production/gateway-served deployments where the gateway emits CORS headers.
- `SIGNALOPS_GATEWAY_URL` — server-side proxy target for `vite.config.ts`
  (defaults to `http://localhost:18000`).

See `.env.example`.

## Views

- **Runs** — recent scheduler runs (AG Grid), detail panel with provider usage,
  config/report JSON, and a provider-requests chart (ECharts).
- **Raw Events** — raw event ledger (AG Grid) with filters and a detail panel
  showing payload and entity hints.
- **Idempotency** — lookup by tenant/source/key; handles 404 cleanly; links to
  the matching raw event.
- **System** — gateway health/readiness, storage-availability probe, API base
  URL, last refresh.

## Testing

```bash
npm test          # vitest run (no tests introduced in this gate)
```

`npm run build` is the minimum validation gate for G030.
