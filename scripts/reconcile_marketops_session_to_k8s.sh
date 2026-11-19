#!/usr/bin/env bash
set -euo pipefail

# Additive, conflict-safe reconciliation of one completed MarketOps session
# from the retained Docker boundary into the K8s production databases. This
# deliberately never truncates, updates, or deletes target rows.
session_date=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    --date) session_date="${2:-}"; shift 2 ;;
    *) printf 'Usage: %s --date YYYY-MM-DD\n' "$0" >&2; exit 2 ;;
  esac
done
[[ "$session_date" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]] || { echo 'session date must be YYYY-MM-DD' >&2; exit 2; }

namespace="${SIGNALOPS_K8S_DATA_NAMESPACE:-signalops-data}"
src_primary="${SIGNALOPS_DOCKER_MARKETOPS_PRIMARY_CONTAINER:-signalops-marketops-postgres-1}"
src_temporal="${SIGNALOPS_DOCKER_MARKETOPS_TEMPORAL_CONTAINER:-signalops-marketops-timescaledb-1}"
dst_primary="${SIGNALOPS_K8S_MARKETOPS_PRIMARY_POD:-marketops-postgres-production-0}"
dst_temporal="${SIGNALOPS_K8S_MARKETOPS_TEMPORAL_POD:-marketops-timescaledb-production-0}"
run_id="marketops-session-reconcile-${session_date}-$(date -u +%Y%m%dT%H%M%SZ)"

docker_psql() { docker exec "$1" psql -X -v ON_ERROR_STOP=1 -U signalops -d "$2" -Atqc "$3"; }
k8s_psql() { kubectl exec -n "$namespace" "$1" -- psql -X -v ON_ERROR_STOP=1 -U signalops -d "$2" -Atqc "$3"; }
k8s_psql_stdin() { kubectl exec -i -n "$namespace" "$1" -- psql -X -v ON_ERROR_STOP=1 -U signalops -d "$2" -c "$3"; }

kubectl get pod -n "$namespace" "$dst_primary" >/dev/null
kubectl get pod -n "$namespace" "$dst_temporal" >/dev/null
docker inspect "$src_primary" >/dev/null
docker inspect "$src_temporal" >/dev/null

copy_table() {
  local source_container="$1" source_db="$2" destination_pod="$3" destination_db="$4" table="$5" predicate="$6"
  local safe_table stage exists inserted
  safe_table="${table//[^a-zA-Z0-9_]/_}"
  stage="mr_${safe_table:0:28}_$$_${RANDOM}"
  exists="$(docker_psql "$source_container" "$source_db" "SELECT (to_regclass('public.$table') IS NOT NULL)::int")"
  [[ "$exists" == "1" ]] || { echo "skipped=$table reason=source_table_missing"; return 0; }
  exists="$(k8s_psql "$destination_pod" "$destination_db" "SELECT (to_regclass('public.$table') IS NOT NULL)::int")"
  [[ "$exists" == "1" ]] || { echo "skipped=$table reason=target_table_missing"; return 0; }
  k8s_psql_stdin "$destination_pod" "$destination_db" "CREATE UNLOGGED TABLE public.$stage (LIKE public.$table INCLUDING DEFAULTS);" >/dev/null
  if ! docker exec "$source_container" psql -X -v ON_ERROR_STOP=1 -U signalops -d "$source_db" -c "COPY (SELECT * FROM public.$table WHERE $predicate) TO STDOUT WITH (FORMAT binary)" | k8s_psql_stdin "$destination_pod" "$destination_db" "COPY public.$stage FROM STDIN WITH (FORMAT binary)" >/dev/null; then
    k8s_psql_stdin "$destination_pod" "$destination_db" "DROP TABLE IF EXISTS public.$stage;" >/dev/null || true
    return 1
  fi
  inserted="$(k8s_psql "$destination_pod" "$destination_db" "WITH inserted AS (INSERT INTO public.$table SELECT * FROM public.$stage ON CONFLICT DO NOTHING RETURNING 1) SELECT count(*) FROM inserted;")"
  k8s_psql_stdin "$destination_pod" "$destination_db" "DROP TABLE public.$stage;" >/dev/null
  printf 'reconciled=%s inserted=%s\n' "$table" "${inserted:-0}"
}

echo "marketops_session_reconcile_started run_id=$run_id session_date=$session_date"

# Temporal ledgers are the authoritative EOD evidence source.
copy_table "$src_temporal" marketops_temporal "$dst_temporal" marketops_temporal normalized_event_ledger "app_id='marketops' AND (observation_time::date=DATE '$session_date' OR normalized_payload->>'observation_date'='$session_date')"
copy_table "$src_temporal" marketops_temporal "$dst_temporal" marketops_temporal signal_ledger "app_id='marketops' AND created_at::date=DATE '$session_date'"

# Primary algorithm/evidence rows. Missing tables are safely skipped to allow
# schema-version differences between the retained source and K8s target.
for table in marketops_eeom_results marketops_evidence marketops_feature_observations marketops_market_states marketops_hypothesis_evaluations marketops_options_capture_sessions marketops_risk_reward_snapshots marketops_task_workflows marketops_task_items marketops_valuation_snapshots marketops_valuation_results sri_segment_snapshots marketops_options_chain_daily marketops_options_distribution_daily subscriber_global_marketops_evidence_runs subscriber_global_marketops_evidence_records; do
  case "$table" in
    marketops_task_items|marketops_task_workflows) predicate="created_at::date=DATE '$session_date'" ;;
    subscriber_global_marketops_evidence_records) predicate="session_date=DATE '$session_date'" ;;
    subscriber_global_marketops_evidence_runs) predicate="session_date=DATE '$session_date'" ;;
    *) predicate="COALESCE(as_of_date, created_at::date)=DATE '$session_date'" ;;
  esac
  # Build a predicate using only columns that exist on the source table.
  columns="$(docker_psql "$src_primary" marketops "SELECT string_agg(column_name, ',') FROM information_schema.columns WHERE table_schema='public' AND table_name='$table'")"
  [[ -n "$columns" ]] || { echo "skipped=$table reason=source_table_missing"; continue; }
  if [[ ",$columns," == *,as_of_date,* ]]; then predicate="as_of_date=DATE '$session_date'"; elif [[ ",$columns," == *,session_date,* ]]; then predicate="session_date=DATE '$session_date'"; elif [[ ",$columns," == *,as_of_time,* ]]; then predicate="as_of_time::date=DATE '$session_date'"; elif [[ ",$columns," == *,created_at,* ]]; then predicate="created_at::date=DATE '$session_date'"; else echo "skipped=$table reason=no_session_column"; continue; fi
  copy_table "$src_primary" marketops "$dst_primary" marketops "$table" "$predicate"
done

echo "marketops_session_reconcile_verified run_id=$run_id session_date=$session_date"
