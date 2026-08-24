"""Real-browser, read-only Subscriber UX smoke checks.

The test is skipped unless its dedicated QA identity is supplied through the
protected environment. It records a HAR, trace, and screenshot only on failure.
"""

from __future__ import annotations

import os
import re
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any
from urllib.parse import parse_qs, urlparse

import pytest
from playwright.sync_api import Browser, Page, Response, TimeoutError as PlaywrightTimeoutError, expect


@dataclass(frozen=True)
class SubscriberUIConfig:
    base_url: str
    username: str
    password: str = field(repr=False)
    watchlist_name: str
    tenant_id: str
    shared_tickers: tuple[str, ...]
    pending_tickers: tuple[str, ...]


def symbols(name: str, value: str) -> tuple[str, ...]:
    parsed = tuple(symbol.strip().upper() for symbol in value.split(",") if symbol.strip())
    if not parsed or any(not re.fullmatch(r"[A-Z.]{1,15}", symbol) for symbol in parsed):
        pytest.skip(f"Subscriber UI smoke has no valid {name}")
    return parsed


def subscriber_ui_config() -> SubscriberUIConfig:
    required = {
        "SIGNALOPS_E2E_USERNAME": os.getenv("SIGNALOPS_E2E_USERNAME", "").strip(),
        "SIGNALOPS_E2E_PASSWORD": os.getenv("SIGNALOPS_E2E_PASSWORD", "").strip(),
        "SIGNALOPS_E2E_WATCHLIST_NAME": os.getenv("SIGNALOPS_E2E_WATCHLIST_NAME", "").strip(),
        "SIGNALOPS_E2E_TENANT_ID": os.getenv("SIGNALOPS_E2E_TENANT_ID", "").strip(),
        "SIGNALOPS_E2E_SHARED_TICKERS": os.getenv("SIGNALOPS_E2E_SHARED_TICKERS", "").strip(),
    }
    missing = [name for name, value in required.items() if not value]
    if missing:
        pytest.skip("Subscriber UI smoke is not configured: " + ", ".join(missing))
    return SubscriberUIConfig(
        base_url=os.getenv("SIGNALOPS_E2E_BASE_URL", "https://signalops.syncratic.io").rstrip("/"),
        username=required["SIGNALOPS_E2E_USERNAME"],
        password=required["SIGNALOPS_E2E_PASSWORD"],
        watchlist_name=required["SIGNALOPS_E2E_WATCHLIST_NAME"],
        tenant_id=required["SIGNALOPS_E2E_TENANT_ID"],
        shared_tickers=symbols("SIGNALOPS_E2E_SHARED_TICKERS", required["SIGNALOPS_E2E_SHARED_TICKERS"]),
        pending_tickers=tuple(
            symbol.strip().upper()
            for symbol in os.getenv("SIGNALOPS_E2E_PENDING_TICKERS", "").split(",")
            if symbol.strip()
        ),
    )


@pytest.fixture(scope="session")
def subscriber_config() -> SubscriberUIConfig:
    return subscriber_ui_config()

@pytest.fixture
def subscriber_page(subscriber_config: SubscriberUIConfig, browser: Browser, request: pytest.FixtureRequest) -> Page:
    artifact_dir = Path(os.getenv("SIGNALOPS_E2E_ARTIFACT_DIR", "/tmp/signalops-e2e-artifacts"))
    artifact_dir.mkdir(mode=0o700, parents=True, exist_ok=True)
    artifact_dir.chmod(0o700)
    stem = request.node.name
    har_path, trace_path, screenshot_path = (artifact_dir / f"{stem}.har", artifact_dir / f"{stem}.zip", artifact_dir / f"{stem}.png")
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


