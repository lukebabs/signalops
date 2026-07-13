# G090 Syncratic Ask Enrichment

Status: implemented — backend/API route and tests complete
Use case: MarketOps Daily Market Surveillance

## Goal

Add the first controlled LLM explanation path for MarketOps Syncratic insights by calling Syncratic Ask with a bounded SignalOps context window.

G090 should prove the integration boundary for one operator/API-triggered context window at a time. It should not introduce automatic batch generation, scheduled Ask jobs, Syncratic Search retrieval, external ingestion, graph writes, alert lifecycle mutation, detector threshold changes, or frontend scope beyond what G089 already renders.

## Product Boundary

G088/G089 created deterministic context windows and deterministic Syncratic insight rows. G090 adds optional LLM synthesis over those deterministic records.

The authority split is:

- SignalOps owns facts: persisted events, signals, alerts, artifacts, graph proposals, labels, back-test records, context windows, evidence digests, and durable insight rows.
- Syncratic Ask owns language synthesis: a generated explanation and optional cited reasoning over the bounded prompt SignalOps sends.
- Operators see both together: deterministic evidence remains inspectable; the Ask-generated explanation is marked as generated synthesis, not detector output.

Search is not part of this gate. The previous Search call was only an auth/connectivity probe for the Syncratic facade.

## Current Inputs

G090 should start from an existing `syncratic_context_windows` row created by G088 materialization.

Required context-window inputs:

- `context_window_id`
- `tenant_id`
- `app_id`
- `domain`
- `use_case`
- `subject_symbol`
- `subject_type`
- `subject_id`
- `window_start`
- `window_end`
- `context_strategy`
- `context_builder_version`
- `signal_types`
- `detector_ids`
- evidence id lists: events, signals, alerts, artifacts, graph proposals, labels
- `baseline_refs`
- `evaluation_refs`
- `promotion_candidate_refs`
- `summary_metrics`
- `evidence_digest`

G090 may fetch bounded details for referenced records when already available through the repository layer, but it must cap counts and serialized size. It must never send raw secrets, bearer tokens, API keys, full raw provider payloads, or privacy-token reveal outputs to Syncratic Ask.

## Prompt Contract

Create a versioned prompt builder, initially:

`marketops.syncratic.ask_prompt.v1`

The prompt should be compact, structured, and reproducible. Recommended shape:

```text
You are assisting a MarketOps surveillance operator.
Explain the following deterministic SignalOps context window.
Use only the facts provided. Do not invent prices, events, filings, news, or recommendations.
Distinguish deterministic evidence from generated interpretation.
Return a concise explanation, operational relevance, and uncertainty notes.

Context metadata:
- tenant/app/domain/use_case
- subject symbol/type/id
- window start/end
- context strategy and builder version
- evidence digest

Evidence summary:
- signal types and detector ids
- counts by evidence type
- severity/confidence aggregates
- key summary metrics
- baseline/evaluation/promotion refs when present

Evidence ids:
- alerts: capped list
- signals: capped list
- events/artifacts/graph proposals/labels: capped lists

Output request:
- title
- summary
- explanation
- recommendation object with one of: observe, review, escalate, no_action
- uncertainty_notes
- cited_evidence_ids
```

The prompt builder must record:

- prompt builder version;
- context window id;
- context evidence digest;
- prompt digest;
- request size metrics;
- caps applied;
- whether record details beyond IDs were included.

## Ask Request

Use `internal/syncratic/userapi.Client.Ask` and `SYNCRATIC_AUTH_MODE=api_key` unless the auth boundary changes.

Initial Ask request mapping:

- `question`: the full structured context-window prompt;
- `scope`: `tenant`;
- `k`: `0` or omitted unless Syncratic Ask requires a positive value;
- `filters`: omitted for G090 because the Syncratic facade currently rejects SignalOps-specific filter keys. SignalOps metadata is embedded in the bounded prompt and persisted locally in `metrics.syncratic_ask`.
- `thread_mode`: `off`.
- `include_refs`: `false`.

G090 should not enable graph-assisted or KEE-assisted Syncratic retrieval unless a later gate explicitly approves retrieval influence. The explanation should be based on the supplied context window.

## Persistence Model

Prefer reusing the existing `syncratic_insights` row for the context window.

When Ask succeeds, update the associated Syncratic insight fields:

