# SignalOps Persistent Storage Monitoring

The Administration → Storage workbench monitors SignalOps-owned persistent volumes only: primary PostgreSQL, TimescaleDB, and Redpanda. It deliberately excludes Docker images, writable container layers, engine logs, and unrelated host volumes.

`signalops-storage-monitor.timer` runs every fifteen minutes for capacity health. It captures the component inventory once per America/New_York calendar day at 02:00 ET; the daily inventory is retained for ninety days. Its collector mounts only the three named volumes read-only and writes one durable snapshot per store to `storage_monitor_snapshots`. It has no Docker socket or writable volume mount.

Usage is measured from regular files in each volume. Capacity and free space are the backing filesystem values, so installations whose three volumes share a filesystem display the same capacity; this is shared capacity, not three independent quotas. Status is `healthy` below 75% used, `warning` from 75% through less than 90%, and `critical` at 90% or higher. An unreadable volume is `unavailable` and carries the collection error.

Install or refresh the user units with `scripts/install_marketops_daily_user_timer.sh`, then verify `systemctl --user list-timers --all | rg storage-monitor`. The Admin Scheduled Jobs table reports the collector result through the normal runtime status file.
