# G120 Algorithm Signal Materialization Request Design

Status: proposed design
Timestamp: 2026-07-16T00:00:00Z

## Purpose

G120 defines the concrete backend architecture for the first future write path that can materialize reviewed `algorithm_signal_proposals` into production `signal.v1` rows.

This is a design gate only. It does not implement database migrations, APIs, workers, signal writes, alerts, insights, graph proposals, frontend controls, policy deployment, or Syncratic integration.

## Current Boundary

The current implemented stack is:

1. `algorithm_results`: immutable algorithm output ledger.
2. `algorithm_signal_proposals`: review candidate ledger.
3. proposal decision API: operator can mark proposals `proposed`, `reviewed`, `rejected`, or `superseded`.
4. proposal summary API: review coverage and high/critical unreviewed visibility.
5. materialization preflight API: read-only readiness, duplicate-risk, blocked, invalid, and would-write counts.
6. frontend preflight spec: read-only operator visibility over preflight results.

No implemented path writes production signals from algorithm proposals.

## Design Decision

Do not overload `algorithm_signal_proposals` as the materialization execution ledger.

G120 recommends adding a separate `algorithm_signal_materializations` ledger and treating materialization as an explicit request with auditable execution state.

This keeps three concepts separate:

- proposal review: human/operator review of candidate evidence;
- materialization request: bounded intent to write production signals;
- production signal lifecycle: existing `signal_ledger` and downstream alert/insight derivation.

## Allowed Source State

For the first implementation, only proposals with status `reviewed` should be eligible for materialization request creation.

Do not add `accepted` yet. G112 intentionally avoided acceptance semantics, and G118 already treats reviewed proposals as readiness candidates. A later gate can add `approved_for_materialization` if operator workflow needs a second explicit decision before write.

Rejected and superseded proposals must be blocked.

Proposed/unreviewed proposals must be blocked.

## Materialization Ledger

Add a first-class ledger:

`algorithm_signal_materializations`

Recommended columns:

- `materialization_id` text primary key with `algmat_` prefix;
- `tenant_id` text not null;
- `proposal_id` text not null;
- `algorithm_result_id` text not null;
- `execution_request_id` text not null;
- `algorithm_id` text not null;
- `algorithm_version` text not null;
- `proposed_signal_type` text not null;
- `signal_id` text nullable until successful write;
- `materialization_status` text not null;
- `materialization_policy_version` text not null;
- `idempotency_key` text not null;
- `duplicate_of_signal_id` text nullable;
- `requested_by` text not null;
- `requested_at` timestamptz not null;
- `started_at` timestamptz nullable;
- `completed_at` timestamptz nullable;
- `failed_at` timestamptz nullable;
- `error_code` text nullable;
- `error_message` text nullable;
- `request_metadata` jsonb not null default `{}`;
- `preflight_snapshot` jsonb not null default `{}`;
- `signal_payload_preview` jsonb not null default `{}`;
- `created_at` timestamptz not null default now();
- `updated_at` timestamptz not null default now().

Recommended uniqueness:

- unique `(tenant_id, idempotency_key)`;
- unique `(tenant_id, proposal_id, materialization_policy_version)` where status is not terminal duplicate/superseded, if Postgres partial indexes are acceptable;
- index `(tenant_id, materialization_status, updated_at desc)`;
- index `(tenant_id, proposal_id)`;
- index `(tenant_id, signal_id)`.

## Materialization Statuses

Use execution-state names, not review-state names:

- `requested`: request accepted, no write attempted yet;
- `running`: worker/API is actively performing duplicate checks and write;
- `succeeded`: production signal was written;
- `duplicate`: no signal written because an existing signal already covers the evidence;
- `blocked`: request failed preflight or policy checks before writing;
- `failed`: unexpected execution failure;
- `superseded`: replaced by a later materialization request or policy version.

Do not mutate proposal status to `materialized` in the first implementation. The materialization ledger should be the source of truth for write state.

## Idempotency

Materialization must be idempotent across retries and duplicate submissions.