- `title`: generated or fallback deterministic title;
- `summary`: generated summary;
- `explanation`: generated explanation;
- `confidence`: Ask confidence when returned, otherwise keep deterministic confidence;
- `recommendation`: structured recommendation object with Ask metadata;
- `metrics`: merge deterministic metrics with Ask metadata;
- `builder_version`: either keep deterministic builder version and store Ask builder in metrics, or move to a compound value only if existing consumers remain compatible.

Recommended metadata under `metrics.syncratic_ask`:

```json
{
  "enabled": true,
  "ask_query_id": "...",
  "ask_status": "completed",
  "prompt_builder_version": "marketops.syncratic.ask_prompt.v1",
  "prompt_digest": "sha256:...",
  "context_window_id": "...",
  "context_evidence_digest": "...",
  "request_scope": "marketops_signalops_context_window",
  "request_k": 0,
  "included_record_details": false,
  "caps": {
    "max_alert_ids": 20,
    "max_signal_ids": 20,
    "max_event_ids": 20,
    "max_artifact_ids": 20,
    "max_graph_proposal_ids": 20,
    "max_label_ids": 20,
    "max_prompt_bytes": 12000
  },
  "response": {
    "confidence": 0.0,
    "evidence_count": 0,
    "citation_count": 0
  },
  "started_at": "...",
  "completed_at": "...",
  "latency_ms": 0
}
```

If a dedicated Ask-run ledger is introduced, it should be additive and linked by `syncratic_insight_id` and `context_window_id`. Do not block G090 on a broad final storage schema unless implementation shows the existing `metrics`/`recommendation` fields are insufficient.

## API Shape

Add one narrow backend route:

`POST /v1/syncratic/context-windows/{context_window_id}/ask`

Request body:

```json
{
  "tenant_id": "tenant-local",
  "prompt_builder_version": "marketops.syncratic.ask_prompt.v1",
  "max_prompt_bytes": 12000,
  "include_record_details": false,
  "force": false
}
```

Response body:

```json
{
  "syncratic_insight": {},
  "ask_result": {
    "context_window_id": "...",
    "syncratic_insight_id": "...",
    "ask_query_id": "...",
    "ask_status": "completed",
    "prompt_digest": "sha256:...",
    "updated": true,
    "skipped_reason": ""
  }
}
```

Behavior:

- Require gateway auth when enabled, consistent with existing `/v1/syncratic/*` routes.
- Enforce tenant scoping from request/token conventions already used by SignalOps.
- Return `404` when the context window does not exist.
- Return `409` or a successful skipped result when Ask metadata already exists for the same context evidence digest and prompt digest and `force=false`.
- Return `502` for Syncratic facade failures after sanitizing response bodies.
- Never return raw prompt text, API keys, bearer tokens, or long raw Ask payloads.

## Idempotency And Re-Run Rules

G090 must be idempotent for unchanged evidence.

A rerun with the same context window evidence digest, prompt builder version, caps, and prompt digest should not call Syncratic Ask again unless `force=true`.

`force=true` should:

- call Ask again;
- preserve deterministic evidence references;
- update Ask metadata with the new query id/timestamps;
- keep enough previous metadata in `metrics.syncratic_ask.previous` only if compact and useful; otherwise rely on git/journal/API audit for G090 MVP.

## Failure Handling

Failure to call Ask must not corrupt deterministic context windows or remove deterministic insight content.

Failure outcomes:

- context window missing: no writes;
- Syncratic config missing: no writes, actionable error;
- Syncratic auth failure: no writes, sanitized error;
- Ask timeout/network failure: no writes or write compact failure metadata only if an Ask-run ledger exists;
- malformed Ask response: no overwrite of existing explanation unless a usable answer exists.

For G090 MVP, do not add retry queues or automatic retries. Operator/API caller can retry manually.

## Security And Privacy

Required guardrails:

- Do not log prompt text at info level.
- Do not log secrets, bearer tokens, API keys, raw Syncratic responses, or full raw evidence payloads.
- Cap prompt byte size and evidence-id counts.
- Redact or omit raw provider payloads.
- Keep Syncratic Ask calls server-side only; frontend calls SignalOps `/v1/syncratic/*` routes.
- Keep generated explanation visibly tied to deterministic evidence IDs and context evidence digest.

## Frontend Scope

No frontend-agent work is required for the first G090 backend slice.

G089 already renders:

- `title`
- `summary`
- `explanation`
- `metrics`
- `recommendation`
- selected context-window detail

After backend implementation, the existing `/marketops/syncratic` page should display the generated explanation if the selected insight row is updated.

