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
exit "$exit_code"