Recommended idempotency key:

`sha256(tenant_id | proposal_id | materialization_policy_version | sorted_source_event_ids | proposed_signal_type)`

The API may accept a client-provided idempotency key only if it is namespaced and validated, but the server-derived key should remain canonical for the first implementation.

If the same request is repeated:

- return the existing materialization row;
- do not write another signal;
- do not create another alert/insight path.

## Stable Signal ID

Recommended production `signal_id`:

`sig_alg_` + first 24 hex chars of `sha256(tenant_id | proposal_id | materialization_policy_version | sorted_source_event_ids | proposed_signal_type)`

This intentionally aligns with the materialization idempotency inputs so a retry computes the same signal target.

## API Shape

First write-path implementation should expose two API groups:

### Single Proposal Preflight

`GET /v1/algorithms/signal-proposals/{proposal_id}/materialization-preflight?tenant_id={tenant_id}&min_reviewed_ratio=1&policy_version=algorithm_materialization.v1`

Purpose:

- inspect one proposal;
- include duplicate-risk and payload preview;
- do not write.

### Single Proposal Materialization Request

`POST /v1/algorithms/signal-proposals/{proposal_id}/materializations?tenant_id={tenant_id}`

Request body:

```json
{
  "tenant_id": "tenant-local",
  "policy_version": "algorithm_materialization.v1",
  "requested_by": "operator-local",
  "idempotency_key": "optional-client-request-key",
  "metadata": {
    "note": "Materialize reviewed algorithm candidate after preflight."
  }
}
```

Response envelope:

```json
{
  "algorithm_signal_materialization": {
    "materialization_id": "algmat_...",
    "tenant_id": "tenant-local",
    "proposal_id": "algsigprop_...",
    "materialization_status": "succeeded",
    "signal_id": "sig_alg_...",
    "duplicate_of_signal_id": "",
    "materialization_policy_version": "algorithm_materialization.v1",
    "idempotency_key": "...",
    "requested_by": "operator-local",
    "requested_at": "2026-07-16T00:00:00Z",
    "completed_at": "2026-07-16T00:00:00Z",
    "error_code": "",
    "error_message": ""
  }
}
```

### Materialization Reads

`GET /v1/algorithms/signal-materializations?tenant_id={tenant_id}&proposal_id={proposal_id}&status={status}&limit=50`

`GET /v1/algorithms/signal-materializations/{materialization_id}?tenant_id={tenant_id}`

Bulk materialization should not be part of the first write-path gate.

## Authorization

All materialization mutation routes must require operator/admin authorization when auth is enabled.

Read routes can follow existing algorithm read authorization patterns.

Mutation audit fields must include:

- authenticated actor when available;
- request-provided actor only as fallback where current local-dev patterns require it;
- request timestamp;
- policy version;
- idempotency key.

Never persist bearer tokens, auth headers, client secrets, or raw environment variables in materialization metadata.

## Preflight Enforcement

The materialization mutation must re-run server-side checks immediately before writing.

Do not trust a prior G118 UI/API preflight response as authorization to write.

Required blockers:

- missing tenant id;
- missing proposal;
- proposal status is not `reviewed`;
- proposal source event ids are empty;
- proposal payload JSON invalid;
- rationale JSON invalid;
- source algorithm result missing;
- source result tenant mismatch;
- source result algorithm id mismatch;
- source result execution request mismatch;
- existing signal duplicate with same tenant, signal type, and overlapping source event id;
- reviewed ratio below threshold when global threshold is enforced;
- high/critical unreviewed proposal count above threshold when global threshold is enforced.

A duplicate should produce `materialization_status=duplicate`, record `duplicate_of_signal_id`, and avoid writing a new signal.

A policy/preflight blocker should produce `materialization_status=blocked` and avoid writing a signal.

Unexpected write errors should produce `materialization_status=failed`.

## Production Signal Payload Mapping

Initial materialized signal types should remain generic:

