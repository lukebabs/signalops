# SignalOps Frontend Replay Cancellation and Result Accounting Specification

Status: ready for frontend-agent implementation  
Gate: G062  
Author: Codex  
Date: 2026-07-10  
Backend baseline: G061 backend replay hardening

## Purpose

Enhance the existing Replay Jobs UI with the backend capabilities added in G061. Operators must be
able to cancel queued/running replay jobs and inspect the richer replay execution result fields that
now come back from the replay worker.

This is an incremental update to the existing `/replay` operational route. Do not rebuild the page,
create a separate route, introduce mock data, or add worker start/stop controls. Keep the current
SignalOps shell, REST client, TanStack Query patterns, auth behavior, and restrained operational UI
style.

## Current Backend Baseline

G061 added backend replay hardening:

- `POST /v1/replay/jobs/{replay_job_id}/cancel` cancels queued/running replay jobs.
- Replay worker reads temporal rows in bounded batches.
- Replay worker checks cancellation between batches and stops publishing additional records.
- Broker publishes are retried per record.
- Replay job `result` now includes structured execution accounting.

Backend references:

- `docs/api.md`, "Replay Jobs"
- `internal/api/router.go`, replay cancel route
- `internal/storage/storage.go`, replay cancellation contract
- `internal/storage/postgres/repository.go`, `CancelReplayJob` and paginated replay source queries
- `cmd/replay-worker/main.go`, `replayResult` and `replayRecordResult`
- `cmd/replay-worker/main_test.go`, batching/retry/cancellation tests
- `docs/build_journal.md`, G061 entry
- `docs/gate_audit.md`, G061 gate

## Existing Frontend Baseline

G060 implemented the initial `/replay` UI. Build on the existing code:

- Types: `web/src/types.ts`
- API client: `web/src/api/client.ts`
- Query/mutation hooks: `web/src/api/queries.ts`
- Route: `web/src/routes/ReplayJobsRoute.tsx`
- Router: `web/src/router.tsx`
- Shell: `web/src/components/DashboardShell.tsx`
- Shared components: `StatusBadge`, `JsonViewer`, loading/error/empty states, existing button and form styles
- Auth helpers: `web/src/auth/session.tsx`

Do not add a new auth client. Use the existing authenticated same-origin API client.

## Backend API Contract

### Cancel Replay Job

```http
POST /v1/replay/jobs/{replay_job_id}/cancel
Content-Type: application/json
Authorization: Bearer <token>
```

Request body is optional. When supplied, use the existing lifecycle body shape:

```json
{
  "actor": "operator-local",
  "reason": "operator canceled from Replay UI",
  "note": "optional operator note"
}
```

Response status: `200 OK`.

```json
{
  "replay_job": {
    "replay_job_id": "replay-123",
    "status": "canceled",
    "completed_at": "2026-07-10T04:14:46Z",
    "error_message": "canceled by operator-local",
    "result": {
      "canceled": {
        "actor": "operator-local",
        "reason": "operator canceled from Replay UI",
        "canceled_at": "2026-07-10T04:14:46Z"
      }
    }
  }
}
```

The real response includes the full replay job object. Preserve all fields already modeled by G060.

Error behavior:

- `401` when the session is missing/expired. Use existing auth/error behavior.
- `404` when the job does not exist. Show the existing API error surface and refetch the list.
- `200` with existing state may be returned if the job has already moved to a terminal state; render the returned job as authoritative.

### Replay Result Shape

G061 worker result examples may include this successful/non-canceled shape:

```json
{
  "replay_job_id": "replay-test",
  "source_kind": "raw_events",
  "scanned": 3,
  "published": 3,
  "failed": 0,
  "batches": 2,
  "max_records": 3,
  "batch_size": 2,
  "canceled": false,
  "started_at": "2026-07-10T04:00:00Z",
  "completed_at": "2026-07-10T04:00:03Z",
  "records": [
    {
      "source_id": "event-1",
      "key": "tenant:source:event-1",
      "status": "published",
      "topic": "signalops.local.raw.v1",
      "partition": 0,
      "offset": 101,
      "attempts": 1
    },
    {
      "source_id": "event-2",
      "key": "tenant:source:event-2",
      "status": "failed",
      "attempts": 3,
      "error": "publish failed"
    }
  ]
}
```

