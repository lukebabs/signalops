# G083 Back-Test Baselines And Evaluation

Status: specification proposed
Use case: MarketOps Daily Market Surveillance

## Goal

Add the next MarketOps back-test layer: named calibration baselines and evaluation metrics that compare persisted calibration summaries against either a prior accepted baseline or human-reviewed graph proposal labels.

G083 should not promote policies, change detector thresholds, train models, or write graph state. Its job is to make back-test results comparable and auditable so operators can decide whether a later policy/promotion gate is justified.

## Starting Point

G081 provides isolated back-test execution and run-scoped generated outputs.

G082 provides persisted calibration summary snapshots over selected back-test runs:

- `marketops_backtest_calibration_summaries`
- `POST /v1/marketops/backtest-calibration-summaries`
- `GET /v1/marketops/backtest-calibration-summaries`
- `GET /v1/marketops/backtest-calibration-summaries/{summary_id}`
- `/marketops/backtests` UI support for viewing and creating stored snapshots

G080 provides operator decisions over production graph proposals. Those decisions are possible label data, but they are not yet formal evaluation labels.

## Problem

A stored calibration summary is useful, but not sufficient for controlled calibration decisions.

Operators still need to know:

- whether a new summary is better or worse than a known baseline;
- whether auto-accept volume increased without increasing known false positives;
- whether manual review burden decreased;
- whether zero-input or low-coverage runs should be excluded from comparison;
- whether G080 operator decisions can be used as evaluation labels for precision/recall style scoring;
- whether a summary has enough data to support a later policy recommendation.

## Scope

Implement or specify a first-class baseline/evaluation substrate with these concepts:

- Named baseline records.
- Baseline-to-summary comparison records.
- Optional label extraction from G080 graph proposal decisions.
- Evaluation metrics stored as immutable snapshots.
- Read-only APIs for list/detail/compare.
- Operator-facing UI surface for baseline comparison, if backend is implemented in this gate.

## Non-Goals

Do not implement:

- detector threshold promotion;
- policy deployment;
- automatic graph materialization;
- ML training or model registry integration;
- PnL/trading strategy back-tests;
- raw provider re-pulls;
- mutation of production signal, artifact, graph proposal, alert, or insight ledgers.

## Baseline Model

A baseline is a named reference calibration summary selected by an operator.

Suggested table: `marketops_backtest_calibration_baselines`.

Fields:

- `baseline_id`
- `tenant_id`
- `app_id`
- `domain`
- `use_case`
- `name`
- `description`
- `summary_id`
- `detector_id`
- `dataset`
- `scope`
- `status`
- `created_by`
- `created_at`
- `updated_at`

Recommended status values:

- `active`
- `archived`

Baseline records should point to immutable calibration summaries. Updating a baseline should create or update the baseline pointer, not mutate the referenced summary.

## Comparison Model

A comparison captures the difference between a candidate calibration summary and a baseline.

Suggested table: `marketops_backtest_calibration_comparisons`.

Fields:

- `comparison_id`
- `tenant_id`
- `baseline_id`
- `baseline_summary_id`
- `candidate_summary_id`
- `detector_id`
- `dataset`
- `comparison_metrics`
- `recommendation`
- `recommendation_reason`
- `created_by`
- `created_at`

Initial recommendation values:

- `needs_more_data`
- `regression_candidate`
- `improvement_candidate`
- `neutral_candidate`
- `manual_review_required`

These are calibration recommendations only. They must not deploy policy changes.

## Initial Comparison Metrics

At minimum, compare:

- run count delta
- zero-input rate delta
- scanned event delta
- signal yield delta
- graph proposals per signal delta
- policy results per signal delta
- auto-accept share delta
- auto-reject share delta
- manual-review share delta
- supersede share delta
- dominant recommendation change

Recommended guardrails:

- If candidate `run_count` is below a configured minimum, recommendation is `needs_more_data`.
- If candidate `scanned` is zero or mostly zero-input, recommendation is `needs_more_data`.
- If manual-review share increases materially, recommendation is at least `manual_review_required`.
- If auto-accept share increases without label support, recommendation is `manual_review_required`, not `improvement_candidate`.