- `signalops.algorithm.anomaly_candidate`
- `signalops.algorithm.change_point_candidate`
- `signalops.algorithm.forecast_deviation_candidate`
- `signalops.algorithm.classification_candidate`

Do not map to MarketOps DSM taxonomy in the first implementation.

Recommended `signal_ledger` mapping:

- `tenant_id`: proposal tenant.
- `signal_id`: stable `sig_alg_...` id.
- `signal_type`: proposal `proposed_signal_type`.
- `detector_id`: proposal `algorithm_id`.
- `detector_version`: proposal `algorithm_version`.
- `model_version`: materialization policy version or algorithm version, depending on existing ledger convention.
- `event_ids`: proposal source event ids.
- `confidence`: proposal confidence.
- `severity`: proposal severity.
- `correlation_id`: proposal correlation id.
- `causation_id`: proposal id.
- `trace_id`: execution request id or correlation id.
- `supporting_metrics`: compact metrics from proposal payload and algorithm result payload.
- `semantic_evidence`: include algorithm id/version, result id, proposal id, score, confidence, severity, review metadata, and materialization id.
- `evidence`: include normalized event refs, algorithm result id, proposal id, materialization id, and policy version.
- `entities`: derive from proposal payload or normalized event evidence where available.
- `app_id`, `domain`, `use_case`, `source_adapter`, `dataset`: derive from normalized source events where available; otherwise use conservative generic SignalOps metadata and mark missing values in evidence.

If required production signal fields cannot be derived safely, block materialization rather than emitting a low-quality row.

## Alerts And Insights Boundary

The materialization path should only write the production signal row.

Existing signal persistence/lifecycle behavior remains responsible for alert/insight derivation. The materializer must not directly write alert or insight rows.

If the existing lifecycle creates one alert for the materialized signal, that is acceptable. It should not create an insight unless existing multi-event insight logic justifies it.

## Worker Versus Inline Write

First implementation can choose either:

- synchronous API transaction for one proposal; or
- request row plus worker execution.

Recommended first implementation: synchronous single-proposal materialization inside the API transaction after preflight.

Reason:

- bounded blast radius;
- simpler operator feedback;
- easier idempotency tests;
- no new worker lifecycle yet.

A later gate can introduce async/bulk workers after single-proposal semantics are validated.

## Transactions

The implementation should use a database transaction that:

1. creates or retrieves materialization row by idempotency key;
2. locks the proposal row or otherwise ensures consistent read;
3. re-runs preflight checks;
4. checks duplicate signal evidence;
5. writes signal row if eligible;
6. updates materialization row to terminal status.

If signal ledger lives in temporal/Timescale storage while materialization ledger lives in Postgres, document whether cross-database atomicity is possible. If not, implement compensating idempotency and retry behavior rather than pretending both writes are atomic.

## Testing Requirements

A future implementation must include tests for:

- migration up/down;
- materialization ledger insert/read/list;
- idempotent repeated request returns the same materialization;
- stable signal id generation;
- duplicate signal detection blocks second signal write;
- proposed/rejected/superseded proposals cannot materialize;
- missing source events block materialization;
- missing source result blocks materialization;
- tenant mismatch blocks materialization;
- invalid JSON blocks materialization;
- successful signal payload includes required lineage;
- no direct alert/insight/graph writes;
- auth-gated mutation route;
- read routes remain accessible per existing auth policy;
- live smoke with one reviewed, duplicate-free proposal.

## Implementation Gate Recommendation

G121 should implement the storage ledger and read-only materialization list/detail APIs first, without writing production signals.

G122 should implement single-proposal materialization mutation and signal write after G121 storage/read APIs are validated.

This staged path avoids combining schema, read APIs, mutation semantics, and production signal writes in one gate.

## Explicitly Out Of Scope For G120

- Database migrations.
- API implementation.
- Signal ledger writes.
- Alert or insight creation.
- Graph proposal creation.
- Frontend changes.
- Bulk materialization.
- Async materialization worker.
- Policy deployment.
- MarketOps DSM taxonomy remapping.
- Syncratic integration.
