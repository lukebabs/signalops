#!/usr/bin/env bash
set -euo pipefail

job_id="${1:-}"
root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"

case "$job_id" in
  marketops-daily-postclose)
    exec ./scripts/marketops_scheduled_job.sh "$job_id" "Weekdays 18:01:55" America/New_York ./scripts/marketops_daily_postclose.sh --write
    ;;
  marketops-intraday)
    exec ./scripts/marketops_scheduled_job.sh "$job_id" "Weekdays every 15 minutes, 09:30-20:00" America/New_York ./scripts/marketops_intraday_monitor.sh
    ;;
  marketops-fmp-continuation)
    exec ./scripts/marketops_scheduled_job.sh "$job_id" "Daily 04:00" America/New_York ./scripts/marketops_fmp_continuation.sh
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