A canceled job may instead have cancellation metadata merged into `result.canceled`:

```json
{
  "replay_job_id": "replay-test",
  "source_kind": "raw_events",
  "scanned": 1,
  "published": 1,
  "failed": 0,
  "batches": 1,
  "max_records": 3,
  "batch_size": 1,
  "completed_at": "2026-07-10T04:14:46Z",
  "canceled": {
    "actor": "operator-local",
    "reason": "operator canceled from Replay UI",
    "canceled_at": "2026-07-10T04:14:46Z"
  }
}
```

Important: historical replay jobs may have the older G059 result shape with only
`replay_job_id`, `source_kind`, `scanned`, `published`, `max_records`, and `completed_at`. The UI
must tolerate missing G061 fields.

## Required Frontend Changes

### 1. Types

Update `web/src/types.ts` with additive replay result types. Keep existing replay job fields stable.

Required suggested types:

```ts
export type ReplayRecordStatus = 'published' | 'failed' | string;

export interface ReplayRecordResult {
  source_id: string;
  key: string;
  status: ReplayRecordStatus;
  topic?: string;
  partition?: number;
  offset?: number;
  attempts?: number;
  error?: string;
}

export interface ReplayCancellationResult {
  actor?: string;
  reason?: string;
  canceled_at?: string;
}

export interface ReplayResult {
  replay_job_id?: string;
  source_kind?: ReplaySourceKind | string;
  scanned?: number;
  published?: number;
  failed?: number;
  batches?: number;
  max_records?: number;
  batch_size?: number;
  canceled?: boolean | ReplayCancellationResult;
  started_at?: string;
  completed_at?: string;
  records?: ReplayRecordResult[];
  [key: string]: unknown;
}

export interface ReplayJobCancelRequest {
  actor?: string;
  reason?: string;
  note?: string;
}
```

If `ReplayJob.result` is currently typed loosely, narrow it to `ReplayResult | null | undefined` only
where doing so does not cause broad churn. Do not rewrite unrelated types.

### 2. API Client

Add to `web/src/api/client.ts`:

```ts
export function cancelReplayJob(replayJobId: string, body?: ReplayJobCancelRequest): Promise<ReplayJobResponse>
```

Implementation requirements:

- URL-encode `replayJobId`.
- Use `POST /v1/replay/jobs/{id}/cancel`.
- Send JSON only with allowed fields.
- Preserve existing auth headers and same-origin behavior.
- Reuse existing `post` helper if available.

### 3. Query/Mutation Hook

Add to `web/src/api/queries.ts`:

```ts
export function useCancelReplayJob(): UseMutationResult<ReplayJobResponse, Error, { replayJobId: string; reason?: string; note?: string }>;
```

Mutation behavior:

- Include the actor from existing identity helper if the route passes it, or let the backend resolve actor from auth/header/body according to existing patterns.
- On mutate, optimistically mark the matching replay job as `canceled` only if the current status is `queued` or `running`.
- Store previous cache state and roll back on error.
- On success, update the detail cache with returned `replay_job`.
- Invalidate the replay jobs list queries and the specific replay job detail query.
- Disable duplicate in-flight cancel requests for the same selected job.

Do not add retry semantics to the mutation itself beyond the app's existing query client defaults.

### 4. Replay Detail Action

Update `web/src/routes/ReplayJobsRoute.tsx` detail panel:

- Show a cancel action only for `queued` and `running` jobs.
- Hide or disable cancel for `succeeded`, `failed`, and `canceled`.
- Button label: `Cancel` or equivalent concise action.
- Use an existing destructive/action style if one exists; otherwise use a restrained button state consistent with the app.
- Include a short confirmation step before sending the request. This can be an inline confirmation row, popover/dialog, or confirm button state. Do not use `window.confirm` if the app already has a modal/dialog pattern.
- The confirmation should allow an optional reason/note only if it can be implemented without clutter. A default reason such as `operator canceled from Replay UI` is acceptable.
- While cancellation is in flight, disable the button and show existing loading affordance.
- After success, show the returned `canceled` status and refetch list/detail.
- After error, keep the previous status and show the existing API error style.

