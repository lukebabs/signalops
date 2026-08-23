"""Real-browser smoke for Syncratic Ask on MarketOps narratives.

The test uses the protected tenant-local QA identity. It exercises the live UI
path, clicks the normal Ask action only (`force=false`), and verifies the gateway
returns a governed Ask response without exposing upstream bodies.
"""

from __future__ import annotations

import os
import re
from pathlib import Path

import pytest
from playwright.sync_api import Browser, Page, TimeoutError as PlaywrightTimeoutError, expect


@pytest.fixture(scope="session")
def syncratic_config() -> tuple[str, str, str]:
    username = os.getenv("SIGNALOPS_E2E_ADMIN_USERNAME", "").strip()
    password = os.getenv("SIGNALOPS_E2E_ADMIN_PASSWORD", "").strip()
    base_url = os.getenv("SIGNALOPS_E2E_BASE_URL", "https://signalops.syncratic.io").rstrip("/")
    if not username or not password:
        pytest.skip("Syncratic Ask UI smoke is not configured")
    return base_url, username, password


@pytest.fixture
def syncratic_page(syncratic_config: tuple[str, str, str], browser: Browser, request: pytest.FixtureRequest) -> Page:
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
    heading = page.get_by_role("heading", name="Syncratic Insights")
    last_error: Exception | None = None
    for attempt in range(3):
        try:
            page.goto(f"{base_url}/marketops/syncratic", wait_until="domcontentloaded", timeout=30_000)
            if heading.is_visible(timeout=5_000):
                return
            sign_in = page.get_by_role("button", name="Sign in")
            if sign_in.is_visible(timeout=10_000):
                sign_in.click()
            if heading.is_visible(timeout=3_000):
                return
            username_input = page.locator("#username, input[name='username']").or_(
                page.get_by_role("textbox", name="Email or username")
            ).first
            username_input.wait_for(state="visible", timeout=30_000)
            username_input.fill(username)
            password_input = page.locator("#password, input[name='password']").or_(
                page.get_by_role("textbox", name="Password")
            ).first
            password_input.fill(password)
            page.locator("#kc-login, input[type='submit']").or_(page.get_by_role("button", name="Continue")).first.click()
            expect(heading).to_be_visible(timeout=30_000)
            return
        except PlaywrightTimeoutError as exc:
            last_error = exc
            page.wait_for_timeout(1_000 * (attempt + 1))
    if last_error is not None:
        raise last_error
    raise AssertionError("Syncratic Ask UI smoke could not reach the login page after retries")


def test_syncratic_daily_narrative_ask_smoke(syncratic_page: Page, syncratic_config: tuple[str, str, str]) -> None:
    login(syncratic_page, syncratic_config)
    expect(syncratic_page.get_by_role("heading", name="Daily Overview narratives")).to_be_visible(timeout=30_000)

    card = syncratic_page.locator("button").filter(has_text="Inspect artifacts and context").first
    expect(card).to_be_visible(timeout=30_000)
    card.click()

    expect(syncratic_page.get_by_text("Ask Syncratic AI").first).to_be_visible(timeout=30_000)
    ask_button = syncratic_page.get_by_role("button", name=re.compile(r"^Ask Syncratic AI$")).first
    expect(ask_button).to_be_visible(timeout=30_000)

    with syncratic_page.expect_response(
        lambda response: response.request.method == "POST"
        and re.search(r"/v1/syncratic/context-windows/[^/]+/ask$", response.url),
        timeout=60_000,
    ) as response_info:
        ask_button.click()
    response = response_info.value
    assert response.status == 200, f"{response.url} returned HTTP {response.status}: {response.text()[:500]}"
    payload = response.json()
    ask_result = payload.get("ask_result")
    assert isinstance(ask_result, dict), "Ask response did not include ask_result"
    assert ask_result.get("ask_status") in {"completed", "skipped"}, f"unexpected Ask status: {ask_result}"
    assert ask_result.get("prompt_digest"), "Ask response did not include prompt_digest"
    assert "raw_prompt" not in payload and "upstream_body" not in payload

    body = syncratic_page.locator("body")
    expect(body).to_contain_text(re.compile("Ask completed|Skipped"), timeout=60_000)
