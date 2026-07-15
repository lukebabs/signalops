# Frontend-Agent Spec: G109 Algorithm Execution Visibility UI

## Goal

Add read-only analyst/operator visibility for the SignalOps algorithm layer implemented in G106-G108.

This UI must expose algorithm definitions, execution requests, execution summaries, and result evidence without adding execution controls, tuning controls, policy deployment, signal conversion, or new backend behavior.

## Existing Backend APIs

Use existing endpoints only:

- `GET /v1/algorithms/definitions?tenant_id={tenant_id}&algorithm_type={type}&runtime_type={runtime_type}&status={status}&limit=50`
- `GET /v1/algorithms/definitions/{algorithm_id}?tenant_id={tenant_id}`
- `GET /v1/algorithms/execution-requests?tenant_id={tenant_id}&algorithm_id={algorithm_id}&status={status}&correlation_id={correlation_id}&limit=50`
- `GET /v1/algorithms/execution-requests/{execution_request_id}?tenant_id={tenant_id}`
- `GET /v1/algorithms/execution-requests/{execution_request_id}/summary?tenant_id={tenant_id}&limit=10`
- `GET /v1/algorithms/results?tenant_id={tenant_id}&algorithm_id={algorithm_id}&execution_request_id={execution_request_id}&result_type={result_type}&severity={severity}&correlation_id={correlation_id}&limit=50`
- `GET /v1/algorithms/results/{algorithm_result_id}?tenant_id={tenant_id}`

## UI Placement

Add the feature inside the existing MarketOps/back-test/operator area where calibration and evidence review already live.

Suggested label: `Algorithms`.

Do not create a marketing page, landing page, or separate product shell.

## Primary Workflow

The analyst should be able to:

1. View registered algorithm definitions.
2. Select an algorithm definition.
3. View recent execution requests for that algorithm.
4. Select an execution request.
5. View the execution summary.
6. Inspect top result rows and result payload details.
7. Open a full result row for lineage fields.

## Required Views

### Algorithm Definitions

Show a compact list/table with:

- algorithm id
- name
- algorithm type
- runtime type
- version
- status
- input features
- input event types

Filters:

- status
- algorithm type
- runtime type

Default status filter should include `draft` and `active` if the UI supports multi-select. If not, default to all statuses.

### Execution Requests

When an algorithm is selected, show execution requests filtered by `algorithm_id`.

Columns:

- execution request id
- status
- algorithm version
- requested by
- correlation id
- created at
- updated at
- window ref
- feature refs

Filters:

- status
- correlation id

### Execution Summary

When an execution request is selected, call:

`GET /v1/algorithms/execution-requests/{execution_request_id}/summary?tenant_id={tenant_id}&limit=10`

Show:

- status
- requested by
- result count
- max score
- max confidence
- severity counts
- config JSON, collapsed by default
- result JSON, collapsed by default
- top result rows

Top result row columns:

- result id
- result type
- score
- confidence
- severity
- created at
- source event ids
- feature value ids

### Result Detail

When a result row is selected, show:

- result payload JSON
- source event ids
- feature value ids
- evidence refs
- correlation id
- algorithm id/version
- execution request id

Payload JSON should be formatted and collapsible.

## UX Requirements

- Read-only only.
- No “Run”, “Tune”, “Promote”, “Deploy”, “Accept”, “Reject”, or “Convert to Signal” controls.
- Use badges for statuses and severities.
- Use compact tables, not large cards.
- Use existing app layout, auth handling, API client, loading states, and error patterns.
- Empty states should be factual:
  - “No algorithm definitions found.”
  - “No execution requests found.”
  - “No algorithm results found.”
- Errors should show the backend error message where safe, matching existing UI conventions.
- Preserve tenant handling from the existing app.
- Do not expose secrets, bearer tokens, raw auth headers, or environment variables.

## Out Of Scope

- Starting algorithm executions from UI.
- Editing algorithm definitions.
- Editing execution requests.
- Runtime policy deployment.
- Threshold tuning.
- Model training.
- New backend endpoints.
- Signal/artifact/alert/insight conversion.
- Syncratic Ask/Search integration.
- Graph proposal review changes.

## Validation Checklist

Frontend-agent should verify:

- Algorithm definitions load.
- Selecting a definition filters execution requests by algorithm id.
- Selecting an execution request loads its summary endpoint.
- Summary displays result count, severity counts, max score, max confidence, and top results.
- Selecting a top result displays formatted result payload and lineage.
- Empty states render correctly.
- API errors render without crashing.
- Responsive layout works at desktop and mobile widths.
- No mutation controls were introduced.
