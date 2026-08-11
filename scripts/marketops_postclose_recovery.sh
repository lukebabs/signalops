#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

usage() {
  printf '%s\n' 'Usage: scripts/marketops_postclose_recovery.sh [--date YYYY-MM-DD]'
}

session_date=""
while (($# > 0)); do
  case "$1" in
    --date) session_date="$2"; shift 2 ;;
    --help|-h) usage; exit 0 ;;
    *) printf 'unknown argument: %s\n' "$1" >&2; usage >&2; exit 2 ;;
  esac
done

if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  . ./.env
  set +a
fi

timezone="${MARKETOPS_DAILY_TIMEZONE:-America/New_York}"
max_attempts="${MARKETOPS_POSTCLOSE_RECOVERY_MAX_ATTEMPTS:-2}"
state_dir="${MARKETOPS_POSTCLOSE_RECOVERY_STATE_DIR:-$ROOT_DIR/runtime/postclose-recovery}"
status_dir="${SIGNALOPS_SCHEDULE_STATUS_DIR:-$ROOT_DIR/runtime/scheduled-jobs}"
daily_lock="${MARKETOPS_DAILY_LOCK_FILE:-/tmp/signalops-marketops-daily.lock}"
recovery_lock="${MARKETOPS_POSTCLOSE_RECOVERY_LOCK_FILE:-/tmp/signalops-marketops-postclose-recovery.lock}"

[[ "$max_attempts" =~ ^[1-9][0-9]*$ ]] || { printf 'MARKETOPS_POSTCLOSE_RECOVERY_MAX_ATTEMPTS must be a positive integer\n' >&2; exit 2; }

if [[ -z "$session_date" ]]; then
  session_date="$(TZ="$timezone" date '+%F')"
  if [[ "$(TZ="$timezone" date '+%H%M%S')" -lt 180000 ]]; then
    session_date="$(TZ="$timezone" date -d "$session_date -1 day" '+%F')"
  fi
fi
[[ "$session_date" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]] || { printf 'invalid session date: %s\n' "$session_date" >&2; exit 2; }
[[ "$(date -u -d "$session_date" '+%F' 2>/dev/null)" == "$session_date" ]] || { printf 'invalid session date: %s\n' "$session_date" >&2; exit 2; }
(( $(TZ="$timezone" date -d "$session_date" '+%u') <= 5 )) || { printf 'session date must be a weekday: %s\n' "$session_date" >&2; exit 2; }

for command in docker flock date mkdir mv; do
  command -v "$command" >/dev/null 2>&1 || { printf 'required command not found: %s\n' "$command" >&2; exit 2; }
done

mkdir -p "$state_dir" "$status_dir"
exec 9>"$recovery_lock"
flock -n 9 || { printf 'post-close recovery already running\n'; exit 0; }

compact() { tr -d '[:space:]'; }
active_symbols="$(docker compose exec -T postgres psql -U signalops -d signalops -Atc "SELECT string_agg(ticker, ',' ORDER BY universe_priority, rank) FROM (SELECT DISTINCT ON (ticker) ticker, universe_priority, rank FROM marketops_universal_assets WHERE tenant_id='tenant-local' AND is_active ORDER BY ticker, universe_priority, rank) canonical;" | compact)"
[[ -n "$active_symbols" ]] || { printf 'active equity universe is empty\n' >&2; exit 4; }
expected="$(docker compose exec -T postgres psql -U signalops -d signalops -Atc "SELECT count(DISTINCT ticker) FROM marketops_universal_assets WHERE tenant_id='tenant-local' AND is_active;" | compact)"
[[ "$expected" =~ ^[1-9][0-9]*$ ]] || { printf 'invalid active universe count: %s\n' "$expected" >&2; exit 4; }

risk_counts="$(docker compose exec -T postgres psql -U signalops -d signalops -Atc "SELECT (SELECT count(DISTINCT result_payload->>'symbol') FROM algorithm_results WHERE tenant_id='tenant-local' AND algorithm_id='signalops.algorithms.risk_reward_temporal_v1' AND (result_payload->>'observation_time')::date=DATE '$session_date') || '|' || (SELECT count(DISTINCT symbol) FROM marketops_risk_reward_snapshots WHERE tenant_id='tenant-local' AND session_date=DATE '$session_date');" | compact)"
IFS='|' read -r risk_results risk_snapshots <<< "$risk_counts"
risk_results="${risk_results:-0}"
risk_snapshots="${risk_snapshots:-0}"

