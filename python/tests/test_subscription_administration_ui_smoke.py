"""Read-only browser evidence for the platform Subscription Administration workspace."""

from __future__ import annotations

import os
import re
from pathlib import Path

import pytest
from playwright.sync_api import Browser, Page, expect

from test_subscription_enforcement_canary_ui import bearer


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
        "Syncratic Ask",
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
    by_label = {str(row.get("label", "")): row for row in rows}
    for label in ["Dashboard", "Risk/Reward", "Intraday conditions", "FMP annual financials"]:
        row = by_label[label]
        assert row.get("expected_freshness"), f"{label} has no expected freshness contract"
        assert row.get("dependency_job_id"), f"{label} has no dependency job contract"
        assert row.get("staleness_explanation"), f"{label} has no freshness explanation"
        assert row.get("next_step"), f"{label} has no next-step guidance"
    assert by_label["Risk/Reward"].get("run_now_job_id") == "marketops-risk-reward"
    assert by_label["Signal Assurance"].get("run_now_enabled") is False
    scheduled_jobs = payload.get("marketops_operations_health", {}).get("scheduled_jobs")
    assert isinstance(scheduled_jobs, list), "operations-health response did not include scheduled_jobs rows"
    scheduled_by_label = {str(row.get("label", "")): row for row in scheduled_jobs}
    operations_monitor = scheduled_by_label.get("MarketOps operations monitor")
    assert operations_monitor, "MarketOps operations monitor scheduled-job row is missing"
    assert operations_monitor.get("run_now_enabled") is True, operations_monitor
    warm_eod = scheduled_by_label.get("MarketOps warm EOD baseline")
    assert warm_eod, "MarketOps warm EOD scheduled-job row is missing"
    if warm_eod.get("status") == "degraded":
        for key in ["normalized", "expected", "missing", "max_missing", "session_date", "missing_symbols"]:
            assert key in warm_eod, f"warm-EOD degraded detail missing {key}: {warm_eod}"

    expect(admin_page.get_by_role("heading", name="MarketOps Operations Health")).to_be_visible(timeout=30_000)
    body = admin_page.locator("body")
    for label in sorted(expected_labels):
        expect(body).to_contain_text(label, timeout=30_000)
    for label in ["Expected freshness", "Dependency", "Latest evidence", "Why / next step", "Run recovery", "Status only"]:
        expect(body).to_contain_text(label, timeout=30_000)
    warm_eod_row = admin_page.get_by_role("row", name=re.compile(r"^MarketOps warm EOD baseline\b"))
    expect(warm_eod_row).to_contain_text("bounded_provider_gap", timeout=30_000)
    expect(warm_eod_row).to_contain_text("normalized", timeout=30_000)
    expect(warm_eod_row).to_contain_text("missing", timeout=30_000)
    operations_monitor_row = admin_page.get_by_role("row", name=re.compile(r"^MarketOps operations monitor\b"))
    expect(operations_monitor_row).to_contain_text("Run now", timeout=30_000)


def test_marketops_admin_can_run_operations_monitor_from_ui(admin_page: Page, admin_config: tuple[str, str, str]) -> None:
    if os.getenv("SIGNALOPS_ADMIN_RUN_NOW_SMOKE_ACK") != "approved":
        pytest.skip("set SIGNALOPS_ADMIN_RUN_NOW_SMOKE_ACK=approved to run the live Admin run-now smoke")
    login_marketops_admin(admin_page, admin_config)
    operations_monitor_row = admin_page.get_by_role("row", name=re.compile(r"^MarketOps operations monitor\b"))
    expect(operations_monitor_row).to_contain_text("Run now", timeout=30_000)
    with admin_page.expect_response(
        lambda response: response.request.method == "POST"
        and "/v1/administration/scheduled-jobs/marketops-operations-monitor/run-now" in response.url,
        timeout=30_000,
    ) as response_info:
        operations_monitor_row.get_by_role("button", name="Run now").click()
    response = response_info.value
    assert response.status == 202, f"{response.url} returned HTTP {response.status}: {response.text()}"
    payload = response.json()
    run = payload.get("run")
    assert isinstance(run, dict), payload
    assert run.get("job_id") == "marketops-operations-monitor", payload
    assert run.get("action") == "scheduler-run-now:marketops-operations-monitor", payload
    assert str(run.get("runner", "")).startswith("unix:") or "signalops-deploy-agent" in str(run.get("runner", "")), payload
    expect(admin_page.locator("body")).not_to_contain_text("scheduled_job_start_failed")



