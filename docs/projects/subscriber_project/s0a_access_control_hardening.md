# Sprint S0-A — Access-control Hardening

Status: implementation complete locally; formal deployment exit remains pending the required workload-identity, browser, and cross-tenant evidence recorded in [the S0-A exit checklist](s0a_exit_checklist.md).

## Objective

Extend the existing OIDC/JWT and tenant-grant foundation without introducing a second identity system or changing the current tenant-owned MarketOps data model.

## Implemented slices: canonical tenant binding and gateway body guard

The gateway now provides `requestTenant` and `requireRequestTenant` in `internal/api/auth.go`.

In authenticated mode they:

1. Treat the verified principal `tenant_id` as authoritative.
2. Use that tenant when a handler value is absent.
3. Reject a handler value that conflicts with the principal.
4. Preserve existing supplied-tenant behavior when authentication is disabled for local development.

The access-administration surface now uses this control for listing grants, listing grant audit, creating/updating grants, and revoking grants. In particular, a super-admin token scoped to one tenant can no longer submit a different `tenant_id` in the access-grant JSON body.

The administration notifications and notification-email settings surfaces now use the same binding for their tenant-bearing reads and mutations. An authenticated request may omit `tenant_id` and is scoped to the verified principal; an explicit conflicting tenant remains rejected.

The MarketOps opportunity list, detail, disposition-history, and disposition-write routes now also derive omitted tenant scope from the verified principal. For every existing route that uses `replayActor`, authenticated audit fields now use the verified principal actor or subject rather than a request header or body value; local development retains its prior caller-supplied fallback.

Replay-job list, status, and create requests now inherit the verified tenant. Replay-job detail and cancellation first verify that the stored record belongs to that tenant and return the same not-found response for foreign records, preventing cross-tenant job enumeration or control.

All MarketOps backtest and calibration payload mutations now bind `tenant_id` to the verified principal before validation: campaigns, runs, calibration summaries, baselines, comparisons, promotion candidates, readiness snapshots, evaluations, and evaluation-label sync. Existing referenced-record checks therefore evaluate against authoritative tenant scope. Promotion-candidate decisions have no payload tenant, so they now load and verify candidate ownership before mutation and return not found for a foreign candidate.

Algorithm definitions and execution requests now bind their JSON tenant to the verified principal. Algorithm signal-proposal decisions and materializations retain their local body-or-query compatibility path, but resolve that value authoritatively when authenticated; their reviewer and requester fields use the verified principal through the shared actor resolver.

Alert and insight lifecycle routes now load the stored record before mutation and verify it belongs to the authenticated tenant. Foreign records use the ordinary not-found response and are not mutated; local development remains compatible when no principal is present.

Authenticated raw-event ingestion now resolves the request tenant before persistence and rewrites the published JSON payload with that canonical tenant. This keeps the broker event, raw-event ledger, and idempotency evidence aligned even when a caller omits `tenant_id`.

MarketOps graph-proposal list and detail routes now bind an omitted tenant to the authenticated principal and suppress foreign records with the ordinary not-found response. Both canonical and DSM proposal-decision routes verify stored ownership before mutation; the repository mutation itself now qualifies the update by `proposal_id` and `tenant_id` as defense in depth.

MarketOps DSM artifact lists now bind to the authenticated tenant, and artifact detail routes suppress foreign records with the ordinary not-found response. This protects the underlying evidence and provenance payloads associated with each artifact.

Signal Assurance assertion, evaluation, effectiveness, observation, and recommendation reads now resolve their tenant scope from the authenticated principal. Assertion-detail and evaluation routes continue to use tenant-qualified retrieval, so a foreign assertion ID returns the ordinary not-found response rather than revealing provenance or effectiveness evidence.

MarketOps algorithm-evaluation runs, result and outcome evidence, and backfill-campaign reads now bind an omitted tenant to the authenticated principal. Tenant-qualified run and campaign lookup suppresses foreign identifiers through the ordinary not-found response.

MarketOps signal-outcome lists and detail reads now bind an omitted tenant to the authenticated principal. The tenant-qualified outcome lookup continues to suppress foreign identifiers through the ordinary not-found response.

MarketOps hypothesis definitions and evaluations, plus feature, state, transition, and evidence lists, now bind an omitted tenant to the authenticated principal. State, lineage, and evidence detail routes verify stored ownership before responding, returning the ordinary not-found response for foreign records.

Raw-event, normalized-event, and signal ledger lists now bind to the authenticated tenant. Their detail routes verify stored ownership before responding, returning the ordinary not-found response for a foreign record ID.