write_risk_status() {
  local status="$1" detail="$2" now temp
  now="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
  temp="$status_dir/.marketops-risk-reward.tmp"
  printf '{"job_id":"marketops-risk-reward","schedule":"Post-close stage; completion-guarded","timezone":"%s","status":"%s","session_date":"%s","risk_reward_results":%s,"risk_reward_snapshots":%s,"expected_symbols":%s,"detail":"%s","updated_at":"%s"}\n' "$timezone" "$status" "$session_date" "$risk_results" "$risk_snapshots" "$expected" "$detail" "$now" > "$temp"
  mv "$temp" "$status_dir/marketops-risk-reward.json"
}

if completion_output="$(./scripts/marketops_universal_completion_gate.sh "$session_date" "$active_symbols" "$expected" 2>&1)"; then
  write_risk_status "succeeded" "universal completion gate passed"
  rm -f "$state_dir/$session_date.attempts"
  printf '%s\n' "$completion_output"
  printf 'post-close recovery: session %s is complete; no retry required\n' "$session_date"
  exit 0
fi

printf '%s\n' "$completion_output" >&2
write_risk_status "recovery_needed" "universal completion gate incomplete"

if command -v systemctl >/dev/null 2>&1 && systemctl --user is-active --quiet signalops-marketops-daily.service; then
  printf 'post-close recovery: primary post-close service is active; deferring retry\n'
  exit 0
fi

if flock -n "$daily_lock" -c true; then
  :
else
  printf 'post-close recovery: primary workflow lock is held; deferring retry\n'
  exit 0
fi

attempt_file="$state_dir/$session_date.attempts"
attempts=0
if [[ -f "$attempt_file" ]]; then
  attempts="$(<"$attempt_file")"
fi
[[ "$attempts" =~ ^[0-9]+$ ]] || { printf 'invalid recovery attempt state for %s\n' "$session_date" >&2; exit 5; }
if (( attempts >= max_attempts )); then
  write_risk_status "failed" "recovery attempt budget exhausted"
  printf 'post-close recovery: attempt budget exhausted for %s (%s attempts)\n' "$session_date" "$attempts" >&2
  exit 6
fi

attempts=$((attempts + 1))
temp_attempt="$attempt_file.tmp"
printf '%s\n' "$attempts" > "$temp_attempt"
mv "$temp_attempt" "$attempt_file"
write_risk_status "recovering" "starting bounded recovery attempt $attempts of $max_attempts"
printf 'post-close recovery: starting attempt %s of %s for %s\n' "$attempts" "$max_attempts" "$session_date"

export MARKETOPS_DAILY_ACKNOWLEDGE_WRITES=true
bash ./scripts/marketops_scheduled_job.sh marketops-daily-postclose "Weekdays 18:01:55" "$timezone" ./scripts/marketops_daily_postclose.sh --date "$session_date" --write

risk_counts="$(docker compose exec -T postgres psql -U signalops -d signalops -Atc "SELECT (SELECT count(DISTINCT result_payload->>'symbol') FROM algorithm_results WHERE tenant_id='tenant-local' AND algorithm_id='signalops.algorithms.risk_reward_temporal_v1' AND (result_payload->>'observation_time')::date=DATE '$session_date') || '|' || (SELECT count(DISTINCT symbol) FROM marketops_risk_reward_snapshots WHERE tenant_id='tenant-local' AND session_date=DATE '$session_date');" | compact)"
IFS='|' read -r risk_results risk_snapshots <<< "$risk_counts"
risk_results="${risk_results:-0}"
risk_snapshots="${risk_snapshots:-0}"
if completion_output="$(./scripts/marketops_universal_completion_gate.sh "$session_date" "$active_symbols" "$expected" 2>&1)"; then
  write_risk_status "succeeded" "universal completion gate passed after recovery"
  rm -f "$attempt_file"
  printf '%s\n' "$completion_output"
  exit 0
fi

printf '%s\n' "$completion_output" >&2
write_risk_status "failed" "recovery completed without universal completion"
exit 7
