# Syncratic Context Windows UI Specification

Status: ready for frontend-agent implementation
Gate: G089 frontend follow-up for G088
Author: Codex
Date: 2026-07-13
Backend baseline: `393b097 Implement G088 Syncratic context windows`
Auth/API-key baseline: `e13fdce Support Syncratic API key auth mode`

## Purpose

Wire the SignalOps frontend to the G088 backend APIs so operators can see Syncratic synthesized insights as pattern-level explanations over deterministic context windows.

The UI must make the product distinction clear:

- Alerts are event-level operational work items.
- Syncratic insights are multi-record explanations over bounded evidence windows.

This is a SignalOps UI task over `/v1/syncratic/*`. It must not call the external Syncratic user facade directly, ingest external data, generate LLM narratives, write graph state, mutate alert lifecycle, deploy policies, or edit detector thresholds.

## Existing Context

G088 added backend/API support for:

- durable `syncratic_context_windows` records;
- durable `syncratic_insights` records;
- selective materialization over the MarketOps Top 50 universe;
- idempotent evidence digest skip behavior;
- authenticated create/list/detail APIs.

The frontend already has MarketOps shell/navigation, authenticated `/v1/*` API client behavior, React Query patterns, and read/review surfaces for DSM artifacts, graph proposals, back-tests, baselines, evaluations, and promotion candidates.

## Backend Contract

Use the existing authenticated same-origin `/v1/*` API client pattern. Do not add a new auth mechanism. Do not use `internal/syncratic/userapi` from frontend code.

### List Syncratic Insights

```http
GET /v1/syncratic/insights?tenant_id=tenant-local&app_id=marketops&domain=market_data&use_case=daily_market_surveillance&limit=50
```

Response envelope:

```json
{
  "syncratic_insights": [
    {
      "syncratic_insight_id": "synins_...",
      "tenant_id": "tenant-local",
      "app_id": "marketops",
      "domain": "market_data",
      "use_case": "daily_market_surveillance",
      "context_window_id": "synctx_...",
      "insight_type": "marketops.syncratic.multi_event_context",
      "subject_type": "ticker",
      "subject_id": "AAPL",
      "subject_symbol": "AAPL",
      "status": "active",
      "severity": "medium",
      "confidence": 0.75,
      "title": "AAPL Syncratic context",
      "summary": "AAPL has 2 supporting signals and 1 supporting alerts in the symbol_signal_cluster_5d window.",
      "explanation": "This insight was synthesized from a deterministic Syncratic context window over persisted SignalOps and MarketOps evidence.",
      "supporting_alert_ids": ["alert-..."],
      "supporting_signal_ids": ["sig-..."],
      "supporting_event_ids": ["evt-..."],
      "supporting_artifact_ids": ["artifact-..."],
      "related_graph_proposal_ids": ["graphprop-..."],
      "related_label_ids": ["label-..."],
      "metrics": {},
      "recommendation": {},
      "builder_version": "syncratic.context_builder.v1",
      "created_at": "2026-07-13T00:00:00Z",
      "updated_at": "2026-07-13T00:00:00Z"
    }
  ]
}
```

Supported filters:

- `tenant_id`
- `app_id`
- `domain`
- `use_case`
- `context_window_id`
- `insight_type`
- `subject_symbol`
- `status`
- `limit`

### Get Syncratic Insight

```http
GET /v1/syncratic/insights/{syncratic_insight_id}
```

Response envelope:

```json
{
  "syncratic_insight": {}
}
```

### List Context Windows

```http
GET /v1/syncratic/context-windows?tenant_id=tenant-local&app_id=marketops&domain=market_data&use_case=daily_market_surveillance&limit=50
```

Response envelope:

