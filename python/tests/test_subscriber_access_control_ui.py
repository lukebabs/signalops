"""Read-only browser/API evidence for subscriber tenant isolation.

This smoke uses real OIDC browser sessions but performs only GET requests. It
proves tenant-bearing MarketOps routes are bound to the token tenant while the
platform Subscription Administration route remains available only to the
subscription-admin identity.
"""

from __future__ import annotations

import json
import os
import re
from pathlib import Path
from typing import Any

import pytest
from playwright.sync_api import Browser, BrowserContext, Page, expect

PILOT_TENANT = "tenant-pilot-b"
LOCAL_TENANT = "tenant-local"
OIDC_STORAGE_KEY = "oidc.user:https://auth.syncratic.co/realms/syncratic:signalops-web"


@pytest.fixture(scope="session")
def config() -> dict[str, str]:
    values = {
        "base_url": os.getenv("SIGNALOPS_E2E_BASE_URL", "https://signalops.syncratic.io").rstrip("/"),
        "pilot_username": os.getenv("SIGNALOPS_E2E_PILOT_USERNAME", "").strip(),
        "pilot_password": os.getenv("SIGNALOPS_E2E_PILOT_PASSWORD", "").strip(),
        "admin_username": os.getenv("SIGNALOPS_E2E_ADMIN_USERNAME", "").strip(),
        "admin_password": os.getenv("SIGNALOPS_E2E_ADMIN_PASSWORD", "").strip(),
    }
    if not all(values.values()):
        pytest.skip("Subscriber access-control smoke credentials are not configured")
    return values


@pytest.fixture
def browser_context(browser: Browser, request: pytest.FixtureRequest) -> BrowserContext:
    artifact_dir = Path(os.getenv("SIGNALOPS_E2E_ARTIFACT_DIR", "/tmp/signalops-e2e-artifacts"))
    artifact_dir.mkdir(mode=0o700, parents=True, exist_ok=True)
    artifact_dir.chmod(0o700)
    har_path = artifact_dir / f"{request.node.name}.har"
    context = browser.new_context(record_har_path=str(har_path), record_har_mode="minimal")
    context.tracing.start(screenshots=True, snapshots=True, sources=True)
    try:
        yield context
    except Exception:
        for index, page in enumerate(context.pages):
            page.screenshot(path=str(artifact_dir / f"{request.node.name}-{index}.png"), full_page=True)
        context.tracing.stop(path=str(artifact_dir / f"{request.node.name}.zip"))
        raise
    else:
        context.tracing.stop()
        har_path.unlink(missing_ok=True)
    finally:
        context.close()


def sign_in(page: Page, *, base_url: str, username: str, password: str, path: str, heading: str | re.Pattern[str]) -> None:
    page.goto(f"{base_url}{path}", wait_until="domcontentloaded")
    heading_locator = page.get_by_role("heading", name=heading)
    if heading_locator.is_visible(timeout=5_000):
        return
    sign_in_button = page.get_by_role("button", name="Sign in")
    if sign_in_button.is_visible(timeout=10_000):
        sign_in_button.click()
    username_input = page.locator("#username, input[name='username']").or_(page.get_by_role("textbox", name="Email or username")).first
    username_input.wait_for(state="visible", timeout=30_000)
    username_input.fill(username)
    password_input = page.locator("#password, input[name='password']").or_(page.get_by_role("textbox", name="Password")).first
    password_input.fill(password)
    page.locator("#kc-login, input[type='submit']").or_(page.get_by_role("button", name="Continue")).first.click()
    expect(heading_locator).to_be_visible(timeout=30_000)


def bearer(page: Page) -> str:
    raw = page.evaluate("(key) => sessionStorage.getItem(key)", OIDC_STORAGE_KEY)
    assert raw, "OIDC session was not persisted in browser storage"
    token = json.loads(raw).get("access_token", "")
    assert token, "OIDC access token is missing"
    return token


def api(page: Page, path: str) -> tuple[int, dict[str, Any]]:
    response = page.evaluate(
        """async ({path, token}) => {
          const response = await fetch(path, {headers: {Authorization: "Bearer " + token}});
          return {status: response.status, body: await response.json().catch(() => ({}))};
        }""",
        {"path": path, "token": bearer(page)},
    )
    return int(response["status"]), response["body"]


def assert_error(page: Page, path: str, status_code: int, error_code: str) -> None:
    status, body = api(page, path)
    assert status == status_code, f"{path} returned HTTP {status}: {body}"
    assert body.get("error") == error_code, f"{path} returned unexpected error body: {body}"


def assert_ok(page: Page, path: str) -> dict[str, Any]:
    status, body = api(page, path)
    assert status == 200, f"{path} returned HTTP {status}: {body}"
    return body


def test_pilot_and_local_tokens_are_tenant_bound(browser_context: BrowserContext, browser: Browser, config: dict[str, str]) -> None:
    pilot = browser_context.new_page()
    sign_in(
        pilot,
        base_url=config["base_url"],
        username=config["pilot_username"],
        password=config["pilot_password"],
        path="/marketops/watchlists",
        heading="Watchlists",
    )
    assert_ok(pilot, f"/v1/tenants/{PILOT_TENANT}/marketops/subscriber/lists")
    assert_ok(pilot, f"/v1/tenants/{PILOT_TENANT}/marketops/assets/signal-overview?universe_group=all_active&window=10_trade_days")
    assert_error(pilot, f"/v1/tenants/{LOCAL_TENANT}/marketops/subscriber/lists", 403, "tenant_mismatch")
    assert_error(pilot, f"/v1/tenants/{LOCAL_TENANT}/marketops/assets/signal-overview?universe_group=all_active&window=10_trade_days", 403, "tenant_mismatch")
    assert_error(pilot, f"/v1/administration/subscriptions?tenant_id={PILOT_TENANT}", 403, "insufficient_role")

    local_context = browser.new_context(record_har_mode="minimal")
    local = local_context.new_page()
    try:
        sign_in(
            local,
            base_url=config["base_url"],
            username=config["admin_username"],
            password=config["admin_password"],
            path="/admin/system",
            heading="MarketOps Operations Health",
        )
        assert_ok(local, f"/v1/tenants/{LOCAL_TENANT}/marketops/subscriber/lists")
        assert_error(local, f"/v1/tenants/{PILOT_TENANT}/marketops/subscriber/lists", 403, "tenant_mismatch")
        assert_error(local, f"/v1/tenants/{PILOT_TENANT}/marketops/assets/signal-overview?universe_group=all_active&window=10_trade_days", 403, "tenant_mismatch")
        assert_ok(local, f"/v1/administration/subscriptions?tenant_id={LOCAL_TENANT}")
        assert_ok(local, f"/v1/administration/subscriptions?tenant_id={PILOT_TENANT}")
    finally:
        local_context.close()
