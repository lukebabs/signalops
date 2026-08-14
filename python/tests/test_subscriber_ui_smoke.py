"""Real-browser, read-only Subscriber UX smoke checks.

The test is skipped unless its dedicated QA identity is supplied through the
protected environment. It records a HAR, trace, and screenshot only on failure.
"""

from __future__ import annotations

import os
import re
from dataclasses import dataclass
from pathlib import Path

import pytest
from playwright.sync_api import Browser, Page, expect


@dataclass(frozen=True)
class SubscriberUIConfig:
    base_url: str
    username: str
    password: str
    watchlist_name: str


def subscriber_ui_config() -> SubscriberUIConfig:
    required = {
        "SIGNALOPS_E2E_USERNAME": os.getenv("SIGNALOPS_E2E_USERNAME", "").strip(),
        "SIGNALOPS_E2E_PASSWORD": os.getenv("SIGNALOPS_E2E_PASSWORD", "").strip(),
        "SIGNALOPS_E2E_WATCHLIST_NAME": os.getenv("SIGNALOPS_E2E_WATCHLIST_NAME", "").strip(),
    }
    missing = [name for name, value in required.items() if not value]
    if missing:
        pytest.skip("Subscriber UI smoke is not configured: " + ", ".join(missing))
    return SubscriberUIConfig(
        base_url=os.getenv("SIGNALOPS_E2E_BASE_URL", "https://signalops.syncratic.io").rstrip("/"),
        username=required["SIGNALOPS_E2E_USERNAME"],
        password=required["SIGNALOPS_E2E_PASSWORD"],
        watchlist_name=required["SIGNALOPS_E2E_WATCHLIST_NAME"],
    )


@pytest.fixture
def subscriber_page(browser: Browser, request: pytest.FixtureRequest) -> Page:
    artifact_dir = Path(os.getenv("SIGNALOPS_E2E_ARTIFACT_DIR", "/tmp/signalops-e2e-artifacts"))
    artifact_dir.mkdir(mode=0o700, parents=True, exist_ok=True)
    artifact_dir.chmod(0o700)
    stem = request.node.name
    har_path, trace_path, screenshot_path = (artifact_dir / f"{stem}.har", artifact_dir / f"{stem}.zip", artifact_dir / f"{stem}.png")
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


def login(page: Page, config: SubscriberUIConfig) -> None:
    page.goto(f"{config.base_url}/marketops/watchlists", wait_until="domcontentloaded")
    if "auth." in page.url or "login-actions" in page.url:
        page.locator("#username, input[name='username']").first.fill(config.username)
        page.locator("#password, input[name='password']").first.fill(config.password)
        page.locator("#kc-login, input[type='submit']").first.click()
    page.wait_for_url(re.compile(re.escape(config.base_url) + r"/marketops/.*"), timeout=30_000)
    expect(page.get_by_role("heading", name="Watchlists")).to_be_visible(timeout=30_000)


def selected_watchlist_name(page: Page) -> str:
    selector = page.locator("label:has-text('Watchlist') select")
    expect(selector).to_be_visible(timeout=30_000)
    return selector.evaluate("element => element.options[element.selectedIndex].textContent || ''")


def test_subscriber_watchlist_context_propagates_across_marketops(subscriber_page: Page) -> None:
    config = subscriber_ui_config()
    login(subscriber_page, config)
    subscriber_page.get_by_role("button", name=re.compile(re.escape(config.watchlist_name))).click()
    use_across = subscriber_page.get_by_role("button", name="Use across MarketOps")
    if use_across.is_visible():
        use_across.click()
    expect(subscriber_page.get_by_role("button", name="Used across MarketOps")).to_be_visible(timeout=30_000)

    for route in ("assets", "dashboard", "eroc", "valuation", "earnings-opportunities", "state"):
        subscriber_page.goto(f"{config.base_url}/marketops/{route}", wait_until="domcontentloaded")
        assert config.watchlist_name in selected_watchlist_name(subscriber_page)
