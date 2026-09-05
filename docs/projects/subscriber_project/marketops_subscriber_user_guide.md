# MarketOps Subscriber User Guide

Status: production-readiness draft for pilot users and tenant administrators.

## Purpose

MarketOps gives each subscriber a personalized analytical workspace over one centrally governed MarketOps data plane. Users create watchlists and choose the analytical depth allowed by their subscription tier; the platform centrally manages assets, prices, algorithms, provider polling, and evidence.

MarketOps is a research and intelligence system. It does not provide trading advice, brokerage execution, or a guarantee that every security has every enrichment available at all times.

## Core concepts

| Concept | Meaning |
|---|---|
| Global catalog | The centrally governed asset list. Watchlists reference this catalog; they do not create duplicate assets. |
| Tenant default list | The shared default list for a tenant. If a user has no private list, MarketOps should fall back to this list. |
| Private watchlist | A user-owned list visible only to that user, plus authorized platform controls. |
| Warm asset | An asset covered by central end-of-day collection and EOD algorithms. |
| Hot asset | An asset explicitly selected in at least one watchlist; hot assets are eligible for intraday monitoring. |
| Cold asset | A governed catalog asset not yet centrally warmed for MarketOps evidence. Adding it to an authorized list creates one global activation request. |
| Freshness | The latest completed-session evidence available for a view. Weekends and market holidays do not create new EOD or intraday evidence. |

## Subscription tiers

MarketOps subscriptions increase analytical depth rather than selling separate copies of market data.

| Tier | Intended access |
|---|---|
| Explorer | Market Dashboards, Public Signals, Sector Rotation Discovery, Limited Watchlists. |
| Professional | Explorer plus Value Intelligence, Distressed Opportunity Intelligence, Earnings Opportunity Intelligence, Sector Rotation Details, Options Signals, Earnings Calendar, and Research Reports. |
| Institutional | Professional plus Signal Assurance Analytics, Portfolio Analysis, Batch Screening, Historical Replay, Strategy Validation, Custom Universes, APIs, unlimited/governed watchlists, and White-Label Controls. |

The gateway is authoritative. If a feature is outside the user’s tier, the browser may show it as locked and direct API Access returns a subscription error.

## Main user workflows

### 1. Open MarketOps

1. Sign in.
2. Open MarketOps.
3. Confirm the Dashboard loads with the expected tenant/user context.
4. If the page shows Not Found after login, sign out only as a temporary workaround and report it. The expected production behavior is direct load after the first login.

### 2. Use watchlists

1. Open Assets.
2. If you have private watchlists, the first private/custom watchlist should be selected by default.
3. If you do not have private watchlists, the tenant default list should be selected.
4. Select a different watchlist when you want the Dashboard and related MarketOps views to follow a different asset scope.

Expected behavior:

- Assets, Dashboard, Review Queue, Value Intelligence, Distressed Opportunity Intelligence, Earnings Opportunity Intelligence, and other asset-scoped views should align to the selected watchlist where that view is watchlist-scoped.
- Sector Rotation Intelligence remains a platform/global sector view, not a private-watchlist-only view.
- Material Events should show events related to assets in the selected scope when such events exist.

### 3. Add an asset

1. Search the governed catalog.
2. Add the asset to a permitted list.
3. If the asset already has central evidence, it should become usable from the shared global source.
4. If the asset is cold, the system records one global activation request and shows queued or warming-up status until central evidence is ready.

Important boundaries:

- The browser does not call data providers.
- Adding an asset is a watchlist membership action, not tenant-local provider polling.
- Admin asset onboarding remains controlled; ordinary users should not perform provider-authoritative asset administration.

### 4. Interpret freshness and status

Freshness should be read by completed market session, not by wall-clock date alone.

| Status | Meaning |
|---|---|
| Current | The view has the latest expected evidence for the completed market session or is correctly idle outside market hours. |
| Pending / warming up | The asset or enrichment is being activated or does not yet have enough central evidence. |
| Partial / degraded | The job completed with bounded provider gaps or incomplete enrichment; the UI/admin view should show the reason. |
| Stale | The view is behind the expected completed session and needs operator review. |

Weekend example: Saturday should not produce a new EOD or intraday date. Friday’s completed session remains the latest expected market date until the next trading day completes.

## Analytical surfaces

| Surface | User interpretation |
|---|---|
| Dashboard | Portfolio/watchlist-level summary of current MarketOps context. |
| Assets | Watchlist-driven asset cards and drill-downs. |
| Review Queue | Operational list of signals/opportunities that need analyst attention. Stale evidence should be filtered or clearly identified. |
| Value Intelligence | Valuation-oriented algorithm evidence and scoring. |
| Distressed Opportunity Intelligence | EROC/distress-oriented opportunity evidence. |
| Earnings Opportunity Intelligence | Earnings-event opportunity evidence. |
| Sector Rotation Intelligence | ETF/sector progression and makeup intelligence. |
| Market Structure Intelligence | Market state, intraday conditions, and structural context. |
| Signal Assurance | Operator/institutional viability workbench for benchmark analysis of signal outcomes. The operational view excludes pre-August 20, 2026 outcomes, defaults to 10 trading days, and caps the chart window at 20 trading days. |

## Tenant administrator workflows

Tenant administrators should focus on tenant-level controls:

- default watchlist policy;
- Institutional seat assignment when enabled;
- tenant-scoped usage visibility;
- user support and escalation;
- tenant branding/custom universe controls when released.

Tenant administrators should not bypass the global catalog, manually poll providers, or grant themselves platform subscription-administration rights.

## Platform administrator workflows

Platform administrators use Administration, not MarketOps, for platform controls:

- Subscription Administration;
- user enrollment and plan/tier governance;
- product features and limits;
- tenant contracts and Institutional seats;
- user activity review for login, logout, feature views, and MarketOps mutations;
- audit trail review;
- System / Operations Health and Scheduled Jobs.

Activity monitoring is operational metadata, not content capture. The platform records immutable subject, tenant, event type, feature/route, status, correlation, and timestamp. It does not store passwords, tokens, cookies, request bodies, response bodies, or provider payloads.

## What to report

Report these as product/operations issues:

- Dashboard, Assets, Review Queue, or SAF show different latest completed-session dates without an explicit reason.
- A private list shows another user’s private assets.
- A tenant can read another tenant’s MarketOps route.
- A user can access a feature outside their tier.
- A scheduled-job status is failed without a visible degraded/skip reason.
- A market holiday or weekend produces unexpected EOD/intraday jobs.



## Profile and Settings — 2026-09-04

MarketOps now separates ordinary subscriber controls from operational tooling.

- Use **Profile** to review identity, tenant, enrollment state, current subscription, feature access, watchlist context, limits, billing actions, and refund request options.
- Use **Settings** for user-centric account, subscription upgrade, billing/refund, preference, and watchlist-default workflows.
- Use **Pricing** when entering from enrollment or a locked feature prompt. Pricing and Settings use the same package presentation.
- Operational asset onboarding, research backtest controls, and the Signal Assurance operational workbench are no longer part of user Settings or normal subscriber navigation; they live in **MarketOps Tools** for operator/admin users.
