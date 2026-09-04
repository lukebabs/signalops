# MarketOps Subscription Commerce Model

Status: foundation deployed and verified on 2026-08-17; Administration governance for enrolled users, tenant contracts, seats, tier policy, limits, and audit visibility was added on 2026-08-19. Self-service Explorer/Professional Checkout-start is now implemented and guarded by server-side Stripe configuration; entitlement activation remains deliberately webhook-authoritative until controlled paid-flow evidence is retained. See [release evidence](subscription_commerce_foundation_release_evidence_2026-08-17.md).

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

The Platform Subscription Admin API is implemented as a fail-closed, signed-role-only governance boundary for subject plans, Institutional contracts, seats, product/tier feature policy, limits, product active state, and audit visibility. Its controlled operator UI is in the Administration workbench at /admin/subscriptions, never in MarketOps. Browser self-service activation is allowed only through the webhook-confirmed Stripe Checkout path described below; direct plan mutation remains admin-only.


## Administration governance surface — 2026-08-19

The Administration workbench now exposes a real governance view rather than only blind provisioning forms:

- tenant-filtered enrolled subject subscriptions with current plan/status evidence;
- Institutional tenant contracts and assigned seats;
- tier/product policy cards showing billing scope, active state, trial days, revision, feature policy, and limit policy;
- product-policy mutation for feature alignment and limits, with revision increment and audit evidence;
- tenant-scoped subscription audit trail for provisioning and policy changes.

This governance surface does not enable provider polling or broad subscription enforcement by itself. Enforcement remains controlled by `SIGNALOPS_SUBSCRIPTIONS_ENABLED`, while B2C enrollment activation is separately controlled by `SIGNALOPS_SUBSCRIBER_B2C_REQUIRE_SUBSCRIPTION`. The gateway remains authoritative for feature checks.

The first Stripe integration slice was admin-managed billing, documented in [Stripe Admin-Managed Billing](stripe_admin_managed_billing.md). The current commerce path extends that foundation with constrained browser Checkout-start for Explorer and Professional. The browser can request a Checkout Session, but it cannot grant access; only governed administration or verified Stripe webhook reconciliation can activate a subscription.

## Upgrade journey foundation — 2026-08-24

Migration `000160_subscriber_upgrade_interactions` adds the first product-led subscription journey ledger. The browser can record authenticated, tenant-scoped upgrade prompt impressions and clicks through `POST /v1/marketops/subscriptions/upgrade-interactions`. Records are RLS-protected in the dedicated MarketOps database and surfaced in Administration > Subscriptions > Upgrade funnel for source attribution, click-through review, and user-level context.

Locked MarketOps feature gates now present contextual upgrade copy and route users to `/marketops/pricing`, preserving the source feature and return URL. The pricing page reads configured subscription products and Stripe price IDs from the existing subscription product API. Explorer and Professional plan cards now start server-created Stripe Checkout through `POST /v1/tenants/{tenant_id}/marketops/subscription/checkout`; the endpoint remains fail-closed unless Stripe runtime configuration is present and product price IDs are mapped. The Stripe return page `/marketops/subscription/return?session_id=...` polls effective subscription state and explicitly treats the redirect as activation-pending evidence only. Entitlements still change only through governed administration or verified Stripe webhook reconciliation; the frontend redirect never grants access.

Production validation on 2026-09-04 confirmed this boundary with Playwright: tenant-pilot-b can view the pricing journey, Stripe Product/Price identifiers are visible as public catalog metadata, Checkout readiness renders correctly, and the authenticated B2C flow routes `subscription_missing` users to Pricing. Earlier upgrade-interaction evidence remains available in Administration > Subscriptions > Upgrade funnel after filtering to the relevant tenant.

The Stripe webhook path remains authoritative for future automatic entitlement updates. The current canary evidence proves invalid signatures fail closed before persistence, while a valid signed synthetic event can be recorded as `unmatched` without creating access for unknown Stripe subscriptions.

## Keycloak B2C enrollment slice — 2026-08-25

Public account creation now has an application-side enrollment resolver. Keycloak remains the identity provider and must emit a verified email, immutable subject, `tenant_id`, roles, and `signalops-api` audience. SignalOps then owns the MarketOps enrollment decision through `GET /v1/session/enrollment`.

The resolver self-provisions only the configured B2C tenant, default `tenant-local`, and only after `email_verified=true`. Under the production Option B policy selected on 2026-08-25, registration is not a subscription activation event. The resolver may create identity/access scaffolding, but an active Explorer or Professional subscription must come from governed administration or verified Stripe webhook reconciliation. The activation gate is controlled by `SIGNALOPS_SUBSCRIBER_B2C_REQUIRE_SUBSCRIPTION`, default `true`, so self-enrolled B2C users without an effective subscription resolve to `subscription_missing` and are routed to Pricing even if broader paid-feature enforcement remains off. The legacy auto-Explorer behavior is behind `SIGNALOPS_SUBSCRIBER_B2C_AUTO_ACTIVATE_EXPLORER=true` and defaults off.

SMS MFA remains an identity-provider control, but it is deferred for the current low-friction enrollment phase. Keycloak owns SMS enrollment, phone verification, and login challenge through the custom `CONFIGURE_SMS_MFA` required action and `syncratic-sms-otp-authenticator` when MFA is later approved. SignalOps must not request SMS MFA for ordinary public registration in the current phase and must never mark phone numbers verified.

See [Keycloak B2C Enrollment Flow](keycloak_b2c_enrollment.md).


## Stripe boundary
Customer-facing price display is separate from Stripe billing identifiers. `subscriber_subscription_products` stores monthly and annual display-price labels used by `/marketops/pricing`; Stripe Product and Price IDs remain the server-side billing authority selected by the Gateway from `product_key` plus `billing_period`. Admin may inspect and govern both values, but ordinary subscriber Pricing screens must not expose raw `price_...` identifiers.


Stripe is used as billing evidence, signed webhook reconciliation, and constrained self-service Checkout for Explorer/Professional. Checkout starts from an authenticated tenant-scoped subject and writes an internal `subscriber_checkout_sessions` ledger row. Stripe receives only an opaque `checkout_ref` plus non-identity product/billing metadata; tenant ID, subject, and authorization state remain in the dedicated MarketOps database. The intended state transition rules are:

1. Checkout/session correlation creates a pending internal checkout record. A verified Stripe subscription webhook resolves the opaque checkout reference and creates or updates the subject subscription.
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

- Stripe customer portal, checkout abandonment/expiration worker, retry/dead-letter handling, and billing telemetry beyond the checkout ledger, admin-managed webhook ledger, and upgrade-intent ledger.
- Tenant-facing seat-management UI, Stripe price editing, controlled commercial overrides beyond the platform-admin governance boundary, and customer self-service plan management beyond Checkout-start.
- Research-report generation/storage, portfolio CSV ingestion, batch-screening UI, custom-universe selector, API-key lifecycle, and shared-tenant branding controls.
- SRI discovery/detail response shaping beyond the current endpoint boundary.
- Full production purchase activation remains gated on controlled paid-flow evidence: Checkout completion, signed webhook reconciliation, effective subscription activation, return-to-context behavior, and Stripe Tax invoice verification.