```json
{
  "context_windows": [
    {
      "context_window_id": "synctx_...",
      "tenant_id": "tenant-local",
      "app_id": "marketops",
      "domain": "market_data",
      "use_case": "daily_market_surveillance",
      "subject_type": "ticker",
      "subject_id": "AAPL",
      "subject_symbol": "AAPL",
      "window_start": "2026-07-01T00:00:00Z",
      "window_end": "2026-07-14T00:00:00Z",
      "context_strategy": "symbol_signal_cluster_5d",
      "context_builder_version": "syncratic.context_builder.v1",
      "signal_types": ["marketops.dsm.volatility_expansion"],
      "detector_ids": ["marketops.dsm.taxonomy_v1"],
      "event_ids": ["evt-..."],
      "signal_ids": ["sig-..."],
      "alert_ids": ["alert-..."],
      "artifact_ids": ["artifact-..."],
      "graph_proposal_ids": ["graphprop-..."],
      "label_ids": ["label-..."],
      "baseline_refs": [],
      "evaluation_refs": [],
      "promotion_candidate_refs": [],
      "summary_metrics": {},
      "evidence_digest": "sha256...",
      "idempotency_key": "tenant-local|daily_market_surveillance|...",
      "status": "active",
      "created_at": "2026-07-13T00:00:00Z",
      "updated_at": "2026-07-13T00:00:00Z"
    }
  ]
}
```

Supported filters:

- `tenant_id`
- `app_id`
- `domain`
- `use_case`
- `subject_symbol`
- `context_strategy`
- `status`
- `limit`

### Get Context Window

```http
GET /v1/syncratic/context-windows/{context_window_id}
```

Response envelope:

```json
{
  "context_window": {}
}
```

### Selective Materialization

```http
POST /v1/syncratic/materialize
Content-Type: application/json
```

Request:

```json
{
  "tenant_id": "tenant-local",
  "universe_group": "top50_megacap",
  "context_strategy": "symbol_signal_cluster_5d",
  "context_builder_version": "syncratic.context_builder.v1",
  "window_start": "2026-07-01T00:00:00Z",
  "window_end": "2026-07-14T00:00:00Z",
  "min_evidence_count": 2,
  "max_assets": 50,
  "max_candidate_windows": 50,
  "max_context_windows": 10,
  "max_insights": 10
}
```

Response envelope:

```json
{
  "materialization": {
    "tenant_id": "tenant-local",
    "universe_group": "top50_megacap",
    "context_strategy": "symbol_signal_cluster_5d",
    "context_builder_version": "syncratic.context_builder.v1",
    "window_start": "2026-07-01T00:00:00Z",
    "window_end": "2026-07-14T00:00:00Z",
    "scanned_assets": 5,
    "candidate_windows": 1,
    "materialized_context_windows": 1,
    "materialized_insights": 1,
    "skipped_below_threshold": 4,
    "skipped_unchanged": 0,
    "skipped_budget_cap": 0,
    "context_window_ids": ["synctx_..."],
    "syncratic_insight_ids": ["synins_..."]
  }
}
```

Materialization action is optional for the frontend implementation. If included, it must be intentionally bounded and operator-triggered. It must not run automatically on page load.

## Required Implementation

### 1. Types

Add TypeScript types near other MarketOps API types.

Core types:

```ts
export type SyncraticInsightStatus = 'active' | 'reviewed' | 'dismissed' | 'archived' | 'superseded';
export type SyncraticContextWindowStatus = 'active' | 'archived' | 'superseded';
export type SyncraticSeverity = 'info' | 'low' | 'medium' | 'high' | 'critical';

export interface SyncraticInsight { ... }
export interface SyncraticContextWindow { ... }
export interface SyncraticMaterializationResult { ... }
```

Use `unknown` for flexible JSON fields:

- `metrics`
- `recommendation`
- `summary_metrics`
- `baseline_refs`
- `evaluation_refs`
- `promotion_candidate_refs`

Response envelope types:

