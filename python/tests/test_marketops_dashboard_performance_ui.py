"""Authenticated Dashboard performance guardrails."""

from __future__ import annotations

import os
import re
import time

import pytest
from playwright.sync_api import Browser, Page, expect


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
    heading = page.get_by_role("heading", name="MarketOps Dashboard")
    if heading.is_visible(timeout=5_000):
        return
    page.get_by_role("button", name="Sign in").click()
    page.locator("#username, input[name='username']").or_(page.get_by_role("textbox", name="Email or username")).first.fill(username)
    page.locator("#password, input[name='password']").or_(page.get_by_role("textbox", name="Password")).first.fill(password)
    page.locator("#kc-login, input[type='submit']").or_(page.get_by_role("button", name="Continue")).first.click()
    page.wait_for_url(re.compile(re.escape(base_url) + r"/marketops/dashboard"), timeout=30_000)


def test_dashboard_reel_is_non_blocking(browser: Browser, config: tuple[str, str, str]) -> None:
    page = browser.new_page(viewport={"width": 1440, "height": 1200})
    try:
        started = time.perf_counter()
        signal_overview_done: list[float] = []

        def observe(response) -> None:
            if "/marketops/assets/signal-overview?" in response.url:
                signal_overview_done.append((time.perf_counter() - started) * 1000)

        page.on("response", observe)
        login(page, config)
        page.get_by_test_id("dashboard-market-intelligence-reel").wait_for(timeout=30_000)
        reel_ms = (time.perf_counter() - started) * 1000
        expect(page.get_by_role("heading", name="MarketOps Dashboard")).to_be_visible()
        assert reel_ms < 3_000, f"Dashboard Market Intelligence reel blocked for {reel_ms:.0f}ms"
        page.wait_for_timeout(5_000)
        assert signal_overview_done, "Dashboard did not request signal overview"
    finally:
        page.context.close()
