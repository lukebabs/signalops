# Subscriber Project — Mobile User Readiness Sprint

Status: planned production-readiness sprint.

Owner: MarketOps subscriber product and frontend engineering.

Scope: subscriber-facing MarketOps mobile web. Administration, operator run-now controls, and platform governance screens are explicitly out of scope for this sprint.

## Purpose

MarketOps subscribers are expected to be primarily mobile users. The product must therefore support a phone-first analyst journey for reading the market, reviewing watchlist intelligence, inspecting evidence, and using Syncratic explainability without requiring desktop layout assumptions.

This sprint turns mobile usability from an incidental responsive behavior into a formal production gate.

## Current position

The application already uses responsive layout primitives, card/detail patterns, scroll-contained tables, and some mobile-aware flows. Historical validation exists for older 375px views, but the current subscription product has changed materially since then: watchlists, global data-plane projections, SAF, Syncratic narratives, subscription gating, and B2C enrollment all changed the user journey.

The product is therefore not mobile-production-ready until current subscriber routes pass a dedicated mobile acceptance suite and the highest-friction views are remediated.

## Non-goals

- No Admin Workbench mobile optimization.
- No Subscription Administration mobile optimization.
- No platform operator run-now workflow on phone.
- No native iOS/Android app build in this sprint.
- No mobile-only market-data behavior or separate mobile entitlement model.

Admin remains a desktop-recommended/operator workflow until a separate Admin Mobile sprint is approved.

## Primary mobile user journeys

### M0 — First authenticated landing

The user signs in, lands on MarketOps, and immediately sees a useful mobile dashboard without needing to zoom, rotate, or scroll past operational noise.

Acceptance:

- Login callback returns to the intended MarketOps route.
- Dashboard loads within the mobile viewport with the most important subscriber context first.
- Syncratic narrative digest is available but not full-screen dominant.
- Market Intelligence reel is readable in the right/secondary content area on wider screens and stacks below core content on phones.
- No route shows a blank shell while data is still available.

### M1 — Watchlist-first navigation

The user sees the selected private watchlist first. If no private watchlist exists, the tenant default appears. The same selection governs Dashboard, Assets, Market State, EROC, Value Intelligence, Distressed Opportunity Intelligence, Earnings Opportunity Intelligence, Opportunities, Signal Assurance, and Syncratic narratives.

Acceptance:

- Watchlist selection is visible and changeable on mobile.
- “Use across MarketOps” is reachable without bottom-page scrolling.
- Watchlist context survives reload and route changes.
- Cold/warming assets remain truthful and do not render fabricated placeholders.

### M2 — Daily market read

The user can answer: “What changed today, where should I focus, and why?”

Acceptance:

- Dashboard mobile presents a concise daily read.
- Market State, Risk/Reward, Review Queue/Opportunities, SRI, and Syncratic narrative snippets are reachable from the mobile dashboard.
- Cards expose the first useful explanation inline and link to full Syncratic Intelligence when deeper context is needed.
- The user does not need to open Admin or raw evidence pages to understand freshness.

### M3 — Asset intelligence drilldown

The user can inspect an asset from a list and understand its current evidence without fighting tables.

Acceptance:

- Assets render in card/list format on mobile, not table-first.
- Selecting an asset opens a focused detail panel or inline detail with an obvious close/back path.
- Price, Market State, Value Intelligence, Distressed Opportunity Intelligence, EROC, EEOM, Options/Risk-Reward, and SAF snippets are stacked in readable sections.
- Long evidence tables are either summarized or horizontally scroll inside their own container, never causing page-level overflow.

### M4 — Signal Assurance viability

The user can review signal effectiveness progression without desktop-scale charts.

Acceptance:

- SAF summary, 10/20-day filters, daily progression chart, and analyst drilldown are visible at phone width.
- “Inspect observations” opens directly below the selected row/card and can be closed easily.
- Chart labels remain legible or intentionally simplified on mobile.
- Dark and light mode contrast is readable.

### M5 — Sector Rotation Intelligence

The user can review sector progression and ETF context on mobile.

Acceptance:

- SRI cards stack cleanly.
- ETF progression charts use reduced x-axis density on mobile.
- ETF makeup tab remains readable through summary + contained constituent table.
- Segment and ETF filters wrap without forcing horizontal page scroll.

### M6 — Syncratic Intelligence

The user can get a natural-language explanation that is connected to current platform evidence.

Acceptance:

- `/marketops/syncratic` works as the full narrative destination from Dashboard cards.
- Narrative list, selected narrative, context metadata, and Ask Explanation are readable on mobile.
- Full explanations are not trapped inside Asset Drilldowns.
- Failed or deterministic explanations are presented without internal implementation labels.

### M7 — Enrollment and subscription path

The mobile user can register, verify SMS MFA, sign in, and understand subscription limits.

Acceptance:

