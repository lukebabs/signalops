# Automated Browser Acceptance

Status: production-readiness control. The suite is committed; it remains inactive until a dedicated QA identity and tenant are provisioned.

## Purpose

The Subscriber Project must not depend on manually exported HAR files for routine release validation. The browser smoke suite drives the real OIDC login screen and validates that the configured private watchlist persists across Assets, Dashboard, EROC, Valuation, EEOM, and Market State. It performs no provider request, catalogue mutation, list creation, or data backfill.

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

## Production gate

Run this suite after Gateway/Web deployment and before declaring a subscriber release accepted. A passing browser smoke validates UX and authorization propagation; it does not replace the separate global analytical-data-plane, provider, scheduler, parity, or recovery gates.
