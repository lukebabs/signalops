# Subscriber Project

Status: active implementation project. The global catalog, subscriber-list foundation, user Profile/Settings flow, Stripe Checkout/Portal/refund-intake surfaces, and first webhook-authoritative paid activation evidence are deployed behind controlled production gates. Scheduled-job operations status is stored in the dedicated MarketOps database; repo-local runtime JSON is retained only as ignored fallback/debug output.

## Goal

Turn MarketOps into a subscription service in which customers receive an individualized experience over one centrally governed market-data and algorithm-processing plane.

The platform will maintain one global catalog of Massive-eligible US common stocks and process the configured EOD MarketOps baseline once per covered symbol. The approved top 1,000 eligible assets form the **warm EOD baseline**; an eligible cold asset selected in an authorized watchlist creates one globally deduplicated coverage-activation request and warms centrally. A tenant and its users consume authorized projections of that shared data. They will not create copies of assets, prices, features, states, algorithm outputs, or provider pulls.

## Product decisions recorded

- The initial global catalog scope is exchange-listed US common stocks that Massive can govern and identify. The centrally processed warm EOD baseline is capped at the approved top 1,000 eligible assets; other governed eligible assets remain cold until demand activates them.
- The **hot intraday tier** is separate: it is the deduplicated union of assets in an explicitly saved MarketOps watchlist selection. Warm membership alone never produces intraday provider polling.
- Where a warm asset needs initial price history, its one-time central EOD backfill uses the existing analyst-watchlist default: 50 prior weekdays plus the selected completed-session end date (at most 51 weekday observations before market holidays). It is equity EOD only, never options or intraday, and is not a recurring schedule or a five-year reconstruction.
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

- [Keycloak B2C enrollment flow](keycloak_b2c_enrollment.md): public sign-up, enrollment resolver, Explorer auto-provisioning, and guardrails.
- [MarketOps Subscriber User Guide](marketops_subscriber_user_guide.md): user-centric guide for subscribers, tenant administrators, and platform administrators covering watchlists, freshness, subscription tiers, analytical surfaces, and support escalation.
- [MarketOps Production Engineering Handbook](marketops_production_engineering_handbook.md): engineering/operator guide for production source-of-truth, scheduler controls, post-close completion, access/subscription controls, validation, and current open readiness items.
- [Subscription Commerce Model](subscription_commerce_model.md): Explorer, Professional, and Institutional product decisions; tier enforcement, RBAC separation, Administration governance, Stripe boundary, rollout, and deferred work.
- [MarketOps Subscription User Journey v1.0](../../MarketOps_Subscription_User_Journey_v1.0_Code_Agent_Spec.md): product/UX specification for Visitor, Explorer, Researcher, Professional, and Institutional subscription progression; contextual upgrade triggers; DAR metrics; Stripe checkout/webhook journey; and phased acceptance criteria.
- [Stripe Billing and Checkout](stripe_admin_managed_billing.md): controlled Stripe product/customer/subscription mapping, signed webhook reconciliation, and opaque-reference Explorer/Professional Checkout without customer portal.
- [Subscriber User Activity Monitoring](subscriber_user_activity_monitoring.md): append-only login/logout/feature-view/mutation activity visibility in Subscription Administration, with privacy and retention boundaries.
- [Syncratic Ask Readiness Checklist](syncratic_ask_readiness_checklist.md): production-readiness controls for prompt governance, AI Gateway policy/catalog alignment, idempotency, browser smoke validation, and failure handling.
- [Mobile User Readiness Sprint](mobile_user_readiness_sprint.md): subscriber-facing mobile-web production gate, excluding Admin/operator workflows while preserving a path toward PWA/native-app viability.
- [SAF-2 multi-horizon signal usefulness sprint](saf_multi_horizon_usefulness_sprint.md): planned Signal Assurance enhancement that evaluates confirmed assertions across 1/5/10/20 trading-day horizons using materialization, MFE, MAE, benchmark-relative performance, lifecycle states, and versioned usefulness scoring instead of superficial one-day hit/miss interpretation.