def login(page: Page, config: SubscriberUIConfig) -> None:
    heading = page.get_by_role("heading", name="Watchlists")
    last_error: Exception | None = None
    for attempt in range(3):
        try:
            page.goto(f"{config.base_url}/marketops/watchlists", wait_until="domcontentloaded", timeout=30_000)
            if page.url.startswith("chrome-error://"):
                page.wait_for_timeout(1_000 * (attempt + 1))
                continue
            if heading.is_visible(timeout=5_000):
                return
            sign_in = page.get_by_role("button", name="Sign in")
            if sign_in.is_visible(timeout=10_000):
                sign_in.click()
            if page.url.startswith("chrome-error://"):
                page.wait_for_timeout(1_000 * (attempt + 1))
                continue
            if heading.is_visible(timeout=3_000):
                return
            username = page.locator("#username, input[name='username']").or_(page.get_by_role("textbox", name="Email or username")).first
            username.wait_for(state="visible", timeout=30_000)
            username.fill(config.username)
            password = page.locator("#password, input[name='password']").or_(page.get_by_role("textbox", name="Password")).first
            password.fill(config.password)
            submit = page.locator("#kc-login, input[type='submit']").or_(page.get_by_role("button", name="Continue")).first
            submit.click()
            expect(heading).to_be_visible(timeout=30_000)
            return
        except PlaywrightTimeoutError as exc:
            last_error = exc
            if page.url.startswith("chrome-error://") or attempt < 2:
                page.wait_for_timeout(1_000 * (attempt + 1))
                continue
            raise
    if last_error is not None:
        raise last_error
    raise AssertionError("Subscriber UI smoke could not reach the login page after retries")


def selected_watchlist_name(page: Page) -> str:
    selector = page.get_by_test_id("marketops-watchlist-selector").locator("select")
    expect(selector).to_be_visible(timeout=30_000)
    return selector.evaluate("element => element.options[element.selectedIndex].textContent || ''")


def visit_for_response(page: Page, url: str, endpoint_fragment: str) -> Response:
    with page.expect_response(
        lambda response: response.request.method == "GET" and endpoint_fragment in response.url,
        timeout=30_000,
    ) as response_info:
        page.goto(url, wait_until="domcontentloaded")
    response = response_info.value
    assert response.status == 200, f"{response.url} returned HTTP {response.status}"
    return response


def assert_watchlist_context(response: Response, config: SubscriberUIConfig) -> dict[str, Any]:
    payload = response.json()
    context = payload.get("watchlist_context")
    assert isinstance(context, dict), f"{response.url} did not return watchlist_context"
    assert context.get("selection_mode") == "list", f"{response.url} did not use the selected list"
    assert context.get("list_name") == config.watchlist_name, f"{response.url} used the wrong list"
    actual_tickers = {str(item.get("ticker", "")).upper() for item in context.get("items", [])}
    assert set(config.shared_tickers + config.pending_tickers).issubset(actual_tickers), f"{response.url} omitted configured fixture symbols"
    return payload


def assert_selected_watchlist(page: Page, config: SubscriberUIConfig) -> None:
    assert config.watchlist_name in selected_watchlist_name(page)


def assert_state_request_scope(response: Response, config: SubscriberUIConfig) -> None:
    query = parse_qs(urlparse(response.url).query)
    assert query.get("tenant_id") == [config.tenant_id]
    assert query.get("symbol") == [config.shared_tickers[0]]


