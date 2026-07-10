# SignalOps Frontend Replay Jobs UI Implementation Specification

Status: ready for frontend-agent implementation  
Gate: G060  
Author: Codex  
Date: 2026-07-10  
Backend baseline: G058 replay job control plane and G059 replay worker execution

## Purpose

Add first-class Replay Jobs UI controls to the existing SignalOps frontend. Operators must be able
to inspect replay job state, create bounded replay requests, and view replay execution results from
the backend APIs added in G058/G059.

This is an operational UI for temporal replay. Do not build a marketing page, mock data, or a
separate frontend package. The implementation must fit the current dashboard shell and existing
REST/TanStack Query patterns.

## Current Backend Baseline

The backend now supports replay job control-plane and execution:

- `POST /v1/replay/jobs` creates a queued replay job in PostgreSQL.
- `GET /v1/replay/jobs` lists replay jobs.
- `GET /v1/replay/jobs/{replay_job_id}` returns one replay job.
- `replay-worker` claims queued jobs, reads matching TimescaleDB rows, republishes through Redpanda,
  and updates job status/result metadata.

Replay execution is asynchronous. Creating a job only queues it. The worker must run separately.
The UI must not pretend a job has executed until the backend returns updated status/result fields.

Backend references:

- `docs/api.md`, "Replay Jobs"
- `cmd/replay-worker/main.go`
- `internal/api/router.go`, replay job routes and DTOs
- `internal/storage/storage.go`, `ReplayJobRecord`
- `migrations/000008_replay_jobs.up.sql`
- `docs/build_journal.md`, G058/G059 entries
- `docs/gate_audit.md`, G058/G059 gates

## Existing Frontend Baseline

Use the current `web/` app and conventions. Relevant files and patterns:

- API client: `web/src/api/client.ts`
- Query hooks: `web/src/api/queries.ts`
- Types: `web/src/types.ts`
- Router: `web/src/router.tsx`
- Shell navigation: `web/src/components/DashboardShell.tsx`
- Shared components: `MetricTile`, `StatusBadge`, `JsonViewer`, loading/error/empty states
- Existing operational routes: `RunsRoute`, `RawEventsRoute`, `NormalizedEventsRoute`, `SignalsRoute`,
  `SourcesRoute`, `PipelinesRoute`, `RulesRoute`

The web container proxies `/v1` to the gateway. Use same-origin API calls only.

Authentication is already enabled in the deployed stack. The existing frontend auth client should
continue to attach bearer tokens. Do not add a new auth implementation for replay.

## Backend API Contract

### List Replay Jobs

```http
GET /v1/replay/jobs?tenant_id={tenant_id}&source_id={source_id}&dataset={dataset}&source_kind={source_kind}&status={status}&limit=50
```

All filters are optional. The UI should send `tenant_id=tenant-local` until tenant selection exists.

Response:

```json
{
  "replay_jobs": [
    {
      "replay_job_id": "replay-g059-raw",
      "tenant_id": "tenant-local",
      "source_id": "src-massive",
      "dataset": "equity_eod_prices",
      "source_kind": "raw_events",
      "replay_mode": "original",
      "status": "succeeded",
      "requested_by": "operator-g059",
      "window_start": "2026-07-07T00:00:00Z",
      "window_end": "2026-07-10T00:00:00Z",
      "started_at": "2026-07-10T03:00:13.866744499Z",
      "completed_at": "2026-07-10T03:00:13.882665132Z",
      "filters": {"validation": "g059"},
      "options": {"max_records": 1},
      "result": {
        "replay_job_id": "replay-g059-raw",
        "source_kind": "raw_events",
        "scanned": 1,
        "published": 1,
        "max_records": 1,
        "completed_at": "2026-07-10T03:00:13.877805066Z"
      },
      "created_at": "2026-07-10T03:00:00Z",
      "updated_at": "2026-07-10T03:00:13.882665132Z"
    }
  ]
}
```

### Replay Job Detail

```http
GET /v1/replay/jobs/{replay_job_id}
```

Response:

```json
{
  "replay_job": { "replay_job_id": "replay-g059-raw" }
}
```

The real object has the full replay job shape shown above.

