# Syncratic Ask Readiness Checklist

Status: production-readiness control.

Last validated: 2026-08-23 UTC.

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
| Browser smoke | `scripts/run_syncratic_ask_ui_smoke.sh` logs in, opens `/marketops/syncratic`, selects a daily narrative, runs normal Ask, and validates the Ask response. | Passing: latest rerun `1 passed in 4.34s`. |
| Failure artifacts | HAR, Playwright trace, and screenshot are retained only on failure and remain protected outside Git. | Implemented by smoke script. |
| Sanitized errors | Browser-visible errors do not expose raw prompts, tokens, secrets, API keys, or upstream response bodies. | Required release invariant. |
| Admin operational visibility | Admin Operations Health exposes a read-only `Syncratic Ask` freshness row with latest success/failure health and context detail. | Deployed and browser-validated on 2026-08-23. |
| Worker data boundary | The background Syncratic worker claims durable jobs from the dedicated MarketOps repository when the MarketOps data boundary is configured. | Source-fixed on 2026-08-23 after queued jobs were found in the dedicated DB while the worker was polling the shared DB. |
| Automatic Ask scope | Automatic worker Ask enrichment is limited to daily narrative contexts: Daily Overview, SRI, Risk/Reward, and Review Queue. Legacy per-asset contexts remain operator-triggered to avoid uncontrolled AI Gateway usage. | Source-fixed on 2026-08-23. |
| Job lifecycle closure | Stale-digest and operator-triggered-only jobs complete with nullable insight/query refs, and completion failures are recorded as `job_completion_failed` instead of leaving rows leased indefinitely. | Source-fixed on 2026-08-23 after stale-digest jobs exposed an empty-string FK completion failure. |

## Job completion lifecycle correction

The readiness rerun exposed a second lifecycle issue: stale-digest daily jobs correctly skipped duplicate Ask calls, but completion passed empty insight/query identifiers. PostgreSQL treated the empty insight identifier as a real foreign-key value, so the completion update failed and the worker ignored that error. The source now stores empty completion identifiers as `NULL`, clears stale error fields on successful completion, and records a visible `job_completion_failed` status if a completion update ever fails again.

Production reconciliation after deployment closed the pre-existing queue backlog without provider polling or AI Gateway calls: 300 legacy per-asset automatic Ask jobs were marked completed with `auto_ask_scope_retired`, and 4 stale-digest daily jobs were marked completed with `stale_evidence_digest_superseded`. The live queue then contained no queued/running tenant-local Syncratic jobs.

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

Production deployment validation passed on 2026-08-23: Gateway deployed through the constrained deployment agent, public `/readyz` returned `200`, and `scripts/run_subscription_admin_ui_smoke.sh` passed with the `Syncratic Ask` Operations Health row included.

## 2026-08-23 readiness rerun

Current validation found that the live browser Ask path is healthy, but the background worker was not draining dedicated MarketOps `syncratic_intelligence_jobs` because it was started with the shared SignalOps repository. The gateway source now starts the worker with the dedicated MarketOps repository when configured.

The same fix constrains automatic worker Ask calls to daily narrative contexts only. This prevents a repository-routing fix from causing the worker to call the AI Gateway for hundreds of legacy per-asset queued contexts. Per-asset Ask remains available as an explicit operator/browser action.

Validation evidence:

- `go test ./cmd/gateway ./internal/api ./internal/syncratic/userapi`
- `npm --prefix web test -- --run src/api/syncratic.test.ts src/lib/syncratic.test.ts`
- `./scripts/run_syncratic_ask_ui_smoke.sh` returned `1 passed in 4.34s`
- Latest completed Ask prompt metadata showed `prompt_bytes=4396`, below the local 10k-byte guard used to stay within the governed 4k-input-token AI Gateway policy.

## Daily-narrative claim boundary

The background worker claim query is scoped to `subject_symbol=MARKETOPS` and the four daily narrative strategies. This is the authoritative cost-control boundary. The in-process worker policy also refuses automatic Ask for non-daily contexts as defense in depth.