- App-hosted login/register path renders cleanly on phone.
- Keycloak SMS MFA required-action screen displays SignalOps branding and compliance disclosure.
- Duplicate existing-user registration routes to login/resolution rather than creating confusion.
- Subscription pricing/upgrade prompts render as concise mobile cards.
- Checkout activation remains governed by the approved Stripe boundary.

## Acceptance viewport matrix

Minimum browser widths:

- 375 × 812 — small iPhone-class viewport.
- 390 × 844 — common iPhone-class viewport.
- 430 × 932 — large phone viewport.

Optional stretch:

- 768 × 1024 — tablet portrait.

Every required route must pass:

- no horizontal page overflow;
- no overlapping text or controls;
- no clipped primary actions;
- no unreadable chart labels;
- no route-level blank state when valid evidence exists;
- dark/light contrast readable;
- selected watchlist context visible or discoverable;
- back/close path for every mobile drilldown.

## Required mobile routes

Subscriber routes:

- `/marketops/dashboard`
- `/marketops/watchlists`
- `/marketops/assets`
- `/marketops/state`
- `/marketops/valuation`
- `/marketops/eroc`
- `/marketops/earnings`
- `/marketops/opportunities`
- `/marketops/sectors`
- `/marketops/signal-assurance`
- `/marketops/syncratic`
- `/marketops/pricing`

Enrollment routes:

- app-hosted `/auth/login?redirect=/marketops/dashboard`
- Keycloak registration form
- Keycloak `CONFIGURE_SMS_MFA` required-action screen
- subscription return/error path

## Automation plan

Add a dedicated mobile Playwright suite instead of overloading the desktop subscriber smoke.

Proposed file:

```text
python/tests/test_subscriber_mobile_ui.py
```

Proposed launcher:

```text
scripts/run_subscriber_mobile_ui_smoke.sh
```

The suite should use the existing real-OIDC QA identity and pre-seeded watchlist fixtures. It should retain failure-only HAR, trace, and screenshots under the protected artifact directory.

Core assertions:

- login succeeds and preserves redirect;
- key subscriber routes load at 375px and 430px;
- document width does not exceed viewport width by more than a small tolerance;
- primary headings/cards are visible;
- selected watchlist context is present where applicable;
- drilldowns can open and close on Assets, SAF, SRI, and Opportunities;
- Syncratic narrative full view is reachable from Dashboard;
- no console/page errors;
- screenshots are retained only on failure unless an evidence mode is explicitly enabled.

## Remediation backlog

Initial likely fixes:

1. Dashboard
   - Prioritize watchlist-scoped daily read.
   - Keep narrative digest lower on page.
   - Ensure Market Intelligence reel stacks below core content on phone.

2. Assets
   - Prefer cards on mobile.
   - Keep detail close/back controls sticky or visible.
   - Avoid table-first asset scanning on phone.

3. SAF
   - Convert dense rows to expandable cards on mobile.
   - Keep chart immediately below analyst drilldown.
   - Simplify x-axis labels for phone widths.

4. SRI
   - Reduce chart label density on phone.
   - Use card-first ETF progression.
   - Keep ETF makeup as summary plus contained table.

5. Syncratic
   - Make full narrative destination obvious.
   - Ensure Ask Explanation is exposed as explainability, not internal “Ask” mechanics.

6. Enrollment
   - Verify SignalOps branding and SMS disclosure on phone.
   - Ensure duplicate-user routing is clear.

## Mobile app viability track

This sprint should preserve a path toward a mobile app, but not start native development.

The near-term product should be built as mobile web with PWA discipline:

- stable responsive routes;
- installable web-app manifest later;
- mobile-safe session renewal;
- offline/error-friendly cached shell later;
- push-notification policy only after alerting/compliance decisions;
- route boundaries that map cleanly into native tabs later.

Native app viability should be evaluated after mobile web proves the core user journey. The main decision points are:

- whether mobile engagement requires push notifications;
- whether biometric/device-bound auth is needed;
- whether app-store distribution improves conversion enough to justify maintenance;
- whether Stripe subscription flows must remain web-first for policy and cost reasons;
- whether chart and drilldown performance is acceptable inside mobile web.

Recommended sequence:

1. Mobile web production gate.
2. PWA readiness review.
3. Native wrapper feasibility study.
4. Native app only if engagement, notification, or distribution evidence justifies it.

## Exit criteria

The sprint exits when:

- mobile Playwright suite exists and passes against production with the configured subscriber QA identity;
- screenshots or traces are retained as controlled evidence;
- all required subscriber routes pass at 375px and 430px;
- Dashboard, Assets, SAF, SRI, Opportunities, Syncratic, and enrollment flows have no blocking mobile usability issues;
- Admin remains explicitly documented as desktop/operator scope;
- production readiness path records mobile subscriber readiness as closed or identifies only non-blocking follow-up polish.
