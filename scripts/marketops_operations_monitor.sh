#!/usr/bin/env bash
set -euo pipefail

# Operational health control for the dedicated MarketOps boundary.
# A non-zero result is intentionally consumed by marketops_scheduled_job.sh,
# which creates an actionable administrator-inbox notification. The WAL check
# performs a bounded pg_switch_wal() probe so low-write periods do not look
# stale while recovery-point archiving is actually functional.
root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
config_path="${SIGNALOPS_PGBACKREST_CONFIG_PATH:-/etc/signalops/marketops-pgbackrest/pgbackrest.conf}"
boundary_env="${SIGNALOPS_MARKETOPS_BOUNDARY_ENV:-/etc/signalops/marketops-boundary.env}"
state_dir="${SIGNALOPS_MARKETOPS_OPERATIONS_STATE_DIR:-/var/lib/signalops/marketops-operations}"
max_backup_age="${MARKETOPS_OPERATIONS_MAX_BACKUP_AGE_SECONDS:-93600}"
max_wal_age="${MARKETOPS_OPERATIONS_MAX_WAL_AGE_SECONDS:-1800}"
max_restore_age="${MARKETOPS_OPERATIONS_MAX_RESTORE_AGE_SECONDS:-2678400}"
max_repository_bytes="${MARKETOPS_OPERATIONS_MAX_REPOSITORY_BYTES:-107374182400}"
max_activation_queue="${MARKETOPS_OPERATIONS_MAX_ACTIVATION_QUEUE:-1000}"
max_activation_age="${MARKETOPS_OPERATIONS_MAX_ACTIVATION_AGE_SECONDS:-86400}"
require_sri="${MARKETOPS_OPERATIONS_REQUIRE_SRI_TIMER:-false}"

for value in "$max_backup_age" "$max_wal_age" "$max_restore_age" "$max_repository_bytes" "$max_activation_queue" "$max_activation_age"; do
  [[ "$value" =~ ^[0-9]+$ ]] || { printf 'Operational monitor thresholds must be non-negative integers.\n' >&2; exit 2; }
done
[[ -r "$config_path" && -r "$boundary_env" ]] || {
  printf 'Dedicated pgBackRest configuration or MarketOps boundary secret is not readable.\n' >&2
  exit 3
}

compose=(docker compose -p signalops --env-file "$boundary_env" -f "$root_dir/compose.yaml" -f "$root_dir/compose.marketops-boundary.yaml" -f "$root_dir/compose.marketops-pgbackrest.yaml")
now_epoch="$(date -u +%s)"
observed_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
failures=0
results=()

escape_json() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//$'\n'/ }"
  printf '%s' "$value"
}

record() {
  local name="$1" state="$2" detail="$3"
  results+=("{\"name\":\"$(escape_json "$name")\",\"state\":\"$(escape_json "$state")\",\"detail\":\"$(escape_json "$detail")\"}")
  printf '%s %s %s\n' "$state" "$name" "$detail"
  [[ "$state" != "failed" ]] || failures=$((failures + 1))
}

pg_info() {
  local service="$1" stanza="$2"
  SIGNALOPS_PGBACKREST_CONFIG_PATH="$config_path" "${compose[@]}" exec -T --user postgres "$service" \
    pgbackrest --stanza="$stanza" info --output=json
}

primary_sql() {
  "${compose[@]}" exec -T marketops-postgres psql -U signalops -d marketops -Atqc "$1"
}

temporal_sql() {
  "${compose[@]}" exec -T marketops-timescaledb psql -U signalops -d marketops_temporal -Atqc "$1"
}

check_backup() {
  local service="$1" stanza="$2" info stop size age
  if ! info="$(pg_info "$service" "$stanza")"; then
    record "backup_${stanza}" failed "pgBackRest info unavailable"
    return
  fi
  stop="$(printf '%s' "$info" | grep -oE '"stop":[0-9]+' | tail -n 1 | cut -d: -f2 || true)"
  size="$(printf '%s' "$info" | grep -oE '"repository":\{"delta":[0-9]+,"size":[0-9]+' | tail -n 1 | sed -E 's/.*"size"://' || true)"
  if [[ ! "$stop" =~ ^[0-9]+$ ]]; then
    record "backup_${stanza}" failed "no completed backup in repository metadata"
  else
    age=$((now_epoch - stop))
    if (( age > max_backup_age )); then
      record "backup_${stanza}" failed "age_seconds=$age threshold_seconds=$max_backup_age"
    else
      record "backup_${stanza}" passed "age_seconds=$age"
    fi
  fi
  if [[ "$size" =~ ^[0-9]+$ ]]; then
    if (( size > max_repository_bytes )); then
      record "repository_${stanza}" failed "bytes=$size threshold_bytes=$max_repository_bytes"
    else
      record "repository_${stanza}" passed "bytes=$size threshold_bytes=$max_repository_bytes"
    fi
  else
    record "repository_${stanza}" failed "repository size unavailable"
  fi
}

