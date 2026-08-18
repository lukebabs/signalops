#!/usr/bin/env bash
set -euo pipefail

job_id="$1"; schedule="$2"; timezone="$3"; shift 3
root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=lib/marketops_trading_calendar.sh
source "$root_dir/scripts/lib/marketops_trading_calendar.sh"
status_dir="${SIGNALOPS_SCHEDULE_STATUS_DIR:-$(pwd)/runtime/scheduled-jobs}"
mkdir -p "$status_dir"
started_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
if marketops_is_weekend "$timezone" && ! marketops_weekend_permitted_job "$job_id"; then
  printf "{\"job_id\":\"%s\",\"schedule\":\"%s\",\"timezone\":\"%s\",\"status\":\"skipped\",\"reason\":\"non_trading_day\",\"started_at\":\"%s\",\"completed_at\":\"%s\",\"exit_code\":0}\n" "$job_id" "$schedule" "$timezone" "$started_at" "$started_at" > "$status_dir/.${job_id}.tmp"
  mv "$status_dir/.${job_id}.tmp" "$status_dir/${job_id}.json"
  printf "skipped non-trading-day job: %s\n" "$job_id"
  exit 0
fi

printf '{"job_id":"%s","schedule":"%s","timezone":"%s","status":"running","started_at":"%s"}\n' "$job_id" "$schedule" "$timezone" "$started_at" > "$status_dir/.${job_id}.tmp"
mv "$status_dir/.${job_id}.tmp" "$status_dir/${job_id}.json"

set +e
if [[ "${SIGNALOPS_MARKETOPS_DATA_PLANE_PREFLIGHT_REQUIRED:-false}" == "true" ]]; then
  "$root_dir/scripts/preflight_marketops_data_plane.sh"
  exit_code=$?
  if [[ "$exit_code" -eq 0 ]]; then
    "$@"
    exit_code=$?
  fi
else
  "$@"
  exit_code=$?
fi
set -e
completed_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
status="succeeded"; [[ "$exit_code" -eq 0 ]] || status="failed"
printf '{"job_id":"%s","schedule":"%s","timezone":"%s","status":"%s","started_at":"%s","completed_at":"%s","exit_code":%s}\n' "$job_id" "$schedule" "$timezone" "$status" "$started_at" "$completed_at" "$exit_code" > "$status_dir/.${job_id}.tmp"
mv "$status_dir/.${job_id}.tmp" "$status_dir/${job_id}.json"

# Governed daily/weekly completions and every job failure become administrator inbox
# events. Recorder failure is non-blocking so it cannot conceal the job result.
if [[ "$status" == "failed" || "$job_id" == "marketops-daily-postclose" || "$job_id" == "marketops-fmp-continuation" || "$job_id" == "marketops-fmp-annual-financial" || "$job_id" == "marketops-sri-refresh" || "$job_id" == "marketops-sri-holdings-refresh" || "$job_id" == "signalops-storage-monitor" || "$job_id" == "signalops-retention-governance" ]]; then
  set +e
  docker compose --profile administration-notifications run --rm administration-notification-recorder \
    --tenant-id "${SIGNALOPS_ADMIN_NOTIFICATION_TENANT_ID:-tenant-local}" \
    --job-id "$job_id" --status "$status" --schedule "$schedule" --timezone "$timezone" \
    --started-at "$started_at" --completed-at "$completed_at" --exit-code "$exit_code"
  notification_exit=$?
  set -e
  if [[ "$notification_exit" -ne 0 ]]; then
    printf 'administrator notification recorder failed for %s (exit %s)\n' "$job_id" "$notification_exit" >&2
  fi
fi

exit "$exit_code"
