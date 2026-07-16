# G122 Algorithm Signal Materialization Write Path

Status: implemented
Timestamp: 2026-07-16T00:00:00Z

## Purpose

G122 implements the first bounded write path from one reviewed `algorithm_signal_proposal` to one production signal ledger row.

The implementation is intentionally single-proposal only. It does not add bulk materialization, async workers, frontend controls, policy deployment, DSM taxonomy remapping, Syncratic integration, graph proposal writes, or direct alert/insight writes.

## Endpoint

`POST /v1/algorithms/signal-proposals/{proposal_id}/materializations?tenant_id={tenant_id}`

Request body:

```json
{
  "tenant_id": "tenant-local",
  "policy_version": "algorithm_materialization.v1",
  "requested_by": "operator-local",
  "idempotency_key": "optional-client-key",
  "metadata": { "note": "operator note" }
}
```

Response envelope:

- `algorithm_signal_materialization`

## Behavior

The route:

1. Loads the proposal.
2. Computes stable materialization id, idempotency key, and signal id from tenant, proposal id, policy version, source event ids, and proposed signal type.
3. Returns an existing materialization on retry.
4. Re-runs server-side preflight using proposal, source result, proposal summary, and existing signals.
5. Records `duplicate` when evidence overlaps an existing signal and does not write a second signal.
6. Records `blocked` when preflight blocks or invalidates the proposal and does not write a signal.
7. Records `running`, writes one production signal ledger row with generic algorithm signal metadata, then records `succeeded` when eligible.
8. Records `failed` if signal writing fails.

## Signal Boundary

G122 writes the signal ledger through `UpsertSignalLedger` only.

It intentionally does not call `PersistSignalLifecycle`, so it does not directly create alerts, insights, DSM artifacts, or graph proposals.

Any future lifecycle derivation from materialized algorithm signals must be designed explicitly in a later gate.

## Signal Mapping

Materialized signals remain generic SignalOps algorithm signals:

- `signalops.algorithm.anomaly_candidate`
- `signalops.algorithm.change_point_candidate`
- `signalops.algorithm.forecast_deviation_candidate`
- `signalops.algorithm.classification_candidate`

The signal payload records:

- proposal id as causation id;
- execution request id as trace id;
- algorithm id/version as detector id/version;
- materialization policy version as model version;
- proposal source event ids;
- proposal confidence and severity;
- proposal/result/materialization lineage in semantic evidence and evidence JSON.

## Authorization

The materialization POST route is an operator/admin mutation when auth is enabled.

## Validation

Implemented validation covers:

- successful reviewed proposal materialization writes one signal and one succeeded materialization;
- idempotent retry returns the existing materialization without duplicate signal/materialization rows;
- duplicate evidence records `duplicate` and writes no second signal;
- unreviewed proposals record `blocked` and write no signal;
- focused API/storage/algorithm package tests;
- full Go test suite;
- JSON schema validation;
- gateway Docker build.

## Explicitly Out Of Scope

- Bulk materialization.
- Async materialization workers.
- Frontend materialization controls.
- Direct alert or insight writes.
- Graph proposal writes.
- DSM taxonomy remapping.
- Policy deployment.
- Syncratic integration.
