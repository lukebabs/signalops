# Tenant-local Legacy Default Preservation — 2026-08-16

## Decision

`tenant-local` retains the existing 132-symbol MarketOps universe as its durable
tenant-default list. The global 1,000-symbol warm cohort is a shared EOD
coverage baseline; it does not replace the tenant-local default selection.

## Applied migration

Migration `000141_subscriber_tenant_local_legacy_default_preservation` was
applied to the dedicated MarketOps primary at `2026-08-16T21:47:54Z`. It
created the single tenant-default list:

- `sublist-tenant-local-legacy-default` — **MarketOps Legacy Default**

The migration imported exactly 132 canonical global memberships, retaining the
legacy `universe_priority` and `rank` as immutable membership provenance and a
stable `legacy_order` from 1 through 132.

## Preservation contract

- The source `marketops_universal_assets` rows remain unchanged: 132 active
  distinct symbols after migration.
- No intraday snapshot, EOD history, feature, algorithm result, schedule,
  entitlement, or feature flag was altered.
- The list references canonical `global_asset_id` values only; it does not copy
  market data into the tenant.
- Each membership records `source_scope = tenant-local`, source ticker, source
  universe priority/rank, canonical mapping, and the preservation-policy version.
- Audit evidence consists of one `create_list` event and 132 `add_asset` events
  under correlation `subscriber-legacy-default-132-v1`.
- The migration fails closed when any active legacy ticker does not resolve to
  exactly one canonical global identity after the immutable alias-resolution
  ledger is applied.

## Coverage interpretation

The historical tenant-local 132-symbol intraday plane remains the serving
compatibility path during migration. The global hot-intraday selector is not yet
active, so no legacy intraday observation is silently claimed as a globally
served result. Mature EOD evidence may be appended to the global evidence ledger
with its original tenant-local provenance, but it must pass parity checks first.

Future hot-intraday work must project the same canonical data to this default
list and every authorized private/default list; it must not create a second
tenant-local market-data copy.
