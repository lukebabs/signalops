# N1 — Production Observability and Recovery Operations

Status: implementation prepared; installation and the first successful monitored run are controlled production actions.

## Purpose

N1 makes the dedicated MarketOps boundary observable without changing market-data collection, tenant authorization, or provider demand. It is the operational gate between a successful backup/restore bootstrap and an unattended service.

## Dedicated operations monitor

`scripts/marketops_operations_monitor.sh` is read-only with respect to market data and provider APIs. It evaluates both dedicated pgBackRest stanzas and writes a root-owned health record to `/var/lib/signalops/marketops-operations/health.json`.

It checks backup and WAL archive age, repository-growth threshold, credential and scheduler failure, restore-rehearsal age, and global coverage-activation queue age/depth. The defaults are 26 hours, 30 minutes, 100 GiB per stanza, 31 days, and 1,000 requests/24 hours respectively. An unavailable metric fails closed.

The systemd service runs through `marketops_scheduled_job.sh`. A non-zero result creates a durable administrator-inbox event for `marketops-operations-monitor`; configured administrator email delivery continues to use the existing inbox/email policy. The monitor never makes a provider call.

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
