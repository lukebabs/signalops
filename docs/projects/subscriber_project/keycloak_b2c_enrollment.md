# Keycloak B2C Enrollment Flow

Status: implementation slice added 2026-08-25. Production enablement still requires Keycloak realm configuration and live browser validation.

## Purpose

The existing Syncratic Keycloak login is reused for public MarketOps enrollment. Keycloak remains the identity provider; SignalOps remains the authority for MarketOps tenant access, Explorer subscription state, watchlist context, and audit evidence.

## Runtime contract

- Browser sign-in uses the configured OIDC Authorization Code + PKCE client. Production must use a Keycloak client that actually exists in the live realm and has the `signalops-api` audience and tenant claim mappers.
- Browser sign-up must be launched through the app/Gateway enrollment facade configured by `VITE_SIGNALOPS_AUTH_SIGNUP_URL`; the SPA must not deep-link directly to Keycloak with `kc_action=register`.
- The production signup facade must live on the app/deployment host, for example `https://signalops.syncratic.io/auth/login?redirect=/marketops/dashboard`, not on `auth.syncratic.co`.
- The SignalOps SPA implements `/auth/login` as the app-hosted facade: it sanitizes the internal `redirect` value, stores the post-login destination, and starts the configured OIDC sign-in flow.
- Create Account uses the same facade with `intent=register`; that intent starts Keycloak's OIDC `registrations` endpoint while preserving PKCE/state callback handling. Keycloak must still have realm registration enabled before the registration form will render.
- The gateway exposes `GET /v1/session/enrollment` for authenticated users before normal MarketOps access is complete.
- The enrollment resolver reads the signed token subject, tenant, display identity, email, and `email_verified` claim.
- Auto-enrollment is restricted to `SIGNALOPS_SUBSCRIBER_B2C_TENANT_ID`, default `tenant-b2c`.
- Verified B2C users may be idempotently provisioned with MarketOps read access and starter watchlist scaffolding, but production readiness requires an active subscription from governed administration or Stripe webhook reconciliation.
- SMS MFA enrollment and login challenge are Keycloak-owned controls. SignalOps may request the `CONFIGURE_SMS_MFA` required action during enrollment, but it must not set `phone_number_verified=true`; only the Keycloak SMS MFA provider may do that after verification.
- The resolver creates a tenant-default starter watchlist only when the B2C tenant has no readable list context.

## Guardrails

- Unauthenticated calls return `401`.
- Missing `tenant_id` still fails before enrollment.
- Unverified email returns `email_verification_required` and performs no provisioning.
- Non-B2C tenants are never self-provisioned; they return the relevant pending state for administrator remediation.
- B2C Explorer auto-activation is disabled by default through `SIGNALOPS_SUBSCRIBER_B2C_AUTO_ACTIVATE_EXPLORER=false`; enabling it is a controlled exception, not the production policy.
- Enrollment calls are rate-limited in the gateway by a hashed tenant/subject/IP key.
- Default public sign-ups must not receive operator, admin, subscription-admin, or tenant-provisioner roles.
- Phone verification is blocking when SMS MFA is required. Registration or required-action cancellation must not bypass MFA setup.
- The custom SMS MFA provider uses Brevo through `BREVO_API`; missing provider credentials must fail closed before production registration is opened.

## Keycloak configuration checklist

1. Enable registration only for the intended public client/flow.
2. Require email verification before access-token sessions are treated as eligible for MarketOps enrollment.
3. Emit exactly one `tenant_id` claim for B2C registrations, set to the configured B2C tenant.
4. Include the `signalops-api` audience in access tokens.
5. Emit `email`, `email_verified`, `preferred_username`, and realm/client roles in the access token.
6. Assign only the minimum viewer role for public registrations.
7. Keep institutional and tenant-admin users on explicit admin provisioning paths.
8. Install the custom SMS MFA provider image and reconcile the browser authentication flow with `syncratic-sms-otp-authenticator`.
9. Configure the required action `CONFIGURE_SMS_MFA` and set `SMS_MFA_ENROLLMENT_POLICY=required_for_new_local_users` only after `BREVO_API` is present.
10. New local users should receive `VERIFY_EMAIL`, `UPDATE_PASSWORD`, and `CONFIGURE_SMS_MFA`; future login SMS challenge should occur only after `sms_mfa_enabled=true` and `phone_number_verified=true`.

