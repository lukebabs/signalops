# Sprint S3 - Lists and Authorization Projection

Status: storage layer complete; no subscriber list API, browser route, UI, feature-flag enablement, or data-collection worker is enabled. The existing MarketOps Assets experience remains unchanged.

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

- No API route is registered.
- No browser UI can create or view a subscriber list.
- No feature flag is enabled.
- No list action can yet enqueue a cold-asset activation.
- No catalog, EOD, options, intraday, scheduler, or legacy MarketOps path changed.

## Next S3 slice

Add API handlers behind an off-by-default, tenant-scoped feature flag. The handlers must bind tenant and subject from the verified principal, require the tenant-administrator guard for tenant-default mutations, invoke only these repository operations, and add API-level cross-tenant and ownership-negative tests. The flag remains disabled until the workload-login preflight and browser evidence are complete.
