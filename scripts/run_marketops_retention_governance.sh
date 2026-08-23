#!/usr/bin/env bash
set -Eeuo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
runtime_env="${1:-$root_dir/.env}"
boundary_env="${SIGNALOPS_MARKETOPS_BOUNDARY_ENV_FILE:-/etc/signalops/marketops-boundary.env}"
job_id="marketops-retention-governance"
schedule="Manual dry run"
timezone="America/New_York"

[[ -r "$runtime_env" ]] || {
  printf 'runtime_env_unreadable: %s\n' "$runtime_env" >&2
  exit 2
}
[[ -r "$boundary_env" ]] || {
  printf 'marketops_boundary_env_unreadable: %s\n' "$boundary_env" >&2
  printf 'Run through the root-owned deployment agent or as root; do not copy protected database secrets into the repository.\n' >&2
  exit 2
}

cd "$root_dir"
# shellcheck source=scripts/lib/marketops_boundary_env.sh
source "$root_dir/scripts/lib/marketops_boundary_env.sh"
load_marketops_boundary_env "$boundary_env"

export SIGNALOPS_MARKETOPS_PRIMARY_DB_SERVICE=marketops-postgres
export SIGNALOPS_MARKETOPS_PRIMARY_DATABASE=marketops
export SIGNALOPS_MARKETOPS_TEMPORAL_DB_SERVICE=marketops-timescaledb
export SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE=marketops_temporal

# shellcheck source=scripts/marketops_schedule_database.sh
source "$root_dir/scripts/marketops_schedule_database.sh"

started_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
run_id="${job_id}-$(date -u +%Y%m%dT%H%M%SZ)"
marketops_record_scheduled_job_status_or_warn "$run_id" "$job_id" "$schedule" "$timezone" "running" "$started_at" "" "" "" "{}" || true

status="succeeded"
reason=""
exit_code=0
output_file="$(mktemp /tmp/signalops-marketops-retention-governance.XXXXXX)"
trap 'rm -f "$output_file"' EXIT

compose_args=(
  docker compose
  --env-file "$runtime_env"
  -f "$root_dir/compose.yaml"
  -f "$root_dir/compose.marketops-boundary.yaml"
  -f "$root_dir/compose.marketops-scheduled-cutover.yaml"
  -f "$root_dir/compose.marketops-retention-governance.yaml"
  --profile marketops-retention-governance
)

set +e
"${compose_args[@]}" run --rm --build marketops-retention-governor \
  --tenant-id tenant-local \
  --policy-id subscriber.user_activity_180d >"$output_file" 2>&1
tenant_local_exit=$?
"${compose_args[@]}" run --rm --build marketops-retention-governor \
  --tenant-id tenant-pilot-b \
  --policy-id subscriber.user_activity_180d >>"$output_file" 2>&1
tenant_pilot_exit=$?
set -e

if [[ "$tenant_local_exit" -ne 0 || "$tenant_pilot_exit" -ne 0 ]]; then
  status="failed"
  reason="retention_governor_failed"
  exit_code=1
fi

completed_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
detail_json="$(printf '{"policy_id":"subscriber.user_activity_180d","mode":"dry_run","tenants":["tenant-local","tenant-pilot-b"],"tenant_local_exit":%s,"tenant_pilot_b_exit":%s}' "$tenant_local_exit" "$tenant_pilot_exit")"
marketops_record_scheduled_job_status_or_warn "$run_id" "$job_id" "$schedule" "$timezone" "$status" "$started_at" "$completed_at" "$exit_code" "$reason" "$detail_json" || true

cat "$output_file"
marketops_primary_psql -P pager=off -c "
SELECT tenant_id, policy_id, mode, status, candidate_rows, affected_rows, started_at, completed_at
FROM retention_runs
WHERE policy_id = 'subscriber.user_activity_180d'
ORDER BY started_at DESC
LIMIT 6;"

if [[ "$status" != "succeeded" ]]; then
  exit "$exit_code"
fi
