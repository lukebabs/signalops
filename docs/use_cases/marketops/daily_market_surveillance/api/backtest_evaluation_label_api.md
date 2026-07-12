# Back-Test Evaluation Label API

G084 synchronizes evaluation labels from reviewed MarketOps DSM graph proposal decisions. The original `marketops_dsm_graph_proposals` rows remain canonical for operator decisions; labels are a normalized evaluation substrate for later label-aware scoring.

## Sync Labels

`POST /v1/marketops/backtest-evaluation-labels/sync`

Required request fields:

- `tenant_id`

Optional request fields:

- `app_id`
- `domain`
- `use_case`
- `status`
- `include_unresolved`
- `limit`
- `requested_by`

When `status` is omitted, sync includes `accepted`, `rejected`, and `superseded` proposals. Set `include_unresolved=true` to include `proposed` rows as unresolved labels.

Decision-to-label mapping:

- `accepted` -> `positive`
- `rejected` -> `negative`
- `superseded` -> `superseded`
- `proposed` -> `unresolved`

The sync is idempotent for each `(source_proposal_id, label_version)` pair.

## List Labels

`GET /v1/marketops/backtest-evaluation-labels`

Supported filters: `tenant_id`, `app_id`, `domain`, `use_case`, `source_proposal_id`, `artifact_id`, `signal_id`, `subject_symbol`, `candidate_type`, `decision_status`, `label`, `label_source`, and `limit`.

## Get Label

`GET /v1/marketops/backtest-evaluation-labels/{label_id}`

Returns one synchronized evaluation label.

## Non-Goals

This API does not perform precision/recall scoring, detector threshold promotion, policy deployment, graph writeback, or model training.