- [Global Catalog and Watchlist Roadmap](global_catalog_watchlists_roadmap.md): target architecture, data flow, options demand design, compatibility-first sprint plan, controls, and acceptance measures.
- [Tenant-local legacy-default preservation — 2026-08-16](tenant_local_legacy_default_preservation_2026-08-16.md): audited, non-destructive preservation of the 132-symbol legacy tenant-default list through canonical global identities.
- [Tenant-local legacy-hot parity foundation — 2026-08-16](tenant_local_legacy_hot_parity_foundation_2026-08-16.md): immutable inventory of the 132 legacy current intraday states and all retained Risk/Reward history; serving and scheduler cutover remain gated.
- [Tenant-local legacy-hot materialization evidence — 2026-08-16](tenant_local_legacy_hot_materialization_evidence_2026-08-16.md): exact central Risk/Reward parity and current-state intraday import, with reader and scheduler release still gated.
- [Global intraday current-state projection evidence — 2026-08-16](global_intraday_current_state_projection_evidence_2026-08-16.md): restricted central reader with payload-based freshness; API, UI, and scheduler cutover remain disabled.
- [Legacy hot-cohort shadow evidence — 2026-08-16](legacy_hot_cohort_shadow_evidence_2026-08-16.md): temporary 125-eligible/132-preserved compatibility cohort and first truthful selector mismatch.
- [FMP annual financial enrichment and VC/DOSM v4](fmp_annual_v4_enrichment.md): disabled, centrally governed annual financial capture and parallel research-model contract; it does not replace the live v3 profile.
- [FMP annual v4 migration evidence — 2026-08-16](fmp_annual_v4_migration_evidence_2026-08-16.md): additive dedicated-primary schema deployment, with no provider, scheduler, Gateway, or tenant-data side effect.
- [FMP annual v4 entitlement preflight — 2026-08-16](fmp_annual_v4_entitlement_preflight_evidence_2026-08-16.md): one-symbol, five-endpoint dry-run proof of Starter annual-data access.
- **Production blocker:** the roadmap records the required global analytical data plane; subscriber production readiness is blocked until globally materialized EOD history and analytics replace legacy tenant-local evidence.
- [Sprint S1 global-catalog shadow](s1_global_catalog_shadow.md): additive identity, immutable provenance, coverage-shadow, controlled seed, and parity evidence.
- [Sprint S2 catalog breadth and EOD planner shadow](s2_catalog_breadth_eod_planner_shadow.md): evidence-gated US-common-stock admission, top-1,000 selection, and cold-activation queue without collection.
- [S2 ranking-source preflight](s2_ranking_source_preflight_2026-08-12.md): provider capability evidence and the required decision to fill the top-1,000 capacity.
- [S2 ranked hot-set plan evidence](s2_ranked_hot_set_plan_evidence_2026-08-12.md): supplied ranking snapshot, governed admission result, and final no-collection shadow-plan record.
- [Sprint S3 lists and authorization projection](s3_lists_authorization_projection.md): RLS-protected tenant-default and private-list foundation, API-only local pilot; still without a browser route.
- [S3 pilot readiness gate](s3_pilot_readiness_gate.md): required pre-enable checks for the dedicated gateway login, named tenant, entitlement, and default list.
- [S3 pilot tier and default-list decision](s3_pilot_tier_default_list_decision_2026-08-12.md): tenant-pilot-b list-only tier and audited governed top-ten seed.
- [S3 pilot preflight and activation evidence](s3_pilot_preflight_evidence_2026-08-12.md): passed local readiness checks and the subsequent tenant-pilot-b API-only gateway activation.
- [Sprint S4 shared-EOD canary](s4_shared_eod_canary.md): disabled-by-default, immutable small-cohort preparation from an existing S2 shadow plan; it has no provider or scheduler execution path.
- [S4 pilot canary preparation evidence](s4_pilot_canary_preparation_evidence_2026-08-13.md): prepared-only NVDA/AAPL cohort from the frozen shadow plan; provider and scheduler execution remain false.
- [S4 canary execution gate](s4_canary_execution_gate.md): immutable two-request controls, per-symbol evidence/parity contract, workload preflight, and an engaged kill switch; provider collection remains disabled.
- [S4 canary execution-gate evidence](s4_canary_execution_gate_evidence_2026-08-13.md): migration and dedicated-workload proof for the live disabled NVDA/AAPL gate; no provider event is permitted.
- [S4 live-canary execution evidence](s4_live_canary_execution_evidence_2026-08-13.md): two-request Massive canary completed with 0/2 raw parity; VWAP and volume revisions are retained as immutable lineage evidence.
- [S4 policy-aware parity evidence](s4_policy_aware_parity_evidence_2026-08-13.md): the original-capture historical context and latest-verified current context both match for AAPL and NVDA (4/4), without another provider request.
- [S4 provider revision policy](s4_provider_revision_policy.md): immutable initial/revised EOD observations and field-level revision deltas; canonical selection is held for review.
- [S4 provider revision evidence](s4_provider_revision_evidence_2026-08-13.md): 4 immutable observations and 12 deltas prove VWAP/volume provider revisions while OHLC remains unchanged.
- [S4 as-of selection policy](s4_as_of_selection_policy.md): deterministic initial capture for historical assurance and latest verified revision for current context.
- [S4 as-of selection evidence](s4_as_of_selection_evidence_2026-08-13.md): deployed AAPL/NVDA resolution proves both contexts carry immutable policy and as-of provenance.
- [S4 historical assurance integration](s4_historical_assurance_integration.md): SAF effectiveness/outcomes and new backtests use the immutable initial-capture contract without historical restatement.
- [S4 historical assurance deployment evidence](s4_historical_assurance_deployment_evidence_2026-08-13.md): clean gateway release and health/auth-boundary verification for the SAF contract.
- [S4 current MarketOps context integration](s4_current_market_context_integration.md): a tenant-authorized, latest-verified EOD projection for the asset-detail API that remains separate from historical evaluation.
- [S4 current MarketOps context deployment evidence](s4_current_market_context_deployment_evidence_2026-08-13.md): deployed Assets detail card, projection checks, and release verification.
- [S4 EOD revision review workflow](s4_revision_review_workflow.md): immutable initial-versus-revised comparison, materiality, and tenant-authorized analyst access.
- [S4 revision review deployment evidence](s4_revision_review_deployment_evidence_2026-08-13.md): narrow projection, data, build, and public release proof.
- [Sprint S5 subscriber catalog search](s5_subscriber_catalog_search.md): pilot-only, active-entitlement-gated global catalog projection with no direct catalog read privilege or provider side effect.
- [S5 pilot capability activation](s5_pilot_capability_activation_2026-08-13.md): live `tenant-pilot-b` policy of catalog search 50 and ten quota-enforced, provider-free EOD activation requests; Options remains disabled.
- [S5 canonical catalog projection](s5_canonical_catalog_projection_2026-08-13.md): source provenance retained while subscriber search returns one canonical governed security.
- [S5 catalog membership repair](s5_catalog_membership_repair_2026-08-13.md): live fix for the private-list add mutation, with retry-safe quota reuse and cross-tenant proof.
- [Sprint S6 — Options demand union](s6_options_demand_union.md): internal shadow-only deterministic demand union; it has no provider, scheduler, entitlement, or capture side effect.
- [S6 Options demand snapshot contract](s6_options_demand_snapshot_contract.md): aggregate-only persistence and execute-only planner boundary for the next shadow slice.
- [S6 Options-demand workload preflight](s6_options_demand_workload_preflight.md): dedicated-login validation for aggregate-only input and append-only shadow storage.
- [S6 Options-demand deployment evidence](s6_options_demand_migration_evidence_2026-08-13.md): applied migration and provider-free dedicated-role rehearsal.
- [S6 Options-demand login evidence](s6_options_demand_login_evidence_2026-08-13.md): provisioned dedicated login and password-authenticated preflight record.
- [S6 Options-demand shadow-run evidence](s6_options_demand_shadow_run_evidence_2026-08-13.md): first real dedicated-login zero-demand snapshot proof.
- [S6 Options-demand quota and fairness](s6_options_demand_quota_fairness.md): numeric per-tenant demand quota and deferred-age carry-forward contract.
- [S6 quota/fairness deployment evidence](s6_options_demand_quota_fairness_evidence_2026-08-13.md): applied migration and restricted-role shadow run.
- [S6 pilot Options-demand shadow evidence](s6_pilot_options_demand_shadow_evidence_2026-08-13.md): bounded ten-asset pilot, non-zero selection, and deferred-age proof with zero capture.
- [S6 Options-capture canary gate](s6_options_capture_canary_gate.md): one-asset disabled capture control plane with no provider or scheduler path.
- [S6 Options-capture gate evidence](s6_options_capture_gate_evidence_2026-08-13.md): applied migration, dedicated identity preflight, and frozen disabled NVDA gate.
- [S6 Options-capture authorization request](s6_options_capture_authorization_request.md): append-only pending review request, distinct from any provider authorization.
- [S6 capture authorization-request evidence](s6_options_capture_authorization_request_evidence_2026-08-13.md): NVDA review request remains pending and non-executable.
- [S6 Options-capture named approval](s6_options_capture_named_approval.md): immutable human approval attestation held pending recovery.
- [S6 named-approval evidence](s6_options_capture_named_approval_evidence_2026-08-13.md): Luke approval is retained with a one-request/no-retry limit and recovery block.
- [Central data, business continuity, and disaster recovery](central_data_business_continuity_disaster_recovery.md): centralized storage model, dependency order, recovery controls, and the required pre-pilot rehearsal.
- [Production backup and restore runbook](production_backup_restore_runbook.md): procurement inputs, encrypted PostgreSQL/PITR backup procedure, restore sequence, and acceptance evidence required before production pilot enablement.
- [N1 production observability and recovery](n1_production_observability_recovery.md): dedicated-boundary health checks, alert delivery, recovery cadence, and host-watch remediation.
- [N1 closure evidence — 2026-08-19](n1_closure_evidence_2026-08-19.md): deployment-agent/Admin run-now closure, activation-queue reconciliation, and pgBackRest recovery-control re-anchor verification.
- [MarketOps scheduled-job status database migration — 2026-08-19](marketops_scheduled_job_status_database_2026-08-19.md): DB-backed scheduler-status source of truth, fallback boundary, and live verification.
- [Production readiness path](production_readiness_path.md): current readiness snapshot, production blockers, P0/P1/P2 gates, and the secure sprint path to production.
- [PR-0 scheduler reconcile evidence — 2026-08-21](pr0_scheduler_reconcile_evidence_2026-08-21.md): approved deployment-agent reconcile action installed and scheduler-status returned clean after stale post-close systemd state was cleared.
- [PR-1 admin operations visibility — 2026-08-21](pr1_admin_operations_visibility_2026-08-21.md): read-only Admin Workbench data-freshness visibility for Dashboard, Market State, Risk/Reward, SRI, SAF, and intraday.
- [PR-2 access and subscription hardening — 2026-08-21](pr2_access_subscription_hardening_2026-08-21.md): real-OIDC read-only tenant-isolation smoke and remaining subscription/access-control exit items.
- [PR-4 production expansion controls — 2026-08-21](pr4_production_expansion_controls_2026-08-21.md): FMP lifecycle, trading-calendar correctness, subscriber administration operationalization, and incident runbook controls.
- [PR-4 production incident runbooks — 2026-08-21](pr4_incident_runbooks_2026-08-21.md): operator response paths for stale data, failed post-close, provider outage, access regression, deployment smoke failure, backup/restore, and FMP annual degradation.
- [Automated browser acceptance](automated_browser_acceptance.md): real-OIDC, read-only subscriber smoke validation with failure-only HAR/trace/screenshot evidence.
- [S4 recovery readiness evidence — 2026-08-13](s4_recovery_readiness_evidence_2026-08-13.md): verified recovery-bucket and IAM controls, unprovisioned safeguards, and the handoff sequence before provider execution.
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

- Historical reconstruction of every asset before the governed coverage policy exists.
- An implied options-history depth or a claim that a configured 10-day window represents 10 captured trading sessions.
- A promise that all catalog assets have options, intraday, fundamental, or complete historical coverage.
- Per-user provider polling, duplicated storage, or implicit entitlement expansion.
- Automated Stripe refund execution, Institutional self-service provisioning, or MFA enforcement without a separately approved gate.
- Trading advice, recommendation, or a change to the research-only boundaries of MarketOps algorithms.

## Success definition

A customer can register through Keycloak, resolve into the governed tenant-local B2C path, activate a subscription through webhook-authoritative Stripe evidence, manage profile/settings/billing/refund-intake from user-facing MarketOps surfaces, find an entitled global asset, add it to a personal list, and see the same centrally calculated EOD MarketOps context that another entitled customer sees—while private list membership remains inaccessible to other users and the system performs no duplicate market-data collection.