### Create Replay Job

```http
POST /v1/replay/jobs
Content-Type: application/json
```

Request body:

```json
{
  "tenant_id": "tenant-local",
  "source_id": "src-massive",
  "dataset": "equity_eod_prices",
  "source_kind": "raw_events",
  "replay_mode": "original",
  "requested_by": "operator-local",
  "window_start": "2026-07-09T00:00:00Z",
  "window_end": "2026-07-10T00:00:00Z",
  "filters": {"symbol": "AAPL"},
  "options": {"max_records": 10}
}
```

Required fields:

- `tenant_id`
- `window_start`
- `window_end`

Defaults applied by backend:

- `source_kind`: `raw_events`
- `replay_mode`: `original`
- `requested_by`: `X-SignalOps-Actor`, then body `requested_by`, then `operator-local`

Supported `source_kind` values:

- `raw_events`
- `normalized_events`
- `signals`

Supported `replay_mode` values:

- `original`
- `latest_compatible`
- `explicit`

Status values:

- `queued`
- `running`
- `succeeded`
- `failed`
- `canceled`

Current worker behavior:

- `raw_events` replays to the raw topic; normalizer reprocesses the message.
- `normalized_events` replays to the normalized topic.
- `signals` replays to the signal topic.
- Replayed payloads include `replay_job_id`, `ingestion_mode: replay`, and `metadata.replay`.
- The worker result records `scanned`, `published`, `max_records`, `source_kind`, and timestamps.

## Required Implementation

### 1. Types

Update `web/src/types.ts` with replay job types. Keep backend strings permissive enough for future
statuses, but expose narrow unions for current controls.

```ts
export type ReplaySourceKind = 'raw_events' | 'normalized_events' | 'signals';
export type ReplayMode = 'original' | 'latest_compatible' | 'explicit';
export type ReplayJobStatus = 'queued' | 'running' | 'succeeded' | 'failed' | 'canceled';

export interface ReplayJob {
  replay_job_id: string;
  tenant_id: string;
  source_id?: string;
  dataset?: string;
  source_kind: ReplaySourceKind | string;
  replay_mode: ReplayMode | string;
  status: ReplayJobStatus | string;
  requested_by: string;
  window_start: string;
  window_end: string;
  started_at?: string;
  completed_at?: string;
  filters: unknown;
  options: unknown;
  result: unknown;
  error_message?: string;
  created_at: string;
  updated_at: string;
}

export interface ReplayJobsResponse {
  replay_jobs: ReplayJob[];
}

export interface ReplayJobResponse {
  replay_job: ReplayJob;
}

export interface ReplayJobCreateRequest {
  tenant_id: string;
  source_id?: string;
  dataset?: string;
  source_kind?: ReplaySourceKind;
  replay_mode?: ReplayMode;
  requested_by?: string;
  window_start: string;
  window_end: string;
  filters?: Record<string, unknown>;
  options?: Record<string, unknown>;
}

export interface ReplayJobFilter {
  tenant_id?: string;
  source_id?: string;
  dataset?: string;
  source_kind?: ReplaySourceKind | '';
  status?: ReplayJobStatus | '';
  limit?: number;
}
```

### 2. API Client

Update `web/src/api/client.ts`.

Add imports for the replay types and methods near existing operational APIs:

```ts
listReplayJobs: (filter: ReplayJobFilter = {}) =>
  get<ReplayJobsResponse>('/v1/replay/jobs', {
    tenant_id: filter.tenant_id ?? 'tenant-local',
    source_id: filter.source_id || undefined,
    dataset: filter.dataset || undefined,
    source_kind: filter.source_kind || undefined,
    status: filter.status || undefined,
    limit: filter.limit ?? 50,
  }),

getReplayJob: (replayJobId: string) =>
  get<ReplayJobResponse>(`/v1/replay/jobs/${encodeURIComponent(replayJobId)}`),

createReplayJob: (body: ReplayJobCreateRequest) =>
  post<ReplayJobResponse>('/v1/replay/jobs', body),
```

If the current client has a shared `post` helper, use it. If it only has GET helpers, add a typed
`post<T>` following the existing auth/error handling pattern. Do not bypass auth token injection.

