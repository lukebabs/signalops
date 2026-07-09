# SignalOps Frontend Alert and Insight Lifecycle Controls Specification

Status: ready for frontend-agent implementation
Gate: G050
Author: Codex
Date: 2026-07-09
Backend baseline: G049 backend alert and insight lifecycle mutation APIs

## Purpose

Extend the existing G048 Alerts and Active Insights frontend pages with operator lifecycle controls
backed by the G049 gateway endpoints. Operators must be able to acknowledge, resolve, and suppress
alerts, and review, dismiss, and archive insights from the current detail/list workflow without
fabricating authentication, streaming, or transition policy that does not exist yet.

This gate is a frontend implementation handoff. The backend API is already implemented, deployed,
validated, journaled, and pushed in G049.

## Current Frontend Baseline

The frontend already has:

- `/alerts` route with alert filters, list/table, metrics, selection, and detail panel.
- `/insights` route with insight filters, list/table, metrics, selection, and detail panel.
- Typed alert/insight records and list/detail client methods.
- TanStack Query hooks for alert and insight list/detail reads.
- Dashboard Open Alerts and Active Insights summaries based on REST queries.
- No working lifecycle action controls.

Do not replace these pages. Add focused lifecycle controls and data mutations in the existing style.

## Backend Contracts

Use same-origin requests through the existing web proxy.

### Existing Read Endpoints

- `GET /v1/alerts?tenant_id={tenant_id}&source_id={source_id}&dataset={dataset}&severity={severity}&status={status}&limit=50`
- `GET /v1/alerts/{alert_id}`
- `GET /v1/insights?tenant_id={tenant_id}&source_id={source_id}&dataset={dataset}&insight_type={insight_type}&status={status}&limit=50`
- `GET /v1/insights/{insight_id}`

### New G049 Mutation Endpoints

Alerts:

- `POST /v1/alerts/{alert_id}/acknowledge`
- `POST /v1/alerts/{alert_id}/resolve`
- `POST /v1/alerts/{alert_id}/suppress`

Insights:

- `POST /v1/insights/{insight_id}/review`
- `POST /v1/insights/{insight_id}/dismiss`
- `POST /v1/insights/{insight_id}/archive`

Each mutation returns the same detail envelope shape as the corresponding read endpoint:

```json
{
  "alert": {
    "alert_id": "alert:signal-g049-high",
    "status": "acknowledged",
    "acknowledged_at": "2026-07-09T00:16:13.80459Z",
    "acknowledged_by": "operator-g049",
    "metadata": {
      "lifecycle": {
        "action": "acknowledge",
        "actor": "operator-g049",
        "note": "G049 acknowledgement validation",
        "mutated_at": "2026-07-09T00:16:13.804580257Z"
      }
    }
  }
}
```

```json
{
  "insight": {
    "insight_id": "insight:signal-g049-high",
    "status": "reviewed",
    "reviewed_at": "2026-07-09T00:16:13.808455Z",
    "reviewed_by": "operator-g049",
    "metadata": {
      "lifecycle": {
        "action": "review",
        "actor": "operator-g049",
        "reason": "G049 review validation",
        "mutated_at": "2026-07-09T00:16:13.808448974Z"
      }
    }
  }
}
```

The examples are partial; the real response includes the full alert or insight object.

### Request Body

The backend accepts an optional JSON body:

```json
{
  "actor": "operator-local",
  "note": "optional operator note",
  "reason": "optional operator reason"
}
```

The gateway actor precedence is:

1. `X-SignalOps-Actor` header
2. request body `actor`
3. default `operator-local`

Until formal auth lands, the frontend must use the placeholder actor value `operator-local` by
default. Do not build user/session identity assumptions.

Recommended frontend mutation body for G050:

```json
{
  "actor": "operator-local",
  "reason": "operator action from SignalOps web"
}
```

A free-text note/reason input is optional for this gate. If implemented, keep it compact and local to
the selected alert/insight action, and never require it before an action can run.

### Error Contract

- Missing alert: `404 alert_not_found`
- Missing insight: `404 insight_not_found`
- Malformed body: `400 invalid_json`
- Storage unavailable: `503 storage_unavailable`
- Unexpected backend failure: `500 query_failed`

The UI must surface failures in the page/action area without clearing the current list or selection.

## Types

Extend existing `web/src/types.ts` or the local API type module with narrow mutation types.

Suggested types:

```ts
export type AlertLifecycleAction = 'acknowledge' | 'resolve' | 'suppress'
export type InsightLifecycleAction = 'review' | 'dismiss' | 'archive'

export interface LifecycleMutationRequest {
  actor?: string
  note?: string
  reason?: string
}

export interface AlertLifecycleMutationOptions extends LifecycleMutationRequest {
  alertId: string
  action: AlertLifecycleAction
}

export interface InsightLifecycleMutationOptions extends LifecycleMutationRequest {
  insightId: string
  action: InsightLifecycleAction
}
```

Keep record `status` fields permissive enough to handle backend strings. Do not make the whole app
fragile if the backend later adds a new lifecycle status.

## API Client Changes

