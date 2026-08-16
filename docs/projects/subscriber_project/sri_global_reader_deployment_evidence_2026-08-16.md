# SRI platform-global reader deployment evidence — 2026-08-16

## Released commits

- `c405429 feat(subscriber): serve SRI from global foundation`
- `f40a4d2 fix(subscriber): grant gateway SRI projection reader`
- `47eed94 fix(subscriber): authorize SRI projection owner`
- `0c26e67 test(subscriber): exercise SRI progression in browser`

## Dedicated MarketOps migration evidence

Applied in order to the dedicated MarketOps primary database:

- `000132_subscriber_global_sri_foundation`
- `000133_subscriber_global_sri_gateway_runtime_grant`
- `000134_subscriber_global_sri_projection_owner_grant`

`000132` idempotently materialized the existing common SRI foundation into the
platform-global data plane:

| Projection | Rows |
| --- | ---: |
| active SRI segments | 19 |
| ETF registry entries | 24 |
| historical segment snapshots | 1,040 |
| issuer holdings snapshots | 60 |
| issuer holdings | 3,500 |

The historical SRI session range remains 2026-05-14 through 2026-08-14.
Migrated snapshots preserve source scope, source tenant, and original snapshot
ID in `input_provenance`.

## Access boundary

Subscriber SRI routes authenticate and bind the request tenant, resolve the
selected authorized watchlist, then read only five security-barrier
`platform-global` SRI views. They return `data_scope: "platform-global"` and
watchlist context. The gateway fails closed with `global_sri_unavailable` if
that reader is not available; it never serves tenant-local SRI as fallback.

The two follow-up grants correct PostgreSQL role semantics without broadening
the runtime boundary: the `signalops` runtime can select the five views, while
the projection owner can evaluate their source tables. The runtime role was
not granted raw SRI table access.

## Acceptance

- Go API/SRI packages and the web production build passed before release.
- Gateway-only deployment completed through the named deployment agent.
- The authenticated read-only pilot browser suite passed: `2 passed in 4.93s`.
  It verifies current selected-watchlist context, platform-global SRI rankings,
  and the interactive ETF progression/history request.

No market-data or issuer-holdings provider call was made by this release.
