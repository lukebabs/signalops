"""Read-only browser smoke for the B2C enrollment entry point.

This intentionally stops at the Keycloak registration screen. It must never
submit registration data or create a user.
"""

from __future__ import annotations

import os
import re
from dataclasses import dataclass
from pathlib import Path
from urllib.parse import parse_qs, urlparse

import pytest
from playwright.sync_api import Browser, Page, TimeoutError as PlaywrightTimeoutError, expect


@dataclass(frozen=True)
class EnrollmentUIConfig:
    base_url: str
    auth_host: str
    client_id: str


def enrollment_ui_config() -> EnrollmentUIConfig:
    auth_host = os.getenv("SIGNALOPS_E2E_AUTH_HOST", "auth.syncratic.co").strip().lower()
    if not auth_host:
        pytest.skip("Keycloak enrollment smoke is missing SIGNALOPS_E2E_AUTH_HOST")
    return EnrollmentUIConfig(
        base_url=os.getenv("SIGNALOPS_E2E_BASE_URL", "https://signalops.syncratic.io").rstrip("/"),
        auth_host=auth_host,
        client_id=os.getenv("SIGNALOPS_E2E_CLIENT_ID", "signalops-web").strip(),
    )


@pytest.fixture
def enrollment_page(browser: Browser, request: pytest.FixtureRequest) -> Page:
    artifact_dir = Path(os.getenv("SIGNALOPS_E2E_ARTIFACT_DIR", "/tmp/signalops-enrollment-e2e-artifacts"))
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


def click_create_account(page: Page, config: EnrollmentUIConfig) -> None:
    page.goto(config.base_url, wait_until="domcontentloaded", timeout=30_000)
    if urlparse(page.url).hostname == config.auth_host:
        page.goto(config.base_url, wait_until="domcontentloaded", timeout=30_000)
    create_account = page.get_by_role("button", name=re.compile(r"create account", re.I))
    expect(create_account).to_be_visible(timeout=30_000)
    create_account.click()


def test_create_account_reaches_keycloak_registration_without_submission(enrollment_page: Page) -> None:
    config = enrollment_ui_config()
    click_create_account(enrollment_page, config)

    try:
        enrollment_page.wait_for_url(
            lambda url: (urlparse(url).hostname or "").lower() == config.auth_host,
            timeout=30_000,
        )
    except PlaywrightTimeoutError:
        raise AssertionError(f"Create account did not redirect to {config.auth_host}; current URL: {enrollment_page.url}") from None

    parsed = urlparse(enrollment_page.url)
    query = parse_qs(parsed.query)
    assert (parsed.hostname or "").lower() == config.auth_host
    if query.get("client_id"):
        assert query["client_id"] == [config.client_id]
    assert "error" not in query, f"Keycloak returned an enrollment error: {query.get('error')}"

    body = enrollment_page.locator("body")
    expect(body).not_to_contain_text(re.compile(r"page not found|invalid parameter|invalid request", re.I))

    registration_markers = [
        enrollment_page.get_by_role("heading", name=re.compile(r"register|create account|sign up", re.I)),
        enrollment_page.locator("#kc-register-form"),
        enrollment_page.locator("input[name='email']"),
        enrollment_page.locator("input[name='username']"),
    ]
    assert any(marker.is_visible(timeout=5_000) for marker in registration_markers), (
        "Create account reached Keycloak, but no registration form marker was visible. "
        "Confirm realm registration is enabled for the syncratic realm/client."
    )
