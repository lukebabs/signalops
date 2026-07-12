# G080 Operator Graph Proposal Review Workflow

Status: implemented
Use case: MarketOps Daily Market Surveillance

## Goal

Add a bounded operator review workflow for persisted MarketOps DSM graph proposals without writing to a production graph database.

G080 builds on G079 first-class `marketops_dsm_graph_proposals` records and the existing decision endpoint. Operators can mark a proposal `accepted`, `rejected`, `superseded`, or restore it to `proposed` from the DSM Workbench. The workflow updates proposal review metadata only.

## Inputs

- `marketops_dsm_graph_proposals.proposal_id`
- `marketops_dsm_graph_proposals.status`
- `marketops_dsm_graph_proposals.reviewed_by`
- `marketops_dsm_graph_proposals.decision_note`
- `marketops_dsm_graph_proposals.decided_at`
- Existing authenticated `POST /v1/marketops/dsm/graph-proposals/{proposal_id}/decision` endpoint

## Deliverables

- Frontend API client mutation for graph proposal decisions.
- React Query mutation hook with graph proposal cache refresh.
- DSM Workbench inline review controls in the expanded graph proposal row.
- Optional review note entry.
- Status-aware buttons for accept, reject, supersede, and restore-to-proposed.
- API client tests for decision endpoint path encoding, bearer auth, payload shape, and local auth-disabled actor fallback.
- Build journal and gate audit updates.

## Acceptance Criteria

- Operators can update proposal status from the existing MarketOps DSM Workbench graph proposal ledger.
- The UI sends only `status` and optional `note` to the decision endpoint.
- Auth-enabled requests rely on the bearer token for actor derivation.
- Auth-disabled local requests send the existing `X-SignalOps-Actor: operator-local` fallback.
- Successful mutations invalidate graph proposal list/detail caches so review metadata refreshes.
- The UI does not add graph visualization, graph editing, or production graph database writes.

## Deferred Work

- Production graph materialization.
- Graph database writes.
- Graph canvas or topology editor.
- Multi-operator review history beyond the current latest-decision metadata fields.
- Bulk proposal decisions.

## Validation

- `cd web && npm test`: passed.
- `cd web && npm test -- src/api/marketopsGraphProposals.test.ts`: passed.
- `cd web && npm run build`: passed.
- `cd web && npm audit --audit-level=low`: 0 vulnerabilities.
