# MarketOps scheduled-job status database migration — 2026-08-19

Status: deployed and verified.

## Decision

MarketOps scheduled-job operational status is stored in the dedicated
MarketOps primary database. Repository-local runtime JSON is no longer the
operational record; it remains only an ignored fallback/debug artifact.

This keeps scheduler state aligned with the MarketOps data boundary. The
application, Admin workbench, and scheduled-job wrappers now interact with the
same dedicated MarketOps database instead of deriving production truth from
files under `runtime/scheduled-jobs/`.

## Implemented boundary

- Migration `000154_marketops_scheduled_job_statuses` creates:
  - `marketops_scheduled_job_statuses` for the latest status of each scheduled
    job.
  - `marketops_scheduled_job_runs` for run-level history.
- The Gateway reads scheduled-job status from the MarketOps query repository
  first.
- If the MarketOps status table is unavailable, the Gateway falls back to the
  ignored runtime JSON artifacts to preserve operator visibility during a
  database outage.
- Scheduled wrappers write `running`, terminal success/failure/skipped, and
  recovery-detail states to the database before writing local fallback JSON.
- The systemd units for MarketOps operations monitor, storage monitor, and
  retention governance load the protected MarketOps cutover environment so
  status writes resolve to the dedicated `marketops` database.
- The Admin System workbench marks status-only jobs as not runnable through
  Run Now.

## Production evidence

- Live migration applied to `signalops-marketops-postgres-1`.
- Prior runtime status was seeded into the database:
  - 13 latest status rows;
  - 11 historical run rows.
- `marketops-operations-monitor` wrote a fresh database-backed success row:
  - started: `2026-08-19 04:16:18 UTC`;
  - completed: `2026-08-19 04:16:20 UTC`;
  - runner: `systemd`;
  - exit code: `0`.
- Gateway deployed from commit `5b0871f` and verified healthy.
- Subscriber pilot UI smoke passed after the web rebuild: `2 passed`.

## Verification performed

```text
bash -n scripts/marketops_scheduled_job.sh   scripts/marketops_schedule_database.sh   scripts/marketops_postclose_recovery.sh   deploy/deployment-agent/signalops-deploy-agent

go test ./internal/api ./internal/storage/postgres
npm --prefix web run build
git diff --check
sudo -n signalops-deploy-agent operations-monitor-run
sudo -n signalops-deploy-agent subscriber-pilot-ui-smoke
```

Database verification confirmed status rows for:

- `marketops-daily-postclose`;
- `marketops-intraday`;
- `marketops-operations-monitor`;
- `signalops-storage-monitor`.

## Operator contract

For production operation, the source of truth is:

```text
marketops.marketops_scheduled_job_statuses
marketops.marketops_scheduled_job_runs
```

The local path below is not a production data store:

```text
runtime/scheduled-jobs/
```

If the Admin workbench shows stale scheduler status, first verify the
MarketOps database row and the installed systemd unit environment. Do not
commit runtime JSON artifacts to the repository.
