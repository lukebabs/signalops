# Subscriber Project - Central Data, Business Continuity, and Disaster Recovery

Status: architecture and recovery contract. The S3 list tables are deployed in the existing SignalOps PostgreSQL database, and the local deployment-equivalent gateway exposes an API-only pilot for tenant-pilot-b. No subscriber-list UI or production rollout is enabled. This document defines the required continuity posture for the current API-only pilot and any future production rollout. It does not assert that a backup, replica, recovery-time objective, or recovery-point objective has already been provisioned.

## Purpose

MarketOps is one centrally operated data and algorithm plane, not a per-customer collection of copied databases. A subscriber's watchlist is a small, tenant-isolated preference projection over that central plane.

This design supports continuity by recovering one governed catalog and one set of shared evidence, then restoring tenant/user preferences and authorization. It avoids recovering a separate asset, price, signal, or algorithm-output copy for every subscriber.

## Centralized storage model

| Data class | System of record | Ownership and recovery meaning |
| --- | --- | --- |
| Global asset identity, source links, reference observations, eligibility, ranking snapshots, coverage plans, and activation queue | Existing SignalOps PostgreSQL database | Platform-owned shared records. Restore once; never replay from an individual tenant watchlist. |
| Prices, raw/normalized events, EOD evidence, algorithm outputs, signal assurance, and options captures | Existing SignalOps PostgreSQL database and established SignalOps persistence paths | Platform-owned shared evidence. Restore according to the primary platform recovery runbook and preserve immutable provenance. |
| Tenant-default lists, private lists, memberships, and list audit | Existing SignalOps PostgreSQL database, S3 forced-RLS tables | Tenant-private preferences. Restore after the shared catalog so each membership can resolve its global asset ID. |
| OIDC identity, tenant_id claim, and SignalOps access grants | Keycloak and the established access-grant store | Authorization control plane. Restore and validate before exposing subscriber projections. The list tables do not replace identity or access grants. |
| Provider credentials, database credentials, and signing/configuration secrets | Deployment secret-management system | Recover through the secret-management runbook. Never place credentials in database backup exports, audit records, or this repository. |
| Provider reference data and future provider pulls | External provider plus retained reference provenance | Re-fetch only through governed workers after recovery. A browser/user action must never trigger a recovery-time provider pull. |

The S3 tables contain only list metadata, immutable membership references, and audit provenance. They do not contain price histories, signal copies, algorithm results, or provider credentials.

## Dependency and recovery order

```text
Identity and secrets
        |
        v
SignalOps PostgreSQL control plane
        |
        +--> shared global catalog and evidence
        |          |
        |          v
        +--> tenant-private S3 lists and audits
                   |
                   v
Gateway authorization and feature flags
                   |
                   v
Subscriber pilot read/write routes (only when explicitly enabled)
```

A list membership is not an asset-onboarding record. It refers to a pre-existing global_asset_id. Therefore the shared catalog must be available before S3 list routes are re-enabled. If the global record has not recovered, the route returns a truthful unavailable/deferred state; it must not create a tenant-local substitute or invent coverage.

## Recovery principles

1. **Restore centrally, project selectively.** Recover the shared catalog and evidence one time, then restore tenant-private preferences under RLS.
2. **Fail closed.** Keep S3 feature flags off until identity, gateway access grants, database roles, RLS policies, and tenant-scope checks pass.
3. **Preserve provenance.** Do not rewrite source observations, ranking checksums, eligibility decisions, coverage plans, activation requests, or audit records to make a restore appear current.
4. **No duplicate collection.** Recovery does not permit per-tenant or per-user provider polling. Restart any future shared worker from its durable queue/checkpoint and idempotency key.
5. **No silent reconstruction.** A missing global asset or missing shared EOD evidence is reported as unavailable/deferred. It is not inferred from a list name, ticker string, or cached browser state.
6. **RLS remains active.** A restore must retain table ownership, revoked public grants, forced RLS, and policies. Restoring rows without those controls is not a successful subscriber recovery.
7. **Audit survives.** List and entitlement audits are business-continuity evidence, not disposable telemetry. Restore them with their tenant scope, actor, mutation, and correlation data.

## Required recovery runbook

Before enabling an S3 pilot, operations must own and rehearse a runbook with these stages:

1. Declare the incident, disable the subscriber-list feature flag, and stop any future list writer or activation worker.
2. Restore/validate identity and secret-management dependencies; verify the JWT issuer, signing keys, intended audience, tenant claim mapper, and access grants.
3. Restore the SignalOps PostgreSQL database from the approved backup/point-in-time recovery procedure.
4. Apply the exact schema migrations required by the restored release, including 000088 through 000095, before any gateway is allowed to use subscriber tables.
5. Verify subscriber roles and grants:
   - migration owner, gateway, catalog-sync, and global-EOD roles are non-superuser and do not bypass RLS;
   - the gateway has access only to tenant-private tables plus the minimum required key-reference privilege;
   - the gateway has no direct global catalog, plan, ranking, provider, or worker-table access.
6. Verify forced RLS under a non-owner gateway login:
   - no private row is visible with no tenant scope;
   - tenant A cannot list, read, mutate, or enumerate tenant B's lists or memberships;
   - connection reuse does not retain a prior tenant context.
7. Reconcile shared catalog integrity before list use:
   - every restored membership resolves to exactly one global asset ID;
   - no membership creates a duplicate asset;
   - no coverage plan has changed from shadow to enabled merely because of restore.
8. Re-enable the gateway with S3 still disabled. Enable a pilot tenant only after the application-level authorization tests and business owner approval pass.
9. Resume a future shared worker only through its documented, idempotent reconciliation path. Record backlog, retries, provider requests, failures, and reconciliation result.

## Validation evidence

A successful recovery must record:

- incident identifier, recovery operator, database backup/PITR reference, restore time, and deployed revision;
- schema migration and RLS-policy verification;
- gateway workload identity and grant verification;
- a cross-tenant negative test and a subject-ownership negative test;
- membership-to-global-asset reconciliation counts, including unresolved/deferred count;
- shared coverage/queue state before and after recovery;
- provider-call count during recovery; it should be zero unless an explicitly approved, governed worker reconciliation begins after the platform is stable; and
- the authority and timestamp approving each feature-flag re-enable.

## Objectives and retention

RTO, RPO, backup frequency, backup retention, cross-region replication, and the recovery environment are operational commitments that must be selected with platform ownership before a subscriber pilot is enabled. They are intentionally not fabricated here.

The pilot gate requires a documented target for each, an assigned owner, evidence that backups are encrypted and access-controlled, and a restore rehearsal against a non-production environment. A production list route must not be enabled merely because the schema exists.

## S3 rollback versus disaster recovery

A normal S3 rollback disables the feature flag and routes without deleting watchlists, memberships, audits, global catalog evidence, or coverage state. Disaster recovery restores the same central records from the approved backup/PITR path, validates isolation, and then selectively re-enables the feature.

Neither process deletes central evidence to make a tenant appear clean. Tenant data removal follows a separate, authorized retention/deletion process; it is not part of deployment rollback or incident recovery.
