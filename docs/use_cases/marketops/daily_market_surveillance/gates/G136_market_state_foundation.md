# G136 Market State Foundation

Status: implemented backend foundation

Date: 2026-07-19

## Purpose

G136 creates the first persisted Market State Intelligence abstractions described by `docs/design/syncratic_marketops_market_state_intelligence_architecture_v1.md`.

The gate establishes versioned, idempotent storage and read-only APIs for feature definitions, feature observations, canonical market states, state transitions, and reusable evidence. It does not yet calculate those objects from provider data.

## Implemented Scope

- Migration `000028_marketops_market_state_foundation` creates five tenant-aware ledgers.
- Shared quality states are explicit: `usable`, `usable_with_warning`, `partial`, `sparse`, `stale`, `invalid`, `missing`, and `not_applicable`.
- Feature observations support numeric, text, or boolean values with dimensions, quality details, source events, source artifacts, and calculation-run lineage.
- Market states retain exact feature-observation IDs, schema version, completeness, quality, eligible-hypothesis references, build run, and deterministic key.
- State transitions retain current/baseline state references, lookback, values, rarity statistics, persistence, direction, quality, and calculation lineage.
- Evidence records retain versioned statements, domain/direction scores, payload, source feature IDs, source transition IDs, and deterministic keys.
- PostgreSQL repository upserts use deterministic-key conflicts for idempotent rebuilds.
- Shared identity utilities generate stable kind-scoped IDs and deterministic keys using canonical components; JSON dimensions are canonicalized before identity generation.
- Read endpoints expose filters by tenant, asset, symbol, session range, version, domain, quality, type, and dimensions where applicable.
- State lineage resolves referenced feature observations and aggregates source event/artifact IDs while reporting missing references explicitly.

## Read API

- `GET /v1/marketops/features/definitions`
- `GET /v1/marketops/features/observations`
- `GET /v1/marketops/states`
- `GET /v1/marketops/states/{market_state_id}`
- `GET /v1/marketops/states/{market_state_id}/lineage`
- `GET /v1/marketops/transitions`
- `GET /v1/marketops/evidence`
- `GET /v1/marketops/evidence/{evidence_id}`

Session filters use `session_start=YYYY-MM-DD` and `session_end=YYYY-MM-DD`. Feature-observation dimensions use a JSON object query value and apply containment matching.

## Safety Boundary

G136 does not add:

- provider acquisition or provider calls;
- automatic scheduling or Top 50 fanout;
- feature, state, transition, or evidence materialization endpoints;
- hypothesis evaluation;
- signal proposals or production signal writes;
- graph mutation;
- Syncratic context changes;
- frontend work.

Existing proposal review, signal materialization, graph review, quality gates, and evidence-purity controls remain unchanged.

## Validation

- Focused API and PostgreSQL repository tests cover filters, DTOs, lineage resolution, missing lineage, invalid date/dimension input, typed-value validation, quality states, score bounds, completeness, and JSON shapes.
- Full Go test suite passes in the repository Go 1.22 container.
- JSON schema validation remains unchanged and passes.
- Migration applies cleanly to the local SignalOps PostgreSQL instance.
- Fresh-schema apply/down validation confirms rollback order and tenant-scoped transition foreign keys.
- Rebuilt gateway passed its embedded Go test stage; authenticated feature-definition list smoke returned HTTP `200` with the expected empty ledger envelope.

## Next Gate

G137 should implement one bounded AAPL vertical slice from existing persisted equity/options evidence into feature observations, one canonical market state, state transitions, and evidence. It must prove point-in-time lineage, deterministic rebuilds, and quality blocking before any hypothesis evaluator is introduced.
