"""Authenticated UX regression for governed SAF benchmark coverage."""

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
        pytest.skip("Signal Assurance UI smoke is not configured")
    return base_url, username, password


def login(page: Page, config: tuple[str, str, str]) -> None:
    base_url, username, password = config
    page.goto(f"{base_url}/marketops/tools", wait_until="domcontentloaded")
    if page.get_by_role("heading", name="MarketOps Tools").is_visible():
        return
    sign_in = page.get_by_role("button", name="Sign in")
    if sign_in.is_visible():
        sign_in.click()
    account = page.locator("#username, input[name='username']").or_(page.get_by_role("textbox", name="Email or username")).first
    account.wait_for(state="visible", timeout=30_000)
    account.fill(username)
    secret = page.locator("#password, input[name='password']").or_(page.get_by_role("textbox", name="Password")).first
    secret.fill(password)
    page.locator("#kc-login, input[type='submit']").or_(page.get_by_role("button", name="Continue")).first.click()
    page.wait_for_url(re.compile(re.escape(base_url) + r"/marketops/tools"), timeout=30_000)
    expect(page.get_by_role("heading", name="MarketOps Tools")).to_be_visible(timeout=30_000)


def benchmark_coverage_response(response: Response) -> bool:
    return (
        response.request.method == "GET"
        and "/v1/marketops/signal-assurance/effectiveness?" in response.url
        and "source=LEGACY" in response.url
        and "dimension=benchmark_coverage" in response.url
    )


def test_signal_assurance_tools_view_honors_operational_cutoff(browser: Browser, config: tuple[str, str, str]) -> None:
    base_url, _, _ = config
    page = browser.new_page()
    try:
        login(page, config)
        expect(page.get_by_label("MarketOps tools sections")).to_be_visible(timeout=30_000)
        with page.expect_response(benchmark_coverage_response, timeout=30_000) as response_info:
            page.get_by_role("button", name=re.compile("Signal Assurance")).click()
        response = response_info.value
        assert response.status == 200, f"{response.url} returned HTTP {response.status}"
        payload: dict[str, Any] = response.json()
        rows = payload.get("effectiveness")
        assert payload.get("operational_cutoff_date") == "2026-08-20", payload
        assert isinstance(rows, list) and rows, "SAF did not return post-cutoff benchmark-coverage rows"
        dimensions = {str(row.get("dimension_value", "")) for row in rows}
        assert all(dimension for dimension in dimensions), dimensions
        expect(page.get_by_role("heading", name="Signal Assurance")).to_be_visible(timeout=30_000)
        expect(page.locator("body")).to_contain_text("last 10 trading days", timeout=30_000)
        expect(page.locator("body")).to_contain_text("August 20, 2026", timeout=30_000)
        expect(page.get_by_label("Window")).to_contain_text("Last 10 trading days", timeout=30_000)
        expect(page.get_by_label("Window")).to_contain_text("Last 20 trading days", timeout=30_000)
        expect(page.get_by_label("Window")).not_to_contain_text("All observations")
        expect(page.locator("body")).to_contain_text("Usefulness", timeout=30_000)
        expect(page.locator("body")).to_contain_text("saf_usefulness.v1", timeout=30_000)
        expect(page.locator("body")).not_to_contain_text("Aug 19")
    finally:
        page.context.close()
