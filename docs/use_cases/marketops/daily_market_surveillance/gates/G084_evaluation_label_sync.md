# G084 Evaluation Label Sync

Status: validated — backend/API label sync implemented
Use case: MarketOps Daily Market Surveillance

## Goal

Create the first label substrate for MarketOps back-test evaluation by normalizing G080 graph proposal decisions into evaluation labels.

## Scope

- Add `marketops_backtest_evaluation_labels` storage.
- Sync labels from reviewed `marketops_dsm_graph_proposals` rows.
- Expose sync/list/detail APIs.
- Keep graph proposal decisions canonical and unchanged.

## Label Mapping

- `accepted` -> `positive`
- `rejected` -> `negative`
- `superseded` -> `superseded`
- `proposed` -> `unresolved`, only when explicitly included or requested

## Non-Goals

- Precision/recall scoring
- Detector threshold promotion
- Policy deployment
- Graph writeback
- ML training or model registry integration
- PnL/trading simulation

## API

- `POST /v1/marketops/backtest-evaluation-labels/sync`
- `GET /v1/marketops/backtest-evaluation-labels`
- `GET /v1/marketops/backtest-evaluation-labels/{label_id}`

## Completion Criteria

- Labels can be synced idempotently from decided graph proposals.
- Labels can be listed and fetched by id.
- Authenticated API smoke validates sync/list/detail.
- Existing graph proposal decision workflow remains unchanged.