## UI behavior

- Unauthenticated users always see **Sign in**. They see **Create account** only when `VITE_SIGNALOPS_AUTH_SIGNUP_URL` is configured for the deployment.
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
- It verifies that the **Create account** action is visible only when `VITE_SIGNALOPS_AUTH_SIGNUP_URL` is configured.
- It follows that action to the configured app/Gateway enrollment facade.
- It asserts that the flow reaches the governed registration/enrollment entry point without using a raw Keycloak `kc_action=register` URL.
- It never submits the registration form and never creates a user.

Useful overrides:

```bash
SIGNALOPS_E2E_BASE_URL=https://signalops.syncratic.io
SIGNALOPS_E2E_AUTH_HOST=auth.syncratic.co
SIGNALOPS_E2E_CLIENT_ID=<configured-live-client-id>
VITE_SIGNALOPS_AUTH_SIGNUP_URL=https://signalops.syncratic.io/auth/login?redirect=/marketops/dashboard
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
- `self_enrollment.eligible=true`;
- for new B2C subjects without a governed subscription, `state=subscription_missing` when subscription enforcement is enabled.
- ready users reach the MarketOps Dashboard.

The smoke does not create a Keycloak user and does not touch Stripe. If the supplied B2C QA subject is not already enrolled, the SignalOps resolver may perform its designed idempotent B2C self-enrollment mutations: MarketOps read access, Explorer subject subscription, and starter tenant-default watchlist if missing. That is why the runner requires `SIGNALOPS_B2C_ENROLLMENT_SMOKE_ACK=approved`.

The same browser test can validate an already-provisioned non-B2C QA account by explicitly overriding the expected tenant and self-enrollment state. For the current `SIGNALOPS_WEB` pilot account, use:

```bash
source scripts/lib/dotenv.sh
load_dotenv .env
SIGNALOPS_B2C_ENROLLMENT_SMOKE_ACK=approved \
SIGNALOPS_B2C_WEB="$SIGNALOPS_WEB" \
SIGNALOPS_B2C_WEB_PASS="$SIGNALOPS_WEB_PASS" \
SIGNALOPS_E2E_B2C_TENANT_ID=tenant-pilot-b \
SIGNALOPS_E2E_EXPECT_CAN_SELF_ENROLL=false \
./scripts/run_keycloak_b2c_enrollment_authenticated_smoke.sh
```

This proves the authenticated enrollment resolver and protected Dashboard entry work for the existing pilot account; it does not prove the B2C self-enrollment branch.

## Live Keycloak B2C QA evidence — 2026-08-25

Live Keycloak was inspected through the `keycloak` container using `kcadm.sh`; no secrets or tokens were printed. The controlled B2C QA identity `luke.babarinde@gmail.com` now has:

- `emailVerified=true`;
- realm role `signalops:viewer`;
- user attribute `tenant_id=["tenant-b2c"]`.

The then-current `signalops-web` client had the required token mappers during the 2026-08-25 QA inspection:

- `signalops-api-audience`, emitting `signalops-api` in the access token audience;
- `tenant-id-from-user-attribute`, emitting the user `tenant_id` attribute into access/userinfo/introspection tokens.

The `signalops-api` client exists and is enabled as a bearer-only API client.

Current realm-level public enrollment posture after the 2026-08-25 form enablement:

- `registrationAllowed=true`;
- `verifyEmail=true`;
- `/signalops/viewers` is configured as a default group for new registrants.

This only renders and protects the Keycloak registration form. It does not complete production enrollment by itself: the running Keycloak image still lacks the `CONFIGURE_SMS_MFA` provider/action, and new registrants still need governed `tenant_id=tenant-b2c` assignment before SignalOps can accept their token.

## HAR finding: registration entrypoint mismatch — 2026-08-25

`har/auth.syncratic.co.har` showed a failed registration launch before any form submission, cookie capture, SMS MFA step, or successful callback. The primary failure was a raw Keycloak authorization request returning `400` with `client_id=signalops-web`, `redirect_uri=https://signalops.syncratic.io/auth/callback`, and `kc_action=register`. A separate `404` came from opening `/auth/login?redirect=/app/admin` on `auth.syncratic.co`; that route belongs on the app/Gateway host, not the raw Keycloak host.

