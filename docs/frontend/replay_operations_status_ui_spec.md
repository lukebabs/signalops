# SignalOps Frontend Replay Operations Status UI Specification

Status: ready for frontend-agent implementation  
Gate: G065  
Author: Codex  
Date: 2026-07-10  
Backend baseline: G064 replay operations observability backend

## Purpose

Surface replay-worker health and replay operations activity in the existing SignalOps UI using the
G064 backend endpoint. Operators should be able to see whether the always-on replay worker is alive,
whether replay jobs are accumulating, and what job the worker most recently claimed or completed.

This is an incremental operational visibility gate. Do not redesign Dashboard, create a new route,
add mock data, or add worker start/stop controls. Use the existing authenticated API client,
TanStack Query patterns, Dashboard metric style, and System/Health route conventions.

## Backend Baseline

G064 added a protected backend endpoint:

```http
GET /v1/replay/status?tenant_id={tenant_id}&limit=20
Authorization: Bearer <token>
```

The endpoint returns:

```json
{
  "replay_status": {
    "generated_at": "2026-07-10T06:03:00Z",
    "job_counts": {
      "queued": 0,
      "running": 0,
      "succeeded": 3,
      "failed": 0,
      "canceled": 0
    },
    "workers": [
      {
        "worker_id": "signalops-replay-worker",
        "status": "idle",
        "health": "online",
        "process_started_at": "2026-07-10T06:01:36Z",
        "last_seen_at": "2026-07-10T06:09:57Z",
        "last_claimed_at": "2026-07-10T05:13:35Z",
        "last_claimed_replay_job_id": "replay-123",
        "last_completed_at": "2026-07-10T05:13:35Z",
        "last_completed_replay_job_id": "replay-123",
        "last_error_at": null,
        "last_error_message": "",
        "metadata": {
          "one_shot": false,
          "max_records": 50,
          "batch_size": 50,
          "publish_max_attempts": 3,
          "poll_interval": "5s"
        },
        "created_at": "2026-07-10T06:01:36Z",
        "updated_at": "2026-07-10T06:09:57Z"
      }
    ],
    "latest_jobs": []
  }
}
```

`health` values are backend-derived strings: `online`, `stale`, or `error`. `status` values are
worker state strings: `idle`, `running`, `error`, or `stopping`.

Endpoint behavior:

- Requires auth like other `/v1` APIs.
- Public unauthenticated request returns `401`.
- `limit` controls workers/latest job response size and should be modest in UI.
- `tenant_id` should come from the existing tenant helper.

Backend references:

- `docs/api.md`, Replay Jobs section
- `internal/api/router.go`, `GET /v1/replay/status`
- `internal/storage/storage.go`, `ReplayWorkerHeartbeatRecord`, `ReplayJobStatusCount`
- `cmd/replay-worker/main.go`, heartbeat updates
- `migrations/000009_replay_worker_heartbeats.up.sql`
- `docs/build_journal.md`, G064 entry
- `docs/gate_audit.md`, G064 gate

## Existing Frontend Baseline

Use the current `web/` app and conventions:

- Types: `web/src/types.ts`
- API client: `web/src/api/client.ts`
- Query hooks: `web/src/api/queries.ts`
- Dashboard route: `web/src/routes/DashboardRoute.tsx`
- System/Health route: `web/src/routes/SystemRoute.tsx`
- Components: `MetricTile`, `StatusBadge`, `RefreshButton`, `ErrorState`, existing table/list styling
- Auth/tenant helpers: `web/src/auth/session.tsx`
- Format helpers: `web/src/lib/format.ts`

Keep same-origin API calls. Do not add a new auth implementation.

## Required Frontend Changes

### 1. Types

Add replay operations status types to `web/src/types.ts`.

Suggested names:

```ts
export type ReplayWorkerHealth = 'online' | 'stale' | 'error' | string;
export type ReplayWorkerStatus = 'idle' | 'running' | 'error' | 'stopping' | string;

export interface ReplayWorkerStatusRecord {
  worker_id: string;
  status: ReplayWorkerStatus;
  health: ReplayWorkerHealth;
  process_started_at: string;
  last_seen_at: string;
  last_claimed_at?: string;
  last_claimed_replay_job_id?: string;
  last_completed_at?: string;
  last_completed_replay_job_id?: string;
  last_error_at?: string;
  last_error_message?: string;
  metadata: unknown;
  created_at: string;
  updated_at: string;
}

export interface ReplayOperationsStatus {
  generated_at: string;
  job_counts: Record<string, number>;
  workers: ReplayWorkerStatusRecord[];
  latest_jobs: ReplayJob[];
}

export interface ReplayOperationsStatusResponse {
  replay_status: ReplayOperationsStatus;
}
```

Keep these additive. Do not break existing replay job types.

### 2. API Client

Add to `web/src/api/client.ts`:

```ts
getReplayStatus(filter: { tenant_id?: string; limit?: number }): Promise<ReplayOperationsStatusResponse>
```

Implementation requirements:

- Use `GET /v1/replay/status`.
- Pass `tenant_id` and `limit` as query parameters.
- Reuse the existing `get` helper so auth headers and error handling stay consistent.
- Do not call this endpoint unauthenticated manually; let existing auth/session behavior govern it.

### 3. Query Hook

Add a query key and hook in `web/src/api/queries.ts`:

```ts
replayStatus: (tenantId: string, limit?: number) => ['replay-status', tenantId, limit]
useReplayStatus({ tenant_id, limit }: { tenant_id: string; limit?: number })
```

