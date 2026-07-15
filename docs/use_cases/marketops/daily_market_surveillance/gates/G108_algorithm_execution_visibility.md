# G108 Algorithm Execution Visibility

Status: implemented backend/API visibility
Timestamp: 2026-07-15T03:04:57Z

## Purpose

G108 adds operator visibility over the generic SignalOps algorithm ledger created in G106 and populated by the G107 z-score runner. The goal is to let operators and analysts inspect what ran and what result evidence was produced before adding more algorithm adapters or a frontend workbench.

## Implemented Scope

- Added read-only execution summary endpoint:
  - `GET /v1/algorithms/execution-requests/{execution_request_id}/summary?tenant_id={tenant_id}&limit=10`
- The summary returns:
  - execution request metadata and lifecycle status;
  - result count over the bounded result scan;
  - severity counts;
  - max score;
  - max confidence;
  - top result rows ordered by score descending.
- `limit` controls the number of `top_results` returned.
- The summary scans up to the repository result cap for the selected execution request.
- Added API test coverage for severity rollup, max score/confidence, result filtering, and top-result ordering.

## Explicitly Out Of Scope

- New algorithm execution behavior.
- New algorithm dependencies.
- API-triggered execution.
- Frontend workbench.
- Signal/artifact/alert/insight conversion.
- Runtime policy deployment.
- Syncratic graph or metadata ingestion.

## Validation

- `docker run --rm -v /home/adminalien/docker/syncratic-core/subsystems/signalops:/workspace -w /workspace golang:1.22-bookworm go test ./internal/api -count=1`

## Next Gate Candidate

The next narrow step should be a frontend-agent specification for algorithm execution/result visibility if the UI needs analyst-facing inspection. If backend-only work continues first, add persisted aggregate summary fields only after real operator usage shows the read-time rollup is insufficient.