MarketOps backtest and calibration read surfaces now bind every list filter to the authenticated tenant: coverage, campaigns, runs, summaries, baselines, comparisons, promotion candidates, readiness snapshots, evaluations, evaluation labels, and per-run signals, graph proposals, and policy results. Identifier-based campaign, run, calibration, promotion, readiness, evaluation, and label detail reads verify stored ownership and return the ordinary not-found response for a foreign identifier.

MarketOps intelligence cohort runs and readiness now inherit tenant scope from the authenticated principal; the cohort-run detail already uses tenant-qualified retrieval. Syncratic context-window and insight lists bind the same scope, while their details verify stored ownership before deriving related evidence and return the ordinary not-found response for foreign IDs. Algorithm registry and evaluation reads now reconcile their required tenant scope with the authenticated principal across definitions, execution requests and summaries, results, materializations, signal-proposal lists and summaries, preflight, and tenant-qualified details.

Alert and insight ledger lists now bind to the authenticated tenant, and their identifier details verify stored ownership before returning the ordinary not-found response for foreign records. Idempotency lookup now resolves the tenant from the principal, passes that canonical value into the tenant-qualified repository lookup, and suppresses a mismatched returned provenance record as not found.

Syncratic context-window creation, insight creation, Ask enrichment, and materialization now bind JSON tenant scope to the authenticated principal. Insight creation and Ask enrichment load the referenced context first; a foreign authenticated context returns the ordinary not-found response, no insight is persisted, and Ask is never called with its evidence. Local development retains its former explicit-tenant validation response when no principal is present.

MarketOps asset-management routes now resolve their tenant path segment against the authenticated principal before display metadata changes, watchlist creation, onboarding, backfill creation, validation, and backfill listing. This protects the subscriber project’s centrally governed asset catalog boundary from a path-tenant escalation, in addition to the gateway path guard.

The tenant-bearing mutation audit is complete. Body-tenant handlers bind an omitted tenant to the principal and reject conflicts; tenant-path handlers are covered by the gateway path guard, with the subscriber-critical asset-management routes also binding in-handler before storage access.

The gateway now exposes reusable list-authorization primitives without enabling any list endpoint: `requireRequestSubject` binds a private resource subject to the immutable authenticated subject and rejects impersonation, while `requireTenantAdministrator` requires the existing SignalOps administrator role for future tenant-default list administration. Local development remains compatible until the future list routes are enabled behind their feature gate.

The authenticated gateway also now inspects a bounded, top-level JSON `tenant_id` on `POST`, `PUT`, `PATCH`, and `DELETE` requests before the handler runs. A conflicting declared tenant is rejected at the gateway, while a valid body is restored unchanged for the handler. This prevents the same body-tenant escalation across the remaining JSON mutation routes while their handler-level binding is audited.

The Subscriber Project now has a separate, pure entitlement and quota evaluator at `internal/subscriber/policy`. It is explicitly distinct from MarketOps read/write grants, recognizes catalog search, shared EOD activation, and Options demand as separate capabilities, and defaults to deny when a matching explicit entitlement and quota are not present. Its deterministic decision includes tenant, subject, capability, units, usage snapshot, quota, policy and provisioning versions, correlation ID, and time so a later durable audit adapter can retain the decision provenance. It has no provisioning source, database table, route, worker integration, reservation behavior, or enabled feature flag. The [entitlement and quota policy contract](entitlement_quota_policy.md) records this boundary.

The Subscriber Project also now has a static least-privilege manifest at `internal/subscriber/worker` for future catalog, shared-EOD, Options-demand, and Options-capture machine principals. It assigns only narrow process scopes and excludes browser, tenant-administration, and unrelated subscriber data authority; it changes no current runner or credential. The [shared-worker identity contract](worker_identities.md) defines the required deployment, provenance, and fail-closed integration boundary.

The project has adopted a hybrid database isolation decision: current tenant-owned data remains application-authorized during compatibility, while every new tenant-private subscriber table must use fail-closed, forced PostgreSQL row-level security before it is enabled. Shared platform records remain separate and can only reach tenants through authorized projections. The [row-level security decision](row_level_security_decision.md) specifies the role, transaction, migration, and verification model.

When JWT enforcement is enabled, gateway startup now fails closed unless issuer, JWKS URL, and audience configuration is structurally valid. The non-mutating `scripts/signalops_oidc_preflight.sh` also verifies that the configured discovery document declares the same issuer and JWKS URL and that its JWKS has an RSA key ID. These checks prevent a partially configured production-like gateway from accepting traffic, but live OIDC client, browser-session, grant, and cross-tenant validation remain required before the Subscriber Project gate can exit.

Focused direct-API tests cover:

