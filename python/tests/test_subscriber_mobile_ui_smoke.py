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
from playwright.sync_api import Browser, Page, TimeoutError as PlaywrightTimeoutError, expect

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
        ("/marketops/profile", "MarketOps profile"),
        ("/marketops/settings", "Manage your MarketOps experience"),
        ("/marketops/sectors", "Sector Rotation Intelligence"),
        ("/marketops/opportunities", "Opportunities"),
        ("/marketops/earnings", "Earnings Opportunity Intelligence"),
        ("/marketops/assurance", "Signal Assurance"),
        ("/marketops/syncratic", "Syncratic Intelligence"),
        ("/marketops/pricing", "Increase analytical depth"),
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





def test_subscriber_mobile_sri_progression_and_makeup(mobile_subscriber_page: Page, subscriber_config: SubscriberUIConfig) -> None:
    login(mobile_subscriber_page, subscriber_config)
    assert_mobile_route(mobile_subscriber_page, f"{subscriber_config.base_url}/marketops/sectors", "Sector Rotation Intelligence")
    mobile_subscriber_page.get_by_role("tab", name="ETF progression").click()
    cards = mobile_subscriber_page.locator("[data-testid^='sri-mobile-etf-card-']")
    expect(cards.first).to_be_visible(timeout=30_000)

    saw_available_holdings = False
    saw_makeup_state = False
    for index in range(min(cards.count(), 8)):
        etf_card = cards.nth(index)
        expect(etf_card).to_contain_text("Open 60-session progression", timeout=30_000)
        etf_card.get_by_role("button").first.click()
        expect(etf_card).to_contain_text("Detail open", timeout=30_000)
        expect(etf_card.get_by_role("tab", name="Progression")).to_be_visible(timeout=30_000)
        etf_card.get_by_role("tab", name="ETF makeup").click()

        holding = etf_card.locator("[data-testid^='sri-mobile-holding-']").first
        unavailable = etf_card.get_by_text("Current ETF makeup is not available")
        try:
            holding.wait_for(state="visible", timeout=10_000)
            expect(holding).to_contain_text("Weight", timeout=30_000)
            saw_available_holdings = True
            saw_makeup_state = True
        except PlaywrightTimeoutError:
            try:
                unavailable.wait_for(state="visible", timeout=10_000)
                saw_makeup_state = True
            except PlaywrightTimeoutError:
                pass

        close_detail = etf_card.get_by_role("button", name="Close ETF detail")
        expect(close_detail).to_be_visible(timeout=30_000)
        close_detail.click()
        expect(etf_card).not_to_contain_text("ETF makeup", timeout=30_000)

        if saw_available_holdings:
            break

    assert saw_makeup_state, "SRI mobile ETF detail did not expose holdings or an explicit unavailable makeup state"
    overflow = mobile_subscriber_page.evaluate(
        "() => Math.max(document.documentElement.scrollWidth, document.body.scrollWidth) - window.innerWidth"
    )
    assert overflow <= 8, f"SRI mobile progression/makeup has {overflow}px horizontal overflow"


def test_subscriber_mobile_opportunities_drilldown(mobile_subscriber_page: Page, subscriber_config: SubscriberUIConfig) -> None:
    login(mobile_subscriber_page, subscriber_config)
    assert_mobile_route(mobile_subscriber_page, f"{subscriber_config.base_url}/marketops/opportunities", "Opportunities")
    queue_button = mobile_subscriber_page.locator("button").filter(has_text=re.compile(r"Score\s+")).first
    if queue_button.is_visible(timeout=10_000):
        queue_button.click()
        expect(mobile_subscriber_page.get_by_text("Opportunity Detail")).to_be_visible(timeout=30_000)
        expect(mobile_subscriber_page.get_by_text("Why now")).to_be_visible(timeout=30_000)
        expect(mobile_subscriber_page.get_by_text("Contributions")).to_be_visible(timeout=30_000)
        back = mobile_subscriber_page.get_by_role("button", name="Back to queue")
        expect(back).to_be_visible(timeout=30_000)
        back.click()
        expect(mobile_subscriber_page.get_by_text("Opportunity Detail")).not_to_be_visible(timeout=30_000)
    else:
        expect(mobile_subscriber_page.get_by_text("No eligible opportunities in this scope")).to_be_visible(timeout=30_000)
    overflow = mobile_subscriber_page.evaluate(
        "() => Math.max(document.documentElement.scrollWidth, document.body.scrollWidth) - window.innerWidth"
    )
    assert overflow <= 8, f"Opportunities mobile drilldown has {overflow}px horizontal overflow"


def test_subscriber_mobile_eeom_history_drilldown(mobile_subscriber_page: Page, subscriber_config: SubscriberUIConfig) -> None:
    login(mobile_subscriber_page, subscriber_config)
    assert_mobile_route(mobile_subscriber_page, f"{subscriber_config.base_url}/marketops/earnings", "Earnings Opportunity Intelligence")
    row = mobile_subscriber_page.locator("[data-testid^='marketops-eeom-row-']").first
    if row.is_visible(timeout=10_000):
        expect(mobile_subscriber_page.get_by_test_id("marketops-eeom-current-table")).to_be_visible(timeout=30_000)
        row.click()
        history = mobile_subscriber_page.get_by_test_id("marketops-eeom-evolution-history")
        expect(history).to_be_visible(timeout=30_000)
        expect(history).to_contain_text("Earnings setup evolution", timeout=30_000)
    else:
        expect(mobile_subscriber_page.get_by_text("No point-in-time-known earnings events")).to_be_visible(timeout=30_000)
    overflow = mobile_subscriber_page.evaluate(
        "() => Math.max(document.documentElement.scrollWidth, document.body.scrollWidth) - window.innerWidth"
    )
    assert overflow <= 8, f"EEOM mobile drilldown has {overflow}px horizontal overflow"

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


def test_subscriber_mobile_pricing_cards(mobile_subscriber_page: Page, subscriber_config: SubscriberUIConfig) -> None:
    login(mobile_subscriber_page, subscriber_config)
    assert_mobile_route(mobile_subscriber_page, f"{subscriber_config.base_url}/marketops/pricing", "Increase analytical depth")
    expect(mobile_subscriber_page.get_by_role("heading", name="Explorer")).to_be_visible(timeout=30_000)
    expect(mobile_subscriber_page.get_by_role("heading", name="Professional")).to_be_visible(timeout=30_000)
    expect(mobile_subscriber_page.get_by_role("heading", name="Institutional")).to_be_visible(timeout=30_000)
    expect(mobile_subscriber_page.get_by_text("Checkout status")).to_be_visible(timeout=30_000)
    monthly_buttons = mobile_subscriber_page.get_by_role("button", name=re.compile(r"Monthly Checkout"))
    expect(monthly_buttons.first).to_be_visible(timeout=30_000)
    overflow = mobile_subscriber_page.evaluate(
        "() => Math.max(document.documentElement.scrollWidth, document.body.scrollWidth) - window.innerWidth"
    )
    assert overflow <= 8, f"Pricing mobile cards have {overflow}px horizontal overflow"

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
