# Syncratic Ask Quality UI Specification

Status: ready for frontend-agent implementation
Gate: G090 frontend follow-up
Author: Codex
Date: 2026-07-14
Backend baseline: `8013f30 Enforce Syncratic context evidence purity`

## Purpose

Update the MarketOps Syncratic UI so operators can understand and safely use the new Syncratic Ask enrichment behavior.

This is a frontend-only handoff over existing same-origin `/v1/syncratic/*` APIs. Do not call the external Syncratic user API directly. Do not add ingestion, graph writes, detector threshold changes, policy deployment, scheduled Ask jobs, or automatic Ask generation across the asset universe.

## Product Semantics

The UI must distinguish three states:

- Deterministic context: SignalOps-built context window and deterministic synthesized insight from persisted ledgers.
- Ask-enriched explanation: Syncratic Ask has generated an explanation over the bounded context window.
- Data-quality blocked explanation: Ask or context materialization found that evidence does not support the context subject, so the operator should treat the result as an evidence-quality issue, not a market insight.

Alerts remain event-level operational records. Syncratic insights are multi-record explanations over evidence windows.

## Backend Contract

Use the existing authenticated API client and tenant hook. Keep `tenant_id`, `app_id=marketops`, `domain=market_data`, and `use_case=daily_market_surveillance` filters consistent with the existing G089 Syncratic UI.

### Existing Insight Fields

`GET /v1/syncratic/insights` and `GET /v1/syncratic/insights/{syncratic_insight_id}` already return:

- `syncratic_insight_id`
- `context_window_id`
- `subject_symbol`
- `title`
- `summary`
- `explanation`
- `supporting_signal_ids`
- `supporting_event_ids`
- `supporting_artifact_ids`
- `related_graph_proposal_ids`
- `related_label_ids`
- `metrics`
- `recommendation`
- `builder_version`
- timestamps

Ask metadata is under:

```json
{
  "metrics": {
    "syncratic_ask": {
      "enabled": true,
      "ask_query_id": "ask-...",
      "ask_status": "completed",
      "prompt_builder_version": "marketops.syncratic.ask_prompt.v1",
      "prompt_digest": "sha256:...",
      "context_window_id": "synctx_...",
      "context_evidence_digest": "...",
      "request_scope": "tenant",
      "request_k": 1,
      "direct_reasoning": true,
      "graph_enabled": false,
      "kee_enabled": false,
      "prompt_bytes": 9709,
      "caps": {},
      "response": {
        "confidence": 0,
        "evidence_count": 0,
        "citation_count": 0
      },
      "latency_ms": 1234
    }
  }
}
```

### Ask Route

```http
POST /v1/syncratic/context-windows/{context_window_id}/ask
Content-Type: application/json
```

Request:

```json
{
  "tenant_id": "tenant-local",
  "max_prompt_bytes": 12000,
  "force": false
}
```

Response:

```json
{
  "ask_result": {
    "context_window_id": "synctx_...",
    "syncratic_insight_id": "synins_...",
    "ask_query_id": "ask-...",
    "ask_status": "completed",
    "prompt_digest": "sha256:...",
    "updated": true,
    "skipped_reason": "",
    "prompt_builder_version": "marketops.syncratic.ask_prompt.v1"
  },
  "syncratic_insight": {}
}
```

Skip response:

```json
{
  "ask_result": {
    "ask_status": "skipped",
    "updated": false,
    "skipped_reason": "unchanged_prompt_and_evidence"
  },
  "syncratic_insight": {}
}
```

Expected errors (verified against `internal/api/router.go` Ask handler):

- `401 unauthorized`: auth expired or missing.
- `404 context_window_not_found`: the context window id does not exist.
- `400 syncratic_ask_invalid`: request/prompt validation failed (e.g. `max_prompt_bytes` out of range, tenant mismatch).
- `502 syncratic_ask_failed`: upstream Syncratic Ask failed; the backend sanitizes the upstream detail to a fixed message and never returns the upstream body.
- `500 syncratic_ask_failed`: other internal enrichment failure (same sanitized code/message).
- `503 syncratic_ask_unavailable`: the Syncratic Ask client is not configured on this gateway.

Note: `400 empty_context_window` is **not** emitted by the Ask route — the Ask route operates on an already-persisted context window. That code is emitted by the context-window create/materialize purity filter (`POST /v1/syncratic/context-windows`) when no pure supporting evidence matches the subject. The UI still handles it defensively under the Ask action (see below) because the operator-facing guidance is identical and the code can surface from the materialize flow operators use alongside Ask.

## Required UI Changes

### Insight Badges

Add compact badges or status chips in the Syncratic insights list/detail:

