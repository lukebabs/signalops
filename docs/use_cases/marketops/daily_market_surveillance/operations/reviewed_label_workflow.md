# Reviewed Label Workflow Operations

This runbook supports G104. It uses existing MarketOps graph proposal review, evaluation-label sync, label-aware evaluation, and calibration-readiness APIs.

## Objective

Increase real reviewed labels from the current G103 level of `7` toward the G094 readiness threshold of `100`, without synthetic labels or threshold changes.

## Cadence

Use staged milestones:

- `25` matched reviewed labels.
- `50` matched reviewed labels.
- `100` matched reviewed labels.

At each milestone, run label sync, a label-aware evaluation, and a readiness snapshot.

## Review Batch Rules

Select proposed graph proposals across multiple symbols, dates, signal types, candidate types, and recommendation types. Avoid counting duplicate `graph_fact_key` decisions as independent evidence for milestone progress.

Recommended first batch size: `18` distinct graph facts, enough to move from `7` to at least `25` if all are reviewable.

## API Loop

1. Review proposals through `POST /v1/marketops/dsm/graph-proposals/{proposal_id}/decision`.
2. Sync reviewed labels through `POST /v1/marketops/backtest-evaluation-labels/sync`.
3. Verify labels through `GET /v1/marketops/backtest-evaluation-labels`.
4. Run a G085 label-aware evaluation through `POST /v1/marketops/backtest-evaluations`.
5. Re-run G094 readiness through `POST /v1/marketops/backtest-calibration-readiness`.

## Quality Checks

- `conflicting_label_ratio <= 0.05`.
- Label set includes positive and negative labels as soon as real rejected proposals exist.
- Superseded labels are retained as review evidence but are not automatic true/false outcomes.
- Re-running sync does not increase counts for the same `(source_proposal_id, label_version)` pair.
- Every milestone report includes matched labels, distinct graph facts, conflict ratio, positive/negative/superseded counts, and readiness status.

## Boundaries

Do not deploy policy, change detector thresholds, write graph database state, generate synthetic labels, or use unresolved/proposed rows as reviewed evidence.