def test_subscriber_watchlist_context_and_global_coverage(subscriber_page: Page) -> None:
    config = subscriber_ui_config()
    login(subscriber_page, config)
    subscriber_page.get_by_role("button", name=re.compile("^" + re.escape(config.watchlist_name) + r"\b")).click()
    expect(subscriber_page.get_by_role("heading", name=config.watchlist_name, exact=True)).to_be_visible(timeout=30_000)
    use_across = subscriber_page.get_by_role("button", name="Use across MarketOps")
    if use_across.is_visible() and use_across.is_enabled():
        use_across.click()
    used_across = subscriber_page.get_by_role("button", name="Used across MarketOps")
    if used_across.is_visible():
        expect(used_across).to_be_visible(timeout=30_000)

    assets = visit_for_response(
        subscriber_page,
        f"{config.base_url}/marketops/assets",
        "/marketops/assets?",
    )
    expect(subscriber_page.get_by_role("heading", name="Assets")).to_be_visible(timeout=30_000)
    assert_selected_watchlist(subscriber_page, config)
    assert_watchlist_context(assets, config)
    subscriber_page.reload(wait_until="domcontentloaded")
    assert_selected_watchlist(subscriber_page, config)
    for ticker in config.shared_tickers:
        row = subscriber_page.get_by_test_id(f"marketops-asset-row-{ticker}")
        expect(row).to_contain_text("Shared", timeout=30_000)
        expect(row).to_contain_text("Global EOD only")
        expect(row).not_to_contain_text("Open Market State")
        expect(row).not_to_contain_text("1-12-31")
        expect(row).not_to_contain_text("Awaiting monitor")
        expect(row).not_to_contain_text("Awaiting EOD analysis")
    pending_coverage = subscriber_page.get_by_test_id("marketops-pending-coverage")
    if config.pending_tickers:
        expect(pending_coverage).to_contain_text("Coverage in progress", timeout=30_000)
        for ticker in config.pending_tickers:
            expect(pending_coverage).to_contain_text(ticker)
            pending_row = subscriber_page.get_by_test_id(f"marketops-asset-row-{ticker}")
            expect(pending_row).to_contain_text("Pending")
            expect(pending_row).to_contain_text("Coverage pending")
            expect(pending_row).not_to_contain_text("Open Market State")
            expect(pending_row).not_to_contain_text("1-12-31")
    else:
        expect(pending_coverage).not_to_be_attached()
    expect(subscriber_page.locator("body")).not_to_contain_text("1-12-31")

    dashboard = visit_for_response(
        subscriber_page,
        f"{config.base_url}/marketops/dashboard",
        "/marketops/assets/signal-overview",
    )
    expect(subscriber_page.get_by_role("heading", name="MarketOps Dashboard")).to_be_visible(timeout=30_000)
    assert_selected_watchlist(subscriber_page, config)
    assert_watchlist_context(dashboard, config)
    expect(subscriber_page.locator("body")).not_to_contain_text("Shared EOD coverage")
    expect(subscriber_page.locator("body")).not_to_contain_text("central completed-session evidence")

    tenant_api = "/v1/tenants/" + config.tenant_id + "/marketops/"
    for route, endpoint, heading in (
        ("eroc", tenant_api + "eroc", "Exhaustive Reversal"),
        ("valuation", tenant_api + "valuation", "Value Intelligence & Distressed Opportunity Intelligence"),
        ("earnings", tenant_api + "earnings-opportunities", "Earnings Opportunity Intelligence"),
    ):
        response = visit_for_response(subscriber_page, f"{config.base_url}/marketops/{route}", endpoint)
        expect(subscriber_page.get_by_role("heading", name=heading)).to_be_visible(timeout=30_000)
        assert_selected_watchlist(subscriber_page, config)
        payload = assert_watchlist_context(response, config)
        if route == "eroc":
            eroc_results = payload.get("results")
            assert isinstance(eroc_results, list) and eroc_results, f"{response.url} returned no global EROC results"
            selected_eroc = [item for item in eroc_results if str(item.get("ticker", "")).upper() == config.shared_tickers[0]]
            assert selected_eroc, f"{response.url} omitted selected global EROC result"
            assert selected_eroc[0].get("data_scope") == "platform-global", f"{response.url} fell back from global EROC"
        if route == "valuation":
            valuation_results = payload.get("results")
            assert isinstance(valuation_results, list) and valuation_results, f"{response.url} returned no global valuation results"
            selected_valuation = [item for item in valuation_results if str(item.get("ticker", "")).upper() == config.shared_tickers[0]]
            assert selected_valuation, f"{response.url} omitted selected global valuation result"
            assert selected_valuation[0].get("data_scope") == "platform-global", f"{response.url} fell back from global valuation"
        if route == "earnings":
            eeom_results = payload.get("results")
            selected_eeom = [item for item in eeom_results if str(item.get("ticker", "")).upper() == config.shared_tickers[0]] if isinstance(eeom_results, list) else []
            if selected_eeom:
                assert selected_eeom[0].get("data_scope") == "platform-global", f"{response.url} fell back from global EEOM"

    state = visit_for_response(
        subscriber_page,
        f"{config.base_url}/marketops/state?symbol={config.shared_tickers[0]}",
        "/v1/marketops/states?",
    )
    assert_selected_watchlist(subscriber_page, config)
    assert_state_request_scope(state, config)
    state_payload = assert_watchlist_context(state, config)
    market_states = state_payload.get("market_states")
    assert isinstance(market_states, list) and market_states, f"{state.url} returned no global Market State"
    selected = [item for item in market_states if str(item.get("symbol", "")).upper() == config.shared_tickers[0]]
    assert selected, f"{state.url} omitted selected global Market State"
    assert selected[0].get("tenant_id") == "platform-global", f"{state.url} fell back from global Market State"
    expect(subscriber_page.locator("body")).to_contain_text(config.shared_tickers[0])

    assurance = visit_for_response(
        subscriber_page,
        f"{config.base_url}/marketops/assurance",
        "/v1/marketops/signal-assurance/effectiveness?",
    )
    expect(subscriber_page.get_by_role("heading", name="Signal Assurance")).to_be_visible(timeout=30_000)
    assurance_payload = assurance.json()
    assert assurance_payload.get("data_scope") == "platform-global", f"{assurance.url} fell back from global Signal Assurance evidence"
    assert isinstance(assurance_payload.get("watchlist_context"), dict), f"{assurance.url} omitted watchlist context"
    effectiveness = assurance_payload.get("effectiveness")
    assert isinstance(effectiveness, list) and effectiveness, f"{assurance.url} returned no global historical outcome cohort"
    assert all(item.get("evidence_source") == "LEGACY" for item in effectiveness), f"{assurance.url} represented unconfirmed SAF assertions as evidence"


