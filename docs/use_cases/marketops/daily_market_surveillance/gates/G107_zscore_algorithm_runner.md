# G107 Z-Score Algorithm Runner

Status: implemented backend/CLI runner
Timestamp: 2026-07-15T02:58:06Z

## Purpose

G107 is the first executable slice of the generic SignalOps algorithm substrate introduced in G106. It runs `signalops.algorithms.zscore_anomaly_v1` over bounded historical normalized events and writes deterministic `algorithm_results` rows.

This is intentionally not a MarketOps DSM detector change. It does not produce `signal.v1`, DSM artifacts, graph proposals, alerts, insights, or runtime policy deployment.

## Implemented Scope

- Added `internal/algorithms` runner package.
- Added `cmd/algorithm-runner` CLI.
- Added Docker build/runtime target `algorithm-runner`.
- Implemented `signalops.algorithms.zscore_anomaly_v1` for one numeric feature at a time.
- Default feature is `daily_return_pct`.
- Reads bounded normalized events through the existing normalized-event query boundary.
- Computes deterministic population mean, standard deviation, z-score, absolute z-score, confidence, and severity.
- Writes one immutable/idempotent `algorithm_results` row per usable sample when `min_samples` is met.
- Creates and updates an `algorithm_execution_requests` lifecycle row from `running` to `succeeded` or `failed`.
- Keeps result IDs stable across reruns for the same tenant, algorithm, version, execution request, event, and feature.

## CLI Example

```bash
signalops-algorithm-runner \
  --execution-request-id algexec-g107-aapl \
  --tenant-id tenant-local \
  --algorithm-id signalops.algorithms.zscore_anomaly_v1 \
  --algorithm-version v1 \
  --feature daily_return_pct \
  --symbols AAPL \
  --window-start 2026-07-09T00:00:00Z \
  --window-end 2026-07-14T00:00:00Z \
  --max-records 50 \
  --batch-size 25 \
  --min-samples 3 \
  --z-threshold 3.0
```

Required environment:

- `SIGNALOPS_DATABASE_URL`
- `SIGNALOPS_TEMPORAL_DATABASE_URL`

## Result Contract

`result_type` is `z_score`.

`result_payload` includes:

- `feature`
- `value`
- `mean`
- `stddev`
- `z_score`
- `abs_z_score`
- `z_threshold`
- `symbol`
- `observation_time`

Lineage fields include:

- `source_event_ids`: normalized event id.
- `feature_value_ids`: `{event_id}:{feature}`.
- `evidence_refs`: `normalized_event:{event_id}`.
- `correlation_id`: execution request id unless overridden.

## Explicitly Out Of Scope

- River, Ruptures, Statsmodels, or Scikit-Learn execution.
- Frontend workbench.
- New API trigger endpoint.
- Signal/artifact/alert/insight conversion.
- Runtime deployment or detector/policy mutation.
- Syncratic graph or metadata ingestion.

## Validation

- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm gofmt -w internal/algorithms/runner.go internal/algorithms/runner_test.go cmd/algorithm-runner/main.go`
- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./internal/algorithms ./cmd/algorithm-runner -count=1`

## Next Gate Candidate

G108 should either add the first frontend/operator read path for algorithm executions/results or implement the next non-stdlib algorithm adapter behind the same result-ledger contract. The safer next step is operator visibility before adding heavier ML dependencies.
