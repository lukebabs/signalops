# Tenant-local Legacy Hot Materialization Evidence — 2026-08-16

Status: append-only global evidence materialization passed; no Gateway-reader or scheduler activation.

## Scope

This follows the immutable parity manifest `subglobalparity_d72978bedcbace8096a8d305`. It moves only mapped legacy evidence to the platform-owned analytical ledger. It does not modify `tenant-local` source rows, memberships, watchlists, schedules, provider activity, or API behavior.

## Risk/Reward parity and materialization

The existing materializer appended the 1,533 Risk/Reward entries not previously represented globally:

- evidence run: `subglobalevrun-1b61c016333fdd3d8e8f3172`
- source scope/execution: `legacy_materialization` / `legacy_materialized`
- algorithm: `signalops.algorithms.risk_reward_temporal_v1` v1
- newly inserted: 1,533
- session range: 2026-07-14 through 2026-08-05

The resulting global restricted Risk/Reward reader now has exact source parity:

| Measure | Result |
| --- | ---: |
| mapped legacy source rows | 2,533 |
| matching global records by canonical asset, session, kind, and fingerprint | 2,533 |
| missing global records | 0 |
| distinct assets | 132 |
| session range | 2026-07-14 through 2026-08-14 |

## Intraday current-state materialization

The restricted materializer was extended to accept `intraday_snapshot` only through `subscriber_global_marketops_legacy_parity_source_v3`; it still fails if its role has direct access to `marketops_intraday_condition_snapshots`.

- evidence run: `subglobalevrun-b68b5885600d6102eb3f6b23`
- source scope/execution: `legacy_materialization` / `legacy_materialized`
- algorithm: `marketops.intraday_conditions` v1
- inserted: 132
- mapped/matched/missing: `132 / 132 / 0`
- logical MarketOps state date: 2026-08-14
- `payload.current_only_source`: true for all 132 records

The source is deliberately current-only. The original `as_of_time`, status, stale marker, conditions, source payload, fingerprint, validation contract, immutable baseline, and manifest provenance are retained in the central record.

### Freshness contract

Legacy snapshots are updated in place and retain their initial row `created_at`. Consequently, the generic evidence-record `observed_at` carries the legacy creation timestamp and is **not** a valid intraday freshness value for this imported evidence. Any later global intraday reader must use the immutable `payload.as_of_time` as its freshness and ordering field, label the result as a current-state snapshot, and never infer a historical intraday timeline from it.

## Safety proofs

- The tenant-local default still contains 132 members.
- Global Risk/Reward materialization is exact, with zero missing source fingerprints.
- Central intraday evidence now contains 132 records, but no Gateway intraday projection, route, or scheduler exists.
- No provider call, active task, user policy, or UI behavior changed.

## Remaining release gates

1. Add a restricted global intraday latest-state projection that orders by `payload.as_of_time`, carries the current-only disclosure, and is separately browser-tested.
2. Define and seed the 132-symbol grandfathered hot cohort; dual-run it with the new watchlist-derived global hot selector.
3. Prove membership, freshness, and writer parity across the two paths before switching a scheduler or UI view.
4. Retire the legacy intraday serving path only after the global projection and dual-run evidence are accepted. Immutable legacy evidence must remain recoverable.
