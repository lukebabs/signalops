# Tenant-local Legacy Hot Parity Foundation — 2026-08-16

Status: applied and inventory-complete; no global materialization or serving cutover has occurred.

## Decision

The existing `tenant-local` 132-symbol MarketOps universe remains that tenant’s durable default list and its mature data must be preserved as central, canonical-identity-linked evidence before any global hot-intraday selector replaces legacy scheduling.

This slice establishes only the first preservation boundary:

- `000141_subscriber_tenant_local_legacy_default_preservation` continues to preserve the 132 list memberships and their original order.
- `000142_subscriber_tenant_local_legacy_hot_parity_foundation` adds a security-barrier parity source for the 132 `all_active` intraday **current states** and permits immutable manifest entries of kind `intraday_snapshot`.
- The controlled parity worker retains its restricted role. It can read the fixed parity view and append manifests, but cannot read `marketops_intraday_condition_snapshots` directly.

No provider was called. No legacy row was changed. No global analytical evidence was written. No Gateway reader, dashboard, scheduler, tenant policy, entitlement, or hot-list selection behavior changed.

## Source inventory

At execution, `tenant-local` contained:

| Source | Scope | Preserved source state |
| --- | --- | --- |
| `marketops_universal_assets` | active legacy universe | 132 distinct symbols |
| `marketops_intraday_condition_snapshots` | `all_active` only | 132 current snapshots, all dated 2026-08-14 |
| `marketops_risk_reward_snapshots` | legacy tenant | 2,533 rows across 132 symbols, sessions 2026-07-14 through 2026-08-14 |

The intraday table is explicitly current-only. Its 132 manifest entries attest to the retained final state, source payload, `as_of_time`, market status, stale flag, condition array, algorithm/version, and source fingerprint. They do **not** claim to reconstruct historical intraday progression.

## Applied evidence

Migration `000142_subscriber_tenant_local_legacy_hot_parity_foundation` was applied to the dedicated MarketOps primary at `2026-08-16T21:57:36Z`.

The controlled run below was immutable-manifest-only:

- parity run: `subglobalparity_d72978bedcbace8096a8d305`
- correlation: `legacy-hot-132-parity-foundation-2026-08-16`
- selected/mapped/unmapped/ambiguous: `1665 / 1665 / 0 / 0`
- manifest fingerprint: `sha256:c098fc94dc9d767b1c66a6187f38dfac056c8035409154ffd22209de8fd44b19`

| Evidence kind | Entries newly manifested | Session range | Mapping result |
| --- | ---: | --- | --- |
| `intraday_snapshot` | 132 | 2026-08-14 | 132 mapped |
| `risk_reward` | 1,533 | 2026-07-14 to 2026-08-05 | 1,533 mapped |

Together with the previously retained 1,000 Risk/Reward manifest entries, the immutable manifest now covers all 2,533 legacy Risk/Reward source rows. A verification run selected zero additional records.

The global Risk/Reward reader still contains only 1,000 materialized records. This is intentional and visible: the remaining 1,533 manifests are `pending_global_materialization`; a manifest is not a serving projection.

## Security proof

After migration, the controlled worker role reported:

- `SELECT` on `subscriber_global_marketops_legacy_parity_source_v3`: allowed.
- direct `SELECT` on `marketops_intraday_condition_snapshots`: denied.

The view is fixed to `tenant-local`, `all_active`, and the active 132-symbol legacy universe. It is owned by the no-login migrator role; `PUBLIC` has no access.

## Follow-on gates

1. Materialize the 1,533 mapped Risk/Reward records through the existing append-only materializer; prove exact session/symbol parity before exposing any broadened reader.
2. Define a separate intraday evidence materializer and latest-state reader contract. It must retain `current_only_source=true`, preserve source fingerprint/provenance, and make no historical-coverage claim.
3. Establish an explicit 132-symbol grandfathered hot cohort, then dual-run it against the future global watchlist-derived hot selector. No scheduler cutover is allowed before count, membership, freshness, and write-parity evidence passes.
4. Only after the above, switch intraday/dashboard consumers to the global evidence projection and browser-contract test both the legacy tenant default and private-list behavior.

Rollback is by restoration from a verified backup, not deletion of manifests or future evidence.
