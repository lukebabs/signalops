# Syncratic Materialization Preview UI Specification

Status: ready for frontend-agent implementation
Gate: G092 frontend follow-up for G091
Author: Codex
Date: 2026-07-14
Backend baseline: `4c31403 Implement G091 budgeted Syncratic materialization`
Live validation baseline: `ef7e26b Document G091 live validation`

## Purpose

Update the MarketOps Syncratic UI so operators can preview budgeted context materialization before creating deterministic Syncratic context windows and insights.

This is a frontend-only handoff over the existing same-origin `/v1/syncratic/materialize` API. Do not call the external Syncratic user API directly. Do not add scheduled materialization, automatic Ask, Search, ingestion, graph writes, detector threshold changes, policy deployment, or new backend storage.

## Product Semantics

G091 made Syncratic materialization explicit and auditable:

- Dry-run preview answers: which assets would materialize, skip, or hit a budget cap?
- Write mode creates deterministic SignalOps/Syncratic context windows and synthesized insight rows only after an operator action.
- Syncratic Ask remains a separate operator-triggered action on a persisted context window. Materialization must not imply an Ask explanation was generated.

The UI should make preview-first operation the normal path. Operators should be able to inspect why each asset was selected or skipped before creating rows.

## Backend Contract

Use the existing authenticated API client, tenant hook, and React Query patterns. Keep `tenant_id`, `app_id=marketops`, `domain=market_data`, and `use_case=daily_market_surveillance` behavior consistent with the existing Syncratic UI.

### Materialize Route

```http
POST /v1/syncratic/materialize
Content-Type: application/json
```

Dry-run request:

```json
{
  "tenant_id": "tenant-local",
  "universe_group": "top50_megacap",
  "context_strategy": "symbol_signal_cluster_5d",
  "context_builder_version": "syncratic.context_builder.v1",
  "window_start": "2026-07-12T00:00:00Z",
  "window_end": "2026-07-14T00:00:00Z",
  "min_evidence_count": 2,
  "max_assets": 10,
  "max_candidate_windows": 50,
  "max_context_windows": 1,
  "max_insights": 1,
  "dry_run": true
}
```

Write-mode request uses the same parameters with `dry_run=false`.

Status codes:

- `200 OK` for dry-run preview.
- `201 Created` for write mode.

Response envelope:

```json
{
  "materialization": {
    "tenant_id": "tenant-local",
    "universe_group": "top50_megacap",
    "context_strategy": "symbol_signal_cluster_5d",
    "context_builder_version": "syncratic.context_builder.v1",
    "window_start": "2026-07-12T00:00:00Z",
    "window_end": "2026-07-14T00:00:00Z",
    "dry_run": true,
    "scanned_assets": 10,
    "candidate_windows": 1,
    "materialized_context_windows": 0,
    "materialized_insights": 0,
    "skipped_below_threshold": 9,
    "skipped_unchanged": 0,
    "skipped_budget_cap": 0,
    "context_window_ids": [],
    "syncratic_insight_ids": [],
    "decisions": [
      {
        "subject_symbol": "AAPL",
        "action": "would_materialize",
        "reason": "eligible",
        "evidence_count": 9,
        "signal_count": 9,
        "alert_count": 0,
        "artifact_count": 0,
        "graph_proposal_count": 0,
        "label_count": 0,
        "critical_alert": false,
        "related_evidence": false,
        "evidence_digest": "...",
        "context_window_id": "synctx_..."
      }
    ]
  }
}
```

Live G091 validation produced this representative dry-run shape: `scanned_assets=10`, `decisions=10`, AAPL `would_materialize`, `materialized_context_windows=0`, `materialized_insights=0`, `skipped_below_threshold=9`. A tight-cap write created one AAPL context and one deterministic insight. A rerun returned `skipped_unchanged=1` and wrote no duplicates.

### Decision Values

Actions:

- `would_materialize`: dry-run only; this candidate would be created in write mode if budget still permits.
- `materialized`: write mode created/updated the deterministic context and insight.
- `skipped`: no write occurred for this asset.

Reasons:

- `eligible`: selected by evidence/budget rules.
- `below_threshold`: evidence did not meet materialization rules.
- `unchanged_evidence_digest`: existing context has the same idempotency key and evidence digest.
- `candidate_budget_cap`: candidate-window cap was exhausted before evaluating this asset as a candidate.
- `materialization_budget_cap`: candidate was eligible but context/insight write cap was exhausted.

The UI should tolerate unknown future `action` or `reason` values by rendering the raw token in a neutral style rather than failing.

## Required UI Workflow

### 1. Preview Controls

Add a controlled materialization panel to the existing MarketOps Syncratic route. It should be compact and operational, not a landing section.

Controls:

- `window_start` and `window_end` date/time inputs or existing date controls.
- `max_assets` numeric input, default `10` or the route's existing bounded default if already present.
- `max_context_windows` numeric input, default `1` for safe operator preview/write.
- `max_insights` numeric input, default same as `max_context_windows`.
- Optional advanced controls for `min_evidence_count`, `max_candidate_windows`, `universe_group`, and `context_strategy`; keep defaults visible but avoid making every field noisy.
- Primary action: `Preview materialization` sends `dry_run=true`.

Do not run preview automatically on page load.

### 2. Preview Results

After dry-run, render aggregate counters prominently:

- scanned assets
- candidate windows
- would materialize count, derived from `decisions[]` action counts
- below-threshold skips
- unchanged skips
- budget-cap skips
- materialized counts, expected to be zero in dry-run

Render a decision table/list with columns or compact cells for:

- symbol
- action
- reason
- evidence count
- signal count
- alert count
- artifact count
- graph proposal count
- label count
- critical alert
- related evidence
- context window id when available

