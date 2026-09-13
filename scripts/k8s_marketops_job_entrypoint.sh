#!/usr/bin/env bash
set -euo pipefail

job_id="${1:-}"
runtime_env_file="${SIGNALOPS_K8S_RUNTIME_ENV_FILE:-/vault/secrets/marketops-worker-runtime.env}"

fail() {
  echo "signalops_k8s_marketops_job_failed: $*" >&2
  exit 2
}

[[ -n "$job_id" ]] || fail "job id is required"
if [[ -f "$runtime_env_file" ]]; then
  # shellcheck disable=SC1090
  source "$runtime_env_file"
fi

export SIGNALOPS_ENV="${SIGNALOPS_ENV:-kubernetes-staging}"
export SIGNALOPS_MARKETOPS_DATA_BOUNDARY_REQUIRED="${SIGNALOPS_MARKETOPS_DATA_BOUNDARY_REQUIRED:-true}"

mode_flag="--execute"
if [[ "${MARKETOPS_K8S_DRY_RUN:-false}" == "true" ]]; then
  mode_flag="--dry-run"
fi

started_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
run_id="${MARKETOPS_K8S_RUN_ID:-${job_id}-k8s-$(date -u +%Y%m%dT%H%M%SZ)}"
schedule_label="${MARKETOPS_K8S_SCHEDULE_LABEL:-Kubernetes staging suspended CronJob}"
timezone_label="${MARKETOPS_K8S_TIMEZONE:-UTC}"
runner_label="${MARKETOPS_K8S_RUNNER_ID:-kubernetes}"
status_database_url="${SIGNALOPS_K8S_STATUS_DATABASE_URL:-${SIGNALOPS_MARKETOPS_DATABASE_URL:-${SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL:-}}}"
status_required="${SIGNALOPS_K8S_STATUS_RECORDING_REQUIRED:-true}"

json_escape() {
  printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g'
}

