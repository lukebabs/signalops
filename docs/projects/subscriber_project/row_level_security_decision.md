# Subscriber Database Row-Level Security Decision

Status: implementation in progress. The least-privilege NOLOGIN group-role bootstrap, role preflight, transaction-local tenant context, and first forced-RLS entitlement/quota schema are implemented; no workload credential, API route, or browser path is enabled. Existing MarketOps data remains unchanged.

## Decision

SignalOps will use a hybrid isolation model for the Subscriber Project. The gateway and tenant-qualified repository queries remain the primary authorization control for existing tenant-owned MarketOps data. Before any new subscriber-private table is enabled, that table must also use PostgreSQL row-level security (RLS) as database defense in depth.

The decision is deliberately not to retrofit all existing tenant-owned and temporal tables during S0-A. The current deployment uses a shared `signalops` database credential across gateway and worker services, and direct SQL calls are not consistently transaction-bound with a database tenant setting. A broad retrofit now would risk an outage or incorrect temporal-data behavior without yet protecting an enabled subscriber path.

## Scope

RLS is mandatory before enabling these new tenant-private classes:

- tenant default lists, private lists, and list memberships;
- entitlement provisioning, quota reservation or usage, and entitlement/decision audit records (the initial inert schema is present under forced RLS);
- tenant-specific coverage projections, preferences, saved views, and subscriber notifications.

The following remain platform-owned shared records and do not receive tenant-membership RLS policies: global asset identity, reference provenance, global coverage state, shared EOD/raw/normalized evidence, canonical algorithm outputs, and deduplicated Options captures. They remain inaccessible to browser callers except through a tenant-authorized projection.

## Required enforcement model

A future migration must create distinct non-owner roles for gateway access, each worker identity, and schema migration. The migration role owns and alters schema but is never used by the gateway or scheduled workers. Application and worker roles must not be superusers, table owners, or able to bypass RLS.

Each tenant-private table must include a non-null `tenant_id`, enable and force RLS, and use a policy that compares the row tenant to a transaction-local setting such as `current_setting(signalops.tenant_id, true)`. Gateway storage code must start a transaction, set that setting from the verified principal using a parameterized local command, perform every tenant-private query or write in that transaction, and commit or roll back before returning the connection to the pool. An absent, malformed, or mismatched setting must yield no rows or a permission failure; it must never become a cross-tenant default.

Workers do not receive arbitrary tenant scope. A tenant-aware worker receives only an immutable authorized planning snapshot and sets a tenant context only for the narrow tenant-private operation it is permitted to perform. Shared-data workers use a separate role with table-level permissions only for their platform-owned inputs and outputs.

## Implemented foundation

- `deploy/postgres/subscriber_project_roles.sql` creates six inert NOLOGIN group roles: a migration owner, gateway, catalog sync, global EOD, Options demand, and Options capture. It intentionally grants no access to current MarketOps tables. Production secret management must attach a distinct workload login to exactly one applicable group role; the current shared superuser credential is not eligible for subscriber-private paths.
- `scripts/subscriber_project_rls_preflight.sh` verifies that each group role is non-login, non-superuser, non-CREATEROLE, and NOBYPASSRLS. It uses a privileged database URL when `SIGNALOPS_SUBSCRIBER_RLS_MIGRATOR_DATABASE_URL` is configured, or the local Compose PostgreSQL service for development.
- `Repository.WithSubscriberTenantScope` starts a transaction and uses `set_config(..., true)` to set `signalops.tenant_id`. Future subscriber-private repositories must use the supplied transaction exclusively so the context clears before a pooled connection is reused.
- `scripts/subscriber_project_gateway_workload_preflight.sh` validates a separately provisioned non-superuser gateway login. It checks the group-role membership and grants, verifies zero private rows without context, and runs a rolled-back cross-tenant probe under forced RLS.
- Migration `000088_subscriber_entitlement_foundation` creates the first tenant-private entitlement, capability, quota-reservation, and decision-audit tables. Each is owned by `signalops_subscriber_migrator`, has forced RLS, and grants access only to `signalops_subscriber_gateway`. It supplies no API route, provisioning workflow, or feature enablement.

## Migration and verification sequence

1. Introduce separate migration, gateway, and per-worker database roles through deployment secret management.
2. The entitlement/quota foundation is the first tenant-private schema with `tenant_id`, forced RLS, explicit gateway grants, and transaction-local tenant context. Subsequent private tables must follow the same model.
3. Exercise direct database and API tests for missing context, a conflicting context, cross-tenant read/write/enumeration, connection-pool reuse, worker scope violation, and migration-role bypass.
4. Add SQL policy/role verification to deployment preflight and record migration, policy version, identity, tenant, and correlation provenance.
5. Repeat the approach table by table. Existing tables are migrated only after query inventory, dual-read parity, temporal-store validation, and rollback review.

## Consequences

This decision does not make the current shared `signalops` deployment safe for subscriber-private data. It blocks enabling subscriber lists, entitlements, quota accounting, and tenant projections until the roles, transaction helper, RLS policies, test suite, and deployment checks above are in place. Application-level tenant binding remains mandatory even after RLS adoption; RLS is defense in depth, not a replacement for gateway authorization or immutable audit provenance.
