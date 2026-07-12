# Back-Test Baseline And Comparison API

G083 adds named calibration baselines and stored baseline-to-candidate comparisons over G082 persisted calibration summaries. These endpoints do not mutate production signal, artifact, graph proposal, alert, or insight ledgers.

## Baselines

`POST /v1/marketops/backtest-calibration-baselines`

Required request fields:

- `tenant_id`
- `name`
- `summary_id`

Optional request fields:

- `baseline_id`
- `description`
- `scope`
- `status`: `active` or `archived`
- `created_by`

The baseline inherits `app_id`, `domain`, `use_case`, `dataset`, and `detector_id` from the referenced calibration summary.

`GET /v1/marketops/backtest-calibration-baselines`

Supported filters: `tenant_id`, `app_id`, `domain`, `use_case`, `dataset`, `detector_id`, `status`, and `limit`.

`GET /v1/marketops/backtest-calibration-baselines/{baseline_id}`

Returns one baseline row.

## Comparisons

`POST /v1/marketops/backtest-calibration-comparisons`

Required request fields:

- `tenant_id`
- `baseline_id`
- `candidate_summary_id`

Optional request fields:

- `comparison_id`
- `created_by`

The stored comparison captures baseline and candidate aggregate snapshots, deterministic deltas, and an advisory recommendation.

Recommendation values:

- `needs_more_data`
- `regression_candidate`
- `improvement_candidate`
- `neutral_candidate`
- `manual_review_required`

`GET /v1/marketops/backtest-calibration-comparisons`

Supported filters: `tenant_id`, `baseline_id`, `dataset`, `detector_id`, `recommendation`, and `limit`.

`GET /v1/marketops/backtest-calibration-comparisons/{comparison_id}`

Returns one stored comparison row.
