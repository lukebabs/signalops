# S5 Pilot Deployment Evidence — 2026-08-13

Status: deployed safely; catalog-use and shared-coverage activation remain deliberately default-deny.

## Release

- Source revision: `9ea948a` (`feat(subscribers): add watchlist catalog flow`)
- Deployment method: an isolated archive of that exact revision was used for the migration inputs and the gateway/web images. Uncommitted Sector Rotation Intelligence work was not included.
- Recreated services: `gateway` and `web` only.
- Rollout time: 2026-08-13 UTC.

## Applied schema

The migration ledger records all required S4/S5 schema changes:

- `000097_subscriber_global_eod_canary`
- `000098_subscriber_catalog_search_projection`
- `000099_subscriber_private_catalog_membership`

These migrations create constrained storage, RLS-scoped functions, and audit paths only. They neither pull provider data nor enable an EOD worker.

## Runtime evidence

- Gateway health returned `status: ok`.
- The public `/marketops/watchlists` route returned the deployed application shell.
- `SIGNALOPS_SUBSCRIBER_LISTS_ENABLED=true`.
- `SIGNALOPS_SUBSCRIBER_LISTS_PILOT_TENANTS=tenant-pilot-b` is the only configured pilot tenant.
- The central governed catalog contained 985 eligible assets at verification.
- The activation-request table contained zero rows at verification.

The authenticated tenant-local Watchlists request remains absent (`404`), as expected outside the named pilot scope. Authentication is evaluated before route-scope disclosure, so an invalid bearer token returns `401` for either tenant and must not be used as an isolation test.

## Deliberate product boundary

`tenant-pilot-b` retains the approved `subscriber-list-pilot-v1` default-deny entitlement:

| Capability | Enabled | Quota |
| --- | ---: | ---: |
| `catalog_search` | No | 0 |
| `eod_activation` | No | 0 |
| `options_demand` | No | 0 |

Therefore the page and protected API surface are deployed, but catalog search and adding catalog assets are correctly denied. No user list action can create an activation request, warm global coverage, or invoke Massive until product ownership approves concrete pilot quotas and enables the corresponding capabilities through the controlled entitlement path.

## Remaining acceptance evidence

When a limited entitlement policy is approved, perform the browser acceptance test with a current pilot-user token:

1. Sign in as a `tenant-pilot-b` MarketOps user, create a private list, search the governed catalog, and add an asset.
2. Confirm tenant-default mutation remains administrator-only and a tenant-local principal still has no subscriber route.
3. Confirm an enabled `eod_activation` request records only an idempotent central activation request; it must not itself call a provider or run an EOD worker.
4. Retain the resulting audit records and verify that disabling the named pilot flag removes access without deleting any lists or audits.
