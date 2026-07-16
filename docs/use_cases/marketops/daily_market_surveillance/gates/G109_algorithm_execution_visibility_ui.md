# G109 Algorithm Execution Visibility UI

Status: implemented and browser-validated
Timestamp: 2026-07-15T04:20:12Z

## Purpose

G109 exposes the generic SignalOps algorithm layer from G106-G108 in the operator UI. It gives analysts read-only visibility into registered algorithms, execution requests, execution summaries, and result evidence.

## Implemented Scope

- Added read-only MarketOps route `/marketops/algorithms` with nav label `Algorithms`.
- Displays algorithm definitions from the existing definitions API.
- Displays execution requests filtered by selected algorithm id.
- Displays execution summaries from `GET /v1/algorithms/execution-requests/{execution_request_id}/summary`.
- Displays top result rows and detailed result payload/lineage.
- Uses existing API client, auth, loading, error, and layout conventions.
- Keeps the UI read-only.

## Browser Validation

Validated on the rebuilt local Docker stack at `http://localhost:15173`.

Confirmed:

- Algorithms view loads.
- `signalops.algorithms.zscore_anomaly_v1` is visible and selectable.
- Execution request `algexec-g109-validate-aapl-openclose` is visible and selectable.
- Summary renders `result_count=3`, severity counts, max score, and top result rows.
- Result detail shows formatted payload and normalized-event lineage.
- No mutation controls were introduced.

## Explicitly Out Of Scope

- Starting algorithm executions from UI.
- Editing algorithm definitions or execution requests.
- Runtime policy deployment.
- Threshold tuning.
- Model training.
- New backend endpoints.
- Signal/artifact/alert/insight conversion.
- Syncratic Ask/Search integration.
- Graph proposal review changes.

## Spec Reconciliation (2026-07-16)

A re-evaluation of the implemented UI against this spec found and fixed two
minor deviations; no behavior or backend contract changed:

- Execution Requests table now shows both `created at` and `updated at`
  (previously only `updated at`). The summarizer already exposed `createdAt`.
- Execution Summary now shows `requested by` (previously omitted from the
  metric strip). Sourced from the summary execution request.

Re-validation: `npx vitest run` (281 tests), `npx tsc --noEmit`, and
`npm run build` all pass.

## References

- Frontend spec: `../../../../frontend/algorithm_execution_visibility_ui_spec.md`
- Backend registry/result ledger: `G106_algorithm_registry_result_ledger.md`
- Z-score runner: `G107_zscore_algorithm_runner.md`
- Execution summary API: `G108_algorithm_execution_visibility.md`
