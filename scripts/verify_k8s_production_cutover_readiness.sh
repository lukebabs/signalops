#!/usr/bin/env bash
set -euo pipefail

mode="${1:---report}"

fail() {
  echo "signalops_k8s_production_cutover_readiness_failed: $*" >&2
  exit 1
}

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_dir"

require_file() {
  local path="$1"
  [[ -f "$path" ]] || fail "missing required evidence file: $path"
}

require_executable() {
  local path="$1"
  [[ -x "$path" ]] || fail "missing executable verifier: $path"
}

require_file docs/projects/subscriber_project/k8s1_openbao_staging_foundation_evidence_2026-09-12.md
require_file docs/projects/subscriber_project/k8s2_staging_app_manifest_evidence_2026-09-12.md
require_file docs/projects/subscriber_project/k8s2_staging_app_parity_evidence_2026-09-12.md
require_file docs/projects/subscriber_project/k8s3_marketops_scheduled_jobs_scaffold_2026-09-12.md
require_file docs/projects/subscriber_project/k8s3_cronjob_unsuspend_resuspend_smoke_2026-09-12.md
require_file docs/projects/subscriber_project/k8s3_provider_cronjob_smoke_2026-09-12.md
require_file docs/projects/subscriber_project/pr3_backup_restore_refresh_evidence_2026-09-12.md
require_file docs/projects/subscriber_project/k8s_production_cutover_parity_plan.md
require_file docs/projects/subscriber_project/k8s_mesh1_istio_readiness_2026-09-13.md
require_file docs/projects/subscriber_project/k8s_mesh2_signalops_staging_route_parity_2026-09-13.md
require_file docs/projects/subscriber_project/k8s_mesh4_ingress_dns_cutover_plan_2026-09-13.md
require_file docs/projects/subscriber_project/k8s_stripe_webhook_route_parity_2026-09-13.md
require_file docs/projects/subscriber_project/k8s_capacity_load_validation_2026-09-13.md
require_file docs/projects/subscriber_project/k8s_capacity_remediation_plan_2026-09-13.md
require_file docs/projects/subscriber_project/k8s4_signal_connect_shadow_smoke_2026-09-13.md
require_file docs/projects/subscriber_project/k8s5_raw_worker_scaffold_2026-09-13.md
require_file docs/projects/subscriber_project/k8s5_raw_worker_processing_shadow_2026-09-13.md

require_executable scripts/verify_k8s_base_scaffold.sh
require_executable scripts/verify_k8s_staging_app_manifests.sh
require_executable scripts/verify_k8s_marketops_scheduled_jobs_manifests.sh
require_executable scripts/verify_k8s_marketops_scheduler_parity_coverage.sh
require_executable scripts/verify_k8s_marketops_dedicated_staging_data_manifests.sh
require_executable scripts/verify_k8s_signalops_connect_staging_manifests.sh
require_executable scripts/verify_k8s_signalops_connect_broker_manifests.sh
require_executable scripts/verify_k8s_signalops_raw_worker_manifests.sh
require_executable scripts/verify_k8s_mesh4_ingress_dns_cutover_plan.sh
require_executable scripts/verify_k8s_mesh1_istio_readiness.sh
require_executable scripts/verify_k8s_mesh2_signalops_staging_route_manifests.sh
require_executable scripts/verify_signalops_shared_postgres_archive_health.sh
require_executable scripts/run_k8s_stripe_webhook_route_parity_smoke.sh
require_executable scripts/run_k8s_capacity_load_validation_smoke.sh
require_executable scripts/verify_k8s_capacity_headroom.sh
require_executable scripts/report_k8s_capacity_requests.sh

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"

base_render="$(kubectl kustomize deploy/kubernetes/base)"
app_render="$(kubectl kustomize deploy/kubernetes/staging/app)"
marketops_jobs_render="$(kubectl kustomize deploy/kubernetes/staging/marketops-jobs)"
marketops_data_render="$(kubectl kustomize deploy/kubernetes/staging/marketops-data)"
connect_render="$(kubectl kustomize deploy/kubernetes/staging/connect)"
connect_broker_render="$(kubectl kustomize deploy/kubernetes/staging/connect-broker)"
raw_worker_render="$(kubectl kustomize deploy/kubernetes/staging/raw-worker)"

count_kind() {
  local rendered="$1"
  local kind="$2"
  printf '%s
' "$rendered" | awk -v kind="$kind" '$0 == "kind: " kind { c++ } END { print c+0 }'
}