### 3. Query Hooks

Update `web/src/api/queries.ts`.

Add query keys:

```ts
replayJobs: (filter: ReplayJobFilter) => ['replay-jobs', filter] as const,
replayJob: (replayJobId: string) => ['replay-job', replayJobId] as const,
```

Add hooks:

```ts
export function useReplayJobs(filter: ReplayJobFilter = { tenant_id: 'tenant-local', limit: 50 }) {
  return useQuery({
    queryKey: queryKeys.replayJobs(filter),
    queryFn: () => api.listReplayJobs(filter),
    refetchInterval: 5000,
  });
}

export function useReplayJob(replayJobId?: string) {
  return useQuery({
    queryKey: queryKeys.replayJob(replayJobId ?? ''),
    queryFn: () => api.getReplayJob(replayJobId!),
    enabled: Boolean(replayJobId),
    refetchInterval: (query) => {
      const status = query.state.data?.replay_job.status;
      return status === 'queued' || status === 'running' ? 3000 : false;
    },
  });
}

export function useCreateReplayJob() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: ReplayJobCreateRequest) => api.createReplayJob(body),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['replay-jobs'] });
      queryClient.setQueryData(queryKeys.replayJob(data.replay_job.replay_job_id), data);
    },
  });
}
```

Adjust the exact invalidation syntax to match the installed TanStack Query version and current
project conventions.

### 4. Route

Create:

```text
web/src/routes/ReplayJobsRoute.tsx
```

Route path:

```text
/replay
```

Navigation label:

```text
Replay
```

Recommended icon from `lucide-react`: `History`, `RotateCcw`, or `RefreshCcwDot`.

The page must include:

- Metrics strip
- Filter controls
- Replay job create form
- Replay jobs table/list
- Detail panel for selected job
- JSON result/options/filter viewers
- Loading, error, and empty states

Use `tenant-local` until tenant selection exists.

### 5. Page Metrics

Render `MetricTile` values from the currently loaded list:

- Replay Jobs: total count
- Queued: `status === 'queued'`
- Running: `status === 'running'`
- Failed: `status === 'failed'`
- Published: sum of numeric `result.published` where present

Keep metric labels compact. Do not invent backend metrics beyond the current list response.

### 6. Filters

Provide compact controls above the table:

- Status segmented/select: All, queued, running, succeeded, failed, canceled
- Source kind segmented/select: All, raw events, normalized events, signals
- Source ID text input
- Dataset text input
- Limit select or numeric input: 25, 50, 100, 200
- Refresh icon button

Do not create a date-range filter for listing yet unless it is local-only. The list API does not
accept list-window filters today.

Changing filters should refetch via the query key. Keep the previous table visible while refetching
if current project patterns support it.

### 7. Create Replay Job Form

Place the create form in a right-side panel on desktop or a top section above the table on mobile.
Do not put a card inside another card.

Required inputs:

- Source kind: segmented/select with `raw_events`, `normalized_events`, `signals`
- Replay mode: segmented/select with `original`, `latest_compatible`, `explicit`
- Window start: datetime-local input
- Window end: datetime-local input
- Source ID: optional text input
- Dataset: optional text input
- Max records: numeric input, default `10`, min `1`, max `200`

Hidden/default body values:

- `tenant_id`: `tenant-local`
- `requested_by`: `operator-local` unless current auth utilities expose a stable username; if they do, use it but fall back to `operator-local`.
- `filters`: `{}` for this gate unless source/dataset are represented here for display. Source/dataset should be sent as top-level fields.
- `options`: `{ "max_records": <value> }`

Validation before submit:

- `window_start` and `window_end` are required.
- `window_end` must be after `window_start`.
- Max records must be between 1 and 200.
- Show inline validation near the form. Do not send invalid requests.

Submit behavior:

- Use `useCreateReplayJob` mutation.
- Disable submit while pending.
- On success, select the returned job and refetch the list.
- Show a concise success state such as queued job id/status.
- On error, show the backend error in the form area without clearing current list/detail.

Important: creating a replay job does not start the worker by itself in all deployments. The UI should
say `Queued` or show returned status. It must not say `Started replay` unless the status returned by
backend is `running`.