- omitted access-grant body tenant binding to the authenticated principal;
- rejection of a conflicting access-grant body tenant before a repository write; and
- omitted access-list query tenant binding to the authenticated principal;
- gateway rejection of a mismatched JSON body tenant before any downstream handler runs; and
- preservation of a valid inspected JSON body for its downstream handler.
- omitted administration-notification list and inbox-state tenants binding to the authenticated principal; and
- omitted notification-email settings read and update tenants binding to the authenticated principal.
- omitted MarketOps opportunity and disposition tenants binding to the authenticated principal; and
- authenticated disposition audit actor binding to the verified principal rather than the request body.
- authenticated replay-job create and list tenant binding, trusted requester identity, and foreign-record detail rejection.
- authenticated MarketOps backtest and calibration payload tenant binding across all mutation routes; and
- foreign promotion-candidate decision rejection before state mutation.
- authenticated algorithm definition, execution-request, proposal-decision, and materialization tenant binding; and
- omitted-tenant algorithm decision binding with a trusted reviewer identity.
- authenticated alert and insight lifecycle ownership checks before mutation; and
- foreign-alert lifecycle rejection without a state change.
- authenticated raw-event tenant injection into the broker payload and persisted ledger evidence.
- authenticated graph-proposal list binding and foreign detail/decision rejection; and
- authenticated graph-proposal decision tenant binding and trusted reviewer attribution.
- authenticated DSM artifact-list tenant binding and foreign artifact-detail rejection.
- authenticated Signal Assurance assertion/evaluation, effectiveness, observation, and recommendation tenant binding; and
- foreign Signal Assurance assertion-detail rejection.
- authenticated algorithm-evaluation run, result, outcome, and backfill read tenant binding; and
- foreign algorithm-evaluation run and backfill-campaign detail rejection.
- authenticated MarketOps outcome-list tenant binding and foreign outcome-detail rejection.
- authenticated hypothesis, feature, market-state, transition, and evidence-list tenant binding; and
- foreign market-state, lineage, and evidence-detail rejection.
- authenticated raw-event, normalized-event, and signal-ledger list tenant binding; and
- foreign core-ledger detail rejection.
- authenticated MarketOps backtest, calibration, promotion, readiness, evaluation, label, and per-run read-filter tenant binding; and
- foreign backtest campaign, run, calibration, promotion, readiness, evaluation, and label detail rejection.
- authenticated intelligence cohort, readiness, Syncratic context/insight, and algorithm registry read-filter tenant binding; and
- foreign intelligence cohort, Syncratic context-window, and Syncratic insight detail rejection.
- authenticated alert and insight-ledger list tenant binding; and
- foreign alert, insight, and idempotency provenance-detail rejection.
- authenticated Syncratic context-window and materialization tenant binding; and
- foreign Syncratic insight/Ask context rejection before persistence or an external Ask call.
- authenticated MarketOps asset display, watchlist, and backfill mutations binding to the principal tenant; and
- foreign tenant-path asset mutation rejection before a repository write.
- principal-bound subject resolution and subject-impersonation rejection; and
- tenant-default administration requiring the existing SignalOps administrator role.

Validation: `go test ./internal/api` passes.

## Safety boundary

This slice adds no migration, list or membership table, entitlement, provider request, scheduler, worker identity, browser behavior, or change to existing data ownership. The gateway continues to reject conflicting tenant values in request paths and queries, and now rejects declared top-level JSON body mismatches before route handling. Handler-specific binding remains required where an omitted tenant must inherit the principal rather than fail ordinary request validation.

## Formal deployment exit requirements

1. Add server-side subject ownership and tenant-administrator authorization to the future private and default list routes, using the implemented principal-bound subject and administrator guards.
2. Complete the entitlement path: RLS-scoped provisioning, atomic idempotent reservation, and durable provisioning/decision audit are implemented but have no enabled route or worker. Add workload-identity enforcement and route or worker integration with gateway-level negative tests.
3. Provision and enforce the static least-privilege service identities through workload credentials, gateway and persistence scope checks, audit, rotation, and negative integration tests.
4. Retain grant and future list-administration audit evidence alongside the entitlement and quota-decision evidence.
5. Complete the adopted tenant-private RLS model: the role bootstrap, role preflight, transaction-local tenant context, and first forced-RLS entitlement/quota tables are implemented and locally cross-tenant tested. Workload login credentials, production grants, and application-level negative integration tests remain required before any subscriber path can be enabled.
6. Run the discovery/JWKS preflight against the intended deployment, then validate live browser session and grants plus the complete cross-tenant negative-test suite; startup configuration shape and discovery consistency are fail-closed but are not proof of deployment readiness.

No Subscriber Project feature flag may enable catalog, list, shared-EOD, or Options-demand behavior until these exit conditions are satisfied. See [the S0-A exit checklist](s0a_exit_checklist.md) for the runnable evidence sequence and rollback.