[[ "$(count_kind "$app_render" Ingress)" -eq 0 ]] || fail "staging app manifests must not define Ingress before cutover approval"
[[ "$(count_kind "$marketops_jobs_render" Ingress)" -eq 0 ]] || fail "staging MarketOps job manifests must not define Ingress"
[[ "$(count_kind "$marketops_data_render" Ingress)" -eq 0 ]] || fail "staging MarketOps data manifests must not define Ingress"
[[ "$(count_kind "$connect_broker_render" Ingress)" -eq 0 ]] || fail "staging Connect broker manifests must not define Ingress"
[[ "$(count_kind "$connect_broker_render" Service)" -eq 1 ]] || fail "staging Connect broker must define exactly one internal Service"
[[ "$(count_kind "$raw_worker_render" Ingress)" -eq 0 ]] || fail "staging raw-worker manifests must not define Ingress"
[[ "$(count_kind "$raw_worker_render" Service)" -eq 0 ]] || fail "staging raw-worker manifests must not define Service"
grep -q "type: ClusterIP" <<<"$connect_broker_render" || fail "staging Connect broker Service must remain ClusterIP"

for rendered_name in app_render marketops_jobs_render marketops_data_render connect_render connect_broker_render raw_worker_render; do
  rendered="${!rendered_name}"
  grep -q 'production-cutover-allowed: "false"' <<<"$rendered" || fail "${rendered_name} missing production-cutover-allowed=false guard"
done

combined_render="${app_render}
${marketops_jobs_render}
${marketops_data_render}"
if grep -q 'vault.hashicorp.com/tls-skip-verify' <<<"$combined_render"; then
  fail "K8S manifests must not use OpenBao tls-skip-verify"
fi

grep -q 'signalops/data/k8s/app/signalops-gateway-runtime-staging' <<<"$app_render" || fail "app OpenBao staging runtime path missing"
grep -q 'signalops/data/k8s/marketops/marketops-worker-runtime-staging' <<<"$marketops_jobs_render" || fail "MarketOps OpenBao staging runtime path missing"
grep -q 'SIGNALOPS_DATABASE_MAX_OPEN_CONNS' <<<"$app_render" || fail "gateway shared DB pool cap missing"
grep -q 'SIGNALOPS_MARKETOPS_DATABASE_MAX_OPEN_CONNS' <<<"$app_render" || fail "gateway MarketOps DB pool cap missing"
grep -q 'concurrencyPolicy: Forbid' <<<"$marketops_jobs_render" || fail "MarketOps CronJob concurrencyPolicy guard missing"
grep -q 'suspend: true' <<<"$marketops_jobs_render" || fail "MarketOps staged CronJobs must remain suspended before cutover"

scripts/verify_k8s_base_scaffold.sh >/dev/null
scripts/verify_k8s_staging_app_manifests.sh >/dev/null
scripts/verify_k8s_marketops_scheduled_jobs_manifests.sh >/dev/null
scheduler_parity_report="$(scripts/verify_k8s_marketops_scheduler_parity_coverage.sh)"
scheduler_parity_status="$(printf '%s\n' "$scheduler_parity_report" | awk -F= '$1=="status"{print $2; exit}')"
scheduler_parity_missing_cronjob="$(printf '%s\n' "$scheduler_parity_report" | awk -F= '$1=="missing_cronjob"{print $2; exit}')"
scheduler_parity_missing_entrypoint="$(printf '%s\n' "$scheduler_parity_report" | awk -F= '$1=="missing_entrypoint"{print $2; exit}')"
scheduler_parity_pending=true
if [[ "$scheduler_parity_status" == "complete" && "$scheduler_parity_missing_cronjob" == "none" && "$scheduler_parity_missing_entrypoint" == "none" ]]; then
  scheduler_parity_pending=false
fi
scripts/verify_k8s_marketops_dedicated_staging_data_manifests.sh >/dev/null
scripts/verify_k8s_signalops_connect_staging_manifests.sh >/dev/null
scripts/verify_k8s_signalops_connect_broker_manifests.sh >/dev/null
scripts/verify_k8s_signalops_raw_worker_manifests.sh >/dev/null
scripts/verify_k8s_mesh4_ingress_dns_cutover_plan.sh >/dev/null
scripts/verify_k8s_mesh1_istio_readiness.sh >/dev/null
scripts/verify_k8s_mesh2_signalops_staging_route_manifests.sh >/dev/null
keycloak_oidc_status="verified"
if ! scripts/verify_keycloak_oidc_discovery_reachability.sh >/dev/null 2>/tmp/signalops-keycloak-oidc-readiness.err; then
  keycloak_oidc_status="blocked"
