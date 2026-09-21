#!/usr/bin/env bash
set -euo pipefail

: "${SIGNALOPS_MARKETOPS_DATABASE_URL:?SIGNALOPS_MARKETOPS_DATABASE_URL is required}"
session_date="${MARKETOPS_SESSION_DATE:-$(date -u -d yesterday +%F 2>/dev/null || date -u +%F)}"
run_id="${MARKETOPS_K8S_RUN_ID:-marketops-task-retry-k8s-$(date -u +%Y%m%dT%H%M%SZ)}"

symbols="$(psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -v ON_ERROR_STOP=1 -At \
  -v session_date="$session_date" <<'SQL'
SELECT symbol
FROM marketops_task_items
WHERE tenant_id = 'tenant-local'
  AND session_date = to_date(:'session_date', 'YYYY-MM-DD')
  AND task_type = 'tactical_posture'
  AND status = 'retry_scheduled'
  AND next_attempt_at <= now()
ORDER BY symbol;
SQL
)"
symbols="$(printf '%s\n' "$symbols" | paste -sd, -)"

dependency_status="$(psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -v ON_ERROR_STOP=1 -At \
  -c "SELECT COALESCE((SELECT status FROM marketops_scheduled_job_statuses WHERE job_id='marketops-daily-postclose' LIMIT 1),'missing')")"

if [[ "$dependency_status" != "succeeded" && "$dependency_status" != "degraded" ]]; then
  psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -v ON_ERROR_STOP=1 \
    -v run_id="$run_id" -v session_date="$session_date" -v dependency_status="$dependency_status" <<'SQL'
INSERT INTO marketops_job_dependency_evaluations
  (evaluation_id, session_date, job_id, state, reason, dependencies, correlation_id)
VALUES
  (:'run_id' || '-evaluation', to_date(:'session_date','YYYY-MM-DD'),
   'marketops-task-retry', 'blocked',
   'dependency marketops-daily-postclose is ' || :'dependency_status',
   jsonb_build_array(jsonb_build_object('job_id','marketops-daily-postclose','status',:'dependency_status')),
   :'run_id')
ON CONFLICT DO NOTHING;
SQL
  echo "marketops_k8s_task_retry_blocked session=${session_date} dependency_status=${dependency_status}"
  exit 0
fi

if [[ -z "$symbols" ]]; then
  echo "marketops_k8s_task_retry_no_due_work session=${session_date}"
  exit 0
fi

if [[ "${MARKETOPS_K8S_DRY_RUN:-false}" == "true" ]]; then
  echo "marketops_k8s_task_retry_dry_run_verified session=${session_date} symbols=${symbols}"
  exit 0
fi

signalops-marketops-tactical-valuation-runner \
  --tenant-id tenant-local \
  --universe-group all_active \
  --session-date "$session_date" \
  --symbols "$symbols" \
  --max-retries 2
