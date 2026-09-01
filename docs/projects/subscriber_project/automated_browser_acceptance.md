# Automated Browser Acceptance

Status: production-readiness control. The suite is committed; it remains inactive until a dedicated QA identity and tenant are provisioned.

## Purpose

The Subscriber Project must not depend on manually exported HAR files for routine release validation. The browser acceptance suite drives the real OIDC login screen and validates that the configured private watchlist persists across Assets, Dashboard, EROC, Value & Distressed Opportunity Intelligence (VC/DOSM), Earnings Opportunity Intelligence (EEOM), Market State, and Signal Assurance. It also verifies each contextual API response, Dashboard analytical scope, honest cold-coverage presentation, and the absence of legacy sentinel timestamps or blank shared-algorithm cells. It performs no provider request, catalogue mutation, list creation, or data backfill.

## Identity boundary

Use a non-human, read-only QA account in an isolated tenant. Do not use a personal account, an administrator, or a production customer identity. The QA account needs MarketOps read access and one pre-seeded private list with safe fixture symbols. Its password must exist only in a root-readable deployment secret; it is never committed, printed, or attached to CI output.

## Pilot account mapping

The read-only launcher `scripts/run_subscriber_pilot_ui_smoke.sh` maps the protected
`SIGNALOPS_WEB` and `SIGNALOPS_WEB_PASS` values to the suite's pilot identity. It
uses the verified `tenant-pilot-b` fixture by default: `First List`, shared
`AAPL,NVDA`, and pending `NOW,SNOW`. It parses `.env` as literal data through
`scripts/lib/dotenv.sh`; it does not source or print that file.

`SIGNALOPS_WEB_ADMIN` and `SIGNALOPS_WEB_PASS_ADMIN` identify the tenant-local
administrator. They are intentionally excluded from the normal subscriber
smoke: production user-journey evidence must be produced by the isolated,
read-only pilot identity. A future tenant-local admin check must be an explicit
separate control with its own narrowly scoped assertions.

## Automated invocation

The deployment-control agent exposes `subscriber-pilot-ui-smoke`, which runs the
pilot contract as the approved host operator with the installed Chromium path.
`marketops-gateway-deploy` runs that same smoke after the Gateway has passed its
dedicated-read cutover verification. The weekday post-close workflow runs it
after the global Dashboard projection gate succeeds whenever the pilot credentials
are configured. A browser failure exits the post-close workflow with a distinct
non-zero status and is therefore visible through the normal scheduled-job and
administrator-notification controls.

## Enrollment smoke

The public B2C enrollment entry point has its own credential-free smoke:

```bash
./scripts/run_keycloak_b2c_enrollment_ui_smoke.sh
```

This check verifies only the unauthenticated front door and governed registration handoff. It does not submit the registration form, create a user, mutate Stripe state, or provision tenant access. Run it after Web/Gateway deployment and after Keycloak registration settings change.

The expected production path is app-host first, for example `https://signalops.syncratic.io/auth/login?redirect=/marketops/dashboard`. The smoke must fail if the Create Account action deep-links directly to raw Keycloak with `kc_action=register`, or if `/auth/login` is opened on `auth.syncratic.co` instead of the app/Gateway host.

When SMS MFA is enabled, this smoke remains non-mutating and validates only that the registration handoff reaches the governed entry point. A separate real-number Keycloak smoke is required to complete `CONFIGURE_SMS_MFA`, verify that cancel/skip cannot bypass required action setup, and confirm the subsequent login SMS OTP challenge. Do not automate or persist real phone numbers in repo artifacts.

The post-registration resolver has a separate opt-in smoke:

```bash
SIGNALOPS_B2C_ENROLLMENT_SMOKE_ACK=approved ./scripts/run_keycloak_b2c_enrollment_authenticated_smoke.sh
```

That check requires an existing B2C QA identity in `SIGNALOPS_B2C_WEB` and `SIGNALOPS_B2C_WEB_PASS`. It may exercise the gateway's idempotent B2C self-enrollment path for that subject, so it is not part of the default public-front-door smoke. For an existing QA identity, the expected safe outcome is `self_enrollment.created=[]`, proving the user was routed through normal login/enrollment resolution rather than re-enrolled.

## Installation

The repository pins the browser-test dependency in `python/requirements-e2e.txt`.

```bash
python3 -m venv .venv
.venv/bin/python -m pip install -r python/requirements-e2e.txt
.venv/bin/python -m playwright install chromium
```

## Protected execution

Supply these variables through a root-readable environment file, not the shell history or `.env` committed to the project:

```text
SIGNALOPS_E2E_BASE_URL=https://signalops.syncratic.io
SIGNALOPS_E2E_USERNAME=<qa-identity>
SIGNALOPS_E2E_PASSWORD=<qa-password>
SIGNALOPS_E2E_WATCHLIST_NAME=<pre-seeded-private-list>
SIGNALOPS_E2E_TENANT_ID=<isolated-qa-tenant-id>
SIGNALOPS_E2E_SHARED_TICKERS=<warm-fixtures,for-example-AAPL,NVDA>
SIGNALOPS_E2E_PENDING_TICKERS=<cold-fixtures,for-example-NOW,SNOW>
SIGNALOPS_E2E_ARTIFACT_DIR=/var/lib/signalops/e2e-artifacts
```

