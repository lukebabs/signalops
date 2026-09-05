"""Controlled production canary for Stripe Checkout startup.

This test creates Stripe Checkout Sessions but does not complete payment. It is
skipped unless SIGNALOPS_STRIPE_CHECKOUT_CANARY=1 is set by the wrapper script.
"""

from __future__ import annotations

import json
import os
from pathlib import Path

import pytest
from playwright.sync_api import Browser, Page, expect

from test_subscriber_ui_smoke import SubscriberUIConfig, login, subscriber_ui_config
from test_subscription_enforcement_canary_ui import bearer


def checkout_artifact_path() -> Path:
    artifact_dir = Path(os.getenv("SIGNALOPS_E2E_ARTIFACT_DIR", "/tmp/signalops-e2e-artifacts"))
    artifact_dir.mkdir(mode=0o700, parents=True, exist_ok=True)
    artifact_dir.chmod(0o700)
    return artifact_dir / "stripe-checkout-canary-refs.json"


@pytest.fixture(scope="session")
def checkout_config() -> SubscriberUIConfig:
    if os.getenv("SIGNALOPS_STRIPE_CHECKOUT_CANARY", "").strip() != "1":
        pytest.skip("Stripe Checkout canary is disabled")
    return subscriber_ui_config()


@pytest.fixture
def checkout_page(checkout_config: SubscriberUIConfig, browser: Browser, request: pytest.FixtureRequest) -> Page:
    artifact_dir = Path(os.getenv("SIGNALOPS_E2E_ARTIFACT_DIR", "/tmp/signalops-e2e-artifacts"))
    artifact_dir.mkdir(mode=0o700, parents=True, exist_ok=True)
    artifact_dir.chmod(0o700)
    context = browser.new_context(
        record_har_path=str(artifact_dir / f"{request.node.name}.har"),
        record_har_mode="minimal",
    )
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
        (artifact_dir / f"{request.node.name}.har").unlink(missing_ok=True)
    finally:
        context.close()


def api_post_checkout(page: Page, tenant_id: str, product_key: str, billing_period: str) -> dict[str, str]:
    response = page.evaluate(
        """async ({tenantId, productKey, billingPeriod, token}) => {
          const response = await fetch(`/v1/tenants/${tenantId}/marketops/subscription/checkout`, {
            method: "POST",
            headers: {Authorization: "Bearer " + token, "Content-Type": "application/json"},
            body: JSON.stringify({product_key: productKey, billing_period: billingPeriod})
          });
          return {status: response.status, body: await response.json().catch(() => ({}))};
        }""",
        {"tenantId": tenant_id, "productKey": product_key, "billingPeriod": billing_period, "token": bearer(page)},
    )
    assert int(response["status"]) == 200, response["body"]
    body = response["body"]
    assert isinstance(body, dict), body
    checkout_url = str(body.get("checkout_url", ""))
    checkout_ref = str(body.get("checkout_ref", ""))
    stripe_session_id = str(body.get("stripe_session_id", ""))
    assert checkout_url.startswith("https://checkout.stripe.com/"), body
    assert checkout_ref.startswith("subcheckout-"), body
    assert stripe_session_id.startswith("cs_"), body
    return {
        "product_key": product_key,
        "billing_period": billing_period,
        "checkout_url": checkout_url,
        "checkout_ref": checkout_ref,
        "stripe_session_id": stripe_session_id,
    }


def test_stripe_checkout_start_creates_sessions_without_payment(checkout_page: Page, checkout_config: SubscriberUIConfig) -> None:
    login(checkout_page, checkout_config)
    checkout_page.goto(f"{checkout_config.base_url}/marketops/pricing", wait_until="domcontentloaded")
    expect(checkout_page.get_by_role("heading", name="Increase analytical depth when the research question requires it.")).to_be_visible(timeout=30_000)

    product_response = checkout_page.evaluate(
        """async (token) => {
          const response = await fetch("/v1/marketops/subscription-products", {headers: {Authorization: "Bearer " + token}});
          return {status: response.status, body: await response.json().catch(() => ({}))};
        }""",
        bearer(checkout_page),
    )
    assert int(product_response["status"]) == 200, product_response["body"]
    products = {item.get("product_key"): item for item in product_response["body"].get("products", [])}
    for product_key in ["explorer", "professional"]:
        product = products.get(product_key)
        assert product, f"missing product {product_key}"
        assert product.get("stripe_product_id"), product
        assert product.get("stripe_monthly_price_id"), product
        assert product.get("stripe_annual_price_id"), product

    refs = [
        api_post_checkout(checkout_page, checkout_config.tenant_id, "explorer", "monthly"),
        api_post_checkout(checkout_page, checkout_config.tenant_id, "professional", "annual"),
    ]
    checkout_artifact_path().write_text(json.dumps(refs, indent=2, sort_keys=True))
