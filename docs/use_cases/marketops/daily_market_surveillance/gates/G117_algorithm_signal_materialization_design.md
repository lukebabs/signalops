# G117 Algorithm Signal Materialization Design

Status: proposed design
Timestamp: 2026-07-16T00:00:00Z

## Purpose

G117 defines the architecture for a future, explicit materialization path from reviewed `algorithm_signal_proposals` into production `signal.v1` rows.

This is a design gate only. It does not implement storage, APIs, workers, signal writes, alerts, insights, graph proposals, policy deployment, or frontend changes.

## Current Boundary

The current algorithm pipeline is intentionally staged:

1. Algorithm runner writes immutable `algorithm_results`.
2. Proposal generator writes idempotent `algorithm_signal_proposals`.
3. Operators review proposals with `proposed`, `reviewed`, `rejected`, or `superseded` statuses.
4. Summary APIs expose review coverage and unresolved high/critical proposal counts.

No step currently writes production signals from algorithm proposals. G117 preserves that boundary while specifying what must exist before any later implementation gate changes it.

## Design Principle

A reviewed algorithm proposal is not automatically a production signal.

Materialization must be:

- explicit;
- auditable;
- idempotent;
- reversible by lifecycle state, not deletion;
- explainable from original algorithm result to final signal row;
- protected from duplicating deterministic DSM signals;
- blocked when review quality or evidence quality is insufficient.

## Required Preconditions

A future materialization implementation should require all of the following before writing any `signal.v1` row:

- Proposal status model includes a materialization-eligible state distinct from `reviewed`, such as `approved_for_materialization`.
- Operator decision records include reviewer, note, timestamp, and optional decision metadata.
- Review coverage threshold is satisfied for the relevant proposal family, algorithm id, and use case.
- High/critical unreviewed proposal count is below an explicit threshold, or the materialization request is scoped to reviewed rows only.
- The source `algorithm_result` still exists and matches the proposal lineage.
- Source normalized event evidence is present and tenant-consistent.
- Proposal payload and rationale JSON are valid and schema-compatible.
- Duplicate detection against existing `signal_ledger` rows is performed.
- Materialization policy version is explicit and recorded.

## Proposed Additional State

G112 intentionally avoided `accepted` because acceptance semantics were not yet defined. A future gate should add review states only after this design is approved.

Recommended additional statuses:

- `approved_for_materialization`: operator approved this proposal for a later materialization action.
- `materialized`: the proposal has produced a production signal row.
- `materialization_failed`: an attempted materialization failed and requires operator review.

Alternative: keep `algorithm_signal_proposals.status` as review state and add a separate `algorithm_signal_materializations` ledger. This is preferred for audit clarity.

## Preferred Materialization Ledger

Add a separate ledger rather than overloading proposal rows:

`algorithm_signal_materializations`

Suggested fields:

- `materialization_id`
- `tenant_id`
- `proposal_id`
- `algorithm_result_id`
- `execution_request_id`
- `algorithm_id`
- `algorithm_version`
- `signal_id`
- `materialization_status`: `planned`, `succeeded`, `failed`, `superseded`
- `materialization_policy_version`
- `idempotency_key`
- `duplicate_of_signal_id`
- `requested_by`
- `requested_at`
- `completed_at`
- `error_message`
- `metadata`
- `created_at`
- `updated_at`

This ledger makes materialization auditable without turning proposal review rows into execution state.

## Signal Mapping

Initial production signal types should remain generic until explicit taxonomy mapping is reviewed:

- `signalops.algorithm.anomaly_candidate`
- `signalops.algorithm.change_point_candidate`
- `signalops.algorithm.forecast_deviation_candidate`
- `signalops.algorithm.classification_candidate`

Do not automatically map these to MarketOps DSM signal types such as accumulation, hedging pressure, pinning risk, or volatility expansion unless a later taxonomy gate defines equivalence rules.

## Signal Payload Requirements

A materialized signal must include:

- `app_id`, `domain`, and `use_case` where derivable from source normalized events or execution config;
- source adapter and dataset lineage where derivable;
- stable `signal_id` derived from tenant, proposal id, policy version, and source event ids;
- signal type from the proposal mapping;
- severity and confidence from the proposal, possibly adjusted by materialization policy;
- event ids from proposal source event ids;
- entities extracted from proposal payload and normalized-event lineage;
- supporting metrics from algorithm result payload;
- semantic evidence explaining algorithm id/version, result type, score, confidence, and review decision;
- evidence refs including proposal id, algorithm result id, execution request id, and normalized event ids;
- correlation id from the source proposal/result;
- metadata with materialization id and policy version.

## Duplicate Detection

Before writing a signal, the materializer must check for duplicates across:

- same tenant;
- same proposed signal type;
- overlapping source event ids;
- same algorithm result id or proposal id;
- same subject/entity when available;
- same materialization policy version;
- existing deterministic DSM signal ids when evidence overlaps.

If a duplicate exists, the materializer should record `duplicate_of_signal_id` in the materialization ledger and avoid writing a second signal row.

## Alerts And Insights Boundary

Materialization should only write the production signal row.

Alert and insight creation should remain owned by the existing signal-persister/lifecycle flow. A materialization worker must not directly create alerts or insights unless a later gate explicitly changes that architecture.

Insights should remain multi-event or aggregate interpretations. A single materialized algorithm signal should not create an insight with the same explanation as the alert.

## Operator Workflow

A safe future workflow:

1. Operator reviews proposals in the G114/G116 UI.
2. Operator marks selected proposals `approved_for_materialization` or creates a materialization request.
3. Materialization preflight summarizes eligible, duplicate, blocked, and invalid proposals.
4. Operator confirms a bounded materialization request.
5. Worker writes materialization ledger rows and idempotent signal rows.
6. Operator reviews resulting signals, alerts, and downstream summaries.

Bulk actions should require preflight and explicit confirmation.

## API Shape For Future Gate

Possible future APIs:

- `POST /v1/algorithms/signal-proposals/{proposal_id}/materialization-preflight`
- `POST /v1/algorithms/signal-proposals/{proposal_id}/materialize`
- `POST /v1/algorithms/signal-materializations/preflight`
- `POST /v1/algorithms/signal-materializations`
- `GET /v1/algorithms/signal-materializations`
- `GET /v1/algorithms/signal-materializations/{materialization_id}`

All mutation routes must require operator/admin authorization when auth is enabled.

## Readiness Gates

Do not implement materialization until at least these readiness questions are answered:

- How many reviewed proposals exist per algorithm id and proposed signal type?
- What share were rejected or superseded?
- Are high/critical unreviewed proposals under control?
- Are duplicate rates acceptable in back-test or smoke runs?
- Are operators comfortable with generic algorithm signal types appearing in the signal ledger?
- Does frontend clearly distinguish reviewed proposals from materialized signals?

G115 summary is the initial backend surface for answering these questions, but it is not sufficient alone to authorize materialization.

## Testing Requirements For Future Implementation

A future implementation must include tests for:

- idempotent materialization id and signal id generation;
- duplicate detection;
- invalid proposal status rejection;
- missing source result rejection;
- missing event lineage rejection;
- tenant mismatch rejection;
- materialization ledger persistence;
- signal ledger payload correctness;
- no direct alert/insight writes;
- auth-gated mutation routes;
- live smoke with one bounded approved proposal.

## Explicitly Out Of Scope For G117

- Database migrations.
- API implementation.
- Worker implementation.
- Signal ledger writes.
- Alert or insight creation.
- Graph proposals.
- Frontend changes.
- Policy deployment.
- Syncratic integration.

## Recommended Next Gate

G118 should either:

- add frontend visibility for G115 summary if G116 is not implemented yet; or
- add a materialization preflight-only backend if the UI and review process have been validated.

Direct signal materialization should wait until a preflight gate proves duplicate/evidence controls are adequate.
