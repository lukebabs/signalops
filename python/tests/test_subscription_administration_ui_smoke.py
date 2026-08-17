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


def test_subscription_administration_is_platform_only(admin_page: Page, admin_config: tuple[str, str, str]) -> None:
    base_url, _, _ = admin_config
    login(admin_page, admin_config)
    expect(admin_page.get_by_role("heading", name="Subscription Administration")).to_be_visible(timeout=30_000)
    expect(admin_page.get_by_role("heading", name="Explorer or Professional subject plan")).to_be_visible()
    expect(admin_page.get_by_role("heading", name="Institutional tenant contract")).to_be_visible()
    expect(admin_page.get_by_role("heading", name="Institutional seat")).to_be_visible()
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