Add mutation methods near the existing alert/insight client functions in `web/src/api/client.ts`.

Suggested client shape:

```ts
mutateAlertLifecycle: ({ alertId, action, ...body }: AlertLifecycleMutationOptions) =>
  post<AlertResponse>(`/v1/alerts/${encodeURIComponent(alertId)}/${action}`, body)

mutateInsightLifecycle: ({ insightId, action, ...body }: InsightLifecycleMutationOptions) =>
  post<InsightResponse>(`/v1/insights/${encodeURIComponent(insightId)}/${action}`, body)
```

If the current client does not have a shared `post` helper, add one consistent with the existing
`get` helper. It must:

- Set `Content-Type: application/json` when sending a body.
- Encode path IDs with `encodeURIComponent` because IDs contain `:`.
- Send `X-SignalOps-Actor: operator-local` or include `actor: 'operator-local'` in the body. Prefer
  the header if it fits the current client pattern; body actor is acceptable for G050.
- Parse and return the backend JSON envelope.
- Preserve existing error handling conventions.

## Query Hooks

Add TanStack Query mutation hooks near the existing alert/insight hooks in `web/src/api/queries.ts`.

Suggested hooks:

```ts
export function useMutateAlertLifecycle() { ... }
export function useMutateInsightLifecycle() { ... }
```

On mutation success:

- Update or invalidate the selected detail query for the mutated record.
- Invalidate alert or insight list queries so filtered counts/tables refresh.
- Invalidate Dashboard alert/insight summary queries if they use separate keys.
- Do not open a second SSE connection.

A simple invalidate/refetch strategy is acceptable. Optimistic updates are optional; if implemented,
keep rollback behavior correct.

## Alerts Page UX

Add lifecycle controls to `/alerts` in the selected alert detail area and, optionally, as compact row
actions if the table can support them without crowding. Detail-panel controls are required; row actions
are optional.

Required alert actions:

- `Acknowledge`
- `Resolve`
- `Suppress`

Use existing button styling and lucide icons if available. Recommended icons:

- `CheckCircle2` for acknowledge
- `CircleCheck` or `Check` for resolve
- `BellOff` or `EyeOff` for suppress

Control behavior:

- Disable all alert action buttons while an alert mutation is in flight.
- Disable `Acknowledge` when the selected alert status is already `acknowledged`, `resolved`, or
  `suppressed`.
- Disable `Resolve` when the selected alert status is already `resolved` or `suppressed`.
- Disable `Suppress` when the selected alert status is already `suppressed`.
- After success, keep the same alert selected and render the updated status/actor/timestamps.
- If a selected alert disappears from the active filter after mutation, keep a truthful detail state or
  select the first remaining row. Document the chosen behavior in the journal.

Display fields after mutation:

- `status`
- `acknowledged_at`
- `acknowledged_by`
- `resolved_at`
- `resolved_by`
- `metadata.lifecycle.action`
- `metadata.lifecycle.actor`
- `metadata.lifecycle.note` or `metadata.lifecycle.reason` when present
- `metadata.lifecycle.mutated_at`

Keep `JsonViewer` for the full metadata object. A compact lifecycle summary above the JSON viewer is
preferred for scanability.

## Insights Page UX

Add lifecycle controls to `/insights` in the selected insight detail area and, optionally, as compact
row actions if the table can support them without crowding. Detail-panel controls are required; row
actions are optional.

Required insight actions:

- `Review`
- `Dismiss`
- `Archive`

Recommended icons:

- `Eye` or `ClipboardCheck` for review
- `XCircle` for dismiss
- `Archive` for archive

Control behavior:

- Disable all insight action buttons while an insight mutation is in flight.
- Disable `Review` when the selected insight status is `reviewed`, `dismissed`, or `archived`.
- Disable `Dismiss` when the selected insight status is already `dismissed` or `archived`.
- Disable `Archive` when the selected insight status is already `archived`.
- After success, keep the same insight selected and render the updated status/actor/timestamps.
- If a selected insight disappears from the active filter after mutation, keep a truthful detail state or
  select the first remaining row. Document the chosen behavior in the journal.

Display fields after mutation:

- `status`
- `reviewed_at`
- `reviewed_by`
- `metadata.lifecycle.action`
- `metadata.lifecycle.actor`
- `metadata.lifecycle.note` or `metadata.lifecycle.reason` when present
- `metadata.lifecycle.mutated_at`

## Dashboard Impact

Dashboard widgets and metric tiles should continue to use REST query data. After a lifecycle mutation,
Dashboard alert/insight query caches should be invalidated so:

- Open Alerts count decreases when an open alert is acknowledged, resolved, or suppressed and the
  dashboard query filters `status=open`.
- Active Insights count decreases when an active insight is reviewed, dismissed, or archived and the
  dashboard query filters `status=active`.

Do not add per-widget SSE subscriptions in G050. The current global dashboard SSE bridge remains the
only SSE connection.

## State, Loading, and Error Requirements

For each mutation:

