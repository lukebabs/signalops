# Sprint S3 ºw^~)Þt Lists and Authorization Projection

Status: foundation in progress. The tenant-default/private-list schema is RLS-protected and has no registered browser or API route. The existing Assets experience remains unchanged.

## First slice

S3 introduces private preference records only:

- one durable tenant-default list per tenant;
- subject-owned private lists;
- membership rows that reference a platform-owned global asset ID;
- immutable list-mutation audit records; and
- forced tenant RLS using the established transaction-local `signalops.tenant_id` scope.

The browser gateway receives CRUD access to tenant-private list tables and only `REFERENCES` privilege on the global-asset key. It receives no global-catalog `SELECT` privilege, no plan/ranking access, and no direct provider credential. Therefore a membership can reference a server-verified global asset without turning the shared catalog into an unrestricted tenant-data query surface.

## Authorization contract

The S0-A gateway primitives remain mandatory for every future list route:

- tenant scope comes from the verified JWT claim;
- a private-list owner must equal the immutable JWT subject;
- a tenant-default-list mutation requires the existing tenant-administrator guard;
- all list and membership reads/writes occur in `WithSubscriberTenantScope`;
- a cross-tenant list identifier returns ordinary not-found behavior; and
- cold-asset activation is evaluated only after a valid membership write, entitlement decision, and quota reservation.

No S3 route is enabled in this slice. The next slice adds storage operations and negative authorization tests before an opt-in pilot route can be registered.

## Deployment evidence

Migration `000095_subscriber_watchlist_foundation` was applied to the local SignalOps PostgreSQL database on 2026-08-12. A rolled-back least-privilege transaction running as `signalops_subscriber_gateway` created a private list and referenced a global asset under `s3-tenant-a`. When the transaction-local tenant context changed to `s3-tenant-b`, the gateway role saw zero lists and zero memberships. The test left no records behind.

This proves the table/RLS boundary and foreign-key privilege, not a user-facing rollout. No route, UI tab, feature flag enablement, scheduler, collection worker, or MarketOps legacy-table mutation was introduced.