- `Deterministic` when `metrics.syncratic_ask` is absent.
- `Ask completed` when `metrics.syncratic_ask.ask_status === "completed"`.
- `Ask skipped` when the latest route response has `ask_status === "skipped"` or the detail metadata indicates unchanged evidence.
- `Data Quality Warning` when the explanation/title/summary contains a data-quality subject mismatch warning.

Recommended detection for data-quality warning:

- case-insensitive match for `data quality warning`;
- or case-insensitive match for `subject mismatch`;
- or explanation states evidence `does not support` the context subject.

Do not infer this from subject symbol alone.

### Detail Panel

In the selected insight detail:

- Keep the current deterministic context metadata and supporting evidence references.
- Show generated explanation content under a labeled section such as `Ask Explanation` when Ask metadata exists.
- Show deterministic summary separately so operators know which text came from SignalOps versus Syncratic Ask.
- Show a prominent warning state when data-quality blocked language is present.
- Show compact Ask metadata:
  - `ask_status`
  - `updated` or last route result if available
  - `skipped_reason`
  - `direct_reasoning`
  - `graph_enabled`
  - `kee_enabled`
  - `prompt_bytes`
  - `prompt_builder_version`
  - `latency_ms`

Never render full prompt text, bearer tokens, API keys, raw Syncratic upstream bodies, or raw provider payloads.

### Ask Action

Add an operator-triggered action if it is not already implemented:

- Button label: `Ask Syncratic`
- Secondary action, if needed: `Regenerate`
- Default request uses `force=false`.
- Use `force=true` only for explicit regenerate action.
- Disable while request is pending.
- On success:
  - update the selected insight detail from the response;
  - invalidate/refetch Syncratic insight and context-window queries;
  - show skipped state if `updated=false`.
- On `401`, rely on existing auth error behavior.
- On `404 context_window_not_found`, show a sanitized not-found message.
- On `400 syncratic_ask_invalid`, show a validation-failed message.
- On `502`/`500 syncratic_ask_failed`, show a sanitized failure message. Do not display upstream response bodies.
- On `503 syncratic_ask_unavailable`, show that Ask is not configured on this gateway.
- On `400 empty_context_window` (defensive — see contract note above; surfaces from the materialize/create flow, not the Ask route itself), show copy equivalent to: `No pure supporting evidence exists for this context subject. Review signal/entity mapping or rematerialize after evidence is corrected.`
- Network failures (`network_error`) show a gateway-unreachable message.

Do not add automatic Ask execution on page load, list load, materialization completion, or asset iteration.

### Materialization Feedback

Where the UI exposes context materialization:

- Show `skipped_below_threshold`, `skipped_unchanged`, and `skipped_budget_cap`.
- Add help text or tooltip for purity filtering:
  - `Subject-scoped contexts require evidence that matches the selected ticker. Evidence mentioning another known ticker is excluded.`
- If materialization returns zero contexts, avoid treating it as a system error. Show it as a no-eligible-evidence outcome unless the API response itself is an error.

## Implementation Boundaries

In scope:

- TypeScript types for Ask result and Ask metadata.
- API client method for `POST /v1/syncratic/context-windows/{id}/ask`.
- UI badges and detail metadata rendering.
- Ask/regenerate action using existing auth and React Query patterns.
- Tests for rendering, request body, skip handling, error handling, and cache invalidation.

Out of scope:

- External Syncratic user API calls from the browser.
- Batch Ask generation.
- Scheduler/queue for Ask.
- Syncratic Search.
- Ingestion into Syncratic.
- Graph writes.
- Alert lifecycle mutation.
- Detector or policy deployment.
- New backend fields unless frontend-agent finds an unavoidable API gap.

## Acceptance Criteria

- `/marketops/syncratic` clearly distinguishes deterministic context text from Ask-generated explanation.
- A completed Ask insight shows Ask metadata without raw prompt/secrets.
- A data-quality-blocked insight is visibly marked and does not look like a valid market thesis.
- `Ask Syncratic` sends `force=false`; explicit regenerate sends `force=true`.
- `updated=false` plus `skipped_reason=unchanged_prompt_and_evidence` is rendered as a skip, not a failure.
- `empty_context_window` is rendered as no eligible pure evidence, not as a generic crash.
- Tests cover positive Ask, skipped Ask, data-quality warning, and sanitized failure.

## Validation Checklist

Run:

```bash
cd web
npm test
npm run build
```

Manual/authenticated validation:

1. Open `/marketops/syncratic`.
2. Select an Ask-completed AAPL insight and confirm it shows Ask metadata and a generated market explanation.
3. Select or create a data-quality-blocked insight and confirm it shows `Data Quality Warning`.
4. Click `Ask Syncratic` on an unchanged context and confirm the UI renders skipped unchanged evidence.
5. Confirm browser network calls are same-origin `/v1/syncratic/*` and include the existing bearer token.
6. Confirm no request goes to `portal.syncratic.co`, `/api/v1/ask`, or any external Syncratic URL from browser code.
