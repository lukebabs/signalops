# Subscription Commerce Foundation Release Evidence — 2026-08-17

Status: foundation deployed and verified; Administration governance for enrolled users and tier policy was added on 2026-08-19; commercial enforcement remains disabled.

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

### Prepared three-tier matrix

A separate, established tenant-local non-admin QA identity now has an active Professional subject plan. The prepared acceptance matrix is therefore:

- Explorer — controlled `tenant-pilot-b` subscriber.
- Professional — controlled `tenant-local` non-admin subscriber.
- Institutional — `luke@strategiclabs.io` as the active `tenant_admin` seat on the `tenant-local` Institutional contract.

Direct subject plans take precedence over Institutional tenant seats, so the three distinct identities avoid masking a tier during testing.

### Remaining pilot gate

Run the live browser/API acceptance matrix and retain the 402 negative-path evidence before requesting feature-flag activation. Professional browser acceptance requires that controlled QA identity’s login credential to be supplied to the protected QA environment; no credential was added or exposed during this provisioning operation.

## Controlled temporary production enforcement canary — 2026-08-17

Status: **passed**. This was a bounded production proof, not a persistent commercial activation.

The named approval authorized a temporary gateway-only enforcement window. The constrained deployment agent enabled the flag only in an isolated temporary Compose environment, ran the browser/API matrix below, restored the controlled pilot subject to Explorer, regenerated the dedicated gateway credential, and restarted the gateway from the normal configuration. At completion the live gateway reported `SIGNALOPS_SUBSCRIPTIONS_ENABLED=false`.

| Tier | Controlled evidence | Result |
| --- | --- | --- |
| Explorer | `tenant-pilot-b` controlled subject | Value Intelligence browser route locked; valuation API returned `402 subscription_feature_required`; public Sector Rotation rankings continued to return `200`. |
| Professional | Same controlled subject, temporarily and audibly changed to Professional | Value Intelligence browser route and valuation API returned normal access; Signal Assurance browser route remained locked and its API returned `402 subscription_feature_required`. |
| Institutional | `luke@strategiclabs.io` active `tenant_admin` seat for the active `tenant-local` Institutional contract | Signal Assurance browser route and effectiveness API returned normal access. |

All temporary Explorer → Professional → Explorer changes were performed through the signed Subscription Administration boundary and retained immutable audit events. No provider request, scheduler, data-plane job, tenant contract, institutional seat, or production dotenv was changed.

### Canary hardening findings closed

- The gateway configuration read `SIGNALOPS_SUBSCRIPTIONS_ENABLED`, but the Compose gateway service did not inject it. The missing environment mapping is now fixed.
- The dedicated subscriber-gateway credential must be refreshed before a gateway-only restart. The canary runner now uses the proven boundary credential preparation flow.
- The runner now waits for both a running container and the gateway `/readyz` endpoint before browser validation. It always restores the normal enforcement-off configuration in its exit path.

The canary action is narrowly allowlisted in the deployment agent. It cannot accept arbitrary commands or persistently enable enforcement. The remaining business decision is a separately named approval to enable commercial enforcement beyond this temporary proof; that approval is not implied by the successful canary.


## Administration governance enhancement — 2026-08-19

The Subscription Administration workbench was expanded from a provisioning-only form into a governance console. It now reads a tenant-scoped subscription snapshot and shows product/tier policies, subject subscriptions, Institutional contracts, seats, and audit history. Platform subscription administrators can update tier feature policy, limit policy, active state, display name, and trial days through a signed, audited API.

Implementation boundaries:

- no new migration was required; the slice uses the existing `000147` subscription commerce tables;
- product-policy updates increment product revision and write audit evidence under the signed administrator tenant;
- provisioning remains signed-role-only through `signalops:subscription_admin`;
- no Stripe Checkout, webhook reconciliation, provider polling, feature-flag activation, or customer self-service upgrade path was added.

Validation retained:

- `go test ./internal/api ./internal/storage/postgres`;
- `npm --prefix web run build`;
- `git diff --check`.


### Administration governance browser validation — 2026-08-19

The live Administration → Subscription governance surface was validated by `luke@strategiclabs.io` on `tenant-local` after the Gateway/Web rebuild and Traefik route restoration. The browser validation confirmed that the page loads and the governance surface is usable for the intended production-administrator workflow.

Validated scope:

- `/admin/subscriptions` loads for the subscription administrator;
- tier cards are visible for Explorer, Professional, and Institutional;
- tenant-local enrollment, contract/seat, and audit sections render;
- the product-policy editor loads the current feature and limit policy;
- public routing is restored through `https://signalops.syncratic.io` with `/readyz` returning 200.

This closes the browser-validation item for the 2026-08-19 subscription governance enhancement. Commercial enforcement remains off until separately approved.
