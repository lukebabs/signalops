#!/usr/bin/env bash
set -euo pipefail

MODE="${1:---dry-run}"
case "$MODE" in
  --dry-run|--execute) ;;
  *) printf 'Usage: %s [--dry-run|--execute]\n' "${0##*/}" >&2; exit 2 ;;
esac

EXPECTED_MARKETOPS_APPROVAL="I, luke@strategiclabs.io, approve copying current Docker production MarketOps databases into the K8s production MarketOps database PVCs, replacing only the K8s production MarketOps target database contents, with no SignalOps shared database copy, no DNS cutover, no public traffic movement, no K8s scheduler enablement, and no provider polling."
EXPECTED_ALL_PLATFORM_APPROVAL="I, luke@strategiclabs.io, approve copying current Docker production SignalOps and MarketOps databases into the K8s production database PVCs, replacing only the K8s production target database contents, with no DNS cutover, no public traffic movement, no K8s scheduler enablement, and no provider polling."

SCOPE="${SIGNALOPS_K8S_PRODUCTION_DB_REPLICATION_SCOPE:-marketops-only}"
case "$SCOPE" in
  marketops-only|all-platform) ;;
  *) printf 'Invalid SIGNALOPS_K8S_PRODUCTION_DB_REPLICATION_SCOPE: %s\n' "$SCOPE" >&2; exit 2 ;;
esac

NAMESPACE="${SIGNALOPS_K8S_PRODUCTION_DATA_NAMESPACE:-signalops-data}"
APP_NAMESPACE="${SIGNALOPS_K8S_PRODUCTION_APP_NAMESPACE:-signalops-app}"

fail() {
  printf 'signalops_k8s_production_database_replication_dry_run_failed: %s\n' "$*" >&2
  exit 1
}

command -v docker >/dev/null 2>&1 || fail 'docker is required'
command -v kubectl >/dev/null 2>&1 || fail 'kubectl is required'

require_clusterip_service() {
  local service="$1"
  local type external_ip
  type="$(kubectl get svc -n "$NAMESPACE" "$service" -o jsonpath='{.spec.type}' 2>/dev/null || true)"
  external_ip="$(kubectl get svc -n "$NAMESPACE" "$service" -o jsonpath='{.status.loadBalancer.ingress[*].ip}' 2>/dev/null || true)"
  [[ "$type" == "ClusterIP" ]] || fail "target service is externally exposed or non-ClusterIP: $service type=$type"
  [[ -z "$external_ip" ]] || fail "target service has external ingress: $service"
}

require_no_k8s_production_traffic() {
  local available
  available="$(kubectl get deploy -n "$APP_NAMESPACE" signalops-gateway-production -o jsonpath='{.status.availableReplicas}' 2>/dev/null || true)"
  [[ -z "$available" || "$available" == "0" ]] || fail "production K8s gateway has available replicas; data-copy gate must not run while K8s production app is serving"
}

source_psql() {
  local container="$1" db="$2" sql="$3"
  docker exec "$container" psql -U signalops -d "$db" -Atc "$sql"
}

target_psql() {
  local pod="$1" container="$2" db="$3" sql="$4"
  kubectl exec -n "$NAMESPACE" "$pod" -c "$container" -- psql -U signalops -d "$db" -Atc "$sql"
}

public_table_count_sql="SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE';"
size_sql="SELECT pg_database_size(current_database());"

marketops_pairs=(
  "signalops-marketops-postgres-1|marketops|marketops-postgres-production-0|postgres|marketops|marketops-postgres-production"
  "signalops-marketops-timescaledb-1|marketops_temporal|marketops-timescaledb-production-0|timescaledb|marketops_temporal|marketops-timescaledb-production"
)
all_platform_pairs=(
  "signalops-postgres-1|signalops|signalops-postgres-production-0|postgres|signalops|signalops-postgres-production"
  "signalops-timescaledb-1|signalops_temporal|signalops-timescaledb-production-0|timescaledb|signalops_temporal|signalops-timescaledb-production"
  "${marketops_pairs[@]}"
)
if [[ "$SCOPE" == "marketops-only" ]]; then
  pairs=("${marketops_pairs[@]}")
else
  pairs=("${all_platform_pairs[@]}")
fi

