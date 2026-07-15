# G106 Algorithm Registry And Result Ledger

Status: implemented backend substrate
Timestamp: 2026-07-15T00:00:00Z

## Purpose

G106 starts the generic SignalOps algorithm substrate described in Section 13 of `docs/signalops_extended_capabilities_spec_v1.md`. The goal is to make algorithms first-class, inspectable platform components rather than hidden MarketOps-only detector code.

This gate does not execute algorithms yet. It creates the registry, execution request ledger, result ledger, seed definitions, and read APIs needed for a later runner and analyst UX.

## Implemented Scope

- Added `algorithm_definitions` for versioned algorithm metadata, runtime type, input features, input event types, schemas, default config, status, and metadata.
- Added `algorithm_execution_requests` for queued/running/succeeded/failed/canceled execution intent and operator-requested config.
- Added immutable/idempotent `algorithm_results` for algorithm outputs, scores, confidence, severity, source event lineage, feature refs, evidence refs, and correlation ids.
- Seeded the initial standard-library algorithm definitions as `draft`:
  - `signalops.algorithms.zscore_anomaly_v1`
  - `signalops.algorithms.river_anomaly_v1`
  - `signalops.algorithms.ruptures_change_point_v1`
  - `signalops.algorithms.statsmodels_forecast_v1`
  - `signalops.algorithms.sklearn_classifier_v1`
  - `signalops.algorithms.sklearn_isolation_forest_v1`
- Added repository interfaces and Postgres implementations for definitions, execution requests, and results.
- Added API routes:
  - `POST /v1/algorithms/definitions`
  - `GET /v1/algorithms/definitions`
  - `GET /v1/algorithms/definitions/{algorithm_id}`
  - `POST /v1/algorithms/execution-requests`
  - `GET /v1/algorithms/execution-requests`
  - `GET /v1/algorithms/execution-requests/{execution_request_id}`
  - `GET /v1/algorithms/results`
  - `GET /v1/algorithms/results/{algorithm_result_id}`
- Added API tests for create/list/get definition flow, create/list/get execution-request flow, and read-only result list/get flow.

## Explicitly Out Of Scope

- Python or container algorithm execution runner.
- New ML dependencies such as River, Ruptures, Statsmodels, or Scikit-Learn.
- Feature store implementation beyond reference fields in execution requests and results.
- Runtime policy deployment or detector mutation.
- Frontend algorithm workbench.
- Syncratic graph or metadata ingestion into SignalOps.
- Automatic conversion of algorithm results into signals, artifacts, alerts, or insights.

## Design Notes

Algorithm results are append-only by `tenant_id` and `algorithm_result_id`. Duplicate result inserts are ignored so reruns can safely replay deterministic output without mutating prior result evidence.

Execution requests are mutable because they model lifecycle state. They can be queued now and later updated by a runner without changing the public API shape.

Definitions are seeded as `draft` so data scientists and operators can inspect intended algorithms before any runtime execution path is activated.

## Validation

- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm gofmt -w internal/storage/storage.go internal/storage/postgres/algorithms.go internal/api/algorithms.go internal/api/router.go internal/api/router_test.go`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./internal/api ./internal/storage/postgres -count=1`

## Next Gate Candidate

G107 should implement the first executable algorithm plugin path using `signalops.algorithms.zscore_anomaly_v1`, with a bounded runner contract that reads existing event/feature evidence, writes `algorithm_results`, and leaves signal/artifact conversion to a later gate.
