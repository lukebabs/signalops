# Back-Test Calibration Readiness API

G094 adds persisted calibration readiness snapshots over existing G081-G086 evidence. These APIs are advisory and audit-oriented. They do not deploy policies, edit detector thresholds, mutate runtime configuration, write graph state, or alter production signal/artifact/proposal ledgers.

## Create Readiness Snapshot

`POST /v1/marketops/backtest-calibration-readiness`

Required fields:

- `tenant_id`
- `baseline_id`
- `comparison_id`

Optional fields:

- `readiness_id`
- `evaluation_id`
- `candidate_id`
- `dataset_scope`
- `universe_group`
- `window_start`
- `window_end`
- `thresholds`
- `requested_by`

The response envelope is `{ "calibration_readiness": ... }` and includes readiness status, reasons, coverage metrics, label metrics, evaluation metrics, thresholds, evidence references, and audit fields.

Readiness values:

- `calibration_ready`
- `needs_more_historical_data`
- `needs_more_labels`
- `label_quality_blocked`
- `regression_detected`
- `manual_review_required`
- `blocked`

`calibration_ready` is still not deployment. It means the evidence is strong enough for continued deployment planning review.

## List Readiness Snapshots

`GET /v1/marketops/backtest-calibration-readiness`

Supported filters: `tenant_id`, `app_id`, `domain`, `use_case`, `baseline_id`, `comparison_id`, `evaluation_id`, `candidate_id`, `detector_id`, `readiness_status`, and `limit`.

## Get Readiness Snapshot

`GET /v1/marketops/backtest-calibration-readiness/{readiness_id}`

Returns one persisted readiness snapshot row.

## Snapshot Semantics

The first implementation computes readiness from:

- succeeded isolated back-test runs for the detector;
- current MarketOps universe assets when `universe_group` is supplied;
- G084 evaluation labels;
- optional G085 label-aware evaluation;
- optional G086 promotion candidate;
- G083 baseline and comparison evidence.

Default thresholds are conservative: `80%` symbol coverage, `20` historical windows, `10` options windows when options are in scope, `100` reviewed labels, `0.8` label coverage, and maximum `0.05` conflicting-label ratio.
