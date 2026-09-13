"""Authenticated regression for Market Intelligence session-boundary labels."""

from __future__ import annotations

import os
import re
from datetime import datetime
from typing import Any

import pytest
from playwright.sync_api import Browser, Page, Response, expect


@pytest.fixture(scope="session")
def config() -> tuple[str, str, str]:
    username = os.getenv("SIGNALOPS_E2E_ADMIN_USERNAME", "").strip()
    password = os.getenv("SIGNALOPS_E2E_ADMIN_PASSWORD", "").strip()
    base_url = os.getenv("SIGNALOPS_E2E_BASE_URL", "https://signalops.syncratic.io").rstrip("/")
    if not username or not password:
        pytest.skip("Market Intelligence UI smoke is not configured")
    return base_url, username, password


def login(page: Page, config: tuple[str, str, str]) -> None:
    base_url, username, password = config
    page.goto(f"{base_url}/marketops/indicator-reel", wait_until="domcontentloaded")
    if page.get_by_role("heading", name="Market Intelligence").is_visible():
        return
    page.get_by_role("button", name="Sign in").click()
    account = page.locator("#username, input[name='username']").or_(page.get_by_role("textbox", name="Email or username")).first
    account.wait_for(state="visible", timeout=30_000)
    account.fill(username)
    page.locator("#password, input[name='password']").or_(page.get_by_role("textbox", name="Password")).first.fill(password)
    page.locator("#kc-login, input[type='submit']").or_(page.get_by_role("button", name="Continue")).first.click()
    page.wait_for_url(re.compile(re.escape(base_url) + r"/marketops/indicator-reel"), timeout=30_000)


def intraday_response(response: Response) -> bool:
    return response.request.method == "GET" and "/marketops/intraday-conditions?" in response.url


def session_label(as_of_time: str) -> str:
    session = datetime.strptime(as_of_time[:10], "%Y-%m-%d")
    return "Completed session · " + session.strftime("%b %-d")


def test_postclose_monitor_conditions_are_not_rendered_as_midnight_live_events(browser: Browser, config: tuple[str, str, str]) -> None:
    base_url, _, _ = config
    page = browser.new_page()
    try:
        login(page, config)
        with page.expect_response(intraday_response, timeout=30_000) as response_info:
            page.reload(wait_until="domcontentloaded")
        response = response_info.value
        assert response.status == 200, f"{response.url} returned HTTP {response.status}"
        payload: dict[str, Any] = response.json()
        snapshots = payload.get("snapshots")
        assert isinstance(snapshots, list), "intraday condition response did not contain snapshots"
        completed = next((snapshot for snapshot in snapshots if snapshot.get("market_status") == "end_of_day" and snapshot.get("conditions")), None)
        assert completed is not None, "expected at least one post-close condition for the tenant-local watchlist"
        expected = session_label(str(completed["as_of_time"]))
        expect(page.locator("body")).to_contain_text(expected, timeout=30_000)
        expect(page.locator("body")).not_to_contain_text(f"{expected.removeprefix('Completed session · ')}, 12:00 AM EDT")
    finally:
        page.context.close()
