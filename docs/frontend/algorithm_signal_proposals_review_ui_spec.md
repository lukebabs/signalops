# Frontend-Agent Spec: G113 Algorithm Signal Proposals Review UI

## Goal

Add operator visibility and review controls for the G111/G112 `algorithm_signal_proposals` workflow inside the existing Algorithms UI.

This UI must let operators inspect algorithm-derived candidate signal proposals, review their evidence, and record review decisions. It must not materialize production signals or create alerts, insights, graph proposals, policies, deployments, or Syncratic work.

## Existing Backend APIs

Use existing endpoints only:

- `GET /v1/algorithms/signal-proposals?tenant_id={tenant_id}&algorithm_id={algorithm_id}&execution_request_id={execution_request_id}&algorithm_result_id={algorithm_result_id}&status={status}&severity={severity}&correlation_id={correlation_id}&limit=50`
- `GET /v1/algorithms/signal-proposals/{proposal_id}?tenant_id={tenant_id}`
- `POST /v1/algorithms/signal-proposals/{proposal_id}/decision?tenant_id={tenant_id}`

Decision request body:

```json
{
  "tenant_id": "tenant-local",
  "status": "reviewed",
  "note": "Reviewed as useful evidence; no production signal materialized.",
  "metadata": {}
}
```

Valid decision statuses:

- `proposed`
- `reviewed`
- `rejected`
- `superseded`

Do not use `accepted`. G112 intentionally did not add acceptance or materialization semantics.

## UI Placement

Extend the existing MarketOps `Algorithms` route from G109. Do not create a new top-level route unless the current app architecture makes an internal tab impossible.

Recommended layout inside `/marketops/algorithms`:

- Keep the current Definitions / Executions / Results workflow intact.
- Add a `Signal Proposals` tab or panel in the same route.
- When an execution request is selected, default the proposals view to `execution_request_id={selected_execution_request_id}`.
- Also allow viewing all recent proposals for the tenant.

## Primary Workflow

The operator should be able to:

1. Open the Algorithms page.
2. Select `Signal Proposals`.
3. Filter proposals by status, severity, algorithm id, execution request id, and correlation id.
4. Select a proposal row.
5. Inspect proposed signal type, score, confidence, severity, payload, rationale, and lineage.
6. Submit one review decision: `reviewed`, `rejected`, `superseded`, or restore to `proposed`.
7. See the row update with `reviewed_by`, `decision_note`, and `decided_at`.

## Required Proposal List

Show a compact table with:

- proposal id
- proposed signal type
- status
- severity
- score
- confidence
- algorithm id
- execution request id
- algorithm result id
- correlation id
- reviewed by
- decided at
- updated at

Filters:

- status
- severity
- algorithm id
- execution request id
- correlation id

Default filters:

- status: `proposed`
- limit: `50`

If the user enters an execution request from the existing execution summary, carry that execution request id into the proposal filter.

## Required Detail Panel

When a proposal is selected, show:

- proposal id
- tenant id
- proposed signal type
- status
- score / confidence / severity
- algorithm id/version
- execution request id
- algorithm result id
- source event ids
- evidence refs
- correlation id
- created by
- reviewed by
- decision note
- decided at
- created at / updated at
- formatted `proposal_payload` JSON
- formatted `rationale` JSON

JSON blocks should be collapsible and formatted. Preserve existing UI patterns for raw JSON display.

## Review Controls

Add a compact review control in the proposal detail panel only.

Controls:

- status selector with `reviewed`, `rejected`, `superseded`, `proposed`
- note text area
- submit button

Behavior:

- Disable submit while request is in flight.
- Require a non-empty note for `rejected` and `superseded`.
- On success, update/invalidate proposal list and detail queries.
- Show backend validation errors safely.
- Use existing auth/bearer handling; do not add a new auth mechanism.

Button copy should be neutral, for example `Save review`.

Do not include buttons labeled `Accept`, `Materialize`, `Create Signal`, `Promote`, `Deploy`, `Create Alert`, or `Create Insight`.

## Visual Semantics

Use badges for:

- status
- severity
- proposed signal type

Status tone guidance:

- `proposed`: neutral/pending
- `reviewed`: positive/complete but not production
- `rejected`: negative
- `superseded`: muted/secondary

Do not imply `reviewed` means accepted or deployed.

## Empty And Error States

Empty states:

- “No algorithm signal proposals found.”
- “Select a proposal to inspect its evidence.”

Errors:

- Follow existing API error rendering patterns.
- Do not expose bearer tokens, auth headers, secrets, raw environment variables, or full upstream internals.

## Out Of Scope

- Production `signal.v1` creation.
- Alert or insight creation.
- Graph proposal creation.
- Algorithm execution triggering.
- Algorithm tuning.
- Policy deployment.
- Syncratic Ask/Search.
- Bulk decisions.
- Multi-review history.
- New backend endpoints.

## Suggested Implementation Areas

Frontend-agent should inspect the existing web implementation first, especially the G109 Algorithms route and API client patterns. Likely touch points include:

- algorithms API client methods
- React Query hooks for algorithm proposal list/detail/decision
- Algorithms route tab/panel state
- proposal table and detail components
- tests for API path construction, query invalidation, status rendering, and decision mutation payload

Use existing app styling and component conventions. Keep tables dense and operational.

## Validation Checklist

Frontend-agent should verify:

- Proposal list loads with default `status=proposed` filter.
- Filters build the correct query params.
- Selecting a proposal loads detail.
- Detail renders payload/rationale JSON and lineage.
- Decision POST sends the correct body and bearer through the existing API client.
- Successful decision updates list/detail state.
- Invalid/backend errors render without crashing.
- Required-note validation works for `rejected` and `superseded`.
- No materialization or production signal controls exist.
- Desktop and mobile layouts remain usable.
- Existing G109 Algorithms UI still works.

## Handoff Note

Backend G111/G112 is already implemented and validated. The local stack has at least one proposal from the G110 live smoke:

- `proposal_id`: `algsigprop_c6c2acad697176d0f438b66e`
- `execution_request_id`: `algexec-g110-ruptures-aapl-openclose`
- `proposed_signal_type`: `signalops.algorithm.change_point_candidate`

The row may currently be `reviewed` from G112 validation. The UI must support filtering beyond `proposed` so reviewed rows remain inspectable.
