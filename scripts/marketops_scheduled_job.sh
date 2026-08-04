#!/usr/bin/env bash
set -euo pipefail

job_id="$1"; schedule="$2"; timezone="$3"; shift 3
status_dir="${SIGNALOPS_SCHEDULE_STATUS_DIR:-$(pwd)/runtime/scheduled-jobs}"
mkdir -p "$status_dir"
started_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
printf '{"job_id":"%s","schedule":"%s","timezone":"%s","status":"running","started_at":"%s"}\n' "$job_id" "$schedule" "$timezone" "$started_at" > "$status_dir/.${job_id}.tmp"
mv "$status_dir/.${job_id}.tmp" "$status_dir/${job_id}.json"

set +e
"$@"
exit_code=$?
set -e
completed_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
status="succeeded"; [[ "$exit_code" -eq 0 ]] || status="failed"
printf '{"job_id":"%s","schedule":"%s","timezone":"%s","status":"%s","started_at":"%s","completed_at":"%s","exit_code":%s}\n' "$job_id" "$schedule" "$timezone" "$status" "$started_at" "$completed_at" "$exit_code" > "$status_dir/.${job_id}.tmp"
mv "$status_dir/.${job_id}.tmp" "$status_dir/${job_id}.json"

# Governed daily/weekly completions and every job failure become administrator inbox
# events. Recorder failure is non-blocking so it cannot conceal the job result.
if [[ "$status" == "failed" || "$job_id" == "marketops-daily-postclose" || "$job_id" == "marketops-fmp-continuation" || "$job_id" == "signalops-storage-monitor" || "$job_id" == "signalops-retention-governance" ]]; then
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
