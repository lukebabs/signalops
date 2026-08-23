# Syncratic Ask Readiness Checklist

Status: production-readiness control.

Last validated: 2026-08-23.

## Purpose

Syncratic Ask is a subscriber-facing explainability layer over persisted MarketOps evidence. It must answer from governed context windows, not from ad hoc provider calls or unbounded prompt payloads.

This checklist defines the minimum controls required before Syncratic Ask can be treated as ready for controlled pilot use or included in a production release gate.

## Scope

In scope:

- MarketOps daily narrative context windows and insights.
- Analyst Ask over persisted daily narrative evidence.
- AI Gateway request validation, idempotency, and commercial-policy acceptance.
- Browser smoke validation for `/marketops/syncratic`.
- Failure handling and artifact retention.

Out of scope:

- Trading advice or portfolio execution.
- Provider polling.
- Graph mutation.
- Signal lifecycle promotion.
- Reconstructing historical Syncratic narratives beyond explicitly materialized MarketOps evidence.

## Readiness controls

| Control | Required state | Current status |
| --- | --- | --- |
| Route availability | Public `/readyz` returns `200`; authenticated `/marketops/syncratic` renders without `404`. | Passing in latest smoke. |
| MarketOps database boundary | Syncratic MarketOps routes use the dedicated MarketOps query repository when the MarketOps boundary is configured. | Implemented and validated. |
| Context availability | Latest completed-session daily narratives have active `syncratic_context_windows` and `syncratic_insights`. | Active for tenant-local 2026-08-21 materialization. |
| Prompt governance | Prompt construction uses compact section summaries, artifact totals, and capped citation samples instead of full lineage expansion. | Implemented. |
| Gateway policy alignment | SignalOps keeps the request below the governed AI Gateway budget of 4k input tokens and 1k output tokens. The local proxy uses a conservative 10k-byte cap as an operational guard, not a policy expansion. | Implemented. |
| Provenance retention | Full evidence lineage remains persisted in MarketOps storage even when the Ask prompt is compacted. | Implemented through context-window lineage refs. |
| Idempotency | Ask calls include `Idempotency-Key`; normal Ask is stable for the same context/prompt digest, while forced regeneration uses a timestamp suffix. | Implemented and smoke-tested. |
| AI Gateway commercial gate | The SignalOps AI Gateway client has an active price-catalog/policy configuration. | Passing after catalog propagation. |
| Browser smoke | `scripts/run_syncratic_ask_ui_smoke.sh` logs in, opens `/marketops/syncratic`, selects a daily narrative, runs normal Ask, and validates the Ask response. | Passing: `1 passed in 1.43s`. |
| Failure artifacts | HAR, Playwright trace, and screenshot are retained only on failure and remain protected outside Git. | Implemented by smoke script. |
| Sanitized errors | Browser-visible errors do not expose raw prompts, tokens, secrets, API keys, or upstream response bodies. | Required release invariant. |
| Admin operational visibility | Admin Operations Health exposes a read-only `Syncratic Ask` freshness row with latest success/failure health and context detail. | Implemented; requires deployment validation. |

## Required validation command

Run this after Syncratic-related Gateway deployments, AI Gateway policy/catalog changes, and before subscription production gates:

```bash
./scripts/run_syncratic_ask_ui_smoke.sh
```

Expected successful result:

```text
1 passed
```

Focused local regression checks, when code changes touch Syncratic request construction or UI behavior:

```bash
go test ./internal/syncratic/userapi ./internal/api
npm --prefix web test -- --run src/api/syncratic.test.ts src/lib/syncratic.test.ts
```

## Failure interpretation

| Symptom | Likely cause | Required response |
| --- | --- | --- |
| `404` on `/marketops/syncratic` | Web route or gateway deployment mismatch. | Rebuild/redeploy the affected container through the constrained deployment path and rerun the smoke. |
| `401` or `403` | QA identity, tenant claim, audience, or SignalOps role mismatch. | Fix identity provisioning; do not bypass tenant/RBAC enforcement. |
| `syncratic_ask_unavailable` | Ask client is not configured for the gateway environment. | Restore AI Gateway endpoint/client configuration. |
| `context_requires_chunking` | Prompt compaction could not fit inside the local policy guard. | Narrow the context window or improve summarization; do not raise the prompt budget as the first fix. |
| `idempotency_key_required` | SignalOps request omitted the AI Gateway idempotency header. | Treat as a code regression. |
| `gateway_price_catalog_not_found` | AI Gateway commercial policy/catalog is missing or has not propagated. | Fix AI Gateway catalog/policy mapping and rerun after propagation. |
| Empty daily narrative list | Daily narrative materialization has not produced active context windows for the latest completed session. | Rerun the approved Syncratic daily narrative materializer without provider polling. |

## Production readiness state

Syncratic Ask is ready for controlled pilot QA when:

1. The required browser smoke passes.
2. The latest completed-session daily narratives are active.
3. AI Gateway policy/catalog is active.
4. Errors are sanitized and actionable.
5. The release journal records the run result and any upstream blocker.

The remaining production-hardening step is deployment validation that Administration renders the `Syncratic Ask` operations-health row in production and that the row reflects the latest completed Ask success or failure category without requiring shell access.
