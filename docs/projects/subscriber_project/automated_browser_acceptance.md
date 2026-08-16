# Automated Browser Acceptance

Status: production-readiness control. The suite is committed; it remains inactive until a dedicated QA identity and tenant are provisioned.

## Purpose

The Subscriber Project must not depend on manually exported HAR files for routine release validation. The browser acceptance suite drives the real OIDC login screen and validates that the configured private watchlist persists across Assets, Dashboard, EROC, Valuation, EEOM, Market State, and Signal Assurance. It also verifies each contextual API response, shared-EOD coverage, honest cold-coverage presentation, and the absence of legacy sentinel timestamps or blank shared-algorithm cells. It performs no provider request, catalogue mutation, list creation, or data backfill.

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
- The Dashboard reports each declared warm symbol under `Shared EOD coverage`.
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

Run this suite after Gateway/Web deployment and before declaring a subscriber release accepted. A passing browser smoke validates UX and authorization propagation; it does not replace the separate global analytical-data-plane, provider, scheduler, parity, or recovery gates.

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
