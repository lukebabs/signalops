"""Read-only browser evidence for the platform Subscription Administration workspace."""

from __future__ import annotations

import os
import re
from pathlib import Path

import pytest
from playwright.sync_api import Browser, Page, expect


@pytest.fixture(scope="session")
def admin_config() -> tuple[str, str, str]:
    username = os.getenv("SIGNALOPS_E2E_ADMIN_USERNAME", "").strip()
    password = os.getenv("SIGNALOPS_E2E_ADMIN_PASSWORD", "").strip()
    base_url = os.getenv("SIGNALOPS_E2E_BASE_URL", "https://signalops.syncratic.io").rstrip("/")
    if not username or not password:
        pytest.skip("Subscription admin UI smoke is not configured")
    return base_url, username, password


@pytest.fixture
def admin_page(admin_config: tuple[str, str, str], browser: Browser, request: pytest.FixtureRequest) -> Page:
    artifact_dir = Path(os.getenv("SIGNALOPS_E2E_ARTIFACT_DIR", "/tmp/signalops-e2e-artifacts"))
    artifact_dir.mkdir(mode=0o700, parents=True, exist_ok=True)
    artifact_dir.chmod(0o700)
    stem = request.node.name
    har_path = artifact_dir / f"{stem}.har"
    trace_path = artifact_dir / f"{stem}.zip"
    screenshot_path = artifact_dir / f"{stem}.png"
    context = browser.new_context(record_har_path=str(har_path), record_har_mode="minimal")
    context.tracing.start(screenshots=True, snapshots=True, sources=True)
    page = context.new_page()
    try:
        yield page
    finally:
        failed = bool(getattr(request.node, "rep_call", None) and request.node.rep_call.failed)
        if failed:
            page.screenshot(path=str(screenshot_path), full_page=True)
            context.tracing.stop(path=str(trace_path))
        else:
            context.tracing.stop()
        context.close()
        if not failed:
            har_path.unlink(missing_ok=True)


def login(page: Page, config: tuple[str, str, str]) -> None:
    base_url, username, password = config
    page.goto(f"{base_url}/admin/subscriptions", wait_until="domcontentloaded")
    if page.get_by_role("heading", name="Subscription Administration").is_visible():
        return
    sign_in = page.get_by_role("button", name="Sign in")
    if sign_in.is_visible():
        sign_in.click()
    username_input = page.locator("#username, input[name='username']").or_(page.get_by_role("textbox", name="Email or username")).first
    username_input.wait_for(state="visible", timeout=30_000)
    username_input.fill(username)
    password_input = page.locator("#password, input[name='password']").or_(page.get_by_role("textbox", name="Password")).first
    password_input.fill(password)
    page.locator("#kc-login, input[type='submit']").or_(page.get_by_role("button", name="Continue")).first.click()
    page.wait_for_url(re.compile(re.escape(base_url) + r"/admin/subscriptions"), timeout=30_000)


def login_marketops_admin(page: Page, config: tuple[str, str, str]) -> None:
    base_url, username, password = config
    page.goto(f"{base_url}/admin/system", wait_until="domcontentloaded")
    if page.get_by_role("heading", name="MarketOps Operations Health").is_visible(timeout=5_000):
        return
    sign_in = page.get_by_role("button", name="Sign in")
    if sign_in.is_visible(timeout=10_000):
        sign_in.click()
    username_input = page.locator("#username, input[name='username']").or_(page.get_by_role("textbox", name="Email or username")).first
    username_input.wait_for(state="visible", timeout=30_000)
    username_input.fill(username)
    password_input = page.locator("#password, input[name='password']").or_(page.get_by_role("textbox", name="Password")).first
    password_input.fill(password)
    page.locator("#kc-login, input[type='submit']").or_(page.get_by_role("button", name="Continue")).first.click()
    expect(page.get_by_role("heading", name="MarketOps Operations Health")).to_be_visible(timeout=30_000)


