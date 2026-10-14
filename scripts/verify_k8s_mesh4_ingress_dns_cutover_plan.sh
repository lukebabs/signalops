#!/usr/bin/env bash
set -euo pipefail

fail() {
  printf 'signalops_k8s_mesh4_ingress_dns_cutover_plan_failed: %s
' "$*" >&2
  exit 1
}

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_dir"

plan="docs/projects/subscriber_project/k8s_mesh4_ingress_dns_cutover_plan_2026-09-13.md"
[[ -r "$plan" ]] || fail "cutover plan is missing"

for required in   'Docker Compose'   'Docker Traefik'   'Istio Gateway API'   'K3s ingress authority'   'signalops.syncratic.io'   'Stripe webhook parity'   'Rollback path'   'production DNS changes'   'production_cutover_allowed=false'
do
  grep -q "$required" "$plan" || fail "plan missing required marker: $required"
done

[[ -x scripts/verify_k8s_mesh1_istio_readiness.sh ]] || fail "Mesh-1 verifier missing"
[[ -x scripts/verify_k8s_mesh2_signalops_staging_route_manifests.sh ]] || fail "Mesh-2 route verifier missing"
[[ -x scripts/run_k8s_mesh3_keycloak_staging_route_smoke.sh ]] || fail "Mesh-3 authenticated route smoke missing"
[[ -x scripts/verify_k8s_production_cutover_readiness.sh ]] || fail "production-readiness verifier missing"

staging_route="deploy/kubernetes/staging/mesh-route/signalops-istio-staging-route.yaml"
[[ -r "$staging_route" ]] || fail "staging route manifest missing"
grep -q 'signalops-staging.syncratic.co' "$staging_route" || fail "staging route hostname missing"
if awk '
  $1 == "hostnames:" { in_hostnames=1; next }
  in_hostnames && $1 !~ /^-/ { in_hostnames=0 }
  in_hostnames && $0 ~ /- signalops[.]syncratic[.]io$/ { found=1 }
  END { exit found ? 0 : 1 }
' "$staging_route"; then
  fail "staging route must not bind production hostname"
fi

echo 'signalops_k8s_mesh4_ingress_dns_cutover_plan_verified'
echo 'production_dns_changed=false'
echo 'production_traffic_moved=false'
echo 'compose_traefik_rollback_required=true'
echo 'staging_hostname=signalops-staging.syncratic.co'
echo 'production_hostname=signalops.syncratic.io'
echo 'production_cutover_allowed=false'
