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

Follow-on validation before production use:

- run a dry-run over a bounded Top 50 smoke window and confirm `decisions[]` matches operator expectations;
- run write mode with a low materialization cap and confirm only the selected budgeted contexts persist;
- confirm no Syncratic Ask calls are emitted by materialization.