request_wal_archive() {
  local label="$1" query_function="$2"
  if ! "$query_function" "SELECT pg_switch_wal()" >/dev/null; then
    record "wal_${label}" failed "pg_switch_wal unavailable"
    return 1
  fi
  sleep 5
  return 0
}

check_wal() {
  local label="$1" query_function="$2" age
  if ! age="$($query_function "SELECT COALESCE(floor(extract(epoch FROM now() - last_archived_time))::bigint,-1) FROM pg_stat_archiver")" || [[ ! "$age" =~ ^[0-9]+$ ]]; then
    record "wal_${label}" failed "last archived WAL time unavailable"
    return
  fi
  if (( age > max_wal_age )); then
    record "wal_${label}" failed "age_seconds=$age threshold_seconds=$max_wal_age"
  else
    record "wal_${label}" passed "age_seconds=$age"
  fi
}

check_backup marketops-postgres marketops-primary
check_backup marketops-timescaledb marketops-temporal
request_wal_archive primary primary_sql && check_wal primary primary_sql
request_wal_archive temporal temporal_sql && check_wal temporal temporal_sql

credentials_result="$(systemctl show signalops-pgbackrest-credentials.service --property=Result --value 2>/dev/null || true)"
if [[ "$credentials_result" == "failed" ]]; then
  record credentials failed "credential refresh service reports failed"
else
  record credentials passed "last_result=${credentials_result:-unknown}"
fi

for unit in signalops-marketops-pgbackrest.service signalops-marketops-boundary-schedule@marketops-intraday.service; do
  result="$(systemctl show "$unit" --property=Result --value 2>/dev/null || true)"
  case "${result:-}" in
    ""|success) record "scheduler_${unit}" passed "last_result=${result:-unknown}" ;;
    *) record "scheduler_${unit}" failed "systemd result=$result" ;;
  esac
done
if [[ "$require_sri" == "true" ]]; then
  for unit in signalops-marketops-boundary-schedule@marketops-sri-refresh.service signalops-marketops-boundary-schedule@marketops-sri-holdings-refresh.service; do
    result="$(systemctl show "$unit" --property=Result --value 2>/dev/null || true)"
    case "${result:-}" in
      ""|success) record "scheduler_${unit}" passed "last_result=${result:-unknown}" ;;
      *) record "scheduler_${unit}" failed "systemd result=$result" ;;
    esac
  done
else
  record scheduler_sri paused "SRI cadence is deliberately disabled pending controlled routing validation"
fi

restore_stamp="$state_dir/restore-rehearsal.json"
if [[ -r "$restore_stamp" ]]; then
  restore_epoch="$(grep -oE '"completed_epoch":[0-9]+' "$restore_stamp" | head -n 1 | cut -d: -f2 || true)"
else
  restore_epoch=""
fi
if [[ "$restore_epoch" =~ ^[0-9]+$ ]]; then
  restore_age=$((now_epoch - restore_epoch))
  if (( restore_age > max_restore_age )); then
    record restore_rehearsal failed "age_seconds=$restore_age threshold_seconds=$max_restore_age"
  else
    record restore_rehearsal passed "age_seconds=$restore_age"
  fi
else
  record restore_rehearsal failed "no durable restore-rehearsal evidence stamp"
fi

if queue="$(primary_sql "SELECT count(*) FILTER (WHERE request_state IN ('queued','warming_up'))::text || '|' || COALESCE(floor(extract(epoch FROM now()-min(requested_at) FILTER (WHERE request_state IN ('queued','warming_up'))))::bigint,0)::text FROM subscriber_global_coverage_activation_requests")"; then
  IFS='|' read -r queue_count queue_age <<< "$queue"
  if [[ "$queue_count" =~ ^[0-9]+$ && "$queue_age" =~ ^[0-9]+$ ]]; then
    if (( queue_count > max_activation_queue || queue_age > max_activation_age )); then
      record coverage_activation_queue failed "count=$queue_count age_seconds=$queue_age"
    else
      record coverage_activation_queue passed "count=$queue_count age_seconds=$queue_age"
    fi
  else
    record coverage_activation_queue failed "invalid queue metrics"
  fi
else
  record coverage_activation_queue failed "queue metrics unavailable"
fi

mkdir -p "$state_dir"
joined="$(IFS=,; printf '%s' "${results[*]}")"
status=healthy
(( failures == 0 )) || status=failed
temp="$state_dir/.marketops-operations-health.$$"
printf '{"observed_at":"%s","status":"%s","failure_count":%s,"checks":[%s]}\n' "$observed_at" "$status" "$failures" "$joined" > "$temp"
install -m 0640 "$temp" "$state_dir/health.json"
rm -f "$temp"

if (( failures > 0 )); then
  printf 'MarketOps operations monitor failed with %s actionable check(s).\n' "$failures" >&2
  exit 1
fi
printf 'MarketOps operations monitor passed.\n'
