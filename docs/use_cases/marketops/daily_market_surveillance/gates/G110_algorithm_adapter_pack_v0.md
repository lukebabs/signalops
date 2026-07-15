# G110 Algorithm Adapter Pack v0

Status: implemented backend runner adapters
Timestamp: 2026-07-15T20:54:51Z

## Purpose

G110 closes the seeded-algorithm execution gap left after G107 by making every G106 seeded algorithm id executable through the existing `signalops-algorithm-runner` and `algorithm_results` ledger contract.

This gate intentionally uses deterministic v0 scoring adapters without adding River, Ruptures, Statsmodels, or Scikit-Learn runtime dependencies. Library-faithful implementations remain calibration follow-ons.

## Implemented Scope

The runner now supports:

- `signalops.algorithms.zscore_anomaly_v1`
- `signalops.algorithms.river_anomaly_v1`
- `signalops.algorithms.ruptures_change_point_v1`
- `signalops.algorithms.statsmodels_forecast_v1`
- `signalops.algorithms.sklearn_classifier_v1`
- `signalops.algorithms.sklearn_isolation_forest_v1`

Implemented deterministic v0 result types:

- `z_score`: population z-score over the bounded window.
- `online_anomaly_score`: online prior-window z-score approximation for River-style streaming anomaly behavior.
- `change_point_score`: adjacent normalized delta approximation for change-point candidate detection.
- `forecast_residual`: linear-trend forecast residual approximation for forecast outlier detection.
- `classifier_label`: deterministic threshold classifier until trained model artifacts and labels are introduced.
- `isolation_score`: median absolute deviation approximation for isolation-style outlier scoring.

All adapters:

- Read bounded normalized-event windows through the existing query boundary.
- Use one numeric feature per execution.
- Persist immutable/idempotent `algorithm_results` rows.
- Preserve normalized-event lineage via `source_event_ids`, `feature_value_ids`, and `evidence_refs`.
- Reuse `algorithm_execution_requests` lifecycle tracking.

## Explicitly Out Of Scope

- Installing external ML libraries.
- Trained model artifacts.
- Hyperparameter search.
- Runtime policy deployment.
- Signal/artifact/alert/insight conversion.
- Frontend changes.
- Syncratic graph or metadata ingestion.

## Validation

- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./internal/algorithms -count=1`

## Follow-On

Library-faithful adapters should be calibrated one at a time after the result-to-signal proposal design is reviewed and after broader historical data coverage is available.

## Live Smoke

Timestamp: 2026-07-15T20:56:59Z

- Ran `signalops.algorithms.ruptures_change_point_v1` through `signalops-algorithm-runner` against bounded AAPL equity rows using `open_close_move_pct`.
- Execution request `algexec-g110-ruptures-aapl-openclose` completed with `scanned=3`, `usable_samples=3`, and `results=2`.
- Persisted result rows use `result_type=change_point_score` with severity counts `critical=1` and `info=1`.
