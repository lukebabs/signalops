"""Bounded production proof for the temporary Subscription enforcement canary.

The controlled tenant-pilot-b Explorer user is temporarily set to Professional
through the normal Subscription Administration UI, then restored to Explorer.
The deployment-agent action owns the temporary gateway flag and restores it.
"""

from __future__ import annotations

import json
import os
import re
from pathlib import Path

import pytest
from playwright.sync_api import Browser, BrowserContext, Page, expect

PILOT_TENANT = "tenant-pilot-b"
PILOT_SUBJECT = "2f437ac3-2cfc-4fe9-b943-198185b4797b"
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
        pytest.skip("Subscription canary browser credentials are not configured")
    return values


@pytest.fixture
def canary_context(browser: Browser, request: pytest.FixtureRequest) -> BrowserContext:
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
    if heading_locator.is_visible():
        return
    sign_in_button = page.get_by_role("button", name="Sign in")
    if sign_in_button.is_visible():
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


def api(page: Page, path: str) -> tuple[int, dict[str, object]]:
    response = page.evaluate(
        """async ({path, token}) => {
          const response = await fetch(path, {headers: {Authorization: "Bearer " + token}});
          return {status: response.status, body: await response.json().catch(() => ({}))};
        }""",
        {"path": path, "token": bearer(page)},
    )
    return int(response["status"]), response["body"]


def set_pilot_plan(page: Page, config: dict[str, str], plan: str) -> None:
    sign_in(page, base_url=config["base_url"], username=config["admin_username"], password=config["admin_password"], path="/admin/subscriptions", heading="Subscription Administration")
    form = page.get_by_role("heading", name="Explorer or Professional subject plan").locator("xpath=ancestor::form")
    form.get_by_role("textbox", name="Tenant ID").fill(PILOT_TENANT)
    form.get_by_role("textbox", name="OIDC subject").fill(PILOT_SUBJECT)
    form.get_by_role("combobox", name="Plan").select_option(plan)
    form.get_by_role("combobox", name="Status").select_option("active")
    form.get_by_role("button", name="Provision subject plan").click()
    expect(form.get_by_role("status")).to_have_text("Provisioning change recorded successfully.", timeout=30_000)


def test_subscription_enforcement_three_tier_canary(canary_context: BrowserContext, browser: Browser, config: dict[str, str]) -> None:
    pilot = canary_context.new_page()
    admin_context = browser.new_context()
    institutional_context = browser.new_context()
    admin = admin_context.new_page()
    try:
        sign_in(pilot, base_url=config["base_url"], username=config["pilot_username"], password=config["pilot_password"], path="/marketops/valuation", heading=re.compile("requires additional analytical depth"))
        status, body = api(pilot, f"/v1/tenants/{PILOT_TENANT}/marketops/valuation?symbol=AAPL")
        assert status == 402 and body.get("error") == "subscription_feature_required"
        pilot.goto(f"{config['base_url']}/marketops/sectors", wait_until="domcontentloaded")
        expect(pilot.get_by_role("heading", name=re.compile("Sector Rotation Intelligence"))).to_be_visible(timeout=30_000)

        set_pilot_plan(admin, config, "professional")
        pilot.goto(f"{config['base_url']}/marketops/valuation", wait_until="domcontentloaded")
        expect(pilot.get_by_role("heading", name="Value Intelligence & Distressed Opportunity Intelligence")).to_be_visible(timeout=30_000)
        status, _ = api(pilot, f"/v1/tenants/{PILOT_TENANT}/marketops/valuation?symbol=AAPL")
        assert status == 200
        pilot.goto(f"{config['base_url']}/marketops/assurance", wait_until="domcontentloaded")
        expect(pilot.get_by_role("heading", name=re.compile("requires additional analytical depth"))).to_be_visible(timeout=30_000)
        status, body = api(pilot, f"/v1/marketops/signal-assurance/effectiveness?tenant_id={PILOT_TENANT}")
        assert status == 402 and body.get("error") == "subscription_feature_required"

        institutional = institutional_context.new_page()
        sign_in(institutional, base_url=config["base_url"], username=config["admin_username"], password=config["admin_password"], path="/marketops/assurance", heading="Signal Assurance")
        status, _ = api(institutional, f"/v1/marketops/signal-assurance/effectiveness?tenant_id={LOCAL_TENANT}")
        assert status == 200
    finally:
        # Keep the controlled pilot's original Explorer state even if any proof fails.
        try:
            set_pilot_plan(admin, config, "explorer")
        finally:
            institutional_context.close()
            admin_context.close()
