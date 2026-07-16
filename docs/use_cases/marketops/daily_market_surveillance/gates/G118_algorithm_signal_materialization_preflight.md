# G118 Algorithm Signal Materialization Preflight

Status: implemented
Timestamp: 2026-07-16T00:00:00Z

## Purpose

G118 implements the first backend safety check for future algorithm signal materialization. It adds a read-only preflight API that evaluates reviewed `algorithm_signal_proposals` and reports whether they are ready for a later materialization gate.

G118 does not write production `signal.v1` rows. It does not create alerts, insights, graph proposals, materialization ledger rows, policy deployments, or frontend changes.

## Endpoint

`GET /v1/algorithms/signal-proposals/materialization-preflight?tenant_id={tenant_id}&algorithm_id={algorithm_id}&execution_request_id={execution_request_id}&algorithm_result_id={algorithm_result_id}&status={status}&severity={severity}&correlation_id={correlation_id}&limit=200&min_reviewed_ratio=1&policy_version=materialization_preflight.v1`

The response envelope is:

`algorithm_signal_materialization_preflight`

## Inputs

The endpoint reuses the existing proposal filters:

- `tenant_id` is required.
- `algorithm_id` is optional.
- `execution_request_id` is optional.
- `algorithm_result_id` is optional.
- `status` is optional.
- `severity` is optional.
- `correlation_id` is optional.
- `limit` defaults to 200 and is capped by the existing API helper.

Additional preflight parameters:

- `min_reviewed_ratio` defaults to `1`.
- `policy_version` defaults to `materialization_preflight.v1`.

## Checks

For each proposal, the preflight checks:

- proposal status is `reviewed`;
- source event ids are present;
- proposal payload JSON is valid;
- rationale JSON is valid;
- source `algorithm_result` exists;
- source result tenant, algorithm id, and execution request match the proposal;
- no existing production signal has the same tenant, proposed signal type, and overlapping source event id.

At the aggregate level, the preflight checks:

- reviewed ratio meets `min_reviewed_ratio`;
- high/critical unreviewed proposal count is zero.

## Response Semantics

Top-level counts include:

- `total_proposals`
- `eligible_count`
- `duplicate_risk_count`
- `blocked_count`
- `invalid_count`
- `would_write_count`
- `reviewed_ratio`
- `min_reviewed_ratio`
- `review_coverage_satisfied`
- `high_critical_unreviewed_count`
- `global_blocking_reasons`
- `item_reason_counts`

Each item includes:

- proposal lineage and proposed signal type;
- `preflight_status`: `eligible`, `duplicate_risk`, `blocked`, or `invalid`;
- reason tokens;
- duplicate signal ids when found;
- source event ids;
- `would_write`.

`would_write` remains false whenever global blockers exist, even for otherwise clean reviewed proposals.

## Boundary

This gate intentionally remains read-only. A later materialization implementation still needs explicit operator approval semantics, a materialization ledger, idempotent writes, and production signal payload tests before any algorithm proposal can become a production signal.

## Validation

Implemented validation covers:

- eligible reviewed proposal blocked by global high/critical unreviewed evidence;
- duplicate-risk detection against existing signal ledger event overlap;
- unreviewed proposal blocking;
- invalid proposal detection for missing evidence/source result;
- filter propagation and response counts.
