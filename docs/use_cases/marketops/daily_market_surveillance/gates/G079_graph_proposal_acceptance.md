# G079 Graph Proposal Acceptance

Status: implemented backend boundary
Use case: MarketOps Daily Market Surveillance

## Goal

Create the first durable review boundary for MarketOps DSM graph target candidates.

G079 extracts graph target candidates from persisted `marketops_dsm_artifacts`, stores them as first-class graph proposal records, exposes read APIs, and supports explicit operator or rule decisions such as accepted/rejected.

## Inputs

- `signal_ledger.graph_targets`
- `marketops_dsm_artifacts.graph_targets`
- `marketops_dsm_artifacts.artifact_id`
- `marketops_dsm_artifacts.signal_id`
- MarketOps metadata: `app_id=marketops`, `domain=market_data`, `use_case=daily_market_surveillance`

## Deliverables

- Migration `000013_marketops_dsm_graph_proposals` for the first-class proposal ledger.
- Storage extraction/upsert logic from persisted DSM artifacts.
- Stable deterministic `proposal_id` generation.
- List/detail graph proposal APIs under `/v1/marketops/dsm/graph-proposals`.
- Decision endpoint and storage mutation for `proposed`, `accepted`, `rejected`, and `superseded` statuses.
- Unit tests for extraction, idempotent identity, malformed candidates, API filters, detail retrieval, and status validation.
- Build journal and gate audit updates.

## Acceptance Criteria

- A persisted G077 smoke artifact with five graph targets materializes five graph proposal rows.
- Replaying the same signal/artifact does not create duplicate proposal rows.
- Node and relationship candidates preserve their original detector-provided properties.
- Accepted/rejected decisions are durable and do not mutate immutable candidate identity fields.
- Existing signal, alert, insight, and artifact persistence behavior remains unchanged.
- Authenticated list/detail APIs can retrieve proposals by artifact id, signal id, symbol, candidate type, and status.

## Deferred Work

- Production graph database writes.
- Graph visualization or editing in DSM Workbench.
- Cross-artifact graph entity resolution beyond deterministic proposal id deduplication.
- Independent graph materialization service.

## Documentation Links

- Architecture: `../architecture/graph_proposal_acceptance.md`
- API: `../api/graph_proposal_api.md`
- Current artifact semantics: `../architecture/signal_artifact_persistence.md`