- Show an in-flight state on the clicked action or action group.
- Prevent duplicate submissions for the same selected record while in flight.
- Preserve the current table/list content while the mutation runs.
- On success, show updated backend-returned state rather than fabricating local status.
- On failure, show a concise error near the action controls and preserve selection.
- Clear action errors after a later successful mutation or selection change.

Use accessible button labels. Do not rely on color alone to indicate status, severity, disabled state,
or success/failure.

## Out of Scope

Do not implement these in G050:

- Authentication, login, RBAC, or real user identity.
- Full audit history beyond the latest `metadata.lifecycle` object.
- Custom state-transition policy beyond disabled buttons described above.
- SSE/WebSocket alert or insight streams.
- Bulk lifecycle actions.
- Confirmation modals unless the frontend-agent determines suppress/dismiss/archive need them for
  local UX consistency.
- Backend changes.
- New package dependencies unless there is a clear existing project need.

## Files Expected To Change

Likely files:

- `web/src/types.ts`
- `web/src/api/client.ts`
- `web/src/api/queries.ts`
- `web/src/routes/AlertsRoute.tsx`
- `web/src/routes/InsightsRoute.tsx`
- `web/src/routes/DashboardRoute.tsx` if query invalidation/key shape requires adjustment
- `web/src/api/alerts_insights.test.ts` or a new focused lifecycle mutation test file
- `docs/build_journal.md`
- `docs/gate_audit.md`

Shared UI components may be added only if they reduce meaningful duplication between the Alerts and
Insights detail panels.

## Testing Requirements

Add or update frontend tests for:

- Alert lifecycle client URL encoding and action mapping.
- Insight lifecycle client URL encoding and action mapping.
- Mutation body/header includes placeholder actor behavior.
- Successful mutation invalidates/refetches relevant alert/insight queries.
- Disabled action behavior for terminal statuses where practical.

Minimum commands:

- `cd web && npm test`
- `cd web && npm run build`
- `cd web && npm audit`
- `docker compose build web`
- `docker compose up -d web`
- `docker compose config --quiet`

If the frontend-agent changes no Compose files, `docker compose config --quiet` is still required as a
regression check.

## Live Validation Requirements

Use the existing backend and web proxy. Validate at least one alert mutation and one insight mutation
through the browser-facing proxy. Prefer fresh validation records if available; otherwise use existing
G049 rows and document their current status.

Suggested API proxy checks:

- `curl -fsS http://localhost:15173/v1/alerts/alert:signal-g049-high`
- `curl -fsS -X POST -H 'Content-Type: application/json' -d '{"actor":"operator-local","reason":"frontend G050 validation"}' http://localhost:15173/v1/alerts/alert:signal-g049-high/suppress`
- `curl -fsS http://localhost:15173/v1/insights/insight:signal-g049-high`
- `curl -fsS -X POST -H 'Content-Type: application/json' -d '{"actor":"operator-local","reason":"frontend G050 validation"}' http://localhost:15173/v1/insights/insight:signal-g049-high/dismiss`

If the selected rows are already suppressed/dismissed, either create a fresh validation signal through
the backend path or validate a different non-terminal action and record the exact row/status used.

## Browser Validation Requirements

Use Playwright or the existing browser validation approach at desktop and 375px mobile widths.
Capture screenshots and verify:

- `/alerts` renders lifecycle action controls in the selected alert detail panel.
- Alert action controls fit without overlap at desktop and mobile widths.
- Clicking a valid alert action updates status/actor/timestamp from the backend response.
- Disabled alert actions are visually and functionally disabled for terminal statuses.
- `/insights` renders lifecycle action controls in the selected insight detail panel.
- Clicking a valid insight action updates status/actor/timestamp from the backend response.
- Disabled insight actions are visually and functionally disabled for terminal statuses.
- Dashboard Open Alerts and Active Insights summaries refresh after relevant mutations.
- Browser console has no errors.
- No extra dashboard SSE connection is opened.
- No horizontal page overflow at 375px.

If Playwright or browser automation is unavailable, document the exact residual validation gap rather
than claiming browser validation.

## Documentation Requirements

Update `docs/build_journal.md` and `docs/gate_audit.md` with a UTC timestamp including:

- Files changed.
- Lifecycle controls implemented.
- Tests/build/audit commands and results.
- Live API/proxy rows used for validation.
- Browser validation results and screenshot locations if applicable.
- Any residual gaps.

## G050 Acceptance Criteria

- Alerts page exposes working Acknowledge, Resolve, and Suppress controls backed by G049 APIs.
- Insights page exposes working Review, Dismiss, and Archive controls backed by G049 APIs.
- Frontend uses real backend responses and does not fabricate lifecycle state.
- Placeholder operator identity is used consistently and documented as temporary.
- Query caches refresh list/detail/dashboard data after successful mutations.
- Loading, disabled, success, and error states are truthful and accessible.
- No additional SSE subscriptions or unsupported auth/audit claims are introduced.
- Tests, production build, npm audit, Compose validation/build/deploy, live proxy checks, and browser
  validation pass or unavailable validation is explicitly recorded.
- `docs/build_journal.md` and `docs/gate_audit.md` receive UTC timestamped G050 evidence.
