#!/usr/bin/env bash
# Shell helpers for MarketOps scheduled jobs. Source from a job script after
# changing to the repository root.
marketops_compose() {
  local root_dir
  root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

  # Scheduled jobs run on the dedicated boundary after cutover. Make that
  # compose topology explicit instead of relying on inherited COMPOSE_FILE,
  # which can silently route a job back to the shared database.
  if [[ "${SIGNALOPS_MARKETOPS_PRIMARY_DB_SERVICE:-}" == "marketops-postgres" ]]; then
    docker compose \
      -f "$root_dir/compose.yaml" \
      -f "$root_dir/compose.marketops-boundary.yaml" \
      -f "$root_dir/compose.marketops-scheduled-cutover.yaml" \
      "$@"
    return
  fi

  docker compose "$@"
}

marketops_primary_psql() {
  marketops_compose exec -T "${SIGNALOPS_MARKETOPS_PRIMARY_DB_SERVICE:-postgres}" \
    psql -U signalops -d "${SIGNALOPS_MARKETOPS_PRIMARY_DATABASE:-signalops}" "$@"
}
marketops_temporal_psql() {
  marketops_compose exec -T "${SIGNALOPS_MARKETOPS_TEMPORAL_DB_SERVICE:-timescaledb}" \
    psql -U signalops -d "${SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE:-signalops_temporal}" "$@"
}


marketops_status_psql() {
  local root_dir service database
  root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
  service="${SIGNALOPS_SCHEDULE_STATUS_DB_SERVICE:-${SIGNALOPS_MARKETOPS_PRIMARY_DB_SERVICE:-postgres}}"
  database="${SIGNALOPS_SCHEDULE_STATUS_DATABASE:-${SIGNALOPS_MARKETOPS_PRIMARY_DATABASE:-signalops}}"
  if [[ "$service" == "marketops-postgres" ]]; then
    docker compose \
      -f "$root_dir/compose.yaml" \
      -f "$root_dir/compose.marketops-boundary.yaml" \
      exec -T "$service" psql -U signalops -d "$database" "$@"
    return
  fi
  marketops_compose exec -T "$service" psql -U signalops -d "$database" "$@"
}


marketops_record_scheduled_job_status() {
  local run_id="$1"
  local job_id="$2"
  local schedule="$3"
  local timezone="$4"
  local status="$5"
  local started_at="$6"
  local completed_at="${7:-}"
  local exit_code="${8:-}"
  local reason="${9:-}"
  local detail_json="${10:-}"
  local runner="${SIGNALOPS_SCHEDULE_RUNNER_ID:-systemd}"
  [[ -n "$detail_json" ]] || detail_json='{}'
  marketops_status_psql -q -v ON_ERROR_STOP=1 \
    -v run_id="$run_id" \
    -v job_id="$job_id" \
    -v schedule="$schedule" \
    -v timezone="$timezone" \
    -v status="$status" \
    -v started_at="$started_at" \
    -v completed_at="$completed_at" \
    -v exit_code="$exit_code" \
    -v reason="$reason" \
    -v detail_json="$detail_json" \
    -v runner="$runner" <<'SQL'
WITH upsert_status AS (
  INSERT INTO marketops_scheduled_job_statuses (
    job_id, schedule, timezone, status, reason, started_at, completed_at,
    exit_code, detail, runner, updated_at
  ) VALUES (
    :'job_id', :'schedule', :'timezone', :'status', COALESCE(:'reason',''),
    NULLIF(:'started_at','')::timestamptz,
    NULLIF(:'completed_at','')::timestamptz,
    NULLIF(:'exit_code','')::integer,
    COALESCE(NULLIF(:'detail_json','')::jsonb, '{}'::jsonb),
    COALESCE(:'runner',''), now()
  )
  ON CONFLICT (job_id) DO UPDATE SET
    schedule = EXCLUDED.schedule,
    timezone = EXCLUDED.timezone,
    status = EXCLUDED.status,
    reason = EXCLUDED.reason,
    started_at = EXCLUDED.started_at,
    completed_at = EXCLUDED.completed_at,
    exit_code = EXCLUDED.exit_code,
    detail = EXCLUDED.detail,
    runner = EXCLUDED.runner,
    updated_at = now()
  RETURNING 1
)
INSERT INTO marketops_scheduled_job_runs (
  run_id, job_id, schedule, timezone, status, reason, started_at, completed_at,
  exit_code, detail, runner, updated_at
) VALUES (
  :'run_id', :'job_id', :'schedule', :'timezone', :'status', COALESCE(:'reason',''),
  NULLIF(:'started_at','')::timestamptz,
  NULLIF(:'completed_at','')::timestamptz,
  NULLIF(:'exit_code','')::integer,
  COALESCE(NULLIF(:'detail_json','')::jsonb, '{}'::jsonb),
  COALESCE(:'runner',''), now()
)
ON CONFLICT (run_id) DO UPDATE SET
  status = EXCLUDED.status,
  reason = EXCLUDED.reason,
  completed_at = EXCLUDED.completed_at,
  exit_code = EXCLUDED.exit_code,
  detail = EXCLUDED.detail,
  runner = EXCLUDED.runner,
  updated_at = now();
SQL
}

marketops_record_scheduled_job_status_or_warn() {
  if ! marketops_record_scheduled_job_status "$@"; then
    printf 'warning: failed to record scheduled job status in MarketOps database for %s\n' "$2" >&2
    return 1
  fi
}
