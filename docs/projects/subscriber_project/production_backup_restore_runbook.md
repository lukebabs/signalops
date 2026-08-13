# Subscriber Project Production Backup and Restore Runbook

Status: required production gate; not yet provisioned.

## Purpose and boundary

SignalOps PostgreSQL is the live system of record for the central asset catalog, MarketOps evidence, Subscriber entitlements, watchlists, memberships, and audit rows. The Subscriber S3 feature does **not** keep live application data in object storage.

Production backup tooling may write encrypted PostgreSQL backup archives and WAL segments to an approved object-storage bucket. That bucket is a recovery destination only: it is not a live database, a browser data source, or a replacement for PostgreSQL. Restoring only Subscriber tables is prohibited because memberships must resolve against the recovered central catalog.

## Procurement decision record

Record these values in the approved production change record before installation:

| Decision | Required value |
| --- | --- |
| Backup tool | `pgBackRest`, or an approved equivalent with physical PostgreSQL backup and PITR support |
| Archive destination | Dedicated encrypted object-storage bucket, separate from application runtime storage |
| Access | Least-privilege backup service identity; no public access; restore access limited to recovery operators |
| RPO | Maximum accepted data-loss interval, including WAL archival lag |
| RTO | Maximum accepted time from incident declaration to validated restoration |
| Retention | Full, differential/incremental, WAL, and monthly retention periods |
| Recovery environment | Isolated non-production target, network-separated from production |
| Owners | Operations owner, backup delegate, security reviewer, and re-enable approver |
| Monitoring | Backup completion/age, WAL archival lag, archive capacity, access/encryption failure, and restore-test age |

Recommended planning defaults—subject to owner approval—are a 15-minute RPO, 4-hour RTO, 35 daily recovery points, and 12 monthly recovery points.

## Backup installation and operating procedure

1. Create the dedicated backup bucket and backup service identity. Require encryption and deny public access.
2. Install and configure `pgBackRest` alongside production PostgreSQL. Put repository credentials and encryption material in the deployment secret manager only; never commit them or add them to `.env`.
3. Configure PostgreSQL WAL archiving through the backup tool. Alert on archival failure or lag beyond the approved RPO.
4. Initialize the repository and take the first full backup. Record its label, database-cluster identity, PostgreSQL version, configuration revision, and verification result.
5. Schedule full plus differential/incremental backups and continuous WAL archival. The schedule must meet the RPO without relying on a workstation or source checkout.
6. Monitor every backup and WAL result. Alert on failed/missing recovery points, repository capacity threshold, or credential/encryption failure.
7. Back up Keycloak realm/client configuration and retain a separate controlled recovery path for deployment secrets; the database backup does not restore either system.
8. Restore to the isolated environment and record the evidence before production Subscriber enablement; repeat on the approved cadence.

Copying a Docker named volume is not an accepted production backup. It does not provide a consistent PostgreSQL recovery point, WAL continuity, off-host durability, encryption, or a tested restore path.

## Incident restore procedure

1. Declare the incident and record its identifier, recovery operator, authority, target recovery timestamp/backup label, and deployed revision.
2. Disable `SIGNALOPS_SUBSCRIBER_LISTS_ENABLED` and stop any future list writer or activation worker. Do not delete tenant preferences, audits, global assets, coverage state, or evidence.
3. Recover identity-provider reachability and deployment secrets. Validate issuer, signing keys, audience, tenant claim mapper, and SignalOps grants before exposing authenticated routes.
4. Restore PostgreSQL to the selected backup or PITR target in the isolated recovery environment. Verify cluster identity, PostgreSQL version compatibility, completion, and backup checksums.
5. Apply the exact migrations for the selected release. The current S3 release requires `000088` through `000096`; do not apply newer migrations merely because they are present in a source checkout.
6. Restore/recreate Subscriber roles and grants using the controlled deployment procedure. Confirm the migration, gateway, catalog-sync, and global-EOD identities are non-superuser and `NOBYPASSRLS`.
7. Run `scripts/subscriber_project_rls_preflight.sh` using the privileged migration connection, then `scripts/subscriber_project_gateway_workload_preflight.sh` using the secret-managed dedicated gateway connection. Attach unedited output to the incident record.
8. Reconcile central integrity: every membership resolves to one global asset ID, no duplicate global asset is created, coverage remains unchanged, and unresolved records remain deferred/unavailable.
9. Deploy the selected clean revision with Subscriber lists still disabled. Validate gateway/web health, tenant binding, subject-ownership denial, and cross-tenant denial.
10. Run `scripts/subscriber_project_s3_pilot_preflight.sh --tenant-id <approved-tenant>` while disabled. Re-enable only the approved tenant after business-owner approval is recorded.
11. Resume shared workers only through governed idempotent reconciliation. Record backlog, retries, provider-request count, and outcome. Provider calls during restoration are zero unless a separate recovery action is approved.

## Restore acceptance evidence

Attach the backup/PITR reference, archive verification, restore timestamps, recovery environment, deployed revision, applied migrations, identity/secrets validation result, RLS preflight output, cross-tenant and subject-ownership negative tests, membership reconciliation counts, feature-flag approvals, and provider-call count to the incident/change record. Do not include credentials or token material.

## Completion criteria

The production S3 pilot remains unavailable until the procurement record is approved, the first encrypted off-host backup completes, WAL archival meets the chosen RPO, monitoring is active, and an isolated restore rehearsal has produced the acceptance evidence above.
