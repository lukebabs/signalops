# MarketOps Subscription Commerce Model

Status: foundation deployed and verified on 2026-08-17; commercial activation remains deliberately disabled until controlled product provisioning and payment-provider readiness have evidence. See [release evidence](subscription_commerce_foundation_release_evidence_2026-08-17.md).

## Product intent

MarketOps subscriptions increase analytical depth. They never sell a separate copy of market data, create tenant-local provider polling, or permit a browser to call Massive, FMP, State Street, or iShares. The global catalog, coverage lifecycle, EOD algorithms, and hot-watchlist selector remain one centrally governed operating plane.

## Launch plans

| Plan | Commercial model | Analytical access | Watchlists |
|---|---|---|---|
| Explorer | Free, no card | Market dashboards, public signals, and Sector Rotation discovery | 3 private lists, 25 assets/list |
| Professional | Stripe monthly or annual; 7-day trial | Explorer plus current validated Value Intelligence, Distressed Opportunity Intelligence, Earnings Opportunity Intelligence, detailed Sector Rotation, options signals, earnings calendar, and saved web/PDF research reports | 20 private lists, 100 assets/list |
| Institutional | Tenant contract with assigned seats | Professional plus Signal Assurance analytics, portfolio analysis from CSV/manual uploads, batch screening, historical replay, strategy validation without customer-executable code, catalog-constrained custom universes, API access, and shared-tenant branding | Fair-use governed limits |

Prices, currencies, Stripe price IDs, trials beyond the product default, and commercial copy are not hard-coded. They are provider/admin configuration so a pricing change does not require a deployment.

## Authorization model

- Explorer and Professional grants are **subject subscriptions** scoped to the authenticated tenant and immutable OIDC subject.
- Institutional access is a **tenant subscription** with a seat assignment for each permitted subject.
- The effective-subscription resolver accepts only `trialing`, `active`, or a `past_due` subscription still inside its configured seven-day grace period.
- Role access and tier access are independent controls. A valid SignalOps role and tenant grant are necessary but cannot substitute for a subscription feature. Conversely, a subscription cannot bypass tenant binding, RBAC, or list ownership.
- The gateway decides from the signed principal, not a browser-supplied plan or tenant.
- A missing subscription is a deliberate `402 subscription_required`; a known plan without a feature is `402 subscription_feature_required`. A database failure is a `503`, never a permissive fallback.

The current implementation uses a default-off `SIGNALOPS_SUBSCRIPTIONS_ENABLED` feature flag. It requires the subscriber catalog gateway database (`SIGNALOPS_SUBSCRIBER_LISTS_ENABLED`) and is therefore safe to deploy ahead of commercial provisioning.

## Enforced capability boundary

The gateway currently guards these server endpoints when the feature flag is on:

| Capability | Required feature | Minimum plan |
|---|---|---|
| Valuation | `value_intelligence` | Professional |
| EROC / distressed analysis | `distressed_opportunity_intelligence` | Professional |
| EEOM | `earnings_opportunity_intelligence` | Professional |
| Options captures | `options_signals` | Professional |
| Detailed SRI drill-down (segment/history/makeup) | `sector_rotation_detail` | Professional |
| SAF | `signal_assurance_analytics` | Institutional |
| Backtest/replay/calibration endpoints | `historical_replay` | Institutional |

Explorer retains dashboard/public-signal access and SRI rankings. The web shell uses the effective product feature policy for locked navigation and direct-route gates; the gateway remains authoritative.

## Operating roles

| Role | Scope |
|---|---|
| Platform Subscription Admin | Products, Stripe mappings, subject overrides, tenant contracts, exception audit, and webhook reconciliation. This must be a dedicated platform role, not a tenant administrator. |
| Tenant administrator | Institutional seat management, default lists, shared branding, portfolio workspace, custom-universe selections, and API credentials within the tenant’s paid allowance. |
| Subscriber | Personal lists and the analytical features granted by the effective subscription. |
| Data-plane worker | Global catalog/coverage/algorithm work only; no subscription administration or browser impersonation. |

The Platform Subscription Admin API is implemented as a fail-closed, signed-role-only provisioning boundary for subject plans, Institutional contracts, and seats. Its controlled operator UI is in the Administration workbench at /admin/subscriptions, never in MarketOps. It is not a browser self-upgrade path. Stripe webhook reconciliation remains the next commerce integration slice.

## Stripe boundary

Stripe will be used only for Professional self-service checkout/billing portal and the signed, idempotent webhook ledger. The intended state transition rules are:

1. Checkout/session correlation creates or updates a subject subscription.
2. A seven-day Professional trial grants access while `trialing`.
3. A successful invoice retains `active`; `past_due` remains usable only for the seven-day grace window.
4. Cancellation retains access through the paid period; subsequent access is removed by the effective-subscription resolver.
5. Every provider event is stored by immutable provider event ID before reconciliation, so a replay cannot double-grant or double-revoke access.

Institutional contracts are provisioned through a tenant-admin workflow and seats—not Stripe Checkout. Portfolio analysis is CSV/manual only, with no broker credential collection in this phase.

## Rollout and rollback

1. Apply migration `000147_subscriber_subscription_commerce_foundation` to the dedicated subscriber gateway database.
2. Verify the products and RLS policies using the subscriber gateway and migrator identities. Assign `signalops:subscription_admin` only to controlled platform operators; it alone may call the cross-tenant provisioning API.
3. Provision Explorer records for the pilot subjects and the appropriate Professional/Institutional records and seats in a controlled, audited operation.
4. Verify each tier with direct API requests and browser acceptance: Explorer public/SRI discovery succeeds while paid paths return 402; Professional paid analysis succeeds while SAF/replay return 402; Institutional assigned seat succeeds.
5. Enable `SIGNALOPS_SUBSCRIPTIONS_ENABLED=true` only for the gateway after the above evidence is complete.

Rollback is one configuration change to `false`. It removes commercial feature enforcement without deleting subscription records, seat history, billing evidence, global data, or watchlist preferences.

## Explicitly deferred work

- Stripe credentials, Checkout, customer portal, webhook verification/reconciliation worker, retry/dead-letter handling, and billing telemetry.
- Tenant-facing seat-management UI, product/price editing, and controlled overrides. The platform provisioning API exists but has no browser self-service upgrade path.
- Research-report generation/storage, portfolio CSV ingestion, batch-screening UI, custom-universe selector, API-key lifecycle, and shared-tenant branding controls.
- SRI discovery/detail response shaping beyond the current endpoint boundary.
- A production purchase flow. There is no hidden self-service billing path in this release.
