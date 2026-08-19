# N1 closure evidence — 2026-08-19

Status: recovery controls materially restored; operations monitor is blocked only by the N3 coverage-activation queue reconciliation decision.

## Closed in this cycle

- Installed the constrained deployment-agent path so approved operations no longer require repeated manual script execution.
- Installed and verified the root-owned Unix-socket bridge for Admin run-now execution.
- Verified `scheduler-status` through `sudo -n signalops-deploy-agent scheduler-status`.
- Verified Admin run-now through the live Gateway: `signalops-storage-monitor` returned `202 accepted` with runner `unix:/run/signalops/deployment-agent.sock`.
- Verified Gateway readiness at `/readyz`.
- Verified subscriber pilot browser smoke after hardening the auth navigation retry for transient Chromium `net::ERR_NETWORK_CHANGED`.

## Fresh evidence

- Gateway readiness returned `status=ready` at `2026-08-19T03:11:47Z`.
- `signalops-storage-monitor` run-now completed successfully at `2026-08-19T03:10:47Z`.
- Scheduler status showed active timers for intraday, post-close, post-close recovery, SRI refresh, SRI holdings refresh, and warm EOD.

## Remaining N1 failures

The operations monitor still fails truthfully. The 2026-08-19 03:11 UTC monitor run reported six actionable checks:

1. `backup_marketops-primary`: pgBackRest info unavailable.
2. `backup_marketops-temporal`: backup age above the 26-hour threshold.
3. `wal_primary`: archived WAL age above the 30-minute threshold.
4. `wal_temporal`: archived WAL age above the 30-minute threshold.
5. `restore_rehearsal`: no durable restore-rehearsal evidence stamp.
6. `coverage_activation_queue`: two activation requests older than the 24-hour threshold.

## Root cause identified

The installed `signalops-marketops-pgbackrest.service` still points at an old temporary release directory:

```text
/tmp/signalops-marketops-recovery-release/scripts/marketops_pgbackrest_backup.sh
```

That stale backup path now executes against MarketOps database containers that do not have `pgbackrest` in `$PATH`, causing:

```text
OCI runtime exec failed: exec failed: unable to start container process: exec: "pgbackrest": executable file not found in $PATH
```

This is a deployment-control drift, not a provider-data failure. The dedicated MarketOps databases remain reachable and the scheduler timers are active.

## Required approval boundary

Closing the remaining N1 recovery loop requires an explicitly approved production recovery-control change because the fix may rebuild/recreate the dedicated MarketOps database containers under the pgBackRest image overlay and reinstall the root-owned systemd backup unit from the current repository path.

The safe target behavior is:

1. re-anchor `signalops-marketops-pgbackrest.service` to the current repository path;
2. ensure backup and restore rehearsal run against MarketOps database containers that include pgBackRest and the mounted root-owned config;
3. run one controlled backup;
4. run one isolated restore rehearsal;
5. rerun the operations monitor;
6. leave the two stale coverage-activation requests for the N3 catalog-activation queue closure unless separately approved.

No provider polling, tenant-data mutation, or database deletion is implied by this recovery-control change.


## Recovery-control execution update — 2026-08-19 03:31 UTC

After the approved recovery-control fix:

- the current repo scripts now reassert the pgBackRest Compose overlay before backup and restore rehearsal;
- isolated restore rehearsal passed for both `marketops-primary` and `marketops-temporal`;
- controlled pgBackRest backup passed for both stanzas;
- the operations monitor reduced from six actionable failures to one.

Latest monitor pass evidence:

- `backup_marketops-primary`: passed, age 109 seconds;
- `repository_marketops-primary`: passed, 308,805,056 bytes;
- `backup_marketops-temporal`: passed, age 21 seconds;
- `repository_marketops-temporal`: passed, 154,393,056 bytes;
- `wal_primary`: passed, age 109 seconds;
- `wal_temporal`: passed, age 21 seconds;
- `credentials`: passed;
- `scheduler_signalops-marketops-pgbackrest.service`: passed;
- `restore_rehearsal`: passed, age 156 seconds.

Remaining monitor failure:

- `coverage_activation_queue`: failed because two pilot activation requests are still `queued` and older than 24 hours.

Read-only inspection showed those two rows are AAPL and NVDA requests from `tenant-pilot-b` private list `sublist_73a0087473df782f499b51e9`. Both underlying global assets already have active `eod_baseline` coverage and current global evidence. Closing this last monitor failure requires a separately approved data-state reconciliation that marks activation requests active when their corresponding global EOD coverage is already active. That is an N3 catalog-activation queue correction, not a provider call or backup operation.

Residual hardening note: the live `signalops-marketops-pgbackrest.service` still shows the old `/tmp/signalops-marketops-recovery-release` path until the deployment agent is reprovisioned or the systemd unit is reinstalled from the current repo. The backup succeeded after the pgBackRest overlay was reasserted, but the root-owned unit should still be re-anchored for permanence.