Sort recommendation:

1. `would_materialize` / `materialized` first
2. budget-cap skips
3. unchanged skips
4. below-threshold skips
5. unknown/other

Use status styling, but keep it restrained:

- eligible/would-materialize: active/accent state
- materialized: success state
- unchanged: neutral state
- below threshold: muted state
- budget cap: warning state

Avoid treating zero materializations as an error when the response itself succeeded.

### 3. Confirmed Write

After a successful dry-run, expose a separate action such as `Materialize selected budget`.

Behavior:

- Uses the same request parameters as the last preview with `dry_run=false`.
- Disable until a preview has succeeded.
- Show a confirmation step or clear inline warning before write mode.
- Disable while pending.
- On success, show write-mode counters and decisions.
- Invalidate/refetch Syncratic insight and context-window list/detail queries.
- If write mode returns `skipped_unchanged`, show it as idempotent success, not failure.

Do not automatically call `POST /v1/syncratic/context-windows/{id}/ask` after materialization.

### 4. Existing Syncratic UI Integration

Place the panel where operators already manage Syncratic contexts/insights. It should complement, not replace:

- context-window list
- insight list/detail
- Ask Syncratic action from G090

After write mode returns new `context_window_ids` or `syncratic_insight_ids`, allow operators to open the created insight/context using existing list/detail patterns. If direct linking is already available, use it; otherwise refetch and let the newly created row appear in the list.

### 5. Errors

Handle existing API/auth behavior:

- `401 unauthorized`: use existing auth/session handling.
- `400 materialize_failed`: show sanitized validation message from backend.
- `503 storage_unavailable` or equivalent existing storage error: show gateway/storage unavailable state.
- Network failure: show gateway unreachable.

Never render bearer tokens, API keys, raw `.env` values, raw provider payloads, or full upstream response bodies.

## Required Types And Client Methods

Add/extend TypeScript types:

```ts
export type SyncraticMaterializationAction =
  | 'would_materialize'
  | 'materialized'
  | 'skipped'
  | string;

export type SyncraticMaterializationReason =
  | 'eligible'
  | 'below_threshold'
  | 'unchanged_evidence_digest'
  | 'candidate_budget_cap'
  | 'materialization_budget_cap'
  | string;

export interface SyncraticMaterializationDecision {
  subject_symbol: string;
  action: SyncraticMaterializationAction;
  reason: SyncraticMaterializationReason;
  evidence_count: number;
  signal_count: number;
  alert_count: number;
  artifact_count: number;
  graph_proposal_count: number;
  label_count: number;
  critical_alert: boolean;
  related_evidence: boolean;
  evidence_digest?: string;
  context_window_id?: string;
}

export interface SyncraticMaterializationResult {
  tenant_id: string;
  universe_group: string;
  context_strategy: string;
  context_builder_version: string;
  window_start: string;
  window_end: string;
  dry_run: boolean;
  scanned_assets: number;
  candidate_windows: number;
  materialized_context_windows: number;
  materialized_insights: number;
  skipped_below_threshold: number;
  skipped_unchanged: number;
  skipped_budget_cap: number;
  context_window_ids?: string[];
  syncratic_insight_ids?: string[];
  decisions: SyncraticMaterializationDecision[];
}
```

Add or update API client method:

```ts
materializeSyncraticContexts(request: SyncraticMaterializationRequest): Promise<SyncraticMaterializationResult>
```

The method must accept `dry_run` and parse `decisions[]`. Preserve existing envelope handling.

## Tests

Add focused frontend tests where the repo already has coverage patterns:

- API client sends `dry_run=true` for preview and parses `decisions[]`.
- API client sends `dry_run=false` for confirmed write.
- Component/helper logic counts decision actions and reasons correctly.
- Dry-run result with zero materialized rows is rendered as successful preview.
- Write-mode unchanged rerun is rendered as idempotent success.
- Ask action is not invoked by preview or write materialization. If direct component assertion is difficult, cover by ensuring only `/v1/syncratic/materialize` is called for these actions.
- Unknown action/reason renders without crashing.

## Manual Validation

Use the local auth-enabled gateway after implementation:

1. Open the MarketOps Syncratic route.
2. Run preview with a bounded smoke request: `max_assets=10`, `max_context_windows=1`, `max_insights=1`, recent validated window.
3. Confirm aggregate counters and `decisions[]` render.
4. Confirm no context/insight write is implied by dry-run.
5. Confirm write action is separate and requires operator intent.
6. Run write mode and confirm created context/insight appears through existing lists.
7. Rerun write mode and confirm unchanged evidence is shown as idempotent success.
8. Confirm no browser request is made to external Syncratic URLs and no Ask route is called automatically.

## Out Of Scope

Do not implement:

- backend API changes;
- scheduled materialization;
- automatic materialization on page load;
- automatic Syncratic Ask after materialization;
- external Syncratic user API calls from browser code;
- Syncratic Search;
- Syncratic ingestion;
- graph writes;
- detector threshold changes;
- policy deployment;
- broad redesign of Alerts/Insights;
- new storage tables or migrations.

## Acceptance Criteria

The frontend-agent implementation is accepted when:

- operators can run a dry-run preview from the MarketOps Syncratic UI;
- the UI renders aggregate counters and per-asset decisions;
- the write action is separate from preview and uses the same parameters with `dry_run=false`;
- successful write mode refetches context/insight data;
- unchanged evidence is presented as idempotent success;
- no automatic Ask, Search, ingestion, graph, detector, policy, or scheduler behavior is introduced;
- tests cover API request shape, result parsing, decision summarization, dry-run success, idempotent write result, and no automatic Ask invocation.