A later frontend gate may add an explicit `Ask Syncratic` button or generated-vs-deterministic badges. Do not include that in G090 unless backend validation shows operators cannot trigger the route safely through API tooling.

## Test Plan

Unit tests:

- prompt builder includes required context metadata and evidence IDs;
- prompt builder caps evidence IDs and prompt bytes;
- prompt digest is stable for unchanged inputs;
- Ask route rejects missing context windows;
- Ask route skips unchanged prompt/evidence when `force=false`;
- Ask route calls `userapi.Ask` when needed and persists explanation/metadata;
- Ask route sanitizes upstream errors;
- existing Syncratic list/detail/materialize routes still pass;
- `internal/syncratic/userapi` Ask request path and API-key auth remain covered.

Validation commands:

- `go test ./internal/syncratic/userapi ./internal/api ./internal/storage/...`
- `go test ./...`
- `git diff --check`

Live smoke, only after unit tests pass:

1. Ensure `SYNCRATIC_AUTH_MODE=api_key`, `SYNCRATIC_API_BASE_URL`, and `SYNCRATIC_CLIENT_SECRET` are configured in ignored env.
2. Select one existing `syncratic_context_windows` row with a linked `syncratic_insights` row.
3. `POST /v1/syncratic/context-windows/{context_window_id}/ask` with a valid operator bearer token.
4. Confirm response includes an updated Syncratic insight.
5. Confirm `/v1/syncratic/insights/{syncratic_insight_id}` returns the generated explanation and Ask metadata.
6. Confirm rerun with `force=false` skips unchanged prompt/evidence without a second Ask call.
7. Confirm `/marketops/syncratic` renders the generated explanation through the existing UI.

## Acceptance Criteria

G090 is accepted when:

- a bounded context window can be sent to Syncratic Ask through a server-side SignalOps route;
- the Ask-generated explanation is persisted against the existing Syncratic insight and linked to deterministic context/evidence digests;
- reruns are idempotent for unchanged evidence unless `force=true`;
- failures do not overwrite deterministic insight content;
- no Syncratic Search enrichment, external ingestion, graph writes, alert lifecycle mutation, frontend direct facade calls, scheduled batch jobs, or automatic universe-wide Ask generation are introduced;
- docs, tests, and live smoke results are recorded in the build journal and gate audit.

## Follow-On Gates

Potential follow-ons after G090:

- G091: operator UI control for Ask generation/re-generation, with generated-vs-deterministic badges.
- G092: budgeted selective Ask materialization queue with per-day caps and freshness rules.
- G093: Ask quality review labels and feedback capture for supervised evaluation.
- G094: optional retrieval policy if Syncratic gains a useful MarketOps corpus and Search becomes relevant.


## Implementation Notes

Implemented in G090:

- `POST /v1/syncratic/context-windows/{context_window_id}/ask`;
- versioned prompt builder `marketops.syncratic.ask_prompt.v1`;
- server-side `internal/syncratic/userapi.Client.Ask` integration;
- idempotent skip for unchanged context evidence digest and prompt digest when `force=false`;
- persistence of generated explanation, recommendation, and `metrics.syncratic_ask` metadata onto `syncratic_insights`;
- sanitized upstream Ask error handling;
- gateway client construction from `SYNCRATIC_*` env only when `SYNCRATIC_API_BASE_URL` is configured;
- live-compatible Ask client behavior: API-key mode sends both `Authorization` and `X-API-Key`, Ask timeout is 60 seconds, `confidence` accepts numeric strings, and the route sends `scope=tenant`, `k=1`, `thread_mode=off`, `include_refs=false`, and no facade filters.

The implementation still keeps scheduled Ask jobs, Syncratic Search enrichment, external ingestion, graph writes, alert lifecycle mutation, and frontend changes out of scope.

## Prompt-Quality Closeout

Validated on `2026-07-13T05:08:00Z` after Syncratic enabled non-human reasoning clients to use the Ask reasoning layer with the intended prompt quality.

The route prompt prefix now uses the direct-validated non-human `CONTEXT_JSON` framing while preserving the bounded deterministic SignalOps context payload, prompt digest, and evidence digest. Authenticated forced Ask against `synctx_47bccf8af8af03a15d4c0d3f` returned HTTP `200`, `ask_status=completed`, `updated=true`, and persisted a `516` character generated explanation instead of `UNKNOWN`. A rerun with unchanged evidence returned `updated=false` and `skipped_reason=unchanged_prompt_and_evidence`.
