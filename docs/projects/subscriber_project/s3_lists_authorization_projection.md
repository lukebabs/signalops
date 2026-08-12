# Sprint S3 - Lists and Authorization Projection

Status: storage and a disabled-by-default pilot API boundary are complete; no browser route, UI, feature-flag enablement, or data-collection worker is enabled. The existing MarketOps Assets experience remains unchanged.

## Delivered storage boundary

S3 adds centralized preference storage inside the existing SignalOps PostgreSQL database:

- one durable tenant-default list per tenant;
- subject-owned private lists;
- idempotent list memberships that reference a platform-owned global asset ID;
- durable list-mutation audit records; and
- forced tenant RLS using the established transaction-local signalops.tenant_id scope.

The browser gateway has CRUD access to tenant-private list tables and only REFERENCES privilege on the global-asset key. It has no global-catalog SELECT privilege, plan/ranking access, provider credential, or worker-table access. A membership can therefore reference a server-verified global asset without turning the shared catalog into an unrestricted tenant-data query surface.

## Repository authorization contract

The repository now provides:

- creation of a subject's private list;
- creation of a tenant-default list, for a caller already approved by the API tenant-administrator guard;
- list reads limited to the tenant default and the requesting subject's private lists;
- membership reads limited to the same authorized list set;
- private-membership add/remove operations constrained to the owning subject; and
- tenant-default membership operations reserved for a caller already approved by the API tenant-administrator guard.

Private list operations return not found for a foreign subject rather than disclosing list existence. Repeating an add returns the existing membership and does not write a second membership or audit event. Every successful create, add, and remove is captured in the tenant-scoped audit table.

## Evidence

Migration 000095_subscriber_watchlist_foundation was applied to the local SignalOps PostgreSQL database on 2026-08-12.

A rolled-back least-privilege transaction running as signalops_subscriber_gateway created a private list and referenced a global asset under s3-tenant-a. After the transaction-local tenant context changed to s3-tenant-b, that role saw zero lists and zero memberships. The probe left no records behind.

The repository integration test also proves private-list subject isolation, tenant-default visibility, idempotent membership addition, private-list membership removal, and no foreign-subject private-list enumeration. It runs only when SIGNALOPS_SUBSCRIBER_RLS_INTEGRATION=1 and uses the local test PostgreSQL connection.

## Explicit boundary

This slice is durable storage and authorization-aware repository code, not a user-facing rollout:

- API routes are compiled but unregistered unless the server feature flag, named pilot tenant, and dedicated subscriber gateway login are all configured.
- No browser UI can create or view a subscriber list.
- No feature flag is enabled.
- No list action can yet enqueue a cold-asset activation.
- No catalog, EOD, options, intraday, scheduler, or legacy MarketOps path changed.

## Remaining S3 pilot gate

The storage and minimal API routes are complete. Before any pilot can be enabled, deployment must provision the dedicated gateway login through secret management, execute the gateway workload preflight, select an entitled pilot tenant, and retain browser evidence for private-list ownership, tenant-default administrator mutation, and cross-tenant isolation. The existing Assets UI remains the default until the separate subscriber list interface and catalog projection are ready.

## Disabled API boundary

The gateway can register the following routes only when all of these conditions are true:

1. SIGNALOPS_SUBSCRIBER_LISTS_ENABLED is explicitly true.
2. The tenant is named in SIGNALOPS_SUBSCRIBER_LISTS_PILOT_TENANTS.
3. SIGNALOPS_SUBSCRIBER_GATEWAY_DATABASE_URL supplies a dedicated least-privilege gateway login.

The gateway refuses to start with the feature flag enabled and no dedicated subscriber gateway database URL. It must not reuse the ordinary gateway database credential for subscriber-private tables.

The initially available, still-disabled routes are:

- GET /v1/tenants/{tenant_id}/marketops/subscriber/lists
- GET /v1/tenants/{tenant_id}/marketops/subscriber/lists/{list_id}/memberships
- POST /v1/tenants/{tenant_id}/marketops/subscriber/private-lists
- POST /v1/tenants/{tenant_id}/marketops/subscriber/tenant-default-list
- POST /v1/tenants/{tenant_id}/marketops/subscriber/lists/{list_id}/memberships
- DELETE /v1/tenants/{tenant_id}/marketops/subscriber/lists/{list_id}/memberships/{global_asset_id}?list_kind=private|tenant_default

Every route binds the tenant and immutable subject from the verified principal. The tenant-default create and membership paths also require the existing tenant-administrator guard. API tests prove routes are absent by default, tenant scope reaches storage unchanged, foreign tenant paths are rejected, and a viewer cannot mutate a tenant-default list.

No frontend calls these routes. The pilot remains disabled until the dedicated workload-login preflight and browser validation evidence are complete.

## Dedicated gateway login evidence

On 2026-08-12, the local SignalOps database received signalops_subscriber_gateway_runtime. It is a LOGIN role, non-superuser, non-CREATEROLE, and NOBYPASSRLS. Its only subscriber membership is signalops_subscriber_gateway with inherited group permissions.

The runtime login has CRUD access to the tenant-private entitlement and S3 list tables, no direct SELECT access to subscriber_global_assets, and no access to legacy MarketOps asset ownership tables. With no tenant context it saw zero private-list rows. In a rolled-back transaction it inserted a private list for subscriber_gateway_probe_a, read one row under that tenant context, switched to subscriber_gateway_probe_b, and read zero rows.

The local DSN is stored only in ignored local configuration. The subscriber list flag remains disabled. Production provisioning must create an equivalent deployment-secret-managed login and run scripts/subscriber_project_gateway_workload_preflight.sh before any pilot flag is enabled.
