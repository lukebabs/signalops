#!/usr/bin/env bash
set -euo pipefail

job_id="${1:-}"
root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"

case "$job_id" in
  preflight)
    source ./scripts/marketops_schedule_database.sh
    primary="$(marketops_primary_psql -Atc "select current_database()")"
    temporal="$(marketops_temporal_psql -Atc "select current_database()")"
    [[ "$primary" == "marketops" && "$temporal" == "marketops_temporal" ]] || {
      printf 'Dedicated scheduler preflight failed: primary=%s temporal=%s\n' "$primary" "$temporal" >&2
      exit 3
    }
    printf 'Dedicated scheduler preflight passed: primary=%s temporal=%s\n' "$primary" "$temporal"
    ;;
  marketops-daily-postclose)
    exec ./scripts/marketops_scheduled_job.sh "$job_id" "Weekdays 18:01:55" America/New_York ./scripts/marketops_daily_postclose.sh --write
    ;;
  marketops-warm-eod)
    export MARKETOPS_WARM_EOD_ACKNOWLEDGE_WRITES=true
    exec ./scripts/marketops_scheduled_job.sh "$job_id" "Weekdays 18:00" America/New_York ./scripts/marketops_warm_eod_refresh.sh --write
    ;;
  marketops-sri-refresh)
    exec ./scripts/marketops_scheduled_job.sh "$job_id" "Weekdays 20:07" America/New_York ./scripts/marketops_sri_refresh.sh
    ;;
  marketops-sri-holdings-refresh)
    exec ./scripts/marketops_scheduled_job.sh "$job_id" "Weekdays 20:20" America/New_York ./scripts/marketops_sri_holdings_refresh.sh
    ;;
  marketops-intraday)
    exec ./scripts/marketops_scheduled_job.sh "$job_id" "Weekdays every 15 minutes, 09:30-20:00" America/New_York ./scripts/marketops_intraday_monitor.sh
    ;;
  marketops-fmp-continuation)
    exec ./scripts/marketops_scheduled_job.sh "$job_id" "Daily 04:00" America/New_York ./scripts/marketops_fmp_continuation.sh
    ;;
  marketops-fmp-annual-financial)
    exec ./scripts/marketops_scheduled_job.sh "$job_id" "Saturday 02:30" America/New_York ./scripts/marketops_annual_financial_task_worker.sh
    ;;
  marketops-task-retry)
    exec ./scripts/marketops_scheduled_job.sh "$job_id" "Weekdays every 15 minutes, 18:30-23:00" America/New_York ./scripts/marketops_tactical_retry.sh
    ;;
  marketops-postclose-recovery)
    exec ./scripts/marketops_scheduled_job.sh "$job_id" "Weekdays every 15 minutes, 18:30-23:00" America/New_York ./scripts/marketops_postclose_recovery.sh
    ;;
  *)
    printf 'Unknown MarketOps scheduled job: %s\n' "$job_id" >&2
    exit 2
    ;;
esac