require_no_k8s_production_traffic
unavailable=0
printf 'signalops_k8s_production_database_replication_%s\n' "${MODE#--}"
for pair in "${pairs[@]}"; do
  IFS='|' read -r source_container source_db target_pod target_container target_db target_service <<<"$pair"
  source_status=ready
  target_status=ready
  source_tables=unavailable
  source_size=unavailable
  target_tables=unavailable
  target_size=unavailable
  if ! docker inspect --format '{{.State.Running}}' "$source_container" 2>/dev/null | grep -q '^true$'; then
    source_status=missing_or_stopped
    unavailable=1
  elif ! docker exec "$source_container" pg_isready -U signalops -d "$source_db" >/dev/null 2>&1; then
    source_status=not_ready
    unavailable=1
  else
    source_tables="$(source_psql "$source_container" "$source_db" "$public_table_count_sql" | tr -d '[:space:]')"
    source_size="$(source_psql "$source_container" "$source_db" "$size_sql" | tr -d '[:space:]')"
  fi
  if ! kubectl get pod -n "$NAMESPACE" "$target_pod" >/dev/null 2>&1; then
    target_status=missing
    unavailable=1
  elif ! kubectl exec -n "$NAMESPACE" "$target_pod" -c "$target_container" -- pg_isready -U signalops -d "$target_db" >/dev/null 2>&1; then
    target_status=not_ready
    unavailable=1
  else
    require_clusterip_service "$target_service"
    target_tables="$(target_psql "$target_pod" "$target_container" "$target_db" "$public_table_count_sql" | tr -d '[:space:]')"
    target_size="$(target_psql "$target_pod" "$target_container" "$target_db" "$size_sql" | tr -d '[:space:]')"
  fi
  printf 'database=%s source_container=%s source_status=%s target_pod=%s target_status=%s service=%s source_tables=%s target_tables=%s source_size_bytes=%s target_size_bytes=%s\n' \
    "$source_db" "$source_container" "$source_status" "$target_pod" "$target_status" "$target_service" "$source_tables" "$target_tables" "$source_size" "$target_size"
done
printf 'mode=%s\nscope=%s\nproduction_traffic_moved=false\nk8s_schedulers_enabled=false\nprovider_polling=false\n' "${MODE#--}" "$SCOPE"
if (( unavailable )); then
  printf 'ready_for_execute=false\n'
  exit 4
fi
printf 'ready_for_execute=true\n'

if [[ "$MODE" == "--dry-run" ]]; then
  exit 0
fi

if [[ "$SCOPE" == "marketops-only" ]]; then
  expected_approval="$EXPECTED_MARKETOPS_APPROVAL"
else
  expected_approval="$EXPECTED_ALL_PLATFORM_APPROVAL"
fi
[[ "${SIGNALOPS_K8S_PRODUCTION_DB_REPLICATION_APPROVAL:-}" == "$expected_approval" ]] || {
  printf 'Missing exact approval for scope=%s. Set SIGNALOPS_K8S_PRODUCTION_DB_REPLICATION_APPROVAL to the matching named approval text.\n' "$SCOPE" >&2
  exit 3
}

run_id="k8s-prod-db-replication-$(date -u +%Y%m%dT%H%M%SZ)"
printf 'signalops_k8s_production_database_replication_started run_id=%s\n' "$run_id"
for pair in "${pairs[@]}"; do
  IFS='|' read -r source_container source_db target_pod target_container target_db target_service <<<"$pair"
  before_tables="$(target_psql "$target_pod" "$target_container" "$target_db" "$public_table_count_sql" | tr -d '[:space:]')"
  printf 'replicating database=%s source=%s target=%s target_tables_before=%s\n' "$source_db" "$source_container" "$target_pod" "$before_tables"
  docker exec "$source_container" pg_dump -U signalops -d "$source_db" --format=custom --no-owner --no-acl \
    | kubectl exec -i -n "$NAMESPACE" "$target_pod" -c "$target_container" -- pg_restore -U signalops -d "$target_db" --clean --if-exists --no-owner --no-acl --exit-on-error
  after_tables="$(target_psql "$target_pod" "$target_container" "$target_db" "$public_table_count_sql" | tr -d '[:space:]')"
  after_size="$(target_psql "$target_pod" "$target_container" "$target_db" "$size_sql" | tr -d '[:space:]')"
  printf 'replicated database=%s target_tables_after=%s target_size_bytes=%s\n' "$target_db" "$after_tables" "$after_size"
done
printf 'signalops_k8s_production_database_replication_verified run_id=%s\n' "$run_id"
printf 'production_traffic_moved=false\nk8s_schedulers_enabled=false\nprovider_polling=false\n'