def test_subscription_administration_governance_surface(admin_page: Page, admin_config: tuple[str, str, str]) -> None:
    expected_products = {"Explorer", "Professional", "Institutional"}
    expected_features = {"Market Dashboards", "Value Intelligence", "Distressed Opportunity Intelligence", "Earnings Opportunity Intelligence", "Signal Assurance Analytics", "APIs", "White-Label Deployment"}

    with admin_page.expect_response(
        lambda response: response.request.method == "GET" and response.url.split("?", 1)[0].endswith("/v1/administration/subscriptions"),
        timeout=30_000,
    ) as response_info:
        login(admin_page, admin_config)
    response = response_info.value
    assert response.status == 200, f"{response.url} returned HTTP {response.status}"
    payload = response.json()
    products = payload.get("products")
    subject_email_pairs = []
    for collection_name in ["subject_subscriptions", "seats", "audit_events", "upgrade_interactions", "refund_requests"]:
        for item in payload.get(collection_name, []) or []:
            subject = str(item.get("subject", ""))
            email = str(item.get("subject_email", ""))
            if subject and email and subject != email:
                subject_email_pairs.append((subject, email))

    assert isinstance(products, list) and len(products) >= 3, "subscription administration did not return product policy rows"
    product_names = {str(product.get("display_name", "")) for product in products}
    assert expected_products.issubset(product_names), f"missing product tiers: {sorted(expected_products - product_names)}"
    for product in products:
        assert isinstance(product.get("feature_policy"), dict), f"{product.get('product_key')} has no feature policy"
        assert isinstance(product.get("limit_policy"), dict), f"{product.get('product_key')} has no limit policy"
        assert "revision" in product, f"{product.get('product_key')} has no revision"

    body = admin_page.locator("body")
    for label in sorted(expected_products):
        expect(body).to_contain_text(label, timeout=30_000)
    expect(body).to_contain_text("signalops:subscription_admin")
    expect(body).to_contain_text("Govern enrolled users")
    expect(body).to_contain_text("Stripe product map")
    expect(body).to_contain_text("Subject subscriptions")

    admin_page.get_by_role("button", name="Tier settings").click()
    expect(body).to_contain_text("Feature policy is the server-side entitlement contract")
    for label in sorted(expected_features):
        expect(body).to_contain_text(label, timeout=30_000)

    admin_page.get_by_role("button", name="Stripe billing").click()
    expect(body).to_contain_text("Admin-managed Stripe billing")
    expect(body).to_contain_text("Map Stripe IDs created in Stripe Dashboard")
    expect(body).to_contain_text("Product mapping")
    expect(body).to_contain_text("Choose a different tier from Configured Stripe products above")
    admin_page.get_by_role("button", name=re.compile("Professional")).click()
    expect(body).to_contain_text("Selected product: Professional")
    admin_page.get_by_role("button", name=re.compile("Explorer")).click()
    expect(body).to_contain_text("Selected product: Explorer")
    expect(body).to_contain_text("Subject Stripe mapping")
    expect(body).to_contain_text("Institutional Stripe mapping")

    admin_page.get_by_role("button", name="Refund requests").click()
    expect(admin_page.get_by_label("Search refunds")).to_be_visible()
    expect(body).to_contain_text("Refund request queue")
    expect(body).to_contain_text("execute any approved refund in Stripe Dashboard")
    expect(body).to_contain_text("Manual Stripe action")

    admin_page.get_by_role("button", name="Users & seats").click()
    for label in ["Explorer or Professional subject plan", "Institutional tenant contract", "Institutional seat", "Subject subscriptions", "Institutional contracts", "Institutional seats", "Selected user activity"]:
        expect(body).to_contain_text(label, timeout=30_000)
    if subject_email_pairs:
        subject, email = subject_email_pairs[0]
        expect(body).to_contain_text(email, timeout=30_000)
        expect(body).not_to_contain_text(subject)

    admin_page.get_by_role("button", name="User activity").click()
    expect(admin_page.get_by_label("Search user activity")).to_be_visible()
    expect(body).to_contain_text("Operational visibility into login, logout")
    expect(body).to_contain_text("Activity events")
    activity_response = admin_page.evaluate(
        """async (token) => {
          const response = await fetch('/v1/administration/subscriptions/activity?tenant_id=tenant-local&limit=100', {headers: {Authorization: 'Bearer ' + token}});
          return {status: response.status, body: await response.json().catch(() => ({}))};
        }""",
        bearer(admin_page),
    )
    assert int(activity_response["status"]) == 200, activity_response["body"]
    labeled = [
        item for item in activity_response["body"].get("summaries", [])
        if item.get("subject_email") or item.get("subject_display_name")
    ]
    uuid_pattern = re.compile(r"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$", re.I)
    for item in labeled:
        label = item.get("subject_email") or item.get("subject_display_name")
        expect(body).to_contain_text(label, timeout=30_000)
        if uuid_pattern.match(str(item.get("subject", ""))):
            expect(body).not_to_contain_text(item["subject"])

    admin_page.get_by_role("button", name="Audit log").click()
    expect(admin_page.get_by_label("Search audit log")).to_be_visible()
    expect(body).to_contain_text("Audit trail")

    admin_page.get_by_role("button", name="Webhook ledger").click()
    expect(admin_page.get_by_label("Search webhooks")).to_be_visible()
    expect(body).to_contain_text("Stripe webhook ledger")