def test_marketops_admin_operations_health_freshness_rows(admin_page: Page, admin_config: tuple[str, str, str]) -> None:
    expected_labels = {
        "Dashboard",
        "Assets analytical coverage",
        "Market State",
        "Risk/Reward",
        "Sector Rotation Intelligence",
        "Signal Assurance",
        "Intraday conditions",
        "FMP annual financials",
    }
    with admin_page.expect_response(
        lambda response: response.request.method == "GET" and "/v1/administration/marketops/operations-health" in response.url,
        timeout=30_000,
    ) as response_info:
        login_marketops_admin(admin_page, admin_config)
    response = response_info.value
    assert response.status == 200, f"{response.url} returned HTTP {response.status}"
    payload = response.json()
    rows = payload.get("marketops_operations_health", {}).get("data_freshness")
    assert isinstance(rows, list), "operations-health response did not include data_freshness rows"
    labels = {str(row.get("label", "")) for row in rows}
    assert expected_labels.issubset(labels), f"missing freshness rows: {sorted(expected_labels - labels)}"

    expect(admin_page.get_by_role("heading", name="MarketOps Operations Health")).to_be_visible(timeout=30_000)
    body = admin_page.locator("body")
    for label in sorted(expected_labels):
        expect(body).to_contain_text(label, timeout=30_000)



def test_subscription_administration_governance_surface(admin_page: Page, admin_config: tuple[str, str, str]) -> None:
    expected_products = {"Explorer", "Professional", "Institutional"}
    expected_tables = {"Subject subscriptions", "Institutional contracts", "Institutional seats", "Audit trail", "Stripe webhook ledger"}
    expected_features = {"Market dashboards", "Value Intelligence", "Distressed Opportunity Intelligence", "Earnings Opportunity Intelligence", "Signal Assurance analytics", "APIs", "White-label deployment"}

    with admin_page.expect_response(
        lambda response: response.request.method == "GET" and "/v1/administration/subscriptions" in response.url,
        timeout=30_000,
    ) as response_info:
        login(admin_page, admin_config)
    response = response_info.value
    assert response.status == 200, f"{response.url} returned HTTP {response.status}"
    payload = response.json()
    products = payload.get("products")
    assert isinstance(products, list) and len(products) >= 3, "subscription administration did not return product policy rows"
    product_names = {str(product.get("display_name", "")) for product in products}
    assert expected_products.issubset(product_names), f"missing product tiers: {sorted(expected_products - product_names)}"
    for product in products:
        assert isinstance(product.get("feature_policy"), dict), f"{product.get('product_key')} has no feature policy"
        assert isinstance(product.get("limit_policy"), dict), f"{product.get('product_key')} has no limit policy"
        assert "revision" in product, f"{product.get('product_key')} has no revision"

    body = admin_page.locator("body")
    for label in sorted(expected_products | expected_tables | expected_features):
        expect(body).to_contain_text(label, timeout=30_000)
    expect(body).to_contain_text("signalops:subscription_admin")
    expect(body).to_contain_text("Govern enrolled users")
    expect(body).to_contain_text("Feature policy is the server-side entitlement contract")
    expect(body).to_contain_text("Admin-managed Stripe billing")
    expect(body).to_contain_text("Map Stripe IDs created in Stripe Dashboard")
    expect(body).to_contain_text("Product mapping")
    expect(body).to_contain_text("Subject Stripe mapping")
    expect(body).to_contain_text("Institutional Stripe mapping")


def test_subscription_administration_is_platform_only(admin_page: Page, admin_config: tuple[str, str, str]) -> None:
    base_url, _, _ = admin_config
    login(admin_page, admin_config)
    expect(admin_page.get_by_role("heading", name="Subscription Administration")).to_be_visible(timeout=30_000)
    expect(admin_page.get_by_role("heading", name="Explorer or Professional subject plan")).to_be_visible()
    expect(admin_page.get_by_role("heading", name="Institutional tenant contract")).to_be_visible()
    expect(admin_page.get_by_role("heading", name="Institutional seat", exact=True)).to_be_visible()
    expect(admin_page.locator("body")).to_contain_text("signalops:subscription_admin")
    assert "/marketops/" not in admin_page.url

    admin_page.goto(f"{base_url}/marketops/settings", wait_until="domcontentloaded")
    expect(admin_page.get_by_role("heading", name="MarketOps Settings")).to_be_visible(timeout=30_000)
    expect(admin_page.locator("body")).not_to_contain_text("Subscription and analytical depth")
    expect(admin_page.locator("body")).not_to_contain_text("Subscription required")
    expect(admin_page.locator("body")).not_to_contain_text("Provision subject plan")

    admin_page.goto(f"{base_url}/marketops/valuation", wait_until="domcontentloaded")
    expect(admin_page.get_by_role("heading", name="Value Intelligence & Distressed Opportunity Intelligence")).to_be_visible(timeout=30_000)
    expect(admin_page.locator("body")).not_to_contain_text("requires additional analytical depth")
