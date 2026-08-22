# Stripe Admin-Managed Billing

Status: implementation slice added on 2026-08-22; migration `000155_subscriber_admin_stripe_billing` is applied and the Administration UI is live. Production webhook activation still requires `STRIPE_WEBHOOK_SECRET` in the gateway runtime and controlled signed-webhook validation.

## Purpose

This slice connects Stripe as billing evidence without enabling customer self-service checkout or a billing portal. Platform subscription administrators create and manage customers/subscriptions in Stripe Dashboard, then map Stripe IDs into SignalOps Subscription Administration. SignalOps remains authoritative for tenant binding, roles, tier policy, seats, and feature enforcement.

## What is included

- Product-level Stripe mapping for product, monthly price, and annual price IDs.
- Subject-level Stripe mapping for Explorer/Professional subscriptions.
- Tenant-level Stripe mapping for Institutional contracts.
- Signed Stripe webhook endpoint at `POST /v1/billing/stripe/webhook`.
- Idempotent webhook ledger in `subscriber_billing_webhook_events`.
- Reconciliation for mapped Stripe subscription IDs only.
- Admin UI visibility for billing mappings and webhook processing state.

## What is not included

- Stripe Checkout.
- Stripe billing portal.
- Customer self-service upgrade/downgrade.
- Automatic user, tenant, subject, or seat creation from Stripe.
- Provider polling or MarketOps data entitlement changes.

## Webhook behavior

The gateway verifies the `Stripe-Signature` header using `STRIPE_WEBHOOK_SECRET`. Invalid signatures return `400` and are not persisted. Supported events are:

- `customer.subscription.created`;
- `customer.subscription.updated`;
- `customer.subscription.deleted`;
- `invoice.payment_succeeded`;
- `invoice.payment_failed`.

A known Stripe subscription ID updates the mapped subject or tenant subscription status and billing dates. An unknown Stripe subscription ID is retained as `unmatched`; it does not create access. Duplicate provider event IDs are idempotent and return the retained processing status.

## Operational flow

1. Platform admin creates product/prices and customer/subscription in Stripe Dashboard.
2. Platform admin opens Administration > Subscriptions.
3. Platform admin maps product price IDs and subject/tenant Stripe IDs.
4. Stripe sends signed webhook events to the gateway.
5. SignalOps records the provider event and reconciles only mapped local subscriptions.
6. Subscription enforcement continues to evaluate local SignalOps records.

## Production configuration

Required before live webhook validation:

```text
STRIPE_WEBHOOK_SECRET=<Stripe endpoint signing secret>
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

## Stripe Tax note

Before selling paid plans broadly, evaluate Stripe Tax and registrations. Do not assume tax is collected just because Billing is enabled; Stripe collects tax only after the required registrations and tax configuration are active.

## Acceptance checks

- Migration `000155_subscriber_admin_stripe_billing` is applied.
- Subscription admin can save product billing IDs.
- Subscription admin can save subject and tenant Stripe IDs.
- Non-mutating webhook preflight reports the current endpoint state.
- A valid signed test webhook updates a mapped subscription after named approval.
- An invalid signature returns `400`.
- An unknown Stripe subscription event records as `unmatched` and creates no access.
- Existing subscription enforcement and tenant-isolation smokes still pass.
