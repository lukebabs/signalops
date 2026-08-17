# Subscription Commerce Foundation Release Evidence — 2026-08-17

Status: foundation deployed and verified; commercial enforcement remains disabled.

## Completed release evidence

- Migration `000147_subscriber_subscription_commerce_foundation` applied to the dedicated Subscriber gateway database at `2026-08-17 02:47:26 UTC`.
- The migration verification confirmed row-level security on every tenant-private subscription table, ownership by `signalops_subscriber_migrator`, the required gateway grants, and no public access to products or billing-webhook records.
- The MarketOps gateway was rebuilt and restarted. Its startup log confirms that MarketOps reads use the dedicated MarketOps data boundary.
- The MarketOps web build was rebuilt and restarted.
- The configured tenant-pilot-b read-only Playwright smoke completed successfully through the constrained deployment agent. It uses the pilot QA account only and does not provision plans, mutate watchlists, invoke data providers, or enable a feature flag.

## Deliberately unchanged

- `SIGNALOPS_SUBSCRIPTIONS_ENABLED` is absent from the runtime configuration and therefore remains `false` by its safe default.
- No customer is locked out and no subscription plan, seat, price, Stripe credential, webhook, or billing state was created.
- Existing MarketOps roles, tenant grants, subscriber lists, global-catalog processing, and provider schedules are unchanged.

## Required activation sequence

1. Assign `signalops:subscription_admin` only to controlled platform operators in Keycloak.
2. Use the signed platform provisioning boundary to create audited Explorer pilot subscriptions, then Professional and Institutional acceptance records with their intended subjects/seats.
3. Run direct API and browser acceptance evidence for all three tiers: Explorer discovery/public paths; Professional analysis paths; Institutional SAF/replay paths.
4. Set `SIGNALOPS_SUBSCRIPTIONS_ENABLED=true` only after the above evidence passes. Rollback remains a single configuration reversal to `false`; subscription and audit records remain intact.

Stripe Checkout, customer portal, signed webhook reconciliation, tenant seat-management UI, and the remaining Institutional work are separate future slices. They are not represented as completed by this release.

## Pilot provisioning evidence — 2026-08-17

The controlled platform subscription administrator provisioned the following records through the Administration workbench. These writes used the signed browser identity and normal gateway mutation boundary; no direct database write, provider call, worker, schedule, or feature-flag change was used.

- `tenant-pilot-b`: an active Explorer subject subscription for the controlled pilot identity.
- `tenant-local`: an active Institutional tenant contract and an active `tenant_admin` seat for `luke@strategiclabs.io`.

The dedicated subscription store retains one immutable audit event for each mutation: `subject_subscription_upserted`, `tenant_subscription_upserted`, and `tenant_subscription_seat_upserted`. The tenant-local seat resolves against the active Institutional contract.

The read-only administrator browser smoke passed after provisioning. `SIGNALOPS_SUBSCRIPTIONS_ENABLED` remains disabled, so these records prepare the acceptance matrix but do not restrict or expand any live analyst capability yet.

### Remaining pilot gate

Professional requires a separate controlled subject because a direct subject plan takes precedence over an Institutional tenant seat. Once that identity is nominated and provisioned, run the full three-tier acceptance matrix and retain the 402 negative-path evidence before requesting feature-flag activation.