## Label Model

G080 decisions can become evaluation labels only after normalization.

Suggested table: `marketops_backtest_evaluation_labels`.

Fields:

- `label_id`
- `tenant_id`
- `source_proposal_id`
- `artifact_id`
- `signal_id`
- `subject_symbol`
- `candidate_type`
- `graph_fact_key`
- `decision_status`
- `label`
- `label_source`
- `labeled_by`
- `labeled_at`
- `label_version`
- `metadata`
- `created_at`

Suggested labels:

- `positive` for accepted graph facts.
- `negative` for rejected graph facts.
- `superseded` for replaced graph facts.
- `unresolved` for proposed/restored graph facts that still need review.

Label caveats:

- An operator decision is contextual, not absolute truth.
- Superseded proposals should not be counted as simple negatives without a replacement fact key.
- Restored/proposed states should not count as positive or negative.
- Labels need a stable `graph_fact_key` so candidate outputs can be matched across runs.

## Graph Fact Key

A graph fact key is the stable identity used to compare a generated proposal with a human label.

Suggested construction:

- Node candidate: `node:{node_id}`.
- Relationship candidate: `relationship:{from_node}:{relationship}:{to_node}`.

The key should be normalized consistently for production proposals and back-test proposals.

## Label-Based Evaluation Metrics

When labels are available, comparison records may include:

- labeled candidate count
- true positives
- false positives
- true negatives
- false negatives
- precision
- recall
- false positive rate
- false negative rate
- manual-review escape rate

MVP rule:

- Do not compute precision/recall unless a minimum labeled sample threshold is met.
- If label coverage is below threshold, emit `needs_more_data` or `manual_review_required`.

## API Boundary

Potential endpoints:

- `POST /v1/marketops/backtest-calibration-baselines`
- `GET /v1/marketops/backtest-calibration-baselines`
- `GET /v1/marketops/backtest-calibration-baselines/{baseline_id}`
- `POST /v1/marketops/backtest-calibration-comparisons`
- `GET /v1/marketops/backtest-calibration-comparisons`
- `GET /v1/marketops/backtest-calibration-comparisons/{comparison_id}`
- `POST /v1/marketops/backtest-evaluation-labels/sync`
- `GET /v1/marketops/backtest-evaluation-labels`

MVP implementation can defer label sync if baseline comparison over summaries is implemented first.

## Frontend Boundary

If implemented in G083, the frontend should add a compact baseline/comparison area to `/marketops/backtests`:

- list active baselines;
- mark a persisted summary as a baseline;
- compare a selected persisted summary to a baseline;
- show comparison deltas and recommendation;
- show label coverage if available.

Do not add threshold promotion or policy deployment controls.

## Acceptance Criteria

G083 is complete when:

- Named baselines can point to immutable calibration summaries.
- A candidate summary can be compared to a baseline and stored as an immutable comparison record.
- Comparison metrics include the initial summary-level deltas.
- Recommendation values are deterministic and explainable.
- Production ledgers remain untouched.
- Authenticated API smoke validates create/list/detail for baselines and comparisons.
- Docs and gate audit explain label caveats and deferred promotion work.

If label extraction is included:

- G080 decisions can be synced into evaluation labels without mutating the original proposal records.
- Label sync preserves actor/time/source proposal metadata.
- Precision/recall metrics are only emitted when label coverage is sufficient.

## Recommended Implementation Order

1. Baseline storage/API over existing G082 summaries.
2. Comparison storage/API with deterministic summary-level deltas.
3. Authenticated smoke for baseline create/list/detail and comparison create/list/detail.
4. UI handoff or minimal UI wiring for read-only comparison.
5. Label extraction spec or implementation depending on available G080 decision volume.

## Follow-On Gate

G084 should focus on label extraction and label-aware scoring if G083 implements only baseline comparison.

If G083 includes labels, G084 should focus on calibration recommendation review workflow, still without automatic policy promotion.
