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
