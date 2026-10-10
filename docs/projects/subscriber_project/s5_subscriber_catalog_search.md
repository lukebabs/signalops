# Sprint S5 — Subscriber Catalog Search

Status: first pilot-only increment: entitlement-gated catalog search. List membership, cold activation, provider collection, options demand, and tenant-default changes are not expanded by this increment.

Migration `000098_subscriber_catalog_search_projection` adds `subscriber_search_global_catalog`, a bounded security-definer projection over eligible global assets and their stored EOD coverage status. The subscriber gateway receives execute permission only; it retains no direct global-catalog `SELECT` permission.

`GET /v1/tenants/{tenant_id}/marketops/subscriber/catalog?q=<query>&limit=<1..50>` is registered only under the existing Subscriber pilot flag and named pilot tenant. It binds the path tenant and authenticated subject, then requires an active tenant entitlement with enabled `catalog_search` capability and a positive quota limit. It returns platform metadata and truthful stored coverage state only. It cannot trigger a provider request, activate a cold asset, reveal another tenant's membership, or mutate a tenant-default list.

The current pilot entitlement intentionally has metered capabilities disabled. Therefore the route fails closed with `subscriber_catalog_not_entitled` until product ownership explicitly enables `catalog_search` for the tenant. This is expected; implementation does not silently change entitlements.

The delivered private membership route requires the separate `eod_activation` entitlement, server-verifies an eligible global asset, preserves private-list ownership, writes one audit record for a new membership, and writes at most one central `queued` activation request for a cold asset. It cannot call Massive or mutate a tenant-default list.
