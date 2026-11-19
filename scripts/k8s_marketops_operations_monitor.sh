#!/usr/bin/env bash
set -euo pipefail

: "${SIGNALOPS_MARKETOPS_DATABASE_URL:?dedicated MarketOps database URL is required}"

# A pod can disappear after writing `running`. Reconcile those leases before
# evaluating health so one lost worker cannot hold the monitor in a permanent
# failure state. The two-hour bound is longer than every production job SLA.
reconciled="$(psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -v ON_ERROR_STOP=1 -Atc "
WITH stale_status AS (
  UPDATE marketops_scheduled_job_statuses
     SET status='recovery_needed', reason='stale_run_timeout',
         completed_at=COALESCE(completed_at, now()), exit_code=42, updated_at=now()
   WHERE status='running' AND updated_at < now() - interval '2 hours'
   RETURNING job_id
), stale_runs AS (
  UPDATE marketops_scheduled_job_runs
     SET status='recovery_needed', reason='stale_run_timeout',
         completed_at=COALESCE(completed_at, now()), exit_code=42, updated_at=now()
   WHERE status='running' AND started_at < now() - interval '2 hours'
   RETURNING run_id
)
SELECT (SELECT count(*) FROM stale_status) || '/' || (SELECT count(*) FROM stale_runs)
")"

failed_jobs="$(psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -v ON_ERROR_STOP=1 -Atc \
  "SELECT count(*) FROM marketops_scheduled_job_statuses
   WHERE job_id <> 'marketops-operations-monitor'
     AND status='failed' AND updated_at >= now() - interval '24 hours'")"
stale_running="$(psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -v ON_ERROR_STOP=1 -Atc \
  "SELECT count(*) FROM marketops_scheduled_job_statuses
   WHERE status='running' AND updated_at < now() - interval '2 hours'")"
dry_run_mutating="$(psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -v ON_ERROR_STOP=1 -Atc \
  "SELECT count(*) FROM marketops_scheduled_job_statuses
   WHERE job_id IN ('marketops-daily-postclose','marketops-risk-reward','marketops-task-retry','marketops-warm-eod')
     AND status='succeeded' AND detail->>'dry_run'='true'
     AND updated_at >= now() - interval '24 hours'")"
stale_intraday="$(psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -v ON_ERROR_STOP=1 -Atc \
  "WITH market_clock AS (
         SELECT now() AT TIME ZONE 'America/New_York' AS local_now
       ), latest AS (
         SELECT max(as_of_time) AS latest
           FROM marketops_intraday_condition_snapshots
          WHERE tenant_id='tenant-local'
       )
   SELECT CASE WHEN extract(isodow FROM local_now) BETWEEN 1 AND 5
                    AND local_now::time >= time '09:30'
                    AND local_now::time <= time '16:15'
                    AND (latest IS NULL OR latest < now() - interval '45 minutes')
               THEN 1 ELSE 0 END
     FROM market_clock, latest")"

printf 'marketops_k8s_operations_monitor failed_jobs=%s stale_running=%s dry_run_mutating=%s stale_intraday=%s reconciled=%s\n' \
  "$failed_jobs" "$stale_running" "$dry_run_mutating" "$stale_intraday" "$reconciled"
[[ "$failed_jobs" == 0 && "$stale_running" == 0 && "$dry_run_mutating" == 0 && "$stale_intraday" == 0 ]]
