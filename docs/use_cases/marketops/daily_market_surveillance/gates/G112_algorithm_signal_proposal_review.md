# G112 Algorithm Signal Proposal Review Lifecycle

Status: implemented
Timestamp: 2026-07-16T00:00:00Z

## Purpose

G112 adds an operator review lifecycle for `algorithm_signal_proposals`, the review boundary introduced in G111. Operators can mark a proposal as reviewed, rejected, superseded, or restore it to proposed. These decisions are audit metadata only.

## Implemented Scope

- Added migration `000025_algorithm_signal_proposal_review` with review metadata:
  - `reviewed_by`
  - `decision_note`
  - `decided_at`
  - `decision_metadata`
- Added storage mutation `MutateAlgorithmSignalProposal`.
- Added API endpoint:
  - `POST /v1/algorithms/signal-proposals/{proposal_id}/decision`
- Added response fields to proposal DTOs:
  - `reviewed_by`
  - `decision_note`
  - `decided_at`
- Updated auth route classification so the decision endpoint requires operator/admin role when auth is enabled.

## Decision Statuses

Valid statuses:

- `proposed`: candidate remains queued for review or is restored for reconsideration.
- `reviewed`: operator reviewed the proposal but did not promote it to any downstream materialized state.
- `rejected`: operator judged the proposal as not useful or not sufficiently supported.
- `superseded`: operator retained the row but marked it replaced by better evidence or a better proposal.

There is intentionally no `accepted` status in this gate. Acceptance into production signal materialization requires a later, explicit materialization gate.

## Explicitly Out Of Scope

- Production `signal.v1` writes.
- Alert or insight creation.
- Graph proposal creation.
- Frontend review workflow.
- Multi-row decision history.
- Runtime policy deployment.

## Validation

- Focused API tests cover decision mutation and invalid status rejection.
- Focused algorithm/generator tests continue to pass against the expanded repository interface.
- Full Go test suite passed.
- JSON schema validation passed.
- Gateway build passed.
- Local migration `000025_algorithm_signal_proposal_review` was applied.
- Authenticated live decision smoke marked proposal `algsigprop_c6c2acad697176d0f438b66e` as `reviewed`; DB verification confirmed reviewer metadata and decision timestamp.

## Next Gate

G113 should add read-only or review-capable frontend visibility for algorithm signal proposals. Production materialization should wait until operators have reviewed enough proposal quality.
