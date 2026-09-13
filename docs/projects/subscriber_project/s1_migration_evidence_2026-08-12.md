# S1 Global Catalog Shadow ºw^~)Þt Migration Evidence

Date: 2026-08-12 UTC  
Environment: live SignalOps PostgreSQL  
Migration: `000091_subscriber_global_catalog_shadow`

## Result

The migration applied successfully in one PostgreSQL transaction and was recorded in `schema_migrations` at `2026-08-12 15:46:49 UTC`.

The following platform-owned tables exist and are owned by `signalops_subscriber_migrator`:

- `subscriber_global_assets`
- `subscriber_global_asset_source_links`
- `subscriber_global_asset_reference_observations`
- `subscriber_global_asset_coverage`
- `subscriber_global_catalog_seed_runs`

## Authorization verification

- `signalops_subscriber_catalog_sync` has the intended read-only `SELECT` grant on `marketops_asset_universe`, and `SELECT/INSERT/UPDATE` on the S1 tables.
- `signalops_subscriber_global_eod` has only the intended shared-record read and coverage-update grants.
- `signalops_subscriber_gateway` has no direct grant on any S1 global-catalog table.
- No browser route, tenant projection, provider collection, scheduler, or seed operation was enabled by the migration.

## Outstanding S1 evidence

The controlled seed has not yet run. Before S1 exit, run it under the separately provisioned catalog-sync workload identity and retain the parity report proving every active compatibility-universe row maps once, with coverage remaining in `shadow` mode.
