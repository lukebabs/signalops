# Syncratic Context Window API

G088 adds deterministic Syncratic context windows and synthesized insights over existing SignalOps and MarketOps ledgers. G090 adds an optional server-side Syncratic Ask enrichment route for one bounded context window at a time.

These APIs do not ingest external data, use Syncratic Search for enrichment, mutate alert lifecycle state, write graph state, deploy policies, or change detector thresholds.

## Materialize Selectively

`POST /v1/syncratic/materialize`

Runs the Top 50 selective materialization flow:

```text
daily Top 50 scan -> candidate counts -> threshold-gated context build -> materialized insight
```

Request fields:

- `tenant_id` required.
- `universe_group` defaults to `top50_megacap`.
- `context_strategy` defaults to `symbol_signal_cluster_5d`.
- `context_builder_version` defaults to `syncratic.context_builder.v1`.
- `window_start` and `window_end` are RFC3339 timestamps.
- `min_evidence_count` defaults to `2`.
- `max_assets`, `max_candidate_windows`, `max_context_windows`, and `max_insights` cap work per request.

The response includes scan/materialization counters:

- `scanned_assets`
- `candidate_windows`
- `materialized_context_windows`
- `materialized_insights`
- `skipped_below_threshold`
- `skipped_unchanged`
- `skipped_budget_cap`

A rerun with the same evidence digest should increment `skipped_unchanged` and create no duplicate rows.

## Context Windows

`POST /v1/syncratic/context-windows`

Creates or refreshes one deterministic context window from existing persisted evidence for a symbol/window.

`GET /v1/syncratic/context-windows`

Lists context windows. Common filters:

- `tenant_id`
- `app_id`
- `domain`
- `use_case`
- `subject_symbol`
- `context_strategy`
- `status`
- `limit`

`GET /v1/syncratic/context-windows/{context_window_id}`

Returns one context window with evidence references, summary metrics, `evidence_digest`, and `idempotency_key`.


## Syncratic Ask Enrichment

`POST /v1/syncratic/context-windows/{context_window_id}/ask`

Calls Syncratic Ask server-side with a compact, bounded prompt built from the deterministic context window, then persists the generated explanation and Ask metadata onto the associated Syncratic insight.

Request fields:

- `tenant_id` optional but must match the context window when provided.
- `prompt_builder_version` defaults to `marketops.syncratic.ask_prompt.v1`.
- `max_prompt_bytes` defaults to `12000` and is capped at `24000`.
- `include_record_details` is accepted for contract stability but the G090 implementation sends IDs and summary metrics only.
- `force` defaults to `false`; unchanged prompt/evidence skips the Ask call.

Response fields:

- `syncratic_insight`: the updated or unchanged Syncratic insight.
- `ask_result.context_window_id`
- `ask_result.syncratic_insight_id`
- `ask_result.ask_query_id`
- `ask_result.ask_status`
- `ask_result.prompt_digest`
- `ask_result.updated`
- `ask_result.skipped_reason`

Idempotency uses the context evidence digest plus prompt digest. With `force=false`, a repeated request for unchanged evidence returns `updated=false` and `skipped_reason=unchanged_prompt_and_evidence` without calling Syncratic Ask again.

Syncratic Ask failures return a sanitized `502 syncratic_ask_failed`; raw upstream response bodies, prompts, bearer tokens, and API keys are not returned.

## Synthesized Insights

`POST /v1/syncratic/insights`

Creates or refreshes a synthesized insight from a persisted context window.

`GET /v1/syncratic/insights`

Lists synthesized insights. Common filters:

- `tenant_id`
- `context_window_id`
- `insight_type`
- `subject_symbol`
- `status`
- `limit`

`GET /v1/syncratic/insights/{syncratic_insight_id}`

Returns one synthesized insight and its supporting context references.

## Auth Boundary

When gateway auth is enabled, all `/v1/syncratic/*` routes require a valid bearer token. Unauthenticated probes should return `401` while `/healthz` remains public.
