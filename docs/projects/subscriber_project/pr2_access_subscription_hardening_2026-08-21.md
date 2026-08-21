# PR-2 Access and Subscription Hardening — 2026-08-21

Status: started; first read-only tenant-isolation browser/API smoke implemented and passed.

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

## Remaining PR-2 items

1. Run the temporary subscription-enforcement canary with fresh named approval. This is intentionally not run automatically because it temporarily enables subscription enforcement and changes the pilot plan before restoring it.
2. Add or retain browser evidence for private-list ownership within the same tenant. Current unit/API coverage binds private-list mutations to the immutable subject, but live browser proof requires a second same-tenant user or a controlled temporary private-list mutation.
3. Expand/verify Subscription Administration as the operator-facing governance surface for enrolled users, tier assignment, entitlement state, quota state, default-list policy, and audit history.
4. Record cross-tenant/private-list evidence in the production-readiness checklist before claiming PR-2 exit.
