# G104 Reviewed Label Workflow Specification

Status: proposed workflow specification.

## Purpose

G103 showed calibration readiness is blocked by reviewed-label volume as well as historical coverage. G104 defines the smallest operator workflow for increasing real reviewed labels from existing MarketOps DSM graph proposal decisions without synthetic labels, threshold relaxation, model training, detector changes, or policy deployment.

## Current State

Latest readiness snapshot: `btready-g103-recheck-20260714185649`.

Relevant G103 metrics:

- Matched labels: `7`.
- Readiness threshold: `100` reviewed labels.
- Conflicting label ratio: `0`.
- Label coverage for the matched evaluation: `1`.
- Readiness status: `needs_more_historical_data`.
- Label-related reason: reviewed label volume or label coverage is below readiness thresholds.

G104 targets the label-volume gap only. Historical coverage remains a separate workstream.

## Existing Building Blocks

Canonical operator decisions:

- `marketops_dsm_graph_proposals` from G079.
- `POST /v1/marketops/dsm/graph-proposals/{proposal_id}/decision` from G080.
- Decision statuses: `accepted`, `rejected`, `superseded`, and restore-to-`proposed`.

Evaluation label substrate:

- `POST /v1/marketops/backtest-evaluation-labels/sync` from G084.
- `GET /v1/marketops/backtest-evaluation-labels` from G084.
- Decision-to-label mapping:
  - `accepted` -> `positive`.
  - `rejected` -> `negative`.
  - `superseded` -> `superseded`.
  - `proposed` -> `unresolved` only when explicitly included.

Evaluation/readiness loop:

- `POST /v1/marketops/backtest-evaluations` from G085.
- `POST /v1/marketops/backtest-calibration-readiness` from G094.

## Label Milestones

Use staged label-volume milestones so operators can inspect quality before pushing to the full readiness threshold:

- Milestone 1: `25` matched reviewed labels.
- Milestone 2: `50` matched reviewed labels.
- Milestone 3: `100` matched reviewed labels.

At each milestone, sync labels, run at least one label-aware evaluation, and create a readiness snapshot. Do not relax readiness thresholds to make an intermediate milestone pass.

## Sampling Rules

Each review batch should be assembled from real persisted graph proposals and should diversify across:

- Subject symbol: prioritize the currently covered symbols first, then expand as new coverage arrives.
- Date/window: avoid reviewing only one run or one market day.
- Signal type: include accumulation, volatility expansion, hedging pressure, speculative call/put pressure, pinning risk, divergence, and any other generated taxonomy types present in proposals.
- Candidate type: include both node candidates and relationship candidates.
- Recommendation type: include `auto_accept_candidate`, `auto_reject_candidate`, `manual_review_required`, and `supersede_candidate` when present.
- Severity/confidence: avoid reviewing only high-confidence/high-severity rows.

A batch should not count duplicate decisions on the same `graph_fact_key` as independent label-quality evidence. Duplicate graph facts can still be useful for consistency review, but milestone counting should be based on distinct graph facts wherever possible.

## Decision Guidance

Accept:

- Use `accepted` when the proposal accurately represents a graph fact supported by the signal/artifact evidence.
- This syncs to a `positive` evaluation label.

Reject:

- Use `rejected` when the proposal is unsupported, misleading, wrong subject, wrong relationship, wrong signal type, or too weak for graph materialization consideration.
- This syncs to a `negative` evaluation label.

Supersede:

- Use `superseded` when the proposal is directionally useful but should be replaced by a better graph fact, newer evidence, cleaner relationship, or corrected entity representation.
- This syncs to a `superseded` evaluation label and is not counted as an automatic true/false scoring outcome.

Restore:

- Use restore-to-`proposed` only to undo an erroneous review decision before the proposal is treated as reviewed evidence.
- Restored proposals should not be counted toward reviewed-label milestones unless later reviewed again.

## Quality Gates

Before treating a milestone as useful calibration evidence, confirm:

- The label sync is idempotent and does not inflate counts on repeated syncs.
- `conflicting_label_ratio` remains at or below `0.05`.
- There is some negative-label coverage; a label set with only positives is useful but weak for calibration quality.
- Superseded labels are tracked but not treated as automatic true/false outcomes.
- Labels are traceable to source proposal ids, artifact ids, signal ids, subject symbols, candidate types, decision statuses, and reviewed actors.
- The review set covers more than one symbol and more than one run as soon as enough proposals exist.

## Operator Workflow

1. Select a review batch from proposed DSM graph proposals using bounded filters: tenant, app/domain/use case, subject symbol, status, signal type, candidate type, and limit.
2. Inspect the supporting signal, artifact evidence, semantic evidence, and graph target fields.
3. Decide each proposal as `accepted`, `rejected`, `superseded`, or leave it `proposed` when there is not enough information.
4. Add a short decision note for rejected or superseded proposals when the reason is not obvious from the evidence.
5. Sync labels with `POST /v1/marketops/backtest-evaluation-labels/sync` using reviewed statuses only.
6. List labels and confirm matched-label count, distinct graph fact count, decision status distribution, and conflict ratio.
7. Run a G085 label-aware evaluation for a relevant back-test run.
8. Run a G094 readiness snapshot and record whether label-related blockers changed.
9. Repeat until the next milestone is reached.

## Frontend-Agent Scope Check

No frontend-agent work is required if the current UI can already:

- list DSM graph proposals with enough filters for batch selection;
- show supporting signal/artifact evidence;
- submit accept/reject/supersede/restore decisions;
- show evaluation label counts or allow operators to verify them through the existing back-test/evaluation views.

Write a frontend-agent specification only if one of those capabilities is missing. The frontend scope must stay limited to operator review ergonomics and label-count visibility. It must not add graph writeback, policy deployment, detector threshold editing, model training, or automatic bulk acceptance.

## Acceptance Criteria

G104 is ready for execution when operators have:

- a documented label milestone target;
- a bounded sampling rule for the next review batch;
- clear decision semantics for accept/reject/supersede/restore;
- a sync/evaluation/readiness loop using existing APIs;
- quality checks for conflict ratio, duplicates, and label distribution;
- a decision on whether frontend-agent work is actually required.

## Non-Goals

- Synthetic labels.
- Threshold relaxation.
- Runtime policy deployment.
- Detector mutation.
- Graph database materialization.
- Model training or supervised-learning pipeline implementation.
- Broad ingestion or historical campaign expansion.
