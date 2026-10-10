"""Istio Gateway API staging route smoke for SignalOps.

The test maps signalops-staging.syncratic.co to a local port-forward of the
Istio public gateway. It validates browser-visible SPA routing and gateway
probes without DNS, TLS, production traffic, or authenticated Keycloak callback
movement.
"""

from __future__ import annotations

import os
import re

import pytest
from playwright.sync_api import expect, sync_playwright


HOSTNAME = "signalops-staging.syncratic.co"


def staging_mesh_base_url() -> str:
    port = os.environ.get("SIGNALOPS_K8S_MESH2_LOCAL_PORT", "").strip()
    if not port:
        pytest.skip("SIGNALOPS_K8S_MESH2_LOCAL_PORT is set by the Mesh-2 smoke runner")
    assert port.isdigit(), "SIGNALOPS_K8S_MESH2_LOCAL_PORT must be numeric"
    return f"http://{HOSTNAME}:{port}"


def test_signalops_mesh2_staging_route_browser_smoke() -> None:
    base_url = staging_mesh_base_url()
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch(
            headless=True,
            args=[f"--host-resolver-rules=MAP {HOSTNAME} 127.0.0.1"],
        )
        context = browser.new_context(ignore_https_errors=False)
        page = context.new_page()
        try:
            page.goto(f"{base_url}/", wait_until="domcontentloaded", timeout=30_000)
            expect(page).to_have_title(re.compile("SignalOps"), timeout=10_000)
            assert "404 page not found" not in page.content().lower()

            page.goto(f"{base_url}/marketops/dashboard", wait_until="domcontentloaded", timeout=30_000)
            expect(page).to_have_title(re.compile("SignalOps"), timeout=10_000)
            body = page.locator("body")
            expect(body).not_to_contain_text("404 page not found", timeout=5_000)
            expect(body).not_to_contain_text("subscriber_watchlist_context_not_found", timeout=5_000)

            for path in ["/healthz", "/readyz"]:
                response = page.goto(f"{base_url}{path}", wait_until="domcontentloaded", timeout=10_000)
                assert response is not None
                assert response.status == 200, f"{path} returned {response.status}: {page.content()[:200]}"
                body_text = page.locator("body").inner_text(timeout=5_000)
                assert "signalops-gateway" in body_text, body_text
        finally:
            context.close()
            browser.close()
