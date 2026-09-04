# Stripe Admin-Managed Billing

Status: admin-managed billing slice added on 2026-08-22 and self-service Checkout backend slice added on 2026-08-25. Migration `000155_subscriber_admin_stripe_billing` is applied; migration `000161_subscriber_stripe_checkout_sessions` adds the internal checkout ledger. The Administration UI is live, `STRIPE_WEBHOOK_SECRET` and `STRIPE_RESTRICTED_API_KEY` are injected into the gateway runtime, controlled signed-webhook validation has passed for unmapped and mapped subscriptions, and the first real live Explorer Checkout payment has activated through webhook-authoritative reconciliation. Product/price mappings are configured for Explorer and Professional.

## Purpose

This slice connects Stripe as billing evidence and as the payment processor for Explorer/Professional Checkout. SignalOps remains authoritative for tenant binding, subject identity, roles, tier policy, seats, and feature enforcement. Stripe receives only an opaque checkout reference for self-service activation; tenant and subject mapping stay in the dedicated MarketOps database. Institutional remains admin/contract managed.

## What is included

- Product-level Stripe mapping for product, monthly price, and annual price IDs.
- Subject-level Stripe mapping for Explorer/Professional subscriptions.
- Tenant-level Stripe mapping for Institutional contracts.
- Signed Stripe webhook endpoint at `POST /v1/billing/stripe/webhook`.
- Idempotent webhook ledger in `subscriber_billing_webhook_events`.
- Authenticated Checkout endpoint at `POST /v1/tenants/{tenant_id}/marketops/subscription/checkout`.
- Pricing UI Checkout controls for Explorer and Professional.
- Stripe return UX at `/marketops/subscription/return?session_id=...` with webhook-authoritative activation polling.
- Opaque checkout ledger in `subscriber_checkout_sessions`.
- Reconciliation for mapped Stripe subscription IDs and webhook-confirmed checkout references.
- Admin UI visibility for billing mappings and webhook processing state.

## What is not included

- Stripe billing portal.
- Customer self-service downgrade/cancel/change-payment workflows.
- Automatic user, tenant, or seat creation from Stripe.
- Provider polling or MarketOps data entitlement changes.


## Refund request workflow — 2026-09-04

Refunds are admin-governed. A subscriber can request a refund from `/marketops/pricing` after they have an effective subscription. SignalOps records the request in `subscriber_refund_requests` and surfaces it in Administration > Subscriptions > Refund requests. The request creates audit evidence and queue visibility; it does not move money.

Current operational boundary:

1. Subscriber submits reason and optional amount.
2. Gateway resolves tenant and immutable OIDC subject from the authenticated token.
3. Gateway creates a tenant-scoped refund-request row against the subscriber's active/trialing/past-due subject subscription.
4. Subscriber receives only safe acknowledgement metadata: request ID, status, amount/currency, and timestamps. Stripe customer/subscription/session IDs are not returned to the subscriber.
5. `signalops:subscription_admin` reviews the request in Admin Subscription.
6. If approved, the admin executes the actual refund in Stripe Dashboard, then records the disposition in SignalOps as `manual_refund_completed` with a note/reference.

The allowed status model is: `requested`, `reviewing`, `approved_for_manual_refund`, `rejected`, `manual_refund_completed`, and `closed`. This intentionally avoids a misleading `refunded` state until SignalOps owns a signed, idempotent Stripe refund executor and refund-webhook reconciliation path.

## Webhook behavior

The gateway verifies the `Stripe-Signature` header using `STRIPE_WEBHOOK_SECRET`. Invalid signatures return `400` and are not persisted. Supported events are:

- `checkout.session.completed`;
- `customer.subscription.created`;
- `customer.subscription.updated`;
- `customer.subscription.deleted`;
- `invoice.payment_succeeded`;
- `invoice.payment_failed`.

A known Stripe subscription ID updates the mapped subject or tenant subscription status and billing dates. A Stripe subscription event or completed Checkout Session with a known opaque `checkout_ref` activates or updates the stored subject subscription for Explorer/Professional. An unknown Stripe subscription ID/reference is retained as `unmatched`; it does not create access. Duplicate provider event IDs are idempotent and return the retained processing status.

## Operational flow

1. Platform admin creates product/prices and customer/subscription in Stripe Dashboard.
2. Platform admin opens Administration > Subscriptions.
3. Platform admin maps product price IDs and subject/tenant Stripe IDs.
4. Stripe sends signed webhook events to the gateway.
5. SignalOps records the provider event and reconciles only mapped local subscriptions.
6. Subscription enforcement continues to evaluate local SignalOps records.

## Production configuration

Required before live webhook/Checkout validation:

```text
STRIPE_WEBHOOK_SECRET=<Stripe endpoint signing secret>
STRIPE_API_KEY=<restricted Stripe secret key with Checkout Session create/read scope>
SIGNALOPS_STRIPE_CHECKOUT_SUCCESS_URL=https://signalops.syncratic.io/marketops/subscription/return?session_id={CHECKOUT_SESSION_ID}
SIGNALOPS_STRIPE_CHECKOUT_CANCEL_URL=https://signalops.syncratic.io/marketops/pricing
```

Use Stripe test mode first. The Stripe secret must be injected as a runtime secret and must not be committed to the repository.

## Canary procedure

Use the non-mutating preflight first:

```bash
scripts/run_stripe_webhook_canary.sh
```

Expected results:

- `{"status":"stripe_webhook_disabled", ...}` means the endpoint is deployed but `STRIPE_WEBHOOK_SECRET` is not configured in the running gateway.
- `{"status":"rejected_invalid_signature", ...}` means the secret is configured and invalid signatures fail closed without persistence.

After the Stripe endpoint signing secret is configured and the planned canary has named approval, run the persistent valid-signature check:

```bash
scripts/run_stripe_webhook_canary.sh --allow-persistent-ledger
```

That command creates or reuses one Stripe webhook ledger row. Use an intentionally unmapped `SIGNALOPS_STRIPE_CANARY_SUBSCRIPTION_ID` to prove unknown subscriptions become `unmatched`, or use a mapped subscription ID to prove reconciliation updates a known local subscription.

Current production canary evidence:

- Unmapped subscription canary `sub_signalops_unmapped_canary_20260822` resolved to `unmatched` and created no subject or tenant access.
- Mapped pilot canary `sub_signalops_mapped_canary_20260822` resolved to `processed` under provider event `evt_signalops_canary_1787425181`.
- The mapped canary updated tenant `tenant-pilot-b`, subject `2f437ac3-2cfc-4fe9-b943-198185b4797b`, to `status=active`, `provisioned_by=stripe-webhook`.
- The mapped canary created `stripe_subscription_reconciled` audit evidence scoped to `tenant-pilot-b`.

## Stripe Tax readiness

Operator-reported status as of 2026-08-23: Stripe Tax information has been configured in Stripe. Because SignalOps currently uses an admin-managed billing model, Stripe remains the operational surface for creating subscriptions and applying tax behavior. SignalOps records the resulting subscription state through mapped IDs and signed webhooks.

Required Stripe-side controls before broad paid launch:

- Tax Settings must have an active head office/origin address.
- Tax registrations must be active in jurisdictions where collection is required. Adding a registration in Stripe records where the business is already registered; it is not a legal registration by itself.
- Explorer and Professional Products must have an appropriate Stripe product tax code selected from Stripe's canonical tax-code list. Do not use `Nontaxable` unless that is the confirmed tax position.
- Monthly and annual Prices must have the intended tax behavior.
- Real Stripe-created subscriptions, invoices, or future Checkout Sessions must have `automatic_tax` enabled.
- Customers must have enough valid address information for Stripe Tax to resolve tax location.

Validation boundary:

- The SignalOps webhook canaries prove signature verification, idempotent ledgering, unmatched subscription safety, mapped subscription reconciliation, and tenant-scoped audit evidence.
- The SignalOps webhook canaries do not prove Stripe calculated or collected tax, because they use synthetic signed webhook payloads rather than real Stripe invoice calculation.
- Before paid launch, validate one test-mode Explorer subscription and one test-mode Professional subscription from Stripe Dashboard and inspect the generated invoice tax results.

Operational note: Stripe Tax calculates and collects tax when configured correctly, but filing/remittance remains an operational responsibility through Stripe filing products, filing partners, or manual processes. Confirm obligations with a tax advisor.


## Real paid activation evidence — 2026-09-04

A live Explorer monthly Checkout payment completed for session `cs_live_a1PXQR4g96ZkMkJFkiOJMVoSQhrES4u2mBRgJaxptYjCyqFkZlFLaICeIN`. Stripe returned `status=complete`, `payment_status=paid`, subscription `sub_1UBoUu8w1ilrMHjMPE6RGdBH`, customer `cus_VCCuc6id2ct0vu`, and metadata `checkout_ref=subcheckout-ac7ea41094f5498af111dfe7`.

Root cause of the initial non-activation was operational, not payment failure: no live Stripe webhook endpoint was configured for SignalOps, so Stripe events did not reach the Gateway. The Gateway was also hardened to support `checkout.session.completed` directly, in addition to subscription and invoice events.

Closure evidence:

- Live Stripe webhook endpoint `we_1UBob38w1ilrMHjMGAiMODEs` was created for `https://signalops.syncratic.io/v1/billing/stripe/webhook` with subscription, invoice, and checkout-completed events enabled.
- The endpoint signing secret was stored only in the gitignored runtime environment and loaded by the redeployed Gateway.
- The real Stripe `customer.subscription.created` event `evt_1UBoUw8w1ilrMHjM3UOV9CAI` was replayed through the signed webhook endpoint and processed successfully.
- `subscriber_checkout_sessions` moved `subcheckout-ac7ea41094f5498af111dfe7` to `webhook_processed` and recorded the Stripe subscription ID.
- `subscriber_subject_subscriptions` now has the B2C subject active on Explorer with `provisioned_by=stripe-webhook`.
- Authenticated B2C Playwright smoke passed with state `marketops_ready`, confirming the user no longer routes to Pricing after activation.

## Checkout-start canary closure — 2026-09-04

After correcting Stripe product tax-code configuration, the controlled live Checkout-start canary passed.

Evidence:

- `scripts/run_stripe_checkout_canary.sh` returned `1 passed` and `stripe_checkout_canary_verified sessions=2`.
- Explorer monthly Checkout Session was created for `price_1UBkkF8w1ilrMHjMacm0rTny` with opaque reference `subcheckout-eca0827e65acdc76af3a9ece`.
- Professional annual Checkout Session was created for `price_1UBkqT8w1ilrMHjMVdNVFLrq` with opaque reference `subcheckout-082a2ba58a4fe5d9b88434a3`.
- Both internal checkout ledger rows are `checkout_started`, `checkout_url_returned=true`, and retain empty `stripe_subscription_id`; this confirms Checkout-start works and access is still webhook-authoritative.

Remaining paid-flow acceptance:

1. Complete one controlled Stripe test/live payment only when the business is ready to create a real subscription.
2. Verify the signed Stripe webhook transitions the matching `checkout_ref` into an active subject subscription.
3. Verify Stripe invoice tax output in Stripe Dashboard.

## Checkout readiness evidence — 2026-09-03

Production evidence on 2026-09-03:

- Explorer product mapping: `prod_V7YY6OYlzF94MO`, monthly `price_1U7JRI9yvXTfstCXUvv6QZkr`, annual `price_1U7JRI9yvXTfstCXukjjXpJ1`.
- Professional product mapping: `prod_V7YZToB4EPMh3e`, monthly `price_1U7JSP9yvXTfstCXmEktmULQ`, annual `price_1U7JSP9yvXTfstCXGujxKmPw`.
- Gateway runtime has `STRIPE_WEBHOOK_SECRET` and Checkout return URLs present.
- Gateway runtime has empty `STRIPE_API_KEY` and empty `STRIPE_RESTRICTED_API_KEY`; therefore `POST /v1/tenants/{tenant_id}/marketops/subscription/checkout` correctly fails closed with `stripe_checkout_disabled`.
- `GET /v1/marketops/subscription-products` now returns `checkout_enabled` so the Pricing UI can disable Checkout controls when runtime configuration is incomplete.
- Read-only Pricing readiness Playwright smoke passed and verified the UI disables Checkout controls while the API key is absent.

To complete the real Checkout-start canary after adding the restricted Stripe key:

```bash
sudo -n signalops-deploy-agent marketops-gateway-deploy
scripts/run_stripe_checkout_canary.sh
```

The canary creates Stripe Checkout Sessions only. It does not complete payment. Entitlement activation still requires a verified Stripe webhook containing the opaque `checkout_ref`.

## Acceptance checks

- Migration `000155_subscriber_admin_stripe_billing` is applied.
- Subscription admin can save product billing IDs.
- Subscription admin can save subject and tenant Stripe IDs.
- Non-mutating webhook preflight reports the current endpoint state.
- A valid signed test webhook updates a mapped subscription after named approval.
- An invalid signature returns `400`.
- An unknown Stripe subscription event records as `unmatched` and creates no access.
- Stripe Tax readiness has been validated in Stripe test mode for Explorer and Professional before broad paid launch.
- Existing subscription enforcement and tenant-isolation smokes still pass.
- Migration `000161_subscriber_stripe_checkout_sessions` is applied before enabling Checkout in production.
- Checkout creates a Stripe Session with only opaque `checkout_ref` metadata and records the internal tenant/subject/product mapping in MarketOps.
- Pricing UI redirects only to the gateway-returned Stripe Checkout URL; it cannot submit arbitrary Price IDs.
- The return page may show activation pending and poll the effective subscription endpoint, but redirect-only success does not grant access.
- A verified subscription webhook with that `checkout_ref` activates the subject subscription.