fi

shared_postgres_archive_health="not_checked"
shared_postgres_archive_reason="not_checked"
shared_postgres_archive_report="$(scripts/verify_signalops_shared_postgres_archive_health.sh 2>/tmp/signalops-shared-postgres-archive-health.err || true)"
if [[ -n "$shared_postgres_archive_report" ]]; then
  shared_postgres_archive_health="$(printf '%s\n' "$shared_postgres_archive_report" | awk -F= '$1=="status"{print $2; exit}')"
  shared_postgres_archive_reason="$(printf '%s\n' "$shared_postgres_archive_report" | awk -F= '$1=="reason"{print $2; exit}')"
else
  shared_postgres_archive_health="unknown"
  shared_postgres_archive_reason="$(tr '\n' ' ' </tmp/signalops-shared-postgres-archive-health.err | sed 's/[[:space:]]\+/ /g' | sed 's/[=,]/_/g')"
fi


capacity_headroom_report="$(scripts/verify_k8s_capacity_headroom.sh 2>/tmp/signalops-k8s-capacity-headroom.err || true)"
capacity_headroom_status="unknown"
capacity_headroom_cpu_pct="unknown"
capacity_headroom_memory_pct="unknown"
if [[ -n "$capacity_headroom_report" ]]; then
  capacity_headroom_status="$(printf '%s\n' "$capacity_headroom_report" | awk -F= '$1=="status"{print $2; exit}')"
  capacity_headroom_cpu_pct="$(printf '%s\n' "$capacity_headroom_report" | awk -F= '$1=="cpu_request_pct"{print $2; exit}')"
  capacity_headroom_memory_pct="$(printf '%s\n' "$capacity_headroom_report" | awk -F= '$1=="memory_request_pct"{print $2; exit}')"
fi

cat <<EOF
signalops_k8s_production_cutover_readiness_report
production_cutover_allowed=false
compose_systemd_production_authority=true
shared_postgres_archive_health=${shared_postgres_archive_health}
shared_postgres_archive_reason=${shared_postgres_archive_reason}
k8s_base_scaffold=verified
k8s_app_manifest_guard=verified
k8s_marketops_jobs_manifest_guard=verified
k8s_marketops_data_manifest_guard=verified
k8s_signal_connect_manifest_guard=verified
openbao_ca_trust=verified
backup_restore_current=verified_2026-09-12
proven_app_parity=port_forward_unauthenticated+mesh_authenticated_https
proven_scheduler_parity=no_provider_and_one_provider_fmp_smoke
pending_authenticated_keycloak_k8s_parity=false
pending_authenticated_keycloak_mesh_route_parity=false
keycloak_oidc_discovery_reachability=$keycloak_oidc_status
pending_broader_marketops_scheduler_parity=${scheduler_parity_pending}
k8s_marketops_scheduler_parity_coverage=${scheduler_parity_status}
k8s_marketops_scheduler_missing_cronjob=${scheduler_parity_missing_cronjob}
k8s_marketops_scheduler_missing_entrypoint=${scheduler_parity_missing_entrypoint}
pending_signal_connect_ingestion_shadow=false
proven_signal_connect_ingestion_shadow=sequential_openbao_broker_db_2026-09-13
k8s_raw_worker_scaffold=verified
pending_raw_worker_processing_shadow=false
proven_raw_worker_processing_shadow=one_message_redpanda_signal_2026-09-13
mesh1_istio_control_plane=verified_2026-09-13
proven_service_mesh_signalops_route_parity=authenticated_keycloak_playwright_2026-09-13
pending_service_mesh_ingress_dns_cutover_plan=false
proven_service_mesh_ingress_dns_cutover_plan=rollback_plan_verified_2026-09-13
pending_stripe_webhook_k8s_route_parity=false
proven_stripe_webhook_k8s_route_parity=synthetic_checkout_completed_2026-09-13
proven_k8s_route_load_smoke=health_ready_webhook_2026-09-13
k8s_capacity_headroom_status=${capacity_headroom_status}
k8s_capacity_headroom_cpu_request_pct=${capacity_headroom_cpu_pct}
k8s_capacity_headroom_memory_request_pct=${capacity_headroom_memory_pct}
k8s_capacity_request_report=available
k8s_capacity_remediation_plan=prepared_2026-09-13
pending_capacity_load_validation=true
EOF

if [[ "$mode" == "--strict" ]]; then
  fail "strict production cutover readiness is intentionally not satisfied; see pending_* rows above"
fi
