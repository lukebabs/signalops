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

require_executable scripts/verify_k8s_base_scaffold.sh
require_executable scripts/verify_k8s_staging_app_manifests.sh
require_executable scripts/verify_k8s_marketops_scheduled_jobs_manifests.sh
require_executable scripts/verify_k8s_marketops_scheduler_parity_coverage.sh
require_executable scripts/verify_k8s_marketops_dedicated_staging_data_manifests.sh
require_executable scripts/verify_k8s_mesh1_istio_readiness.sh
require_executable scripts/verify_k8s_mesh2_signalops_staging_route_manifests.sh
require_executable scripts/verify_signalops_shared_postgres_archive_health.sh

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"

base_render="$(kubectl kustomize deploy/kubernetes/base)"
app_render="$(kubectl kustomize deploy/kubernetes/staging/app)"
marketops_jobs_render="$(kubectl kustomize deploy/kubernetes/staging/marketops-jobs)"
marketops_data_render="$(kubectl kustomize deploy/kubernetes/staging/marketops-data)"

count_kind() {
  local rendered="$1"
  local kind="$2"
  printf '%s
' "$rendered" | awk -v kind="$kind" '$0 == "kind: " kind { c++ } END { print c+0 }'
}

[[ "$(count_kind "$app_render" Ingress)" -eq 0 ]] || fail "staging app manifests must not define Ingress before cutover approval"
[[ "$(count_kind "$marketops_jobs_render" Ingress)" -eq 0 ]] || fail "staging MarketOps job manifests must not define Ingress"
[[ "$(count_kind "$marketops_data_render" Ingress)" -eq 0 ]] || fail "staging MarketOps data manifests must not define Ingress"

for rendered_name in app_render marketops_jobs_render marketops_data_render; do
  rendered="${!rendered_name}"
  printf '%s
' "$rendered" | grep -q 'production-cutover-allowed: "false"' || fail "${rendered_name} missing production-cutover-allowed=false guard"
done

if printf '%s
%s
%s
' "$app_render" "$marketops_jobs_render" "$marketops_data_render" | grep -q 'vault.hashicorp.com/tls-skip-verify'; then
  fail "K8S manifests must not use OpenBao tls-skip-verify"
fi

printf '%s
' "$app_render" | grep -q 'signalops/data/k8s/app/signalops-gateway-runtime-staging' || fail "app OpenBao staging runtime path missing"
printf '%s
' "$marketops_jobs_render" | grep -q 'signalops/data/k8s/marketops/marketops-worker-runtime-staging' || fail "MarketOps OpenBao staging runtime path missing"
printf '%s
' "$app_render" | grep -q 'SIGNALOPS_DATABASE_MAX_OPEN_CONNS' || fail "gateway shared DB pool cap missing"
printf '%s
' "$app_render" | grep -q 'SIGNALOPS_MARKETOPS_DATABASE_MAX_OPEN_CONNS' || fail "gateway MarketOps DB pool cap missing"
printf '%s
' "$marketops_jobs_render" | grep -q 'concurrencyPolicy: Forbid' || fail "MarketOps CronJob concurrencyPolicy guard missing"
printf '%s
' "$marketops_jobs_render" | grep -q 'suspend: true' || fail "MarketOps staged CronJobs must remain suspended before cutover"

scripts/verify_k8s_base_scaffold.sh >/dev/null
scripts/verify_k8s_staging_app_manifests.sh >/dev/null
scripts/verify_k8s_marketops_scheduled_jobs_manifests.sh >/dev/null
scheduler_parity_report="$(scripts/verify_k8s_marketops_scheduler_parity_coverage.sh)"
scheduler_parity_status="$(printf '%s\n' "$scheduler_parity_report" | awk -F= '$1=="status"{print $2; exit}')"
scheduler_parity_missing_cronjob="$(printf '%s\n' "$scheduler_parity_report" | awk -F= '$1=="missing_cronjob"{print $2; exit}')"
scheduler_parity_missing_entrypoint="$(printf '%s\n' "$scheduler_parity_report" | awk -F= '$1=="missing_entrypoint"{print $2; exit}')"
scripts/verify_k8s_marketops_dedicated_staging_data_manifests.sh >/dev/null
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
openbao_ca_trust=verified
backup_restore_current=verified_2026-09-12
proven_app_parity=port_forward_unauthenticated+mesh_authenticated_https
proven_scheduler_parity=no_provider_and_one_provider_fmp_smoke
pending_authenticated_keycloak_k8s_parity=false
pending_authenticated_keycloak_mesh_route_parity=false
keycloak_oidc_discovery_reachability=$keycloak_oidc_status
pending_broader_marketops_scheduler_parity=true
k8s_marketops_scheduler_parity_coverage=${scheduler_parity_status}
k8s_marketops_scheduler_missing_cronjob=${scheduler_parity_missing_cronjob}
k8s_marketops_scheduler_missing_entrypoint=${scheduler_parity_missing_entrypoint}
pending_signal_connect_ingestion_shadow=true
mesh1_istio_control_plane=verified_2026-09-13
proven_service_mesh_signalops_route_parity=authenticated_keycloak_playwright_2026-09-13
pending_service_mesh_ingress_dns_cutover_plan=true
pending_capacity_load_validation=true
EOF

if [[ "$mode" == "--strict" ]]; then
  fail "strict production cutover readiness is intentionally not satisfied; see pending_* rows above"
fi
