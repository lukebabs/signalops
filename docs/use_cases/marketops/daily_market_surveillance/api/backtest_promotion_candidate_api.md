# Back-Test Promotion Candidate API

G086 adds promotion candidate records over G083 baseline comparisons and optional G085 label-aware evaluations. These APIs create auditable operator review records only. They do not deploy policies, edit detector thresholds, write graph state, or mutate production signal/artifact/proposal ledgers.

## Create Candidate

`POST /v1/marketops/backtest-promotion-candidates`

Required fields:

- `tenant_id`
- `baseline_id`
- `comparison_id`

Optional fields:

- `candidate_id`
- `evaluation_id`
- `candidate_version`
- `requested_by`

The response envelope is `{ "promotion_candidate": ... }` and includes readiness status, readiness reasons, compact evidence, and audit fields.

Readiness values:

- `ready_for_review`
- `needs_more_data`
- `manual_review_required`
- `regression_detected`
- `blocked`

## List Candidates

`GET /v1/marketops/backtest-promotion-candidates`

Supported filters: `tenant_id`, `app_id`, `domain`, `use_case`, `baseline_id`, `comparison_id`, `evaluation_id`, `run_id`, `detector_id`, `dataset`, `readiness_status`, `status`, and `limit`.

## Get Candidate

`GET /v1/marketops/backtest-promotion-candidates/{candidate_id}`

Returns one promotion candidate row.

## Decide Candidate

`POST /v1/marketops/backtest-promotion-candidates/{candidate_id}/decision`

Required field:

- `status`

Allowed decision statuses:

- `approved_for_promotion`
- `rejected`
- `deferred`
- `superseded`

Optional fields:

- `reviewed_by`
- `decision_note`

A decision updates only the promotion candidate audit row.
