"""Read-only browser proof for Stripe Checkout readiness on Pricing.

This test never starts Checkout. It verifies the Pricing UI accurately reflects
the Gateway checkout_enabled state returned by /subscription-products.
"""

from __future__ import annotations

import os
from pathlib import Path

import pytest
from playwright.sync_api import Browser, Page, expect

from test_subscriber_ui_smoke import SubscriberUIConfig, login, subscriber_ui_config
from test_subscription_enforcement_canary_ui import bearer


@pytest.fixture(scope="session")
def pricing_config() -> SubscriberUIConfig:
    return subscriber_ui_config()


@pytest.fixture
def pricing_page(pricing_config: SubscriberUIConfig, browser: Browser, request: pytest.FixtureRequest) -> Page:
    artifact_dir = Path(os.getenv("SIGNALOPS_E2E_ARTIFACT_DIR", "/tmp/signalops-e2e-artifacts"))
    artifact_dir.mkdir(mode=0o700, parents=True, exist_ok=True)
    artifact_dir.chmod(0o700)
    har_path = artifact_dir / f"{request.node.name}.har"
    context = browser.new_context(record_har_path=str(har_path), record_har_mode="minimal")
    context.tracing.start(screenshots=True, snapshots=True, sources=True)
    page = context.new_page()
    try:
        yield page
    except Exception:
        page.screenshot(path=str(artifact_dir / f"{request.node.name}.png"), full_page=True)
        context.tracing.stop(path=str(artifact_dir / f"{request.node.name}.zip"))
        raise
    else:
        context.tracing.stop()
        har_path.unlink(missing_ok=True)
    finally:
        context.close()


def test_stripe_checkout_readiness_state_matches_gateway(pricing_page: Page, pricing_config: SubscriberUIConfig) -> None:
    login(pricing_page, pricing_config)
    pricing_page.goto(f"{pricing_config.base_url}/marketops/pricing", wait_until="domcontentloaded")
    expect(pricing_page.get_by_role("heading", name="Increase analytical depth when the research question requires it.")).to_be_visible(timeout=30_000)

    product_response = pricing_page.evaluate(
        """async (token) => {
          const response = await fetch("/v1/marketops/subscription-products", {headers: {Authorization: "Bearer " + token}});
          return {status: response.status, body: await response.json().catch(() => ({}))};
        }""",
        bearer(pricing_page),
    )
    assert int(product_response["status"]) == 200, product_response["body"]
    products = {item.get("product_key"): item for item in product_response["body"].get("products", [])}
    assert products.get("explorer", {}).get("monthly_display_price") == "$24.99/mo"
    assert products.get("explorer", {}).get("annual_display_price") == "$249/yr"
    assert products.get("professional", {}).get("monthly_display_price") == "$99/mo"
    assert products.get("professional", {}).get("annual_display_price") == "$999/yr"

    body = pricing_page.locator("body")
    for display_price in ["$24.99/mo", "$249/yr", "$99/mo", "$999/yr"]:
        expect(body).to_contain_text(display_price, timeout=30_000)
    expect(body).not_to_contain_text("price_")

    subscription_response = pricing_page.evaluate(
        """async ({tenantId, token}) => {
          const response = await fetch(`/v1/tenants/${tenantId}/marketops/subscription`, {headers: {Authorization: "Bearer " + token}});
          return {status: response.status, body: await response.json().catch(() => ({}))};
        }""",
        {"tenantId": pricing_config.tenant_id, "token": bearer(pricing_page)},
    )
    assert int(subscription_response["status"]) == 200, subscription_response["body"]
    if subscription_response["body"].get("subscription"):
        expect(body).to_contain_text("Need billing help?", timeout=30_000)
        expect(body).to_contain_text("Submit a refund request for administrator review", timeout=30_000)
        expect(pricing_page.get_by_role("button", name="Request refund")).to_be_disabled()

    checkout_enabled = bool(product_response["body"].get("checkout_enabled"))

    monthly_buttons = pricing_page.get_by_role("button", name="Monthly Checkout")
    annual_buttons = pricing_page.get_by_role("button", name="Annual Checkout")
    expect(monthly_buttons.first).to_be_visible(timeout=30_000)
    expect(annual_buttons.first).to_be_visible(timeout=30_000)

    if checkout_enabled:
        expect(pricing_page.locator("body")).not_to_contain_text("Gateway is missing a Stripe API key")
        expect(monthly_buttons.first).to_be_enabled()
        expect(annual_buttons.first).to_be_enabled()
    else:
        expect(pricing_page.locator("body")).to_contain_text("Gateway is missing a Stripe API key", timeout=30_000)
        for index in range(monthly_buttons.count()):
            expect(monthly_buttons.nth(index)).to_be_disabled()
        for index in range(annual_buttons.count()):
            expect(annual_buttons.nth(index)).to_be_disabled()