Then execute:

```bash
set -a
. /etc/signalops/subscriber-ui-smoke.env
set +a
.venv/bin/python -m pytest -q python/tests/test_subscriber_ui_smoke.py
```

On success, the temporary HAR is removed. On failure, the suite retains a HAR, Playwright trace, and full-page screenshot in the configured artifact directory. Those artifacts contain browser session material and must be mode `0700`, excluded from Git, and retained under the operational evidence policy.

## UX contract

The pre-seeded fixture list must contain the exact symbols declared in `SIGNALOPS_E2E_SHARED_TICKERS` and `SIGNALOPS_E2E_PENDING_TICKERS`. The test validates the following user-visible contract:

- The chosen private list survives a full browser reload and remains selected on every MarketOps view.
- Assets identified as shared show central evidence and never show `1-12-31`, `Awaiting monitor`, or `Awaiting EOD analysis`; those states would misleadingly mix legacy tenant-only algorithm gaps into a shared row.
- Cold assets remain visible as `Pending` and appear in the explicit `Coverage in progress` panel. They are not silently omitted or shown as completed data.
- The Dashboard remains scoped to the selected list's analytical breadth and never renders an operational shared-EOD asset dump. Shared EOD provenance is shown only on the selected Assets rows.
- Assets, Dashboard, EROC, Valuation, and EEOM responses all return the saved list context, with the configured fixture symbols in scope. Market State sends the configured tenant and a configured in-scope symbol.
- Signal Assurance returns a non-empty platform-global historical-outcome cohort for the selected list. The test explicitly requires `LEGACY` evidence only until a real confirmed SAF assertion exists; it must never represent an absent assertion as a confirmed result.

A failure of the shared-evidence checks is intentionally a release failure, not a reason to export another HAR. It is the automated proof that the global analytical-data-plane blocker remains open.

## First controlled execution — 2026-08-14

The first live run used the dedicated `testsignalops` identity in `tenant-pilot-b`, with `First List` as the private fixture. The test passed the SignalOps landing page, branded identity-provider authentication, private-list activation, context persistence, Assets route, and contextual Assets API response.

The first substantive assertion failed as intended: `AAPL` was marked `Shared` but rendered `Unavailable`, `Awaiting monitor`, `Awaiting EOD analysis`, and `Regular · 1-12-31 19:03:58 ET`. This is direct production evidence that current shared EOD projection is not yet accompanied by the global analytical evidence plane. The run retained a protected HAR, trace, and screenshot under `/tmp/signalops-e2e-artifacts`; they must not be committed or distributed.

During this run, the public front door initially returned `404` before the application loaded because the Web service had been deployed without `compose.traefik.yaml`. Reapplying the Web-only deployment with that overlay restored public `/` and `/marketops/watchlists` routing to `200`. This is now part of the required deployment command for public Web changes:

```bash
docker compose --env-file .env -p signalops -f compose.yaml -f compose.traefik.yaml \
  up -d --build --no-deps web
```

Result: the Subscriber Project remains blocked from production acceptance by the documented global analytical-data-plane gap. The browser test is now a reproducible release control for that gap.

## UX truth-gate validation — 2026-08-14

After the shared and pending Asset projection remediation, the same isolated `tenant-pilot-b` browser contract passed: authentication, private-list activation, reload persistence, Assets, Dashboard, EROC, valuation, EEOM, and Market State all preserved the selected list and tenant scope. Shared rows now render verified global-EOD provenance and explicitly identify global intraday/Risk-Reward/Market-State evidence as unmaterialized. Pending rows render only their central coverage state. Neither class renders the legacy sentinel timestamp, tenant-local algorithm placeholder, or Market-State link.

This closes the **UX truthfulness** sub-gate only. It does not close the global analytical-data-plane production blocker: the referenced algorithm and historical evidence still need to be materialized platform-wide.

## Production gate

Run this suite after Gateway/Web deployment and before declaring a subscriber release accepted. Also run the controlled Syncratic Ask smoke after Gateway deployments that touch Syncratic, after Syncratic AI Gateway policy/catalog changes, and before subscription production gates. A passing browser smoke validates UX and authorization propagation; it does not replace the separate global analytical-data-plane, provider, scheduler, parity, or recovery gates.

## Mobile subscriber acceptance

The default subscriber smoke validates product correctness and authorization, but it is not sufficient for subscriber mobile production readiness.

The Mobile User Readiness Sprint adds a separate mobile suite focused on phone-first subscriber use. Admin Workbench, Subscription Administration, and operator run-now controls remain out of scope for that suite.

Committed launcher:

```bash
./scripts/run_subscriber_mobile_ui_smoke.sh
```

Committed suite:

```bash
.venv/bin/python -m pytest -q python/tests/test_subscriber_mobile_ui_smoke.py
```

