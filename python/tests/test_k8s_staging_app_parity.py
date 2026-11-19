"""Bounded Kubernetes staging app parity smoke.

This test intentionally avoids authenticated production user journeys. The
staging app port-forward uses localhost, while the live Keycloak client is
configured for the production callback host. Full authenticated parity remains
a separate gate that requires a staging hostname/client redirect.
"""

import os
import re

from playwright.sync_api import Page, expect


def staging_base_url() -> str:
    base_url = os.environ.get("SIGNALOPS_K8S_STAGING_BASE_URL", "").rstrip("/")
    assert base_url, "SIGNALOPS_K8S_STAGING_BASE_URL must be set"
    assert base_url.startswith("http://127.0.0.1:"), "staging parity must use a local port-forward"
    return base_url


def assert_gateway_probe(page: Page, path: str) -> None:
    response = page.request.get(f"{staging_base_url()}{path}", timeout=10_000)
    assert response.status == 200, f"{path} returned {response.status}: {response.text()[:200]}"
    payload = response.json()
    assert payload.get("service") == "signalops-gateway", payload


def test_k8s_staging_spa_shell_and_gateway_proxy(page: Page) -> None:
    base_url = staging_base_url()

    page.goto(f"{base_url}/", wait_until="domcontentloaded", timeout=30_000)
    expect(page).to_have_title(re.compile("SignalOps"), timeout=10_000)
    assert "404 page not found" not in page.content().lower()

    page.goto(f"{base_url}/marketops/dashboard", wait_until="domcontentloaded", timeout=30_000)
    expect(page).to_have_title(re.compile("SignalOps"), timeout=10_000)
    body = page.locator("body")
    expect(body).not_to_contain_text("404 page not found", timeout=5_000)
    expect(body).not_to_contain_text("subscriber_watchlist_context_not_found", timeout=5_000)

    assert_gateway_probe(page, "/healthz")
    assert_gateway_probe(page, "/readyz")
