# Automated Browser Acceptance

Status: production-readiness control. The suite is committed; it remains inactive until a dedicated QA identity and tenant are provisioned.

## Purpose

The Subscriber Project must not depend on manually exported HAR files for routine release validation. The browser acceptance suite drives the real OIDC login screen and validates that the configured private watchlist persists across Assets, Dashboard, EROC, Valuation, EEOM, and Market State. It also verifies each contextual API response, shared-EOD coverage, honest cold-coverage presentation, and the absence of legacy sentinel timestamps or blank shared-algorithm cells. It performs no provider request, catalogue mutation, list creation, or data backfill.

## Identity boundary

Use a non-human, read-only QA account in an isolated tenant. Do not use a personal account, an administrator, or a production customer identity. The QA account needs MarketOps read access and one pre-seeded private list with safe fixture symbols. Its password must exist only in a root-readable deployment secret; it is never committed, printed, or attached to CI output.

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
