# Back-Test Label-Aware Evaluation API

G085 scores a MarketOps back-test run against G084 evaluation labels. It stores aggregate scoring snapshots for later calibration review and does not promote policies or change detector thresholds.

## Create Evaluation

`POST /v1/marketops/backtest-evaluations`

Required request fields:

- `tenant_id`
- `run_id`

Optional request fields:

- `evaluation_id`
- `label_source`
- `requested_by`

The scorer matches back-test graph proposals to labels by `graph_fact_key`. Positive and negative labels contribute to precision/recall style metrics. Superseded and unresolved labels are counted but excluded from automatic true/false outcome rates.

## List Evaluations

`GET /v1/marketops/backtest-evaluations`

Supported filters: `tenant_id`, `app_id`, `domain`, `use_case`, `run_id`, `detector_id`, `dataset`, `recommendation`, and `limit`.

## Get Evaluation

`GET /v1/marketops/backtest-evaluations/{evaluation_id}`

Returns one stored label-aware back-test evaluation.

## Non-Goals

This API does not deploy policy, tune detector thresholds, write graph state, or train models.
