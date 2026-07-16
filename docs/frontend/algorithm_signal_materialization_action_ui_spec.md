# Frontend-Agent Spec: G123 Algorithm Signal Materialization Action UI

## Goal

Add a narrowly scoped single-proposal materialization action to the existing Algorithms / Signal Proposals UI.

This builds on:

- G114 proposal list/detail/review UI.
- G116 proposal summary UI.
- G119 materialization preflight UI.
- G121 materialization ledger read APIs.
- G122 single-proposal materialization mutation.

The UI must let an operator materialize exactly one eligible reviewed algorithm signal proposal after seeing preflight status and confirming intent. It must not add bulk materialization, policy deployment, alert/insight controls, graph controls, Syncratic actions, algorithm execution, or tuning.

## Existing Backend APIs

Use these existing endpoints only:

- `GET /v1/algorithms/signal-proposals/materialization-preflight?tenant_id={tenant_id}&algorithm_id={algorithm_id}&execution_request_id={execution_request_id}&algorithm_result_id={algorithm_result_id}&status={status}&severity={severity}&correlation_id={correlation_id}&limit={limit}&min_reviewed_ratio=1&policy_version=materialization_preflight.v1`
- `POST /v1/algorithms/signal-proposals/{proposal_id}/materializations?tenant_id={tenant_id}`
- `GET /v1/algorithms/signal-materializations?tenant_id={tenant_id}&proposal_id={proposal_id}&limit=50`
- `GET /v1/algorithms/signal-materializations/{materialization_id}?tenant_id={tenant_id}`
- existing signal detail/list API if already wired in the UI; do not invent a new signal endpoint.

POST request body:

```json
{
  "tenant_id": "tenant-local",
  "policy_version": "algorithm_materialization.v1",
  "metadata": {
    "note": "Operator confirmed materialization from proposal detail."
  }
}
```

Response envelope:

```json
{
  "algorithm_signal_materialization": {
    "materialization_id": "algmat_...",
    "proposal_id": "algsigprop_...",
    "materialization_status": "succeeded",
    "signal_id": "sig_alg_...",
    "duplicate_of_signal_id": "",
    "materialization_policy_version": "algorithm_materialization.v1",
    "idempotency_key": "algmat_idem_..."
  }
}
```

## UI Placement

Extend the existing `Signal Proposals` detail panel in the Algorithms route.

Recommended placement:

- Keep G119 read-only preflight panel above the proposal list.
- Add materialization controls only in the selected proposal detail panel.
- Show related materialization ledger rows for the selected proposal near the detail panel, preferably below the review controls or in a compact adjacent section.

Do not create a new top-level route.

## Eligibility Rules

The Materialize action must be enabled only when all of these are true:

- selected proposal exists;
- proposal status is `reviewed`;
- matching G119 preflight item has `preflight_status=eligible`;
- matching preflight item has `would_write=true`;
- no active mutation is in flight;
- user has mutation permission according to existing frontend auth/role helpers.

Disable the action and show compact reason text when:

- proposal is `proposed`, `rejected`, or `superseded`;
- preflight status is `blocked`, `invalid`, or `duplicate_risk`;
- global preflight blockers are active;
- preflight query is loading or failed;
- materialization already exists for the proposal with `succeeded` or `duplicate` status.

Do not rely only on frontend eligibility. The backend remains authoritative.

## Confirmation Flow

Materialization changes production signal state, so require an explicit confirmation dialog or inline confirmation step.

Confirmation content must include:

- proposal id;
- proposed signal type;
- algorithm id/version;
- severity;
- confidence;
- source event count;
- policy version `algorithm_materialization.v1`;
- copy that one production signal may be created;
- copy that alerts/insights/graph proposals are not directly created by this action.

Use concise operator-focused text. Avoid long explanatory prose.

The confirm button should be disabled while the mutation is in flight.

## Required Result Handling

After mutation succeeds:

- invalidate/refetch materialization rows for the selected proposal;
- invalidate/refetch materialization preflight;
- invalidate/refetch proposal detail/list if existing query behavior expects it;
- if `materialization_status=succeeded`, show the materialized `signal_id` and provide a link or selection affordance to the existing signal detail surface if available;
- if `materialization_status=duplicate`, show `duplicate_of_signal_id` and do not imply a new signal was created;
- if `materialization_status=blocked`, show the preflight/blocker reason from the row;
- if `materialization_status=failed`, show sanitized error code/message.

Idempotent retry behavior:

- If the operator repeats the action and backend returns an existing materialization, show it as already materialized/recorded rather than treating it as an error.
- Do not create duplicate UI rows when refetching.

## Materialization Ledger Panel

For the selected proposal, show compact rows from:

`GET /v1/algorithms/signal-materializations?tenant_id={tenant_id}&proposal_id={proposal_id}&limit=50`

Columns/content:

- materialization id;
- status;
- policy version;
- signal id;
- duplicate signal id;
- requested by;
- requested at;
- completed/failed at;
- error code;
- error message, truncated;
- idempotency key, truncated.

Keep JSON fields collapsed by default:

- request metadata;
- preflight snapshot;
- signal payload preview.

## Visual Semantics

Use existing badge/status styles.

Recommended status tones:

- `succeeded`: success;
- `duplicate`: warning/neutral;
- `blocked`: warning;
- `failed`: error;
- `requested`/`running`: in-progress/neutral;
- `superseded`: muted.

The UI must distinguish:

- reviewed proposal;
- eligible preflight candidate;
- materialized production signal;
- duplicate materialization row;
- blocked/failed materialization row.

Do not label a reviewed proposal as accepted or materialized until the materialization ledger says so.

## Loading, Empty, And Error States

Loading:

- Materialization ledger rows should have their own compact loading state.
- Mutation loading must not block unrelated proposal browsing.

Empty:

- If no materialization rows exist for the selected proposal, show: `No materialization records for this proposal.`

Error:

- Follow existing API error patterns.
- Do not expose bearer tokens, auth headers, secrets, raw environment variables, stack traces, or upstream internals.

## Out Of Scope

- Bulk materialization.
- Async worker controls.
- Policy deployment controls.
- Editing policy version beyond the fixed `algorithm_materialization.v1` default.
- Alert or insight creation controls.
- Graph proposal creation controls.
- DSM taxonomy remapping.
- Syncratic Ask/Search.
- Algorithm execution triggering.
- Algorithm tuning.
- New backend endpoints.
- New navigation shell.

## Suggested Implementation Areas

Frontend-agent should inspect the G114/G116/G119 implementation first. Likely touch points:

- API client: add `materializeAlgorithmSignalProposal`, `listAlgorithmSignalMaterializations`, and optional detail fetch if needed.
- React Query hooks: mutation hook plus materialization list query keyed by tenant/proposal id.
- Types: add `AlgorithmSignalMaterialization` and response envelopes.
- Algorithms route/detail panel: render action, confirmation, result state, and materialization ledger rows.
- Existing preflight summarizer: expose helper to find a preflight item by proposal id.
- Tests: path construction, request body, eligibility gates, disabled states, confirmation flow, success/duplicate/blocked/failed rendering, query invalidation, and absence of bulk controls.

## Validation Checklist

Frontend-agent should verify:

- Materialize button appears only in selected proposal detail.
- Button is enabled only for reviewed + eligible + would-write proposal.
- Blocked/invalid/duplicate-risk proposals cannot be submitted.
- Confirmation appears before POST.
- POST path and body match G122.
- Successful response renders materialization status and signal id.
- Duplicate response renders duplicate signal id and no new-signal copy.
- Blocked/failed responses render sanitized reason/error.
- Materialization ledger rows refetch after mutation.
- Existing proposal review workflow still works.
- No bulk materialization, alert, insight, graph, deploy, Syncratic, run, or tune controls are added.
- Desktop and mobile layouts remain usable.

## Handoff Note

Backend G122 is implemented and validated by automated tests and an auth-boundary smoke. Positive live UI validation requires an authenticated operator session and at least one reviewed eligible proposal.
