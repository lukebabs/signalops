# Subscriber Project

Status: discovery and architecture project. This documentation describes a target subscription-service capability; it does not change the deployed MarketOps data contracts.

## Goal

Turn MarketOps into a subscription service in which customers receive an individualized experience over one centrally governed market-data and algorithm-processing plane.

The platform will maintain one global catalog of Massive-eligible US common stocks and process the configured EOD MarketOps baseline once per covered symbol. The approved top 1,000 eligible assets form the hot baseline; an eligible cold asset selected in an authorized watchlist creates one globally deduplicated coverage-activation request and warms centrally. A tenant and its users consume authorized projections of that shared data. They will not create copies of assets, prices, features, states, algorithm outputs, or provider pulls.

## Product decisions recorded

- The initial global catalog scope is exchange-listed US common stocks that Massive can govern and identify. The centrally processed hot baseline is capped at the approved top 1,000 eligible assets; other governed eligible assets remain cold until demand activates them.
- Price-based EOD ingestion, features, and algorithms run centrally for the active global coverage set.
- Each tenant has a shared default watchlist. Each user has private watchlists and sees only those lists plus the tenant default.
- Every watchlist entry references a global asset. Adding a hot symbol to a list is a membership change, not asset onboarding or data collection. Adding an eligible cold symbol also writes the membership, then creates one idempotent, auditable global activation request; the browser never calls Massive and the system shows `queued`/`warming_up` until central evidence is available.
- Tenant RBAC remains the authorization and isolation boundary for browser/API access. The existing OIDC/JWT, immutable subject, tenant claim, per-use-case grant, and audit primitives are the foundation for subscriber access control; the subscriber-specific authorization layer is an explicit production gate.
- The project will not introduce a second identity provider or authorization service. It will extend the existing gateway and grant model with tenant binding, list ownership, entitlements, quotas, and service identities.
- Options data is an entitled, budgeted enrichment. At EOD, the system resolves one deduplicated demand union across eligible users and tenants, polls each selected symbol once, and shares the persisted result with all entitled watchers. Options analytics use the central, prospective capture history: the current default lookback is 10 calendar days, configurable up to 60 calendar days.
- An options symbol starts with no reconstructed chain history. Its history begins with its first selected EOD capture and is shared globally; the subscriber experience must show that the analysis is warming up until the selected window has sufficient captured sessions.
- Intraday and options coverage must never be widened implicitly just because the global EOD catalog is broad.

## Why this is needed

The current MarketOps asset model is tenant-owned and supports direct watchlist onboarding. That works for a single operating tenant but would duplicate identities and trigger tenant-specific work at subscription scale. The Subscriber Project replaces that ownership assumption with a global asset/coverage model and makes tenant/user artifacts lightweight preferences and access controls.

The existing SRI ETF makeup feature is not the product boundary. It is one useful source of current constituents and an example of provenance-preserving market representation. ETF composition must not determine the global catalog or make a user-visible list unless a user or tenant chooses it.

## Document set

- [Global Catalog and Watchlist Roadmap](global_catalog_watchlists_roadmap.md): target architecture, data flow, options demand design, compatibility-first sprint plan, controls, and acceptance measures.
- [Sprint S1 global-catalog shadow](s1_global_catalog_shadow.md): additive identity, immutable provenance, coverage-shadow, controlled seed, and parity evidence.
- [Sprint S2 catalog breadth and EOD planner shadow](s2_catalog_breadth_eod_planner_shadow.md): evidence-gated US-common-stock admission, top-1,000 selection, and cold-activation queue without collection.
- [S2 ranking-source preflight](s2_ranking_source_preflight_2026-08-12.md): provider capability evidence and the required decision to fill the top-1,000 capacity.
- [S2 ranked hot-set plan evidence](s2_ranked_hot_set_plan_evidence_2026-08-12.md): supplied ranking snapshot, governed admission result, and final no-collection shadow-plan record.
- [Sprint S3 lists and authorization projection](s3_lists_authorization_projection.md): RLS-protected tenant-default and private-list foundation, initially without a browser route.
- [S3 pilot readiness gate](s3_pilot_readiness_gate.md): required pre-enable checks for the dedicated gateway login, named tenant, entitlement, and default list.
- [S3 pilot tier and default-list decision](s3_pilot_tier_default_list_decision_2026-08-12.md): tenant-pilot-b list-only tier and audited governed top-ten seed.
- [Central data, business continuity, and disaster recovery](central_data_business_continuity_disaster_recovery.md): centralized storage model, dependency order, recovery controls, and the required pre-pilot rehearsal.
- [MarketOps daily surveillance architecture](../../use_cases/marketops/daily_market_surveillance/architecture/functional_components.md): current production components.
- [Sprint S0 — Baseline and Controls](s0_baseline_and_controls.md): read-only current-state baseline utility, metrics, evidence review, reserved flags, and rollback posture.
- [Sprint S0-A — Access-control Hardening](s0a_access_control_hardening.md): principal-bound tenant enforcement and the deployment gate.
- [Sprint S0-A exit checklist](s0a_exit_checklist.md): required workload-login, OIDC, browser, and cross-tenant evidence before enabling S1.
- [S0-A deployment evidence — 2026-08-12](s0a_deployment_evidence_2026-08-12.md): pilot bootstrap, browser, mutation-denial, and cross-tenant validation record awaiting final approval.
- [Entitlement and quota policy contract](entitlement_quota_policy.md): separate default-deny subscriber product-policy decision model.
- [Shared-worker identity contract](worker_identities.md): least-privilege future worker identities and deployment boundary.
- [Database row-level security decision](row_level_security_decision.md): hybrid subscriber-table RLS boundary and required rollout model.
- [Tenant Use-Case RBAC v1](../../rbac_use_case_access_v1.md): current grant model, gateway enforcement, and access-management audit boundary.
- [MarketOps ETF makeup guide](../../use_cases/marketops/daily_market_surveillance/operations/sri_etf_makeup.md): current issuer snapshot behavior and provenance boundary.
- [Tenant access implementation](../../frontend/auth_integration_spec.md): current authenticated-session behavior and tenant claim model.

## Non-goals

This project does not yet authorize:

- A new public pricing, billing, payment, or account-provisioning implementation.
- Historical reconstruction of every asset before the governed coverage policy exists.
- An implied options-history depth or a claim that a configured 10-day window represents 10 captured trading sessions.
- A promise that all catalog assets have options, intraday, fundamental, or complete historical coverage.
- Per-user provider polling, duplicated storage, or implicit entitlement expansion.
- Trading advice, recommendation, or a change to the research-only boundaries of MarketOps algorithms.

## Success definition

A customer can find an entitled global asset, add it to a personal list, and see the same centrally calculated EOD MarketOps context that another entitled customer sees—while their private list membership remains inaccessible to other users and the system performs no duplicate market-data collection.
