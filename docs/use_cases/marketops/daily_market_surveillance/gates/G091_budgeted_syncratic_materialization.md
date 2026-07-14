# G091 Budgeted Syncratic Context Materialization

Status: implemented - backend/API dry-run selection and decision audit

## Objective

Make Syncratic context materialization explicitly budgeted, previewable, and auditable before operators create context windows or synthesized insight rows.

G091 closes the operational gap left after G088-G090: SignalOps can scan the MarketOps Top 50 universe, but it must not create excessive Syncratic context or Ask work for assets that will never be reviewed.

## Scope

G091 extends the existing same-origin `POST /v1/syncratic/materialize` API. It does not add a scheduler, background worker, new storage table, frontend scope, Syncratic Search, Syncratic ingestion, graph writes, detector changes, policy deployment, or automatic Ask generation.

## Selection Model

The materializer evaluates each active asset in the requested universe and records one decision per scanned asset. The decision explains whether the asset:

- would materialize in dry-run mode;
- materialized in write mode;
- skipped because evidence is below threshold;
- skipped because the context evidence digest is unchanged;
- skipped because the candidate-window budget is exhausted;
- skipped because the context/insight materialization budget is exhausted.

Default selection remains conservative:

- `universe_group`: `top50_megacap`
- `context_strategy`: `symbol_signal_cluster_5d`
- `context_builder_version`: `syncratic.context_builder.v1`
- `min_evidence_count`: `2`
- candidate budgets and materialization budgets are enforced per request.

A context candidate is eligible when it has enough pure signal/alert evidence, a critical alert, or related graph/label evidence. Existing G090 evidence-purity checks still apply through the context builder.

## API Contract

`POST /v1/syncratic/materialize` accepts the existing request fields plus:

- `dry_run`: when `true`, return a preview with no context-window or insight writes.

Dry-run requests return `200 OK`; write-mode requests return `201 Created`.

The response keeps the existing aggregate counters and adds:

- `dry_run`
- `decisions[]`

Each decision includes:

- `subject_symbol`
- `action`: `would_materialize`, `materialized`, or `skipped`
- `reason`: `eligible`, `below_threshold`, `unchanged_evidence_digest`, `candidate_budget_cap`, or `materialization_budget_cap`
- `evidence_count`
- `signal_count`
- `alert_count`
- `artifact_count`
- `graph_proposal_count`
- `label_count`
- `critical_alert`
- `related_evidence`
- `evidence_digest` when a context candidate was built
- `context_window_id` when a context candidate was built

## Acceptance Criteria

G091 is accepted when:

- a dry-run materialization request returns per-asset decisions and does not persist context windows or insights;
- dry-run applies the same candidate and materialization caps that write mode would apply;
- write mode still persists eligible context windows and synthesized insights;
- below-threshold, unchanged-digest, and budget-cap skips are visible in aggregate counters and `decisions[]`;
- G090 Ask remains operator-triggered and is not invoked by materialization.

## Validation

Implemented validation:

- `docker run --rm -v ... golang:1.22-bookworm go test ./internal/api -count=1`: passed.
- `docker run --rm -v ... golang:1.22-bookworm go test ./... -count=1`: passed.

Live validation:

- Gateway was rebuilt/restarted with the G091 route changes.
- Authenticated dry-run over a bounded Top 50 smoke window returned `200 OK`, scanned `10` assets, returned `10` decisions, selected AAPL as `would_materialize`, reported `materialized_context_windows=0`, and reported `materialized_insights=0`.
- Authenticated write mode with `max_context_windows=1` and `max_insights=1` returned `201 Created`, scanned `10` assets, materialized exactly one AAPL context (`synctx_9f96168debca2528ce72efe5`) and one deterministic insight (`synins_467aef31771fd45262d48de8`), and skipped `9` assets below threshold.
- The materialized insight did not include `metrics.syncratic_ask`, confirming materialization did not trigger Syncratic Ask.
- Authenticated rerun of the same write request returned `201 Created`, reported `skipped_unchanged=1`, `materialized_context_windows=0`, and `materialized_insights=0`; the AAPL decision was `skipped` with `reason=unchanged_evidence_digest`.
