# Sprint S0-A — Access-control Hardening

Status: in progress. This document records the implemented first slice and the remaining conditions required to exit the access-control readiness gate.

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

The authenticated gateway also now inspects a bounded, top-level JSON `tenant_id` on `POST`, `PUT`, `PATCH`, and `DELETE` requests before the handler runs. A conflicting declared tenant is rejected at the gateway, while a valid body is restored unchanged for the handler. This prevents the same body-tenant escalation across the remaining JSON mutation routes while their handler-level binding is audited.

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

Validation: `go test ./internal/api` passes.

## Safety boundary

This slice adds no migration, list or membership table, entitlement, provider request, scheduler, worker identity, browser behavior, or change to existing data ownership. The gateway continues to reject conflicting tenant values in request paths and queries, and now rejects declared top-level JSON body mismatches before route handling. Handler-specific binding remains required where an omitted tenant must inherit the principal rather than fail ordinary request validation.

## Remaining work before S0-A exit

1. Complete the tenant-bearing mutation-route audit: apply canonical principal binding where an omitted body tenant must inherit the principal, and add direct-API regression tests for each route class.
2. Add server-side subject ownership and tenant-administrator authorization for future private and default lists.
3. Implement entitlement and quota policy evaluation separately from MarketOps read/write grants.
4. Define least-privilege service identities for scheduled shared processing.
5. Retain grant, entitlement, list-administration, and quota-decision audit evidence.
6. Make and document the database row-level-security defense-in-depth decision.
7. Validate production-like OIDC/JWKS configuration and the complete cross-tenant negative-test suite.

No Subscriber Project feature flag may enable catalog, list, shared-EOD, or Options-demand behavior until these exit conditions are satisfied.
