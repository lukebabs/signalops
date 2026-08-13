# S4 Recovery Readiness Evidence — 2026-08-13

Status: recovery infrastructure partially provisioned; backup and provider execution remain disabled.

## Verified recovery destination

The approved recovery bucket is `signalops-production-postgres-backups-354918409279-us-east-1` in `us-east-1`. Read-only verification established:

| Control | Observed state |
| --- | --- |
| Public access | All four S3 public-access-block controls are enabled. |
| Default encryption | SSE-S3 (`AES256`); SSE-C uploads are blocked. |
| Object-version protection | Bucket versioning is enabled. |
| Backup role | `signalops-postgres-backup` exists and has a one-hour maximum session. |
| Role trust | Only `signalops-postgres-backup-runner` may assume the backup role. |
| Repository permissions | The role may list the dedicated bucket and read/write only its objects. |

The role is deliberately not granted application-database privileges. Its S3 object deletion permission is needed for governed backup-expiration/pruning; lifecycle and backup-tool retention policies must make that deletion bounded and auditable.

## Controls not yet provisioned

1. The bucket has no lifecycle policy. Retention therefore does not yet implement the approved full, differential/incremental, WAL, and monthly recovery-point policy.
2. `pgBackRest` is not installed alongside the PostgreSQL 16 deployment, and PostgreSQL has no configured `archive_mode`, `archive_command`, or verified WAL archival path.
3. No non-root workload credential/profile has been configured on the backup runner to assume `signalops-postgres-backup`. The available host identity is the AWS account root identity; it must not be used for backup operations.
4. Backup age, WAL-archive lag, repository capacity, credential/encryption failure, and restore-test-age monitoring have not been installed.
5. No isolated recovery environment or successful restore rehearsal exists.

## Required completion sequence

1. Configure the backup runner with a non-root programmatic credential that can only call `sts:AssumeRole` for `signalops-postgres-backup`; do not place it in the repository, `.env`, or an image.
2. Install and configure `pgBackRest` with client-side repository encryption and the dedicated bucket prefix. Store its encryption material and AWS credential source in the deployment secret manager.
3. Enable PostgreSQL WAL archiving through `pgBackRest`, take a full backup, and verify archive/checksum status.
4. Apply the owner-approved repository lifecycle and pgBackRest retention policy (planning defaults: 35 daily recovery points and 12 monthly recovery points).
5. Install monitoring for the recovery controls above and validate alert delivery.
6. Restore that backup into the network-isolated recovery target. Run the required identity, RLS, cross-tenant, membership-integrity, and zero-provider-call checks from the production backup/restore runbook.

Only after the resulting evidence is attached to the change record may the S4 execution worker be considered for its separate, named two-request authorization.

## Boundary

No S3 objects, IAM policies, PostgreSQL settings, scheduler settings, feature flags, or provider requests were changed while collecting this evidence. The current AWS caller is the account root identity and was used only for read-only inspection; it is explicitly excluded from the operating procedure.
