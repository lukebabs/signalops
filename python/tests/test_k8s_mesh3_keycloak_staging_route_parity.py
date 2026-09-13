"""Authenticated Keycloak parity smoke through the Istio staging route.

This validates whether the SignalOps SPA can start an OIDC flow from the
Istio-routed staging hostname, authenticate an existing QA user, and receive the
callback on the same staging route. It performs no registration, no Stripe
operation, no provider polling, and no production DNS movement.
"""

from __future__ import annotations

import os
import re
from dataclasses import dataclass, field
from pathlib import Path
from urllib.parse import parse_qs, urlparse

import pytest
from playwright.sync_api import TimeoutError as PlaywrightTimeoutError, expect, sync_playwright


HOSTNAME = "signalops-staging.syncratic.co"


@dataclass(frozen=True)
class Mesh3AuthConfig:
    port: str
    username: str
    password: str = field(repr=False)
    expected_tenant_id: str
    expected_state: str
    artifact_dir: Path

    @property
    def base_url(self) -> str:
        return f"http://{HOSTNAME}:{self.port}"


def config() -> Mesh3AuthConfig:
    port = os.environ.get("SIGNALOPS_K8S_MESH3_LOCAL_PORT", "").strip()
    if not port:
        pytest.skip("SIGNALOPS_K8S_MESH3_LOCAL_PORT is set by the Mesh-3 smoke runner")
    assert port.isdigit(), "SIGNALOPS_K8S_MESH3_LOCAL_PORT must be numeric"
    username = os.environ.get("SIGNALOPS_B2C_WEB", "").strip() or os.environ.get("SYNCRATIC_QA_CLIENT", "").strip()
    password = os.environ.get("SIGNALOPS_B2C_WEB_PASS", "").strip() or os.environ.get("SYNCRATIC_QA_PASS", "").strip()
    assert username, "SIGNALOPS_B2C_WEB or SYNCRATIC_QA_CLIENT is required"
    assert password, "SIGNALOPS_B2C_WEB_PASS or SYNCRATIC_QA_PASS is required"
    artifact_dir = Path(os.getenv("SIGNALOPS_E2E_ARTIFACT_DIR", "/tmp/signalops-mesh3-auth-e2e-artifacts"))
    artifact_dir.mkdir(mode=0o700, parents=True, exist_ok=True)
    artifact_dir.chmod(0o700)
    return Mesh3AuthConfig(
        port=port,
        username=username,
        password=password,
        expected_tenant_id=os.getenv("SIGNALOPS_E2E_B2C_TENANT_ID", "tenant-local").strip(),
        expected_state=os.getenv("SIGNALOPS_E2E_ENROLLMENT_EXPECTED_STATE", "subscription_missing").strip(),
        artifact_dir=artifact_dir,
    )


def fill_keycloak_login(page, cfg: Mesh3AuthConfig) -> None:
    username = page.locator("#username, input[name='username']").or_(page.get_by_role("textbox", name="Email or username")).first
    username.wait_for(state="visible", timeout=30_000)
    username.fill(cfg.username)
    password = page.locator("#password, input[name='password']").or_(page.get_by_role("textbox", name="Password")).first
    password.fill(cfg.password)
    submit = page.locator("#kc-login, input[type='submit']").or_(page.get_by_role("button", name="Continue")).first
    submit.click()


def assert_not_keycloak_redirect_block(page) -> None:
    visible_text = page.locator("body").inner_text(timeout=5_000) if page.locator("body").is_visible(timeout=5_000) else ""
    parsed = urlparse(page.url)
    query = parse_qs(parsed.query)
    if parsed.netloc == "auth.syncratic.co" and re.search(r"invalid|redirect|client|complete request|registration not allowed", visible_text, re.I):
        raise AssertionError(
            "Keycloak rejected the SignalOps Istio staging auth request. "
            f"current_url={page.url!r} visible_text={visible_text[:500]!r} query={query!r}. "
            "Add a staging redirect/web-origin allowance or create a dedicated staging Keycloak client before this gate can pass."
        )


def test_keycloak_login_through_istio_staging_route() -> None:
    cfg = config()
    with sync_playwright() as playwright:
        browser = playwright.chromium.launch(
            headless=True,
            args=[f"--host-resolver-rules=MAP {HOSTNAME} 127.0.0.1"],
        )
        context = browser.new_context(
            record_har_path=str(cfg.artifact_dir / "mesh3_keycloak_staging_route.har"),
            record_har_mode="minimal",
        )
        context.tracing.start(screenshots=True, snapshots=True, sources=True)
        page = context.new_page()
        failed = False
        try:
            page.goto(f"{cfg.base_url}/marketops/dashboard", wait_until="domcontentloaded", timeout=30_000)
            expect(page).to_have_title(re.compile("SignalOps"), timeout=10_000)
            sign_in = page.get_by_role("button", name="Sign in")
            if sign_in.is_visible(timeout=10_000):
                sign_in.click()

            assert_not_keycloak_redirect_block(page)

            try:
                with page.expect_response(
                    lambda response: response.request.method == "GET" and "/v1/session/enrollment" in response.url,
                    timeout=60_000,
                ) as enrollment_info:
                    fill_keycloak_login(page, cfg)
            except PlaywrightTimeoutError:
                assert_not_keycloak_redirect_block(page)
                visible_text = page.locator("body").inner_text(timeout=5_000) if page.locator("body").is_visible(timeout=5_000) else ""
                raise AssertionError(f"Authenticated staging route did not reach enrollment resolver. url={page.url!r} text={visible_text[:500]!r}") from None

            response = enrollment_info.value
            assert response.status == 200, f"{response.url} returned {response.status}: {response.text()[:300]}"
            payload = response.json()
            assert payload.get("tenant_id") == cfg.expected_tenant_id, payload
            assert payload.get("state") == cfg.expected_state, payload
            assert payload.get("email_verified") is True, payload

            if cfg.expected_state == "marketops_ready":
                expect(page.get_by_role("heading", name="MarketOps Dashboard")).to_be_visible(timeout=30_000)
            elif cfg.expected_state == "subscription_missing":
                page.wait_for_url(re.compile(r"/marketops/pricing.*source_feature=enrollment"), timeout=30_000)
                expect(page.get_by_role("heading", name="Increase analytical depth when the research question requires it.")).to_be_visible(timeout=30_000)
            else:
                expect(page.locator("body")).to_contain_text(re.compile(cfg.expected_state.replace("_", " "), re.I))
        except Exception:
            failed = True
            page.screenshot(path=str(cfg.artifact_dir / "mesh3_keycloak_staging_route.png"), full_page=True)
            context.tracing.stop(path=str(cfg.artifact_dir / "mesh3_keycloak_staging_route.zip"))
            raise
        finally:
            if not failed:
                context.tracing.stop()
                (cfg.artifact_dir / "mesh3_keycloak_staging_route.har").unlink(missing_ok=True)
            context.close()
            browser.close()
