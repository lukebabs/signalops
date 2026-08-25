"""Authenticated browser smoke for the B2C enrollment resolver.

This test uses an existing B2C QA account. It does not create a Keycloak user,
does not submit registration, and does not touch Stripe. If the B2C account is
new, the SignalOps enrollment resolver may perform its normal idempotent
self-enrollment mutations for the configured B2C tenant.
"""

from __future__ import annotations

import os
import re
from dataclasses import dataclass, field
from pathlib import Path

import pytest
from playwright.sync_api import Browser, Page, Response, TimeoutError as PlaywrightTimeoutError, expect


@dataclass(frozen=True)
class B2CEnrollmentConfig:
    base_url: str
    username: str
    password: str = field(repr=False)
    tenant_id: str
    expected_state: str
    expected_can_self_enroll: bool


def b2c_enrollment_config() -> B2CEnrollmentConfig:
    username = os.getenv("SIGNALOPS_B2C_WEB", "").strip()
    password = os.getenv("SIGNALOPS_B2C_WEB_PASS", "").strip()
    if not username or not password:
        pytest.skip("B2C enrollment authenticated smoke requires SIGNALOPS_B2C_WEB and SIGNALOPS_B2C_WEB_PASS")
    return B2CEnrollmentConfig(
        base_url=os.getenv("SIGNALOPS_E2E_BASE_URL", "https://signalops.syncratic.io").rstrip("/"),
        username=username,
        password=password,
        tenant_id=os.getenv("SIGNALOPS_E2E_B2C_TENANT_ID", "tenant-b2c").strip(),
        expected_state=os.getenv("SIGNALOPS_E2E_ENROLLMENT_EXPECTED_STATE", "marketops_ready").strip(),
        expected_can_self_enroll=os.getenv("SIGNALOPS_E2E_EXPECT_CAN_SELF_ENROLL", "true").strip().lower() == "true",
    )


@pytest.fixture
def b2c_page(browser: Browser, request: pytest.FixtureRequest) -> Page:
    artifact_dir = Path(os.getenv("SIGNALOPS_E2E_ARTIFACT_DIR", "/tmp/signalops-b2c-enrollment-e2e-artifacts"))
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


def fill_keycloak_login(page: Page, config: B2CEnrollmentConfig) -> None:
    username = page.locator("#username, input[name='username']").or_(page.get_by_role("textbox", name="Email or username")).first
    username.wait_for(state="visible", timeout=30_000)
    username.fill(config.username)
    password = page.locator("#password, input[name='password']").or_(page.get_by_role("textbox", name="Password")).first
    password.fill(config.password)
    submit = page.locator("#kc-login, input[type='submit']").or_(page.get_by_role("button", name="Continue")).first
    submit.click()


def test_b2c_user_resolves_enrollment_state(b2c_page: Page) -> None:
    config = b2c_enrollment_config()
    b2c_page.goto(f"{config.base_url}/marketops/dashboard", wait_until="domcontentloaded", timeout=30_000)
    sign_in = b2c_page.get_by_role("button", name="Sign in")
    if sign_in.is_visible(timeout=10_000):
        sign_in.click()

    with b2c_page.expect_response(
        lambda response: response.request.method == "GET" and "/v1/session/enrollment" in response.url,
        timeout=45_000,
    ) as enrollment_info:
        fill_keycloak_login(b2c_page, config)

    enrollment_response: Response = enrollment_info.value
    assert enrollment_response.status == 200, f"{enrollment_response.url} returned HTTP {enrollment_response.status}"
    payload = enrollment_response.json()
    assert payload.get("tenant_id") == config.tenant_id
    assert payload.get("state") == config.expected_state, payload
    assert payload.get("email_verified") is True
    assert bool(payload.get("can_self_enroll", False)) is config.expected_can_self_enroll

    if config.expected_state == "marketops_ready":
        expect(b2c_page.get_by_role("heading", name="MarketOps Dashboard")).to_be_visible(timeout=30_000)
    else:
        expect(b2c_page.locator("body")).to_contain_text(re.compile(config.expected_state.replace("_", " "), re.I))
