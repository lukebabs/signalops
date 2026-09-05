"""Authenticated tenant-local Dashboard freshness regression."""

from __future__ import annotations

import os
import re
from typing import Any

import pytest
from playwright.sync_api import Browser, Page, Response, expect


@pytest.fixture(scope="session")
def config() -> tuple[str, str, str]:
    username = os.getenv("SIGNALOPS_E2E_ADMIN_USERNAME", "").strip()
    password = os.getenv("SIGNALOPS_E2E_ADMIN_PASSWORD", "").strip()
    base_url = os.getenv("SIGNALOPS_E2E_BASE_URL", "https://signalops.syncratic.io").rstrip("/")
    if not username or not password:
        pytest.skip("MarketOps Dashboard UI smoke is not configured")
    return base_url, username, password


def login(page: Page, config: tuple[str, str, str]) -> None:
    base_url, username, password = config
    page.goto(f"{base_url}/marketops/dashboard", wait_until="domcontentloaded")
    if page.get_by_role("heading", name="MarketOps Dashboard").is_visible():
        return
    page.get_by_role("button", name="Sign in").click()
    account = page.locator("#username, input[name='username']").or_(page.get_by_role("textbox", name="Email or username")).first
    account.wait_for(state="visible", timeout=30_000)
    account.fill(username)
    page.locator("#password, input[name='password']").or_(page.get_by_role("textbox", name="Password")).first.fill(password)
    page.locator("#kc-login, input[type='submit']").or_(page.get_by_role("button", name="Continue")).first.click()
    page.wait_for_url(re.compile(re.escape(base_url) + r"/marketops/dashboard"), timeout=30_000)


def dashboard_response(response: Response) -> bool:
    return response.request.method == "GET" and "/marketops/assets/signal-overview?" in response.url


def states_response(response: Response) -> bool:
    return response.request.method == "GET" and "/v1/marketops/states?" in response.url


def select_legacy_default(page: Page) -> None:
    selector = page.get_by_test_id("marketops-watchlist-selector").locator("select")
    expect(selector).to_be_visible(timeout=30_000)
    legacy_option = selector.locator("option").filter(has_text="MarketOps Legacy Default")
    legacy_id = legacy_option.get_attribute("value")
    assert legacy_id, "tenant-local legacy default is not available to the authenticated user"
    with page.expect_response(
        lambda response: response.request.method == "PUT" and "/subscriber/watchlist-context" in response.url,
        timeout=30_000,
    ) as response_info:
        selector.select_option(legacy_id)
    assert response_info.value.status == 200, "legacy default watchlist selection was not persisted"


def test_dashboard_and_market_state_share_the_latest_completed_session(browser: Browser, config: tuple[str, str, str]) -> None:
    base_url, _, _ = config
    page = browser.new_page()
    try:
        login(page, config)
        select_legacy_default(page)
        with page.expect_response(dashboard_response, timeout=30_000) as response_info:
            page.reload(wait_until="domcontentloaded")
        dashboard = response_info.value
        assert dashboard.status == 200, f"{dashboard.url} returned HTTP {dashboard.status}"
        payload: dict[str, Any] = dashboard.json()
        context = payload.get("watchlist_context")
        assert isinstance(context, dict) and context.get("selection_mode") == "list", "Dashboard did not use a selected watchlist"
        assert context.get("list_name") == "MarketOps Legacy Default", context
        assert payload.get("asset_count") == 132, f"Dashboard did not retain the tenant-local legacy cohort: {payload.get('asset_count')}"
        points = payload.get("risk_reward", {}).get("points", [])
        latest_risk_session = max((str(point.get("trade_date", "")) for point in points), default="")
        assert latest_risk_session, "Dashboard returned no Risk/Reward session"
        expect(page.get_by_role("heading", name="MarketOps Dashboard")).to_be_visible(timeout=30_000)

        with page.expect_response(states_response, timeout=30_000) as response_info:
            page.goto(f"{base_url}/marketops/state?symbol=AAPL", wait_until="domcontentloaded")
        states = response_info.value
        assert states.status == 200, f"{states.url} returned HTTP {states.status}"
        state_rows = states.json().get("market_states", [])
        latest_state_session = max((str(row.get("session_date", ""))[:10] for row in state_rows), default="")
        assert latest_state_session == latest_risk_session, (
            f"Dashboard Risk/Reward is {latest_risk_session}, but Market State is {latest_state_session}"
        )
        expect(page.get_by_role("heading", name="Market State")).to_be_visible(timeout=30_000)
    finally:
        page.context.close()
