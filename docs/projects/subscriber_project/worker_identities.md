# Subscriber Shared-Worker Identity Contract

Status: implemented static identity manifest; no current runner has been migrated, no service-account credential has been provisioned, and no worker behavior has changed.

## Purpose

Subscriber shared processing must run under dedicated machine principals. A worker must receive only the capability scopes required for its bounded job. It must not impersonate a subscriber browser session, receive a tenant administrator role, or use a broad database-owner credential.

The executable manifest is `internal/subscriber/worker`. It is deliberately static and side-effect free so it can be reviewed and tested before the global catalog, lists, coverage planner, or Options-demand scheduler is enabled.

## Identity manifest

| Identity | Permitted scope families | Explicitly excluded |
|---|---|---|
| `subscriber-catalog-reference-sync` | catalog eligibility read and catalog write | lists, entitlements, quotas, provider market-data pulls, browser/API administration |
| `subscriber-global-eod-reconciler` | global coverage read/write, EOD provider read, raw and normalized evidence write | memberships, entitlements, quotas, Options provider data, tenant administration |
| `subscriber-options-demand-planner` | entitlement read, membership snapshot read, quota reservation, decision audit write, demand-plan write | provider pulls, raw evidence, options-capture writes, tenant administration |
| `subscriber-options-capture` | approved demand-plan read, Options provider read, raw evidence and capture write | memberships, entitlements, quota reservation, decision audit, tenant administration |

The manifest returns immutable copies and has tests that prevent an Options capture worker from acquiring membership, entitlement, quota-reservation, or decision-audit authority. An unknown identity is denied by absence from the manifest.

## Deployment contract

Before a shared worker is enabled, its deployment must use a unique workload/service account and a short-lived credential whose subject and scopes match exactly one manifest identity. The worker must be restricted at the gateway and persistence boundary to those scopes, with an audience distinct from subscriber browser access where the identity provider supports it. Credentials must be injected by the deployment environment, never committed to the repository or placed in browser configuration.

Each worker execution must record identity, scope/provisioning version, run ID, correlation ID, input snapshot or plan version, and resulting provenance. Rotation, revocation, and a failed scope check must be observable and must fail closed.

A shared worker must use tenant information only from an authorized immutable planning snapshot; it may not accept a browser-supplied tenant, subject, entitlement, or membership payload. The options capture worker receives only selected global asset IDs from the authorized demand plan, so it cannot enumerate subscriber memberships.

## Activation boundary

This manifest alone does not grant a process access. Enabling a worker requires: service-account provisioning; OIDC or equivalent workload-token validation; gateway and database authorization enforcement; minimal secret delivery; scope-negative integration tests; audit retention; a rollback that disables the worker without deleting shared evidence; and an approved feature gate. Existing MarketOps scheduled jobs remain unchanged until that work is complete.