Operational conclusion: treat this as an auth entrypoint/client reconciliation problem, not an SMS MFA problem. Production must use exactly one of these controlled paths:

1. Preferred: configure `VITE_SIGNALOPS_AUTH_SIGNUP_URL` to the app/Gateway facade URL and let that facade build the Keycloak request with the live Syncratic client.
2. Alternative: reconcile a dedicated Keycloak `signalops-web` browser client with `https://signalops.syncratic.io/auth/callback`, web origin `https://signalops.syncratic.io`, PKCE, `signalops-api` audience, and `tenant_id` mapper before exposing direct OIDC registration.

Until one path is verified in live browser smoke, public self-registration remains closed.

The existing `.env` pilot browser smoke still validates the authenticated resolver for `tenant-pilot-b`. The true B2C self-enrollment browser smoke passed on 2026-08-25 using `SYNCRATIC_QA_CLIENT` / `SYNCRATIC_QA_PASS` mapped to the smoke variables. The already-provisioned QA account resolved to `tenant-b2c`, `marketops_ready`, `email_verified=true`, and `self_enrollment.eligible=true`. Under the production Option B policy, a new B2C account without a governed subscription must resolve to `subscription_missing` once subscription enforcement is enabled.

Existing-user safety validation is part of the smoke contract: an already-provisioned identity must return `self_enrollment.created=[]`. That proves the account is resolving through the login/enrollment path without creating a duplicate access grant or subscription enrollment.


## SMS MFA provider handoff — 2026-08-25

SMS MFA is Keycloak-owned, not application-owned. The custom provider lives under `keycloak/sms-provider`, is built into the provider-enabled Keycloak image with `keycloak/Dockerfile`, and sends SMS OTP through Brevo using `BREVO_API`.

SignalOps integration boundary:

- Gateway may append `CONFIGURE_SMS_MFA` when `SMS_MFA_ENROLLMENT_POLICY=required_for_new_local_users`.
- SignalOps must not directly set `phone_number_verified=true` or otherwise mark phone verification complete.
- Phone verification must be completed by the Keycloak provider before `sms_mfa_enabled=true` and `phone_number_verified=true` allow future login SMS challenge.
- The configured browser flow must include `syncratic-sms-otp-authenticator` after the user has enrolled SMS MFA.

Operational rollout:

1. Ensure `BREVO_API` is present in the Keycloak runtime secret source.
2. Build and redeploy the provider-enabled Keycloak image.
3. For Compose, run `scripts/reconcile_keycloak_sms_mfa.sh`; for Kubernetes, rely on the Helm hook exposed by `auth.smsMfa`.
4. Create a new local-user registration smoke and confirm required actions include `VERIFY_EMAIL`, `UPDATE_PASSWORD`, and `CONFIGURE_SMS_MFA`.
5. Complete registration with a real SMS-capable test number and confirm cancel/skip cannot bypass required-action setup.
6. Sign in again after enrollment and confirm the SMS OTP challenge appears only for users with `sms_mfa_enabled=true` and `phone_number_verified=true`.

Upgrade warning: the Keycloak SPI is built against Keycloak `25.0.6`. Keycloak marks these SPIs as internal, so any Keycloak version upgrade must rebuild the provider image and run a registration/login SMS MFA smoke before production traffic is allowed.

## Deferred production work

- Stripe Checkout and customer portal remain disabled until webhook-confirmed activation is implemented.
- Durable enrollment lifecycle projection beyond the current subscription/access/activity ledgers remains a later enhancement.
- Cross-replica distributed rate limiting should be added if more than one gateway replica serves enrollment traffic.