- `SyncraticInsightsResponse`
- `SyncraticInsightResponse`
- `SyncraticContextWindowsResponse`
- `SyncraticContextWindowResponse`
- `SyncraticMaterializationResponse`

Filter/request types:

- `SyncraticInsightFilter`
- `SyncraticContextWindowFilter`
- `SyncraticMaterializeRequest`

### 2. API Client

Add client methods following the existing `requestJson<T>` / typed envelope pattern:

```ts
listSyncraticInsights(filter?: SyncraticInsightFilter): Promise<SyncraticInsightsResponse>
getSyncraticInsight(insightId: string): Promise<SyncraticInsightResponse>
listSyncraticContextWindows(filter?: SyncraticContextWindowFilter): Promise<SyncraticContextWindowsResponse>
getSyncraticContextWindow(contextWindowId: string): Promise<SyncraticContextWindowResponse>
materializeSyncraticContexts(request: SyncraticMaterializeRequest): Promise<SyncraticMaterializationResponse>
```

List query params should include only defined values. Default list limits should be `50` unless the caller passes another value.

Do not add a new API base URL. Do not call `https://portal.syncratic.co` or `docs/syncratic_user_api_v1.yaml` routes from this frontend task.

### 3. React Query Hooks

Add query keys and hooks near existing MarketOps hooks:

```ts
syncraticInsights(filter)
syncraticInsight(insightId)
syncraticContextWindows(filter)
syncraticContextWindow(contextWindowId)
```

Hooks:

```ts
useSyncraticInsights(filter)
useSyncraticInsight(insightId)
useSyncraticContextWindows(filter)
useSyncraticContextWindow(contextWindowId)
useMaterializeSyncraticContexts()
```

Only enable detail hooks when the id is truthy.

After materialization, invalidate:

- Syncratic insight list/detail queries;
- Syncratic context-window list/detail queries.

Do not invalidate alert lifecycle, graph proposal decision, back-test, calibration, promotion, or production signal queries unless the existing page already depends on them and refetching is necessary for display consistency.

## UI Placement

Recommended route: `/marketops/syncratic`.

Add a MarketOps navigation item labeled `Syncratic Insights` if the app has a MarketOps-local nav. If adding a new top-level route is too large for the current frontend structure, add a tab/section inside the existing MarketOps shell, but do not bury this under Alerts.

The first screen should be the usable insights workspace, not a landing page.

## UI Requirements

### Layout

Build a dense, operational layout suitable for repeated use:

- Left or top filter bar: status, subject symbol, strategy/type, limit.
- Main insight list/table: synthesized insights.
- Detail panel: selected insight and its context window.
- Evidence sections: supporting alerts, signals, events, artifacts, graph proposals, labels.

Do not clone the Alerts UI. The list should emphasize pattern/window/evidence counts, not incident response.

### Insight List

Each row/card should show:

- subject symbol;
- title;
- status;
- severity;
- confidence;
- insight type;
- context window id shortened;
- supporting alert count;
- supporting signal count;
- related graph proposal count;
- updated timestamp.

Use compact severity/status badges consistent with existing UI patterns.

### Insight Detail

When an insight is selected, show:

- full title, summary, and deterministic explanation;
- insight id and type;
- subject symbol/type/id;
- status, severity, confidence;
- builder version;
- metrics JSON block;
- recommendation JSON block;
- supporting record ids grouped by type.

Fetch the context window detail by `context_window_id` and render it in the same detail panel.

### Context Window Detail

Show:

- context strategy;
- window start/end;
- context builder version;
- evidence digest;
- idempotency key;
- signal types;
- detector ids;
- summary metrics;
- evidence reference counts.

Render evidence id lists with copyable monospace ids or compact chips. If existing components support links to Signals, Alerts, Artifacts, or Graph Proposals, link to those views. If not, show ids read-only and do not create new routing work in this gate.

### Materialization Action

Optional but allowed if implemented narrowly.

Add an operator-controlled action such as `Materialize Contexts` with a compact form:

- universe group default `top50_megacap`;
- strategy default `symbol_signal_cluster_5d`;
- window start/end;
- min evidence count default `2`;
- max assets default `50`;
- max context windows default `10`;
- max insights default `10`.

Required safety behavior:

- no automatic execution on page load;
- show the expected caps before submit;
- disable while pending;
- after success, show scan/materialization counters;
- highlight `skipped_below_threshold`, `skipped_unchanged`, and `skipped_budget_cap` as normal outcomes, not errors;
- invalidate Syncratic queries after success.

Do not allow free-form large caps without guardrails. Do not add scheduling controls.

## Empty, Loading, And Error States

Empty insights state:

- Show a compact empty state that says no Syncratic insights exist for the selected filters.
- If materialization action is implemented, provide the bounded materialization control.

Loading state:

- Use existing table/list skeleton or subdued loading row patterns.

Error state:

- Show API errors through the existing error component pattern.
- For `401`, rely on existing auth behavior.
- For materialization failures, keep the operator on the page and preserve form values.

## Non-Goals

Do not implement:

- external Syncratic user-facade Search/Ask calls;
- external ingestion through `/api/v1/ingest/files`;
- privacy token reveal;
- LLM-generated narratives;
- graph writes or graph visualization;
- alert lifecycle mutation;
- Syncratic insight review/dismiss/archive mutation controls, because G088 backend does not expose these mutation routes yet;
- automatic materialization on interval or page load;
- migration/suppression of legacy one-signal `insight_ledger` rows.

## Tests

Add frontend tests consistent with the existing app test stack.

Required API client tests:

- list insights builds `/v1/syncratic/insights` with expected query params;
- get insight URL-encodes ids;
- list context windows builds `/v1/syncratic/context-windows` with expected query params;
- get context window URL-encodes ids;
- materialize posts to `/v1/syncratic/materialize` with the request body;
- response envelopes are parsed correctly;
- bearer-token attachment remains covered by existing central API client tests.

Required query/helper tests:

- query keys are stable for filters;
- detail hooks are disabled without ids;
- materialization mutation invalidates Syncratic insight and context-window queries;
- evidence count helpers handle missing arrays and unknown JSON fields.

Required UI/pure helper tests:

- insight rows distinguish alert count versus signal count;
- context window summary renders digest, builder version, strategy, and window;
- materialization counters classify below-threshold and unchanged skips as non-error outcomes;
- status/severity display helpers handle unknown values defensively.

If the project still lacks a route-render harness, keep route rendering validation manual and cover API/client/helpers in unit tests, following the G079/G086 precedent.

## Manual Validation

After frontend-agent implementation:

1. Sign in to the app with an operator account.
2. Open `/marketops/syncratic` or the chosen MarketOps Syncratic section.
3. Confirm the page requests `/v1/syncratic/insights` with bearer auth.
4. Confirm existing synthesized insight rows render as pattern-level explanations, not alert duplicates.
5. Select an insight and confirm `/v1/syncratic/context-windows/{context_window_id}` is fetched.
6. Confirm context window evidence references, digest, idempotency key, builder version, and summary metrics render.
7. If materialization is implemented, run a bounded materialization and confirm counters render, including skipped quiet assets and unchanged digest skips.
8. Confirm the page does not call external Syncratic `/api/v1/*` routes.

## Acceptance Criteria

The frontend task is complete when:

- operators can list Syncratic synthesized insights;
- selecting an insight shows its context window and evidence references;
- UI copy and layout distinguish Syncratic insights from event-level alerts;
- materialization, if implemented, is bounded, manual, and shows scan/skip counters;
- no external Syncratic user-facade calls are added;
- tests cover API client methods, query hooks/keys, and core render/helper behavior;
- production build passes;
- local authenticated browser/API validation is documented in `docs/build_journal.md` and `docs/gate_audit.md` after implementation.
