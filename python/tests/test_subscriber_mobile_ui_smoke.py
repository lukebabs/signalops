"""Mobile-first Subscriber UX smoke checks.

This is intentionally read-only. It validates that the core subscriber surfaces
load at phone width without a 404/auth dead-end or page-level horizontal
overflow. Admin Workbench is excluded from this mobile gate.
"""

from __future__ import annotations

import os
import re
from pathlib import Path

import pytest
from playwright.sync_api import Browser, Page, expect

from test_subscriber_ui_smoke import SubscriberUIConfig, login, subscriber_ui_config


@pytest.fixture(scope="session")
def subscriber_config() -> SubscriberUIConfig:
    return subscriber_ui_config()


@pytest.fixture(params=[{"width": 375, "height": 812}, {"width": 390, "height": 844}, {"width": 430, "height": 932}], ids=["iphone-small", "iphone-standard", "iphone-large"])
def mobile_viewport(request: pytest.FixtureRequest) -> dict[str, int]:
    return request.param


@pytest.fixture
def mobile_subscriber_page(subscriber_config: SubscriberUIConfig, browser: Browser, request: pytest.FixtureRequest, mobile_viewport: dict[str, int]) -> Page:
    artifact_dir = Path(os.getenv("SIGNALOPS_E2E_ARTIFACT_DIR", "/tmp/signalops-e2e-artifacts"))
    artifact_dir.mkdir(mode=0o700, parents=True, exist_ok=True)
    artifact_dir.chmod(0o700)
    stem = request.node.name
    har_path = artifact_dir / f"{stem}.har"
    trace_path = artifact_dir / f"{stem}.zip"
    screenshot_path = artifact_dir / f"{stem}.png"
    context = browser.new_context(
        viewport=mobile_viewport,
        is_mobile=True,
        has_touch=True,
        record_har_path=str(har_path),
        record_har_mode="minimal",
    )
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
            har_path.unlink(missing_ok=True)
        context.close()


def assert_mobile_route(page: Page, url: str, heading: str) -> None:
    page.goto(url, wait_until="domcontentloaded", timeout=30_000)
    expect(page.get_by_role("heading", name=heading)).to_be_visible(timeout=30_000)
    body = page.locator("body")
    expect(body).not_to_contain_text("404", timeout=5_000)
    expect(body).not_to_contain_text("page not found", timeout=5_000)
    expect(body).not_to_contain_text("subscriber_watchlist_context_not_found", timeout=5_000)
    overflow = page.evaluate(
        "() => Math.max(document.documentElement.scrollWidth, document.body.scrollWidth) - window.innerWidth"
    )
    assert overflow <= 8, f"{url} has {overflow}px horizontal overflow at mobile viewport"


def test_subscriber_mobile_core_journey(mobile_subscriber_page: Page, subscriber_config: SubscriberUIConfig) -> None:
    login(mobile_subscriber_page, subscriber_config)
    routes = [
        ("/marketops/dashboard", "MarketOps Dashboard"),
        ("/marketops/watchlists", "Watchlists"),
        ("/marketops/assets", "Assets"),
        ("/marketops/sectors", "Sector Rotation Intelligence"),
        ("/marketops/assurance", "Signal Assurance"),
        ("/marketops/syncratic", "Syncratic Intelligence"),
    ]
    for path, heading in routes:
        assert_mobile_route(mobile_subscriber_page, f"{subscriber_config.base_url}{path}", heading)




def test_subscriber_mobile_assets_card_drilldown(mobile_subscriber_page: Page, subscriber_config: SubscriberUIConfig) -> None:
    login(mobile_subscriber_page, subscriber_config)
    assert_mobile_route(mobile_subscriber_page, f"{subscriber_config.base_url}/marketops/assets", "Assets")
    first_symbol = subscriber_config.shared_tickers[0]
    card = mobile_subscriber_page.get_by_test_id(f"marketops-asset-mobile-card-{first_symbol}")
    expect(card).to_be_visible(timeout=30_000)
    expect(card).to_contain_text("Inspect", timeout=30_000)
    expect(card).to_contain_text("Current Market Data", timeout=30_000)
    expect(card).to_contain_text("Intraday", timeout=30_000)
    expect(card).to_contain_text("Risk/Reward", timeout=30_000)
    card.get_by_role("button").first.click()
    expect(card).to_contain_text("Close", timeout=30_000)
    overflow = mobile_subscriber_page.evaluate(
        "() => Math.max(document.documentElement.scrollWidth, document.body.scrollWidth) - window.innerWidth"
    )
    assert overflow <= 8, f"Assets mobile drilldown has {overflow}px horizontal overflow"



def test_subscriber_mobile_signal_assurance_drilldown(mobile_subscriber_page: Page, subscriber_config: SubscriberUIConfig) -> None:
    login(mobile_subscriber_page, subscriber_config)
    assert_mobile_route(mobile_subscriber_page, f"{subscriber_config.base_url}/marketops/assurance", "Signal Assurance")
    expect(mobile_subscriber_page.get_by_role("heading", name="Analyst drill-down")).to_be_visible(timeout=30_000)
    expect(mobile_subscriber_page.get_by_test_id("saf-daily-progression")).to_be_visible(timeout=30_000)
    cohort = mobile_subscriber_page.locator("[data-testid^='saf-mobile-cohort-']").first
    expect(cohort).to_be_visible(timeout=30_000)
    expect(cohort).to_contain_text("Inspect", timeout=30_000)
    cohort.get_by_role("button").first.click()
    expect(cohort).to_contain_text("Close", timeout=30_000)
    expect(cohort).to_contain_text("Included observations", timeout=30_000)
    observation = cohort.locator("[data-testid^='saf-mobile-observation-']").first
    expect(observation).to_be_visible(timeout=30_000)
    observation.get_by_role("button").first.click()
    expect(observation).to_contain_text("Observation audit", timeout=30_000)
    close_audit = observation.get_by_role("button", name="Close audit")
    expect(close_audit).to_be_visible(timeout=30_000)
    close_audit.click()
    expect(observation).not_to_contain_text("Observation audit", timeout=30_000)
    overflow = mobile_subscriber_page.evaluate(
        "() => Math.max(document.documentElement.scrollWidth, document.body.scrollWidth) - window.innerWidth"
    )
    assert overflow <= 8, f"SAF mobile drilldown has {overflow}px horizontal overflow"

def test_subscriber_mobile_dashboard_syncratic_handoff(mobile_subscriber_page: Page, subscriber_config: SubscriberUIConfig) -> None:
    login(mobile_subscriber_page, subscriber_config)
    assert_mobile_route(mobile_subscriber_page, f"{subscriber_config.base_url}/marketops/dashboard", "MarketOps Dashboard")
    handoff = mobile_subscriber_page.get_by_role("link", name="Open Syncratic Intelligence").or_(
        mobile_subscriber_page.get_by_role("button", name="Open Syncratic Intelligence")
    ).first
    expect(handoff).to_be_visible(timeout=30_000)
    handoff.click()
    expect(mobile_subscriber_page).to_have_url(re.compile(r"/marketops/syncratic"), timeout=30_000)
    expect(mobile_subscriber_page.get_by_role("heading", name="Syncratic Intelligence")).to_be_visible(timeout=30_000)
    overflow = mobile_subscriber_page.evaluate(
        "() => Math.max(document.documentElement.scrollWidth, document.body.scrollWidth) - window.innerWidth"
    )
    assert overflow <= 8, f"Syncratic handoff destination has {overflow}px horizontal overflow at mobile viewport"
