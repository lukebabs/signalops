# G121 Algorithm Signal Materialization Ledger Reads

Status: implemented
Timestamp: 2026-07-16T00:00:00Z

## Purpose

G121 implements the storage ledger and read-only API surface for future algorithm signal materialization requests.

This gate does not create materialization requests, write production `signal.v1` rows, create alerts or insights, create graph proposals, add frontend behavior, deploy policy, or integrate Syncratic.

## Storage

Added migration `000026_algorithm_signal_materializations`.

The new ledger is:

`algorithm_signal_materializations`

It records:

- materialization id;
- tenant and proposal lineage;
- algorithm result and execution request lineage;
- proposed signal type;
- materialization status;
- materialization policy version;
- idempotency key;
- optional signal id;
- optional duplicate signal id;
- request actor and timestamps;
- optional lifecycle timestamps;
- error code/message;
- request metadata;
- preflight snapshot;
- signal payload preview.

The migration adds indexes for status, proposal, and signal lookup, plus a unique `(tenant_id, idempotency_key)` constraint.

## APIs

Added read-only endpoints:

- `GET /v1/algorithms/signal-materializations?tenant_id={tenant_id}&proposal_id={proposal_id}&algorithm_result_id={algorithm_result_id}&execution_request_id={execution_request_id}&algorithm_id={algorithm_id}&status={status}&signal_id={signal_id}&limit=50`
- `GET /v1/algorithms/signal-materializations/{materialization_id}?tenant_id={tenant_id}`

Response envelopes:

- `algorithm_signal_materializations`
- `algorithm_signal_materialization`

## Boundary

G121 intentionally does not add a `POST` route.

Rows will be written by a future materialization implementation gate after the write-path semantics are implemented and validated.

## Validation

Implemented validation covers:

- storage model and repository compilation;
- read-only list route filter propagation;
- detail route response shape;
- JSON metadata fields in responses;
- adjacent algorithm/proposal package compile compatibility.
