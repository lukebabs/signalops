# G111 Algorithm Result To Signal Proposal Ledger

Status: implemented
Timestamp: 2026-07-16T00:00:00Z

## Purpose

G111 implements the first reviewed boundary between persisted `algorithm_results` and future production signals. Algorithm outputs can now become durable signal proposals, but they do not write `signal.v1` rows, alerts, insights, graph proposals, or runtime policy changes.

## Implemented Substrate

Added `algorithm_signal_proposals` as an idempotent proposal ledger. Each row preserves:

- `proposal_id`
- `tenant_id`
- `algorithm_result_id`
- `execution_request_id`
- `algorithm_id` and `algorithm_version`
- `proposed_signal_type`
- `status` (`proposed`, `reviewed`, `rejected`, `superseded`)
- score, confidence, and severity
- proposal payload and rationale JSON
- source event ids and evidence refs
- correlation id and creator metadata

The table has a unique guard on `(tenant_id, algorithm_result_id, proposed_signal_type)` so reruns do not duplicate unchanged evidence.

## Generator

Added `signalops-algorithm-proposal-generator` as a bounded CLI process. It reads existing `algorithm_results`, applies filters, maps supported result types to candidate signal types, and inserts proposal rows idempotently.

Supported filters:

- tenant id
- algorithm id
- execution request id
- algorithm result id
- result type
- severity
- correlation id
- minimum confidence
- limit

## Candidate Signal Mapping

Initial mappings are deliberately conservative:

- `z_score`, `anomaly_score`, `online_anomaly_score`, and `isolation_score` propose `signalops.algorithm.anomaly_candidate`.
- `change_point_score` proposes `signalops.algorithm.change_point_candidate`.
- `forecast_residual` proposes `signalops.algorithm.forecast_deviation_candidate`.
- `classifier_label` proposes `signalops.algorithm.classification_candidate`.

These remain generic SignalOps algorithm proposal types. They are not automatically treated as MarketOps DSM taxonomy signals.

## Read APIs

Added read-only APIs:

- `GET /v1/algorithms/signal-proposals?tenant_id={tenant_id}&algorithm_id={algorithm_id}&execution_request_id={execution_request_id}&algorithm_result_id={algorithm_result_id}&status={status}&severity={severity}&correlation_id={correlation_id}&limit=50`
- `GET /v1/algorithms/signal-proposals/{proposal_id}?tenant_id={tenant_id}`

No create/update API was added for operators in this gate. Proposal generation is owned by the bounded CLI path.

## Explicitly Out Of Scope

- Writing production `signal.v1` rows.
- Creating alerts or insights.
- Graph proposals.
- Proposal accept/reject/supersede workflow.
- Frontend implementation.
- Automatic policy deployment.

## Validation

- Focused Go tests for proposal generation, stable IDs, idempotent reruns, confidence filtering, unsupported result skipping, and read API filter/get behavior.
- Full Docker build path ran `go test ./...` successfully.
- JSON schema validation passed.
- Local migration `000024_algorithm_signal_proposals` was applied.
- Live generator smoke inserted one proposal from `algexec-g110-ruptures-aapl-openclose`; the idempotency rerun inserted none.
- Authenticated read API smoke returned the persisted proposal through the rebuilt local gateway.

## Next Gate

G112 should add operator review lifecycle around `algorithm_signal_proposals` or read-only frontend visibility. Production signal materialization should wait until proposal quality can be inspected and reviewed.