The mobile suite uses the same protected QA identity and failure-only artifact policy as the subscriber smoke. The current committed slice runs at 375px, 390px, and 430px phone widths and verifies production login plus Dashboard, Watchlists, Assets, Sector Rotation, Signal Assurance, Syncratic Intelligence, Dashboard-to-Syncratic handoff, Assets mobile card drilldown, SAF mobile drilldown, and SRI ETF progression/makeup drilldown with no 404, watchlist-context error, or page-level horizontal overflow. Remaining expansion should add Opportunities drilldowns, enrollment, subscription pricing, and gated-route behavior.

Required subscriber routes are defined in [Mobile User Readiness Sprint](mobile_user_readiness_sprint.md). A passing mobile smoke is required before declaring the subscriber web product ready for primarily mobile users.

## Global EROC reader acceptance — 2026-08-16

After the controlled, algorithm-filtered materialization of 1,346 EROC v6 records, the pilot contract was strengthened to require a non-empty EROC result for a selected shared symbol and `data_scope = platform-global`. The named Gateway deployment and the isolated `tenant-pilot-b` smoke both passed. This proves the subscriber EROC route reads authorized global evidence rather than legacy tenant-local data.

The route is fail-closed. If the global reader is unavailable, it returns `global_eroc_unavailable` rather than substituting a tenant-local result. This test covers one reader type only; the other analytical projections remain independently gated.

## Global Valuation reader acceptance — 2026-08-16

The pilot contract now requires a non-empty Valuation result for a selected shared symbol and `data_scope = platform-global`. The live `tenant-pilot-b` smoke passed after the VC/DOSM-only global projection was materialized and the Gateway restarted. It proves that subscriber Valuation no longer reads a tenant-local copy. Tactical valuation and posture are not included in this proof and remain separately gated.

## Global EEOM reader acceptance — 2026-08-16

The live pilot smoke passed after the EEOM global reader was materialized and deployed. When an EEOM event is present for a selected watchlist symbol, the contract requires `data_scope = platform-global`; it does not manufacture an expectation that the AAPL/NVDA fixture must have a future earnings event. This proves provenance without turning ordinary calendar absence into a false release failure.

## Global Material Events reader acceptance — 2026-08-16

The Material Events API regression contract requires selected-watchlist symbols to be authorized before reading the restricted global projection and requires returned events to carry `data_scope = platform-global`. The live bootstrap proved the Gateway role can read 20 central FMP events; the isolated pilot browser smoke passed after deployment. The smoke does not require a fixture symbol to have an upcoming earnings event, because calendar absence is valid and must not be converted into a false failure.

## Global Signal Assurance historical-outcome acceptance — 2026-08-16

The global SAF projection contains 92 complete directional historical outcomes, of which 46 are directional matches, from the immutable legacy-parity materialization. The isolated pilot list contains seven applicable observations across NOW and NVDA. The reader is filtered by the selected watchlist, reads only the restricted `platform-global` projection, and is fail-closed when that projection is unavailable. There are currently **zero** confirmed SAF assertions; the UI and API explicitly disclose that absence rather than upgrading historical outcomes into SAF assertions. The browser contract now requires the global scope, the persisted watchlist context, a non-empty historical cohort, and `LEGACY` evidence source only.

### SRI platform-global acceptance

For the pilot subscriber, navigate to `/marketops/sectors` and assert that the
rankings response has `data_scope: "platform-global"`, a resolved authorized
`watchlist_context`, non-empty historical snapshots, and the research-only
evidence note. Exercise one ETF progression and issuer-makeup request. A
missing global projection must surface a controlled unavailable error; the UI
must never receive tenant-local SRI as a fallback.


### Fixture reconciliation evidence — 2026-08-16

The controlled tenant-pilot-b `First List` currently contains AAPL and NVDA. The browser-smoke defaults were reconciled to that actual fixture; NOW and SNOW are no longer asserted as list members. The explicit run using the protected pilot identity passed both browser contracts on 2026-08-16: watchlist context propagation/global-coverage assertions and the SRI platform-global projection assertion (2 passed).

## Syncratic Ask browser acceptance — 2026-08-23

The controlled tenant-local Syncratic Ask smoke is available through `scripts/run_syncratic_ask_ui_smoke.sh`. It logs in with `SIGNALOPS_WEB_ADMIN`, opens `/marketops/syncratic`, selects a daily narrative, clicks normal Ask (`force=false`), and verifies the `/v1/syncratic/context-windows/{id}/ask` response. Success removes the temporary HAR; failure retains protected HAR, trace, and screenshot artifacts.

The first live failure identified two upstream gates: missing `Idempotency-Key`, then `gateway_price_catalog_not_found`. SignalOps now sends the required idempotency header, and after the AI Gateway price-catalog configuration propagated the smoke passed: `1 passed in 1.43s`. This smoke is now required after Syncratic-related Gateway deployments, AI Gateway policy/catalog changes, and before subscription production gates.

The standing readiness contract is maintained in [Syncratic Ask Readiness Checklist](syncratic_ask_readiness_checklist.md). That checklist is the source of truth for required controls, expected commands, failure interpretation, and remaining production-hardening work.