### 8. Table/List

Render a plain HTML table matching existing operational pages. Do not introduce AG Grid for this
page.

Columns:

- Job: replay job id, requested by, created timestamp
- Source: source kind, source id, dataset
- Window: start/end
- Mode
- Status
- Result: scanned/published/max records when present
- Updated

Rows should be selectable. Selecting a row loads/sets the detail panel. If the selected job is
`queued` or `running`, the detail hook should poll every few seconds.

Status rendering:

- Use existing `StatusBadge` for status.
- `queued` and `running` should look neutral/in-progress.
- `succeeded` should look successful if the existing badge supports that.
- `failed` should look error/critical.
- Avoid adding a new color-heavy palette.

### 9. Detail Panel

The detail panel must show:

- Replay job id with copy control if copy buttons already exist in the project
- Status and mode
- Tenant, source kind, source id, dataset
- Requested by
- Window start/end
- Created, updated, started, completed timestamps
- Error message if present
- Result summary: scanned, published, max records
- `JsonViewer` sections for filters, options, result

If the job is `queued` or `running`, show a subtle in-progress state and keep refetching. Do not add
cancel/retry buttons; backend cancellation/retry is not implemented yet.

### 10. Dashboard Integration

Add a small replay summary to the Dashboard without disrupting the target layout:

- Add a metric tile or compact widget for Replay Jobs.
- Show counts for queued/running/failed from `GET /v1/replay/jobs?tenant_id=tenant-local&limit=50`.
- Link the tile/widget to `/replay`.

Do not open another SSE connection for replay. Use REST query polling only for this gate.

### 11. Navigation

Add `/replay` to the existing shell navigation near Pipelines/Rules or Health, whichever matches the
current nav order best. Keep labels short.

Expected architecture after this gate:

```text
SignalOps
├── Dashboard
├── Event Explorer / Raw Events
├── Timeline
├── Correlation
├── Insights
├── Pipelines
├── Rules
├── Replay
├── Sources
├── Health
├── Administration
└── Settings
```

Use the app's actual nav labels/routes; do not rename existing routes in this gate.

### 12. Auth/Error Handling

Use the existing authenticated API client. Expected live behavior:

- Without auth, `/v1/replay/jobs` returns `401`.
- With auth, list/detail/create should work for users with current SignalOps access.

Frontend must:

- Preserve current auth redirect/session behavior.
- Show existing auth/API error states if the token expires.
- Not store tokens manually.

### 13. Validation Required From Frontend Agent

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
- Confirm the replay jobs list loads.
- Confirm existing validation job `replay-g059-raw` appears if local data is present.
- Create a new small replay job with `max_records=1` and a narrow window.
- Confirm the new job appears as `queued` after creation.
- If the replay worker is run separately, confirm status updates to `running`/`succeeded` and result
  counters appear.
- Confirm Dashboard replay summary links to `/replay`.
- Confirm mobile viewport has no horizontal overflow.

Optional backend validation command for a queued job:

```bash
docker compose --profile replay run --rm   -e SIGNALOPS_REPLAY_ONESHOT=true   -e SIGNALOPS_REPLAY_MAX_RECORDS=1   replay-worker
```

### 14. Acceptance Criteria

- `/replay` route exists and is reachable from navigation.
- Replay jobs list reads live backend data.
- Filters refetch live backend data.
- Create form sends valid `POST /v1/replay/jobs` requests and handles success/error states.
- Selected job detail displays lifecycle timestamps, filters, options, result, and errors.
- Queued/running detail polling works without opening a new SSE connection.
- Dashboard includes a compact replay summary/link.
- UI does not claim replay execution happened until backend status/result shows it.
- No unsupported cancel/retry/batch controls are present.
- Tests/build/audit validations are recorded in `docs/build_journal.md` and `docs/gate_audit.md` by the frontend agent.

## Explicit Non-Goals

- No replay cancellation button.
- No retry failed replay button.
- No worker start/stop control from the browser.
- No custom filter language editor.
- No streaming replay progress protocol.
- No AG Grid introduction for replay.
- No changes to backend replay worker behavior.
