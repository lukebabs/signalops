# G081 Back-Test Substrate

Status: specification proposed
Use case: MarketOps Daily Market Surveillance

## Goal

Define the first MarketOps back-test substrate before implementation.

G081 is a documentation and architecture gate only. It specifies how SignalOps should run bounded historical MarketOps DSM back-tests for policy calibration without mutating operational ledgers or writing to a production graph database.

The first objective is not trading simulation, PnL analysis, ML model training, or production graph materialization. The first objective is to evaluate whether deterministic graph-proposal review policy can safely reduce manual review volume using historical normalized events.

## Inputs

- Historical `normalized_event_ledger` rows for MarketOps.
- MarketOps metadata:
  - `app_id=marketops`
  - `domain=market_data`
  - `use_case=daily_market_surveillance`
- Initial datasets:
  - `equity_eod_prices`
  - `options_contracts_daily`
- Optional bounded filters:
  - tenant
  - source id
  - observation window
  - symbol list
  - universe group
  - detector id/version
  - max records

## Conceptual Outputs

G081 proposes separate isolated back-test ledgers for future implementation:

- Back-test run records.
- Generated signal records for the run.
- Generated DSM artifact records for the run.
- Generated graph proposal records for the run.
- Policy evaluation records or summaries.
- Aggregate run metrics.

These records must be separate from production ledgers. A back-test must not write to:

- `signal_ledger`
- `alert_ledger`
- `insight_ledger`
- `marketops_dsm_artifacts`
- `marketops_dsm_graph_proposals`
- any production graph database

## Proposed Flow

1. An operator requests a bounded MarketOps back-test run.
2. The runner reads historical normalized events for the requested window and dataset.
3. The runner executes the current MarketOps DSM detector path in a back-test context.
4. The runner extracts signal artifacts and graph proposal candidates using the same deterministic semantics as production.
5. The runner evaluates each proposal against a versioned deterministic policy pack.
6. The runner stores isolated outputs and metrics under a back-test run id.
7. Operators review aggregate results before any future automation policy is promoted.

## First Policy Calibration Model

The initial policy evaluator should be deterministic. It should produce recommendations, not production graph writes.

Initial recommendation classes:

- `auto_accept_candidate`
- `auto_reject_candidate`
- `manual_review_required`
- `supersede_candidate`

Initial policy examples:

- Auto-accept valid node candidates for known MarketOps assets and known DSM signal types.
- Auto-reject malformed candidates missing required node or relationship identity.
- Require manual review for unknown labels, unknown relationship types, low confidence, new detector versions, or unsupported datasets.
- Mark stale or replaced candidates as supersede candidates only when a newer proposal dominates the same graph fact inside the run.

## First Metrics

Each back-test run should produce at least:

- normalized events scanned
- normalized events skipped
- signals generated
- artifacts generated
- graph proposals generated
- auto-accept candidates
- auto-reject candidates
- manual-review-required candidates
- supersede candidates
- failures
- detector mix
- dataset mix
- symbol coverage
- confidence-band distribution

## Acceptance Criteria For This Spec Gate

- The documentation clearly separates back-testing from operational replay.
- The first implementation target is normalized-event based, not raw-provider based.
- Back-test outputs are explicitly isolated from operational ledgers.
- Production graph writes remain deferred.
- Policy calibration is deterministic in the first implementation.
- The next implementation gate can build a thin MVP runner without choosing architecture.

## Deferred Work

- Raw provider re-pulls.
- Raw-event replay through normalization.
- Trading strategy simulation or PnL.
- ML model training.
- Full back-test UI.
- Bulk graph proposal decisions.
- Production graph materialization.
- Graph database writes.

## Recommended Next Gate

G082 should implement a thin MVP back-test runner and storage boundary based on this specification after review approval.
