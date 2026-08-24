# MarketOps Subscription Commerce Model

Status: foundation deployed and verified on 2026-08-17; Administration governance for enrolled users, tenant contracts, seats, tier policy, limits, and audit visibility was added on 2026-08-19. Commercial activation remains deliberately disabled until controlled activation evidence is retained. See [release evidence](subscription_commerce_foundation_release_evidence_2026-08-17.md).

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
| Syncratic interactive explainability (Ask, Regenerate, materialize/enqueue) | `syncratic_explainability` | Professional |
| SAF | `signal_assurance_analytics` | Institutional |
| Backtest/replay/calibration endpoints | `historical_replay` | Institutional |

Explorer retains dashboard/public-signal access, SRI rankings, and read-only persisted Syncratic narratives. The web shell uses the effective product feature policy for locked navigation and direct-route gates; the gateway remains authoritative.

## Operating roles

| Role | Scope |
|---|---|
| Platform Subscription Admin | Products, Stripe mappings, subject overrides, tenant contracts, exception audit, and webhook reconciliation. This must be a dedicated platform role, not a tenant administrator. |
| Tenant administrator | Institutional seat management, default lists, shared branding, portfolio workspace, custom-universe selections, and API credentials within the tenant’s paid allowance. |
| Subscriber | Personal lists and the analytical features granted by the effective subscription. |
| Data-plane worker | Global catalog/coverage/algorithm work only; no subscription administration or browser impersonation. |

The Platform Subscription Admin API is implemented as a fail-closed, signed-role-only governance boundary for subject plans, Institutional contracts, seats, product/tier feature policy, limits, product active state, and audit visibility. Its controlled operator UI is in the Administration workbench at /admin/subscriptions, never in MarketOps. It is not a browser self-upgrade path. Stripe webhook reconciliation remains the next commerce integration slice.


## Administration governance surface — 2026-08-19

The Administration workbench now exposes a real governance view rather than only blind provisioning forms:

- tenant-filtered enrolled subject subscriptions with current plan/status evidence;
- Institutional tenant contracts and assigned seats;
- tier/product policy cards showing billing scope, active state, trial days, revision, feature policy, and limit policy;
- product-policy mutation for feature alignment and limits, with revision increment and audit evidence;
- tenant-scoped subscription audit trail for provisioning and policy changes.

This governance surface still does not enable Stripe Checkout, tenant self-service upgrades, provider polling, or subscription enforcement by itself. Enforcement remains controlled by `SIGNALOPS_SUBSCRIPTIONS_ENABLED`, and the gateway remains authoritative for feature checks.

The first Stripe integration slice is admin-managed billing, documented in [Stripe Admin-Managed Billing](stripe_admin_managed_billing.md). It allows platform admins to map Stripe product/customer/subscription IDs and reconcile signed webhooks for known subscriptions only. It does not create a customer checkout path.

## Upgrade journey foundation — 2026-08-24

Migration `000160_subscriber_upgrade_interactions` adds the first product-led subscription journey ledger. The browser can record authenticated, tenant-scoped upgrade prompt impressions and clicks through `POST /v1/marketops/subscriptions/upgrade-interactions`. Records are RLS-protected in the dedicated MarketOps database and surfaced in Administration > Subscriptions > Upgrade funnel for source attribution, click-through review, and user-level context.

Locked MarketOps feature gates now present contextual upgrade copy and route users to `/marketops/pricing`, preserving the source feature and return URL. The pricing page reads configured subscription products and Stripe price IDs from the existing subscription product API, and explicitly does not start Checkout. This slice captures upgrade intent and plan comparison only; entitlements still change only through governed administration or future webhook-confirmed Checkout.

## Stripe boundary

Stripe will first be used as admin-managed billing evidence and signed webhook reconciliation. Later, Professional self-service checkout/billing portal can be added as a separate release. The intended state transition rules are:

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

- Stripe Checkout, customer portal, retry/dead-letter handling, and billing telemetry beyond the admin-managed webhook ledger and upgrade-intent ledger.
- Tenant-facing seat-management UI, Stripe price editing, controlled commercial overrides beyond the platform-admin governance boundary, and customer self-service upgrade paths.
- Research-report generation/storage, portfolio CSV ingestion, batch-screening UI, custom-universe selector, API-key lifecycle, and shared-tenant branding controls.
- SRI discovery/detail response shaping beyond the current endpoint boundary.
- A production purchase flow. There is no hidden self-service billing path in this release.
