# Graph Proposal Acceptance

Status: proposed for G079  
Use case: MarketOps Daily Market Surveillance

## Purpose

MarketOps DSM signals currently carry graph target candidates as evidence. G079 should introduce the first backend boundary that turns those candidates into reviewable, durable graph proposals.

This is not graph mutation. It is an acceptance/storage layer for graph proposal records derived from persisted DSM artifacts and signal graph targets.

## Current State

The detector emits `graph_targets` on `signal.v1` payloads. The current MarketOps taxonomy detector emits candidates such as:

- `node_candidate` for a ticker node like `ticker:AAPL`
- `node_candidate` for a DSM signal type node like `signal_type:marketops.dsm.pinning_risk`
- `node_candidate` for an artifact node like `artifact:{artifact_id}`
- `relationship_candidate` from ticker to signal type
- `relationship_candidate` from signal type to artifact

G077 persists those candidates as JSON inside `marketops_dsm_artifacts.graph_targets`. G078 displays them through DSM Workbench artifact context. There is no first-class proposal status, reviewer state, deduplication ledger, or accepted/rejected history yet.

## Proposed G079 Boundary

Add a first-class graph proposal ledger that stores one row per proposed node or relationship candidate.

The ledger should be derived from persisted MarketOps DSM artifacts, not directly from raw normalized events. `marketops_dsm_artifacts` remains the source artifact record, while the new graph proposal ledger becomes the operator/review boundary for graph candidates.

A proposal row should preserve:

- stable proposal id
- tenant/app/domain/use-case metadata
- artifact id and signal id
- candidate type: `node_candidate` or `relationship_candidate`
- candidate identity fields, such as `node_id` or `from`/`relationship`/`to`
- labels and properties JSON
- confidence, severity, signal type, detector id, subject symbol, and source event ids
- status, reviewer metadata, decision timestamps, and rejection/acceptance notes

## Status Model

Use a small explicit lifecycle:

- `proposed`: extracted and awaiting review
- `accepted`: accepted for later graph materialization
- `rejected`: rejected by an operator or rule
- `superseded`: replaced by a newer deterministic proposal

G079 should not write to a graph database. An `accepted` proposal means the proposal is approved for a later graph materialization gate.

## Stable Identity

Proposal ids should be deterministic so replaying the same signal/artifact is idempotent.

Recommended identity inputs:

- artifact id
- signal id
- candidate type
- node id for node candidates
- from, relationship, and to for relationship candidates

The storage layer should upsert on proposal id. Replays can refresh evidence and timestamps without creating duplicates.

## Extraction Rules

Only extract proposals from MarketOps Daily Market Surveillance artifacts:

- `app_id=marketops`
- `domain=market_data`
- `use_case=daily_market_surveillance`
- `artifact_type=marketops.dsm.signal_artifact.v1`

Ignore malformed candidates rather than failing signal persistence. Invalid candidates should be counted or logged so quality issues are visible during validation.

## Non-Goals

G079 should not:

- create or mutate a production graph database
- infer new relationships beyond the detector-provided candidates
- replace `signal_ledger` or `marketops_dsm_artifacts`
- add frontend graph editing unless a separate frontend task is explicitly scoped

## Validation Expectations

A complete G079 backend should prove:

- migrations create and rollback the graph proposal ledger
- persisted artifacts with five current detector graph targets produce five proposal rows
- replaying the same artifact is idempotent
- malformed graph targets do not break signal persistence
- list/detail APIs enforce auth and tenant scoping
- accepted/rejected status updates are durable and auditable