Behavior:

- Poll every 5 seconds, matching replay job list cadence.
- Treat it as REST polling, not SSE.
- Keep `staleTime` short or unset; operator status should be fresh.
- On manual dashboard refresh, this query must refetch with the rest of the dashboard.

### 4. Dashboard Integration

Update `web/src/routes/DashboardRoute.tsx`.

Use `useReplayStatus({ tenant_id: TENANT_ID, limit: 5 })` alongside existing `useReplayJobs`.

Add minimal visibility without expanding the dashboard layout aggressively:

- Keep the existing Replay Jobs metric tile.
- Prefer deriving its hint from `replay_status.job_counts` when available instead of counting the list response.
- Include worker health in the hint if concise:
  - Example: `worker online · 0 queued · 0 running · 0 failed`
  - If multiple workers: summarize worst health (`error` > `stale` > `online`) and count workers.
- If the status query errors, keep current replay jobs list behavior and show `status unavailable` or existing unreachable styling.

In the Processing Health panel, add a compact replay row/field:

- Label: `Replay worker`
- Value: worst worker health (`online`, `stale`, `error`, or `unknown`)
- Supporting line: last seen timestamp and last completed job id when present.

Do not add a new Dashboard section or a large table in Dashboard. Detailed rows belong on System.

### 5. System/Health Route Integration

Update `web/src/routes/SystemRoute.tsx` with a fuller replay operations block.

Use `useTenant()` and `useReplayStatus({ tenant_id: TENANT_ID, limit: 10 })`.

Update refresh behavior:

- `refreshAll()` should refetch replay status.
- Refresh button loading should include replay status fetching.

Add compact metric tiles:

- `Replay Worker` — worst health or `unknown`
- `Replay Queue` — queued count
- `Replay Running` — running count
- `Replay Failed` — failed count
- `Replay Last Seen` — latest worker `last_seen_at`

Add a small table/list for workers below metric tiles:

Columns/fields:

- Worker ID
- Health
- Status
- Last seen
- Last claimed job
- Last completed job
- Last error

Behavior:

- If no workers: show an empty state or compact text `No replay worker heartbeat recorded`.
- If query errors: use existing `ErrorState` or compact error display.
- Long worker/job IDs should wrap or truncate without horizontal overflow.
- Use `StatusBadge` where it fits, otherwise restrained text styling.
- Do not put cards inside cards; follow existing System route density.

### 6. Helpers

Add pure helpers only if useful. Suggested `web/src/lib/replayStatus.ts`:

```ts
export function replayJobCount(status: ReplayOperationsStatus | undefined, key: string): number
export function worstReplayWorkerHealth(workers: ReplayWorkerStatusRecord[]): ReplayWorkerHealth | 'unknown'
export function latestReplayWorkerSeenAt(workers: ReplayWorkerStatusRecord[]): string | undefined
```

Health ordering:

1. `error`
2. `stale`
3. `online`
4. `unknown`

Keep helpers small and tested if introduced.

### 7. Tests

Add lightweight tests consistent with the existing test setup.

Recommended coverage:

- API client builds `GET /v1/replay/status?tenant_id=tenant-local&limit=5`.
- Worst health helper returns `error` over `stale` over `online`.
- Count helper handles missing statuses as zero.
- Optional component test only if existing route tests already make it cheap.

Do not introduce a heavy new test framework.

## Validation Required From Frontend Agent

Run and record:

```bash
cd web
npm test
npm run build
npm audit --json
```

Validate Compose/web proxy:

```bash
docker compose -f compose.yaml -f compose.traefik.yaml config --quiet
docker compose -f compose.yaml -f compose.traefik.yaml build web
docker compose -f compose.yaml -f compose.traefik.yaml up -d --force-recreate web
curl -fsS http://localhost:15173/system
curl -fsS http://localhost:15173/
```

Authenticated browser validation:

- Sign in through the existing auth flow.
- Open Dashboard.
- Confirm Replay Jobs tile still renders and includes worker health/status context.
- Confirm Processing Health includes replay worker health and last seen/last completed context.
- Open System/Health route.
- Confirm replay worker metrics render.
- Confirm worker table/list shows `signalops-replay-worker`, health, status, last seen, and metadata-derived behavior if displayed.
- Confirm manual refresh refetches replay status.
- Confirm mobile viewport has no horizontal overflow.

Unauthenticated API sanity check:

```bash
curl -i https://signalops.syncratic.io/v1/replay/status?tenant_id=tenant-local
```

Expected: `401 Unauthorized`.

## Acceptance Criteria

- Frontend API client exposes `getReplayStatus` using the existing authenticated request path.
- Query hook polls `GET /v1/replay/status` and participates in manual refresh.
- Dashboard replay tile or Processing Health panel surfaces replay worker health concisely.
- System/Health route shows replay worker health, queue counts, running/failed counts, last seen, and worker activity.
- UI tolerates no heartbeat records, stale workers, error workers, and missing job count keys.
- Existing Dashboard, Replay Jobs, and System behavior continues to work.
- Tests/build/audit validations pass and are recorded in `docs/build_journal.md` and `docs/gate_audit.md` by the frontend agent.

## Explicit Non-Goals

- No backend changes.
- No worker start/stop/restart controls from the browser.
- No retry failed replay job button.
- No bulk cancellation controls.
- No SSE stream for replay worker status.
- No broad Dashboard redesign.
- No new route unless the existing System route cannot reasonably contain the details.
