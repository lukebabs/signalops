# G085 Label-Aware Back-Test Evaluation

Status: validated — backend/API scoring substrate implemented
Use case: MarketOps Daily Market Surveillance

## Goal

Use G084 evaluation labels to score MarketOps back-test policy recommendations against human-reviewed graph proposal decisions.

## Scope

- Add stored back-test evaluation records.
- Match run-scoped generated graph proposals to labels by graph fact key.
- Score automatic recommendations against positive/negative labels.
- Expose create/list/detail APIs.

## Metrics

- candidate count
- labeled count
- positive/negative/superseded/unresolved counts
- true positive / false positive / true negative / false negative
- manual review count
- precision, recall, specificity, accuracy, label coverage
- advisory recommendation

## Non-Goals

- Detector threshold promotion
- Policy deployment
- Graph writeback
- ML training
- PnL/trading simulation

## API

- `POST /v1/marketops/backtest-evaluations`
- `GET /v1/marketops/backtest-evaluations`
- `GET /v1/marketops/backtest-evaluations/{evaluation_id}`
