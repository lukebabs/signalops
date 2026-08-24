"""Authenticated Dashboard regression for inline Syncratic explainability."""

from __future__ import annotations

import os
import re
from dataclasses import dataclass
from pathlib import Path

import pytest
from playwright.sync_api import Browser, Page, TimeoutError as PlaywrightTimeoutError, expect


@dataclass(frozen=True, repr=False)
class DashboardNarrativeConfig:
    base_url: str
    username: str
    password: str


@pytest.fixture(scope="session")
def config() -> DashboardNarrativeConfig:
    username = os.getenv("SIGNALOPS_E2E_ADMIN_USERNAME", "").strip()
    password = os.getenv("SIGNALOPS_E2E_ADMIN_PASSWORD", "").strip()
    base_url = os.getenv("SIGNALOPS_E2E_BASE_URL", "https://signalops.syncratic.io").rstrip("/")
    if not username or not password:
        pytest.skip("MarketOps Dashboard Syncratic narrative UI smoke is not configured")
    return DashboardNarrativeConfig(base_url=base_url, username=username, password=password)


@pytest.fixture
def dashboard_page(browser: Browser, request: pytest.FixtureRequest) -> Page:
    artifact_dir = Path(os.getenv("SIGNALOPS_E2E_ARTIFACT_DIR", "/tmp/signalops-e2e-artifacts"))
    artifact_dir.mkdir(mode=0o700, parents=True, exist_ok=True)
    artifact_dir.chmod(0o700)
    har_path = artifact_dir / f"{request.node.name}.har"
    screenshot_path = artifact_dir / f"{request.node.name}.png"
    context = browser.new_context(record_har_path=str(har_path), record_har_mode="minimal")
    page = context.new_page()
    try:
        yield page
    finally:
        failed = bool(getattr(request.node, "rep_call", None) and request.node.rep_call.failed)
        if failed:
            page.screenshot(path=str(screenshot_path), full_page=True)
        context.close()
        if not failed:
            har_path.unlink(missing_ok=True)


def login(page: Page, config: DashboardNarrativeConfig) -> None:
    base_url = config.base_url
    username = config.username
    password = config.password
    heading = page.get_by_role("heading", name="MarketOps Dashboard")
    last_error: Exception | None = None
    for attempt in range(3):
        try:
            page.goto(f"{base_url}/marketops/dashboard", wait_until="domcontentloaded", timeout=30_000)
            if heading.is_visible(timeout=5_000):
                return
            page.get_by_role("button", name="Sign in").click()
            account = page.locator("#username, input[name='username']").or_(
                page.get_by_role("textbox", name="Email or username")
            ).first
            account.wait_for(state="visible", timeout=30_000)
            account.fill(username)
            page.locator("#password, input[name='password']").or_(
                page.get_by_role("textbox", name="Password")
            ).first.fill(password)
            page.locator("#kc-login, input[type='submit']").or_(
                page.get_by_role("button", name="Continue")
            ).first.click()
            expect(heading).to_be_visible(timeout=30_000)
            return
        except PlaywrightTimeoutError as exc:
            last_error = exc
            page.wait_for_timeout(1_000 * (attempt + 1))
    if last_error is not None:
        raise last_error
    raise AssertionError("Dashboard Syncratic narrative smoke could not reach the Dashboard after retries")


def test_dashboard_risk_reward_explainability_renders_full_structured_narrative(
    dashboard_page: Page,
    config: DashboardNarrativeConfig,
) -> None:
    login(dashboard_page, config)
    digest = dashboard_page.locator("section").filter(has_text="Syncratic narrative digest").first
    expect(digest).to_be_visible(timeout=30_000)

    risk_reward_card = dashboard_page.get_by_test_id("dashboard-syncratic-card-marketops-risk-reward-daily-v1")
    expect(risk_reward_card).to_be_visible(timeout=30_000)
    risk_reward_card.click()

    panel = dashboard_page.get_by_test_id("dashboard-syncratic-explainability")
    expect(panel.get_by_text("Risk/Reward").first).to_be_visible(timeout=30_000)
    expect(panel.get_by_text("Syncratic Explainability").or_(panel.get_by_text("Deterministic Explainability")).first).to_be_visible(timeout=30_000)
    explanation = panel.locator("p.whitespace-pre-wrap").first
    expect(explanation).to_be_visible(timeout=30_000)

    text = explanation.text_content() or ""
    compact = " ".join(text.split())
    assert "Executive summary:" in text, compact
    assert "Contextual read:" in text, compact
    assert "Top drivers:" in text, compact
    assert re.search(r"Analyst follow-ups?:", text), compact
    assert "\n" in text, "Dashboard rendered the explainability as a compact single-line summary"
    assert len(compact) >= 500, f"Dashboard explainability is too thin: {compact}"

    lower = compact.lower()
    for forbidden in [
        "they want me",
        "the task is to",
        "the context includes",
        "the prompt",
        "the json provided",
        "with score",
        "confidence 0.",
    ]:
        assert forbidden not in lower, f"Dashboard explainability contains meta or raw metric output {forbidden!r}: {compact}"
