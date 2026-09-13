# N1 — Production Observability and Recovery Operations

Status: implementation deployed for N1 closure. Monitor health, recovery-control re-anchoring, and DB-backed scheduled-job status have production evidence.

## Purpose

N1 makes the dedicated MarketOps boundary observable without changing market-data collection, tenant authorization, or provider demand. It is the operational gate between a successful backup/restore bootstrap and an unattended service.

## Dedicated operations monitor

`scripts/marketops_operations_monitor.sh` is read-only with respect to market data and provider APIs. It evaluates both dedicated pgBackRest stanzas and writes a root-owned health record to `/var/lib/signalops/marketops-operations/health.json`.

It checks backup and WAL archive age, repository-growth threshold, credential and scheduler failure, restore-rehearsal age, and global coverage-activation queue age/depth. The defaults are 26 hours, 30 minutes, 100 GiB per stanza, 31 days, and 1,000 requests/24 hours respectively. An unavailable metric fails closed.

The systemd service runs through `marketops_scheduled_job.sh`. A non-zero result creates a durable administrator-inbox event for `marketops-operations-monitor`; configured administrator email delivery continues to use the existing inbox/email policy. The monitor never makes a provider call.

Scheduled-job status is persisted to the dedicated MarketOps primary database through `marketops_scheduled_job_statuses` and `marketops_scheduled_job_runs`. Local JSON under `runtime/scheduled-jobs/` is ignored fallback/debug output only. The Admin System workbench must treat the MarketOps database as the source of truth for operational scheduler status.

Install it disabled, then run and review it manually:

```bash
sudo ./scripts/install_marketops_operations_monitor.sh
sudo systemctl start signalops-marketops-operations-monitor.service
sudo systemctl status signalops-marketops-operations-monitor.service --no-pager -l
```

Only enable the hourly timer after the health record and first alert exercise are reviewed:

```bash
sudo ./scripts/install_marketops_operations_monitor.sh --enable
```

## Restore evidence and cadence

After a full successful isolated rehearsal, `marketops_pgbackrest_restore_rehearsal.sh` records a root-owned completion stamp at `/var/lib/signalops/marketops-operations/restore-rehearsal.json`. The monitor fails if the evidence is older than 31 days. It never initiates a rehearsal automatically.

## Host directory-watch remediation

The recurring `Failed to allocate directory watch: Too many open files` warning is a host inotify/file-descriptor capacity condition, not a successful scheduler result. `scripts/provision_signalops_systemd_watch_limits.sh` persists bounded inotify limits and `DefaultLimitNOFILE=1048576`; it deliberately leaves `systemctl daemon-reexec` for an approved maintenance window. Verify the warning is absent after that re-exec and a scheduler/backup run.

## S3 and secret continuity

S3 is a recovery-artifact repository, not the operational data store. Before N1 exit, record a bucket lifecycle policy compatible with 12 full and 35 differential recovery points and retain all required WAL. Also record owners and protected recovery procedures for Keycloak realm/client/mapper configuration and deployment-secret references. Never place raw tokens, database passwords, cipher material, or bootstrap credentials in source control or browser artifacts.

## N1 exit evidence

1. A successful manual monitor run and administrator-notification exercise.
2. One scheduled backup and one scheduled monitor run with timestamps and deployed revision.
3. A restore stamp newer than 31 days.
4. Documented S3 lifecycle and Keycloak/deployment-secret recovery ownership.
5. No host watch-limit warning after approved remediation.

## First controlled monitor evidence — 2026-08-14

The monitor was installed disabled, then invoked manually through the scoped deployment-control agent. Its first run proved the backup, repository, credential, primary-WAL, and dedicated-intraday scheduler controls. It also created the expected durable administrator-inbox warning notification (`scheduler:marketops-operations-monitor:failed`).

The first result failed closed on three conditions: temporal WAL was 20,438 seconds old, no durable restore-rehearsal stamp existed (the prior successful rehearsal predated stamp creation), and two global activation requests were older than the 24-hour queue threshold. A provider-free pgBackRest check immediately archived temporal WAL and reduced that check to 8 seconds. The controlled rerun therefore left exactly two active failures: the missing fresh rehearsal stamp and the aged activation queue. Those requests remain intact for N3; they were not suppressed, deleted, or relabeled.

The bounded inotify/file-limit configuration was persisted and applied. The monitor timer is enabled hourly because the alert path has been verified; it is expected to remain warning-state until the two truthful failures are resolved. `systemctl daemon-reexec`, a post-reexec scheduler/backup verification, and the actual removal of the host warning remain an approved-maintenance action.

## Watch-limit remediation evidence — 2026-08-14

The bounded inotify settings were applied and systemd was re-executed under explicit approval. The subsequent dedicated scheduler preflight passed with `primary=marketops` and `temporal=marketops_temporal`; its post-reexec journal contains no `Failed to allocate directory watch` warning. A later scheduled backup remains the final independent cadence confirmation.


## DB-backed scheduler status evidence — 2026-08-19

Migration `000154_marketops_scheduled_job_statuses` was applied to the
dedicated MarketOps primary database. Existing ignored runtime artifacts were
seeded into the new tables, and `marketops-operations-monitor` subsequently
wrote a fresh success row directly from the installed systemd wrapper.

This closes the prior repository/runtime-file ambiguity: production scheduler
status is database-backed, while file artifacts remain local fallback/debug
evidence during a database outage.