def test_subscriber_sri_uses_platform_global_projection(subscriber_page: Page) -> None:
    config = subscriber_ui_config()
    login(subscriber_page, config)
    rankings = visit_for_response(
        subscriber_page,
        f"{config.base_url}/marketops/sectors",
        "/v1/marketops/sectors/rankings?",
    )
    expect(subscriber_page.get_by_role("heading", name="Sector Rotation Intelligence")).to_be_visible(timeout=30_000)
    payload = assert_watchlist_context(rankings, config)
    assert payload.get("data_scope") == "platform-global", f"{rankings.url} fell back from global SRI"
    snapshots = payload.get("snapshots")
    assert isinstance(snapshots, list) and snapshots, f"{rankings.url} returned no platform-global SRI snapshots"
    segment_id = snapshots[0].get("segment_id")
    assert isinstance(segment_id, str) and segment_id, f"{rankings.url} returned an invalid SRI segment"

    subscriber_page.get_by_role("tab", name="ETF progression").click()
    row = subscriber_page.locator("tr[aria-expanded]").first
    with subscriber_page.expect_response(
        lambda response: response.request.method == "GET" and f"/v1/marketops/sectors/{segment_id}/history?" in response.url,
        timeout=30_000,
    ) as response_info:
        row.click()
    history = response_info.value
    assert history.status == 200, f"{history.url} returned HTTP {history.status}"
    history_payload = assert_watchlist_context(history, config)
    assert history_payload.get("data_scope") == "platform-global", f"{history.url} fell back from global SRI history"
    assert isinstance(history_payload.get("snapshots"), list) and history_payload["snapshots"], f"{history.url} returned no global SRI history"