def test_subscription_administration_is_platform_only(admin_page: Page, admin_config: tuple[str, str, str]) -> None:
    base_url, _, _ = admin_config
    login(admin_page, admin_config)
    expect(admin_page.get_by_role("heading", name="Subscription Administration")).to_be_visible(timeout=30_000)
    admin_page.get_by_role("button", name="Users & seats").click()
    expect(admin_page.get_by_role("heading", name="Explorer or Professional subject plan")).to_be_visible()
    expect(admin_page.get_by_role("heading", name="Institutional tenant contract")).to_be_visible()
    expect(admin_page.get_by_role("heading", name="Institutional seat", exact=True)).to_be_visible()
    expect(admin_page.locator("body")).to_contain_text("signalops:subscription_admin")
    assert "/marketops/" not in admin_page.url

    admin_page.goto(f"{base_url}/marketops/settings", wait_until="domcontentloaded")
    expect(admin_page.get_by_role("heading", name="Manage your MarketOps experience")).to_be_visible(timeout=30_000)
    expect(admin_page.get_by_role("tab", name="Subscription")).to_be_visible(timeout=30_000)
    expect(admin_page.locator("body")).not_to_contain_text("Provision subject plan")

    admin_page.goto(f"{base_url}/marketops/tools", wait_until="domcontentloaded")
    expect(admin_page.get_by_role("heading", name="MarketOps Tools")).to_be_visible(timeout=30_000)
    expect(admin_page.get_by_role("heading", name="Analyst assets")).to_be_visible(timeout=30_000)

    admin_page.goto(f"{base_url}/marketops/valuation", wait_until="domcontentloaded")
    expect(admin_page.get_by_role("heading", name="Value Intelligence & Distressed Opportunity Intelligence")).to_be_visible(timeout=30_000)
    expect(admin_page.locator("body")).not_to_contain_text("requires additional analytical depth")