record_status() {
  local status="$1"
  local completed_at="${2:-}"
  local exit_code="${3:-}"
  local reason="${4:-}"

  if [[ -z "$status_database_url" ]]; then
    [[ "$status_required" != "true" ]] || fail "status database URL is required for Kubernetes scheduled-job parity"
    echo "warning: skipping Kubernetes job status record because status database URL is empty" >&2
    return 0
  fi
  command -v psql >/dev/null 2>&1 || {
    [[ "$status_required" != "true" ]] || fail "psql is required for Kubernetes scheduled-job parity"
    echo "warning: skipping Kubernetes job status record because psql is unavailable" >&2
    return 0
  }

  psql "$status_database_url" -q -v ON_ERROR_STOP=1 \
    -v run_id="$run_id" \
    -v job_id="$job_id" \
    -v schedule="$schedule_label" \
    -v timezone="$timezone_label" \
    -v status="$status" \
    -v started_at="$started_at" \
    -v completed_at="$completed_at" \
    -v exit_code="$exit_code" \
    -v reason="$reason" \
    -v mode="$mode_flag" \
    -v dry_run="$dry_run_json" \
    -v runner="$runner_label" <<'SQL'
WITH upsert_status AS (
  INSERT INTO marketops_scheduled_job_statuses (
    job_id, schedule, timezone, status, reason, started_at, completed_at,
    exit_code, detail, runner, updated_at
  ) VALUES (
    :'job_id', :'schedule', :'timezone', :'status', COALESCE(:'reason',''),
    NULLIF(:'started_at','')::timestamptz,
    NULLIF(:'completed_at','')::timestamptz,
    NULLIF(:'exit_code','')::integer,
    jsonb_build_object('mode', :'mode', 'dry_run', (:'dry_run')::boolean),
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
  jsonb_build_object('mode', :'mode', 'dry_run', (:'dry_run')::boolean),
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

command_args=()
case "$job_id" in
  marketops-intraday)
    command_args=(
      signalops-marketops-intraday-monitor
      --tenant-id "${MARKETOPS_INTRADAY_TENANT_ID:-tenant-local}"
      --universe-group "${MARKETOPS_INTRADAY_UNIVERSE_GROUP:-all_active}"
      --max-symbols "${MARKETOPS_INTRADAY_MAX_SYMBOLS:-200}"
    )
    if [[ "$mode_flag" == "--dry-run" ]]; then
      command_args+=(--dry-run)
    fi
    ;;
  marketops-sri-refresh)
    command_args=(
      signalops-marketops-sri-runner
      --tenant-id "${SIGNALOPS_SRI_OUTPUT_TENANT_ID:-platform-global}"
      --input-tenant-id "${SIGNALOPS_SRI_INPUT_TENANT_ID:-tenant-local}"
      --as-of "${MARKETOPS_SESSION_DATE:-$(date -u +%F)}"
    )
    ;;
  marketops-sri-holdings-refresh)
    command_args=(
      signalops-marketops-sri-holdings-runner
      --tenant-id "${SIGNALOPS_SRI_OUTPUT_TENANT_ID:-platform-global}"
    )
    ;;
  marketops-fmp-annual-financial)
    command_args=(
      signalops-subscriber-global-annual-financial-task-worker
      "$mode_flag"
      --max-assets "${MARKETOPS_FMP_ANNUAL_MAX_ASSETS:-1000}"
      --max-retries "${MARKETOPS_FMP_ANNUAL_MAX_RETRIES:-2}"
      --session-date "${MARKETOPS_SESSION_DATE:-}"
      --correlation-id "${MARKETOPS_FMP_ANNUAL_CORRELATION_ID:-$run_id}"
    )
    ;;
  marketops-saf-benchmark)
    command_args=(
      signalops-subscriber-global-saf-benchmark-materializer
      "$mode_flag"
      --max-observations "${MARKETOPS_SAF_BENCHMARK_MAX_OBSERVATIONS:-500}"
      --calculation-version "${MARKETOPS_SAF_BENCHMARK_CALCULATION_VERSION:-saf_benchmark.k8s_staging}"
      --correlation-id "${MARKETOPS_SAF_BENCHMARK_CORRELATION_ID:-k8s-staging-cronjob}"
    )
    ;;

  marketops-daily-postclose)
    [[ "$mode_flag" == "--dry-run" ]] || fail "marketops-daily-postclose requires a separately approved production K8s scheduler cutover"
    command_args=(
      bash
      -ec
      'session_date="${MARKETOPS_SESSION_DATE:-$(date -u +%F)}"; counts="$(psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -v ON_ERROR_STOP=1 -Atc "SELECT (SELECT count(*) FROM marketops_scheduled_job_statuses) || chr(124) || (SELECT count(*) FROM subscriber_global_warm_eod_assets)")"; echo "marketops_k8s_daily_postclose_dry_run_verified session=${session_date} status_and_warm_counts=${counts}"'
    )
    ;;
  marketops-postclose-recovery)
    [[ "$mode_flag" == "--dry-run" ]] || fail "marketops-postclose-recovery requires a separately approved production K8s scheduler cutover"
    command_args=(
      bash
      -ec
      'session_date="${MARKETOPS_SESSION_DATE:-$(date -u +%F)}"; latest="$(psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -v ON_ERROR_STOP=1 -Atc "SELECT COALESCE(max(completed_at)::text,\$none\$none\$none\$) FROM marketops_scheduled_job_runs WHERE job_id IN (\$daily\$marketops-daily-postclose\$daily\$,\$risk\$marketops-risk-reward\$risk\$)")"; echo "marketops_k8s_postclose_recovery_dry_run_verified session=${session_date} latest_dependency_completion=${latest}"'
    )
    ;;
  marketops-risk-reward)
    [[ "$mode_flag" == "--dry-run" ]] || fail "marketops-risk-reward requires a separately approved production K8s scheduler cutover"
    command_args=(
      bash
      -ec
      'session_date="${MARKETOPS_SESSION_DATE:-$(date -u +%F)}"; snapshot_count="$(psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -v ON_ERROR_STOP=1 -Atc "SELECT COALESCE(count(*),0) FROM marketops_scheduled_job_statuses")"; echo "marketops_k8s_risk_reward_dry_run_verified session=${session_date} status_rows=${snapshot_count}"'
    )
    ;;
  marketops-fmp-continuation)
    [[ "$mode_flag" == "--dry-run" ]] || fail "marketops-fmp-continuation requires a separately approved production K8s scheduler cutover"
    command_args=(
      bash
      -ec
      'workflow_count="$(psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -v ON_ERROR_STOP=1 -Atc "SELECT count(*) FROM marketops_scheduled_job_runs WHERE job_id=\$job\$marketops-fmp-annual-financial\$job\$")"; echo "marketops_k8s_fmp_continuation_dry_run_verified annual_workflow_rows=${workflow_count}"'
    )
    ;;
  marketops-operations-monitor)
    [[ "$mode_flag" == "--dry-run" ]] || fail "marketops-operations-monitor is K8s-staging dry-run only until production scheduler cutover is approved"
    command_args=(
      bash
      -ec
      'psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -v ON_ERROR_STOP=1 -Atc "SELECT current_database() || chr(124) || count(*)::text FROM marketops_scheduled_job_statuses"; psql "$SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL" -v ON_ERROR_STOP=1 -Atc "SELECT current_database()"; echo marketops_k8s_operations_monitor_dry_run_verified'
    )
    ;;
  marketops-retention-governance)
    [[ "$mode_flag" == "--dry-run" ]] || fail "marketops-retention-governance is K8s-staging dry-run only until production scheduler cutover is approved"
    command_args=(
      bash
      -ec
      'signalops-retention-governor --tenant-id tenant-local --policy-id subscriber.user_activity_180d; signalops-retention-governor --tenant-id tenant-pilot-b --policy-id subscriber.user_activity_180d; echo marketops_k8s_retention_governance_dry_run_verified'
    )
    ;;

  marketops-task-retry)
    [[ "$mode_flag" == "--dry-run" ]] || fail "marketops-task-retry is K8s-staging dry-run only until production scheduler cutover is approved"
    command_args=(
      bash
      -ec
      'session_date="${MARKETOPS_SESSION_DATE:-$(date -u -d yesterday +%F 2>/dev/null || date -u +%F)}"; due_count="$(psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -v ON_ERROR_STOP=1 -Atc "SELECT count(*) FROM marketops_task_items WHERE tenant_id=\$tenant\$tenant-local\$tenant\$ AND session_date=to_date(\$session\$${session_date}\$session\$,\$format\$YYYY-MM-DD\$format\$) AND task_type=\$task\$tactical_posture\$task\$ AND status=\$status\$retry_scheduled\$status\$ AND next_attempt_at <= now()")"; echo "marketops_k8s_task_retry_dry_run_verified session=${session_date} due_retries=${due_count}"'
    )
    ;;
  marketops-warm-eod)
    [[ "$mode_flag" == "--dry-run" ]] || fail "marketops-warm-eod is K8s-staging dry-run only until production scheduler cutover is approved"
    command_args=(
      bash
      -ec
      'warm_count="$(psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -v ON_ERROR_STOP=1 -Atc "SELECT count(*) FROM subscriber_global_warm_eod_assets")"; [[ "$warm_count" =~ ^[0-9]+$ ]] || exit 4; echo "marketops_k8s_warm_eod_dry_run_verified warm_assets=${warm_count}"'
    )
    ;;
  *)
    fail "unsupported MarketOps Kubernetes job id: $job_id"
    ;;
esac

dry_run_json=false
if [[ "$mode_flag" == "--dry-run" ]]; then
  dry_run_json=true
fi
record_status "running" "" "" ""

set +e
"${command_args[@]}"
exit_code=$?
set -e

completed_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
status="succeeded"
reason=""
if [[ "$exit_code" -ne 0 ]]; then
  status="failed"
fi
record_status "$status" "$completed_at" "$exit_code" "$reason"
exit "$exit_code"