Copy constraints:

- Do not add visible explanatory tutorial text.
- Use concise operational labels.
- Do not claim records already published were reverted; cancellation only stops additional replay work.

### 5. Result Summary

Enhance the selected replay job detail result display with a compact summary before the existing raw
`JsonViewer`:

Required fields when present:

- `scanned`
- `published`
- `failed`
- `batches`
- `max_records`
- `batch_size`
- `started_at`
- `completed_at`
- cancellation actor/reason/canceled_at when `result.canceled` is an object or job status is `canceled`

Layout requirements:

- Use compact metrics or a simple key/value grid consistent with the existing detail panel.
- Missing values should render as `-` or be omitted consistently with current route conventions.
- Keep the raw result `JsonViewer` available for audit/debugging.

### 6. Per-Record Result Sample

If `result.records` exists and is non-empty, render a compact table/list in the detail panel.

Columns/fields:

- Status
- Source ID
- Key
- Attempts
- Topic
- Partition
- Offset
- Error

Behavior:

- Cap visual height with internal scrolling if there are many records.
- Long keys/errors must truncate or wrap without horizontal overflow.
- Use status styling consistent with `StatusBadge` or existing badge components.
- Keep raw JSON visible after the table for complete audit context.

### 7. List/Table Integration

Update the replay jobs list/table if needed:

- Include `canceled` status in status filtering and badges.
- Keep row selection stable when an optimistic cancellation occurs.
- Queued/running polling should continue; terminal `canceled` should stop detail polling like `succeeded`/`failed`.
- Metrics should include canceled count if the existing summary already groups statuses.

### 8. Dashboard Integration

If the Dashboard replay summary currently shows queued/running/failed only, include canceled where it
fits without expanding the dashboard layout. Do not redesign the Dashboard.

### 9. Tests

Add or update frontend tests where the current test setup supports it. At minimum cover pure helpers
or query/client behavior if component tests are not already established.

Recommended coverage:

- `cancelReplayJob` calls the expected endpoint and method.
- Replay result summary handles both G059 and G061 result shapes.
- Cancel button is shown only for `queued`/`running` jobs.
- Optimistic cancellation rolls back on mutation error if the existing query test harness supports it.

Do not introduce a heavy new test framework for this gate.

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
docker compose -f compose.yaml -f compose.traefik.yaml up -d web
curl -fsS http://localhost:15173/replay
```

Authenticated browser validation:

- Sign in through the existing auth flow.
- Navigate to `/replay`.
- Confirm replay jobs list loads.
- Select a queued or running replay job, or create a narrow replay job for validation.
- Confirm cancel action appears only while status is `queued` or `running`.
- Trigger cancellation and confirm status changes to `canceled`.
- Confirm the detail panel displays cancellation metadata when present.
- Confirm result summary displays scanned/published/failed/batches fields when present.
- Confirm `result.records` sample renders without horizontal overflow.
- Confirm terminal jobs do not show active cancel controls.
- Confirm mobile viewport has no horizontal overflow.

Backend helper for creating/running a small replay job may be coordinated with the backend agent if
needed. Do not hard-code validation IDs in frontend code.

## Acceptance Criteria

- `POST /v1/replay/jobs/{replay_job_id}/cancel` is wired through the authenticated frontend API client.
- Replay cancel mutation exists with optimistic/refetch behavior and safe rollback on failure.
- `/replay` detail panel exposes cancel action only for cancelable statuses.
- Cancel action has disabled/in-flight/error/success states.
- Replay result summary displays G061 counters while tolerating older G059 result objects.
- Per-record replay result samples render when present.
- Canceled status is represented consistently in badges, filters, polling, and metrics.
- Existing replay create/list/detail behavior continues to work.
- Tests/build/audit validations pass and are recorded in `docs/build_journal.md` and `docs/gate_audit.md` by the frontend agent.

## Explicit Non-Goals

- No browser control for starting/stopping the replay worker service.
- No retry failed replay job button.
- No bulk cancellation.
- No streaming replay progress protocol.
- No custom replay filter language editor.
- No backend changes.
- No broad Dashboard redesign.
