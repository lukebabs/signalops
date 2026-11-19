# PR-2 Access and Subscription Hardening — 2026-08-21

Status: closed for the configured production QA identities.

## Scope

PR-2 hardens the subscriber access-control and subscription boundary before wider pilot expansion.

The sprint covers:

- tenant-bound route enforcement for real OIDC browser sessions;
- private-list and tenant-default authorization evidence;
- Explorer, Professional, and Institutional entitlement behavior;
- Subscription Administration governance for product/tier policy, subject plans, tenant contracts, seats, and audit evidence;
- repeatable canaries that leave production state clean.

## First slice — read-only tenant-boundary smoke

Added `python/tests/test_subscriber_access_control_ui.py` and runner `scripts/run_subscriber_access_control_ui_smoke.sh`.

The smoke logs in with the protected QA identities already configured in `.env`:

- `SIGNALOPS_WEB` / `SIGNALOPS_WEB_PASS`: tenant-pilot-b subscriber identity.
- `SIGNALOPS_WEB_ADMIN` / `SIGNALOPS_WEB_PASS_ADMIN`: tenant-local administrator/subscription-admin identity.

It performs only GET requests. It does not create watchlists, alter subscriptions, change tenant contracts, start jobs, or call providers.

Assertions:

1. The pilot identity can read tenant-pilot-b subscriber lists.
2. The pilot identity can read tenant-pilot-b MarketOps signal overview.
3. The pilot identity receives `403 tenant_mismatch` for tenant-local subscriber lists.
4. The pilot identity receives `403 tenant_mismatch` for tenant-local MarketOps signal overview.
5. The pilot identity receives `403 insufficient_role` for Subscription Administration.
6. The tenant-local administrator can read tenant-local subscriber lists.
7. The tenant-local administrator receives `403 tenant_mismatch` for tenant-pilot-b tenant data routes.
8. The platform subscription administrator can read Subscription Administration snapshots for tenant-local and tenant-pilot-b, because that is the intentionally controlled cross-tenant administration boundary.

## Verification

```text
scripts/run_subscriber_access_control_ui_smoke.sh
.                                                                        [100%]
1 passed in 3.55s

go test ./internal/api ./internal/storage/postgres
ok   github.com/lukebabs/signalops/internal/api              (cached)
ok   github.com/lukebabs/signalops/internal/storage/postgres  (cached)

bash -n scripts/run_subscriber_access_control_ui_smoke.sh
```

## Current evidence position

The live browser/API evidence now proves that ordinary tenant-bearing MarketOps routes are bound to the signed token tenant for the two configured QA identities.

It also proves that cross-tenant Subscription Administration is not a general tenant-data bypass: it is isolated to `/v1/administration/subscriptions` and requires `signalops:subscription_admin` or equivalent platform admin authority.

## Temporary subscription-enforcement canary — 2026-08-21

Named approval:

```text
I, luke@strategiclabs.io, approve one temporary production subscription-enforcement canary for tenant-pilot-b, with automatic restoration to enforcement-off and pilot Explorer state.
```

Execution path:

```text
sudo -n signalops-deploy-agent subscription-enforcement-canary
```

Result:

```text
subscription_enforcement_canary_enabled
.                                                                        [100%]
1 passed in 3.90s
subscription_enforcement_canary_verified
subscription_enforcement_canary_restored
```

The canary temporarily enabled gateway subscription enforcement, established the controlled pilot Explorer baseline, verified Explorer denial for Professional-only features, temporarily changed the pilot to Professional, verified Professional access to Value Intelligence, verified Professional denial for Institutional-only Signal Assurance analytics, verified tenant-local Institutional/admin Signal Assurance access, and restored the pilot to Explorer with enforcement off.

Post-restore verification:

```text
https://signalops.syncratic.io/readyz       -> 200
https://signalops.syncratic.io/admin/system -> 200
sudo -n signalops-deploy-agent scheduler-status -> clean
scripts/run_subscriber_access_control_ui_smoke.sh -> 1 passed in 3.64s
scripts/run_subscription_admin_ui_smoke.sh        -> 2 passed in 2.20s
```

Direct container environment inspection with `sudo docker exec` was not passwordlessly available in this session, so restoration was verified by the canary's own restoration marker plus post-restore route, scheduler, tenant-isolation, and Admin browser smokes.

## Closure evidence — private-list ownership and governance surface

The read-only tenant-isolation smoke was strengthened to decode the real browser JWT subject and verify returned private lists are owned by that subject.

Evidence:

```text
scripts/run_subscriber_access_control_ui_smoke.sh
.                                                                        [100%]
1 passed in 3.62s
```

This verifies:

- the tenant-pilot-b QA identity has at least one private list;
- every returned pilot private list has `owner_subject` equal to the signed JWT `sub`;
- the tenant-local administrator response contains no foreign private-list owner if private lists are present;
- tenant-pilot-b and tenant-local remain mutually denied from each other's tenant-bearing MarketOps routes with `403 tenant_mismatch`;
- the pilot subscriber remains denied from Subscription Administration with `403 insufficient_role`;
- the platform subscription administrator can read the intentional cross-tenant subscription-admin snapshots.

The Subscription Administration smoke was strengthened to assert the operator governance surface and API snapshot, including product tiers, feature policies, limit policies, revisions, subject subscriptions, tenant contracts, seats, audit trail, and key entitlement labels.

Evidence:

```text
scripts/run_subscription_admin_ui_smoke.sh
...                                                                      [100%]
3 passed in 3.06s
```

Targeted backend tests remain clean:

```text
go test ./internal/api ./internal/storage/postgres
ok   github.com/lukebabs/signalops/internal/api              (cached)
ok   github.com/lukebabs/signalops/internal/storage/postgres  (cached)

bash -n scripts/run_subscriber_access_control_ui_smoke.sh scripts/run_subscription_admin_ui_smoke.sh scripts/run_subscription_enforcement_canary_ui.sh
```

## Closure position

PR-2 is closed for the configured production QA identities. The evidence now covers tenant isolation, private-list owner-subject projection, Subscription Administration role denial for non-admin users, controlled cross-tenant subscription administration, tier-enforcement canary behavior, automatic restoration, and the operator governance surface.

A future expansion with a second same-tenant non-admin identity can add stronger adversarial same-tenant private-list leakage proof, but it is no longer blocking this PR-2 loop because source/API tests already bind private-list reads and mutations to the immutable subject, and live evidence confirms returned private-list owner subjects match the signed token.
