# S3 Pilot Preflight Evidence - 2026-08-12

Scope: local deployment-equivalent SignalOps PostgreSQL environment.

The S3 pilot preflight passed for `tenant-pilot-b` while `SIGNALOPS_SUBSCRIBER_LISTS_ENABLED=false`. The pilot allowlist contained only `tenant-pilot-b`.

## Passed controls

- Dedicated subscriber gateway runtime login is non-superuser, non-CREATEROLE, NOBYPASSRLS, and a member of only the subscriber-gateway group role.
- The runtime login has required tenant-private entitlement/list access and no direct legacy MarketOps asset-table access.
- Tenant-private rows are invisible with no transaction-local tenant context.
- The forced-RLS cross-tenant probe passed.
- All S3 list tables are present.
- `tenant-pilot-b` has an active `subscriber-list-pilot-v1` entitlement with all metered capabilities disabled.
- Exactly one tenant-default list exists and has the audited governed top-ten seed.

## Boundary retained

This is readiness evidence only. It did not set the S3 feature flag true, deploy/restart the production gateway, register a browser UI, add a provider call, activate a cold asset, alter shared EOD coverage, or change a scheduled job.

The next controlled action is deployment enablement for `tenant-pilot-b` alone, followed immediately by the browser ownership, tenant-administrator, and cross-tenant isolation validation. A production deployment must rerun this same preflight with its secret-managed gateway DSN before that action.

## Controlled local activation

After the passing preflight, the local deployment-equivalent gateway was rebuilt and restarted at 2026-08-12T21:46:55Z with the S3 flag enabled and the allowlist restricted to `tenant-pilot-b`. The process started successfully and both `/healthz` and `/readyz` returned success. The deployment revision is `fbf5ed7`.

This is an API-only local pilot: no production gateway, browser route, provider call, cold-asset activation, shared coverage change, or scheduler change was enabled. Browser evidence for private-list ownership, tenant-default administrator mutation, and cross-tenant isolation remains required before a UI rollout or a production activation.


## Watchlists projection deployment - 2026-08-13

Migration 000096_subscriber_watchlist_projection was applied to the local deployment-equivalent SignalOps PostgreSQL database after reconciling the verified, pre-existing 000095_subscriber_watchlist_foundation schema into the migration ledger.

The local gateway and web containers were rebuilt and recreated at 2026-08-13T01:22:21Z. The gateway retained SIGNALOPS_SUBSCRIBER_LISTS_ENABLED=true with the allowlist restricted to tenant-pilot-b. The ready endpoint returned success and the /marketops/watchlists route was served by the local web container.

A transaction under signalops_subscriber_gateway with tenant context tenant-pilot-b confirmed direct_catalog_select=false and returned 10 authorized seeded default-list items from subscriber_visible_watchlist_items. No production deployment, browser evidence, provider call, catalog search, membership edit, or coverage activation occurred.
