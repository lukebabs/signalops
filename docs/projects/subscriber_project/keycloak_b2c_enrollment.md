# Keycloak B2C Enrollment Flow

Status: implementation slice added 2026-08-25. Production enablement still requires Keycloak realm configuration and live browser validation.

## Purpose

The existing Syncratic Keycloak login is reused for public MarketOps enrollment. Keycloak remains the identity provider; SignalOps remains the authority for MarketOps tenant access, Explorer subscription state, watchlist context, and audit evidence.

## Runtime contract

- Browser sign-in uses the existing OIDC Authorization Code + PKCE flow for `signalops-web`.
- Browser sign-up uses the same OIDC client and callback, adding Keycloak `kc_action=register` to start the registration action without bypassing PKCE/state handling.
- The gateway exposes `GET /v1/session/enrollment` for authenticated users before normal MarketOps access is complete.
- The enrollment resolver reads the signed token subject, tenant, display identity, email, and `email_verified` claim.
- Auto-enrollment is restricted to `SIGNALOPS_SUBSCRIBER_B2C_TENANT_ID`, default `tenant-b2c`.
- Verified B2C users are idempotently provisioned with MarketOps read access and an active Explorer subject subscription.
- The resolver creates a tenant-default starter watchlist only when the B2C tenant has no readable list context.

## Guardrails

- Unauthenticated calls return `401`.
- Missing `tenant_id` still fails before enrollment.
- Unverified email returns `email_verification_required` and performs no provisioning.
- Non-B2C tenants are never self-provisioned; they return the relevant pending state for administrator remediation.
- Enrollment calls are rate-limited in the gateway by a hashed tenant/subject/IP key.
- Default public sign-ups must not receive operator, admin, subscription-admin, or tenant-provisioner roles.

## Keycloak configuration checklist

1. Enable registration only for the intended public client/flow.
2. Require email verification before access-token sessions are treated as eligible for MarketOps enrollment.
3. Emit exactly one `tenant_id` claim for B2C registrations, set to the configured B2C tenant.
4. Include the `signalops-api` audience in access tokens.
5. Emit `email`, `email_verified`, `preferred_username`, and realm/client roles in the access token.
6. Assign only the minimum viewer role for public registrations.
7. Keep institutional and tenant-admin users on explicit admin provisioning paths.

## UI behavior

- Unauthenticated users see both **Sign in** and **Create account**.
- Authenticated users must resolve enrollment before the protected MarketOps router mounts.
- Pending states show specific remediation language for email verification, tenant access, subscription setup, or watchlist setup.
- Ready users enter the normal MarketOps Dashboard flow.

## Browser preflight

Run the enrollment smoke after the gateway/web deployment and after Keycloak registration is enabled:

```bash
./scripts/run_keycloak_b2c_enrollment_ui_smoke.sh
```

The smoke is intentionally non-mutating:

- It opens the public SignalOps entry point.
- It verifies that the **Create account** action is visible.
- It follows that action to the configured Keycloak host.
- It asserts that the registration form or registration markers are visible.
- It never submits the registration form and never creates a user.

Useful overrides:

```bash
SIGNALOPS_E2E_BASE_URL=https://signalops.syncratic.io
SIGNALOPS_E2E_AUTH_HOST=auth.syncratic.co
SIGNALOPS_E2E_CLIENT_ID=signalops-web
SIGNALOPS_E2E_ARTIFACT_DIR=/tmp/signalops-enrollment-e2e-artifacts
```

Failure artifacts are retained under `SIGNALOPS_E2E_ARTIFACT_DIR`; successful runs remove the generated HAR.

## Authenticated resolver smoke

After a dedicated B2C QA account exists in Keycloak, run the authenticated resolver smoke:

```bash
SIGNALOPS_B2C_ENROLLMENT_SMOKE_ACK=approved \
SIGNALOPS_B2C_WEB=<existing-b2c-qa-email> \
SIGNALOPS_B2C_WEB_PASS=<existing-b2c-qa-password> \
./scripts/run_keycloak_b2c_enrollment_authenticated_smoke.sh
```

This test signs in through Keycloak, waits for `GET /v1/session/enrollment`, and asserts:

- the response tenant is the configured B2C tenant, default `tenant-b2c`;
- the enrollment state matches `SIGNALOPS_E2E_ENROLLMENT_EXPECTED_STATE`, default `marketops_ready`;
- `email_verified=true`;
- `can_self_enroll=true`;
- ready users reach the MarketOps Dashboard.

The smoke does not create a Keycloak user and does not touch Stripe. If the supplied B2C QA subject is not already enrolled, the SignalOps resolver may perform its designed idempotent B2C self-enrollment mutations: MarketOps read access, Explorer subject subscription, and starter tenant-default watchlist if missing. That is why the runner requires `SIGNALOPS_B2C_ENROLLMENT_SMOKE_ACK=approved`.

## Deferred production work

- Stripe Checkout and customer portal remain disabled until webhook-confirmed activation is implemented.
- Durable enrollment lifecycle projection beyond the current subscription/access/activity ledgers remains a later enhancement.
- Cross-replica distributed rate limiting should be added if more than one gateway replica serves enrollment traffic.
