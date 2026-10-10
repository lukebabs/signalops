# Sprint S1 ºw^~)Þt Global Catalog Shadow

Status: shadow schema, controlled seed, and parity evidence complete; no new browser path, subscriber projection, or provider collection is enabled.

## Delivered foundation

Migration `000091_subscriber_global_catalog_shadow` introduces platform-owned, additive tables for:

- stable global asset identity keyed by provider/source and provider symbol;
- compatibility-universe source links, so every active current MarketOps row has an explicit mapping rather than an inferred duplicate;
- immutable reference observations, each with a fingerprint, source coordinates, seed run, and provenance; and
- an EOD coverage registry in `shadow` execution mode. A current active source row is recorded as observed `active`; it does **not** mean the new global EOD planner is enabled.

The tables are intentionally not tenant-RLS tables: they contain platform-owned shared records, as decided in [the row-level-security decision](row_level_security_decision.md). `PUBLIC` and the browser gateway receive no grant. Only the inert catalog-sync role may write them and it receives one read-only grant on \`marketops_asset_universe\` to seed the compatibility mapping; the future global-EOD role receives narrowly scoped read/coverage-update access. Neither role is a login or an enabled worker.

## Controlled seed and parity evidence

After the migration-owner and catalog-sync workload identities are separately provisioned and preflighted, a controlled operator may seed the compatibility universe:

```sh
go run ./cmd/subscriber-global-catalog-seed --execute \
  --source-tenant-id tenant-local \
  --actor subscriber-catalog-reference-sync \
  --correlation-id s1-shadow-<change-id>
```

The command refuses to mutate without `--execute`. It reads `marketops_asset_universe`, writes only the new S1 tables, and emits a JSON count report. It does not call Massive, enqueue collection, alter tenant-owned universe rows, alter scheduled jobs, or create an API/browser route.

The seed run records source-row count, active-row count, distinct global identities, inserted identities, and immutable reference-observation count. The S1 exit evidence must establish:

1. every active `marketops_asset_universe` row for the selected compatibility tenant has one source-link/global-asset mapping;
2. no source/provider-symbol pair maps to more than one global identity; and
3. every coverage row remains `execution_mode = shadow`.

No eligibility status is promoted by the seed. Imported records start as `discovered` because S1 has not yet run the provider and US-common-stock governance validation required for `eligible`. S2 owns breadth expansion, top-1,000 admission, and coverage-planner shadowing.

## Rollback

Disable or do not invoke the manual seed command. Existing MarketOps reads and jobs are unaffected. Before any subsequent sprint enables a projection or worker, retain the seed-run and reference-observation provenance. The migrationéÝyø§yÙs down file is reserved for a non-production rollback before dependent S2+ tables exist; production recovery should first disable the path and preserve the evidence.
