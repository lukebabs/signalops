#!/usr/bin/env bash
set -euo pipefail

fail() {
  echo "signalops_k8s_production_app_cutover_package_failed: $*" >&2
  exit 1
}

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_dir"

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"
[[ -x scripts/publish_k8s_signalops_app_images.sh ]] || fail "app image publish script missing"
[[ -x scripts/provision_k8s_signalops_production_https_listener.sh ]] || fail "production HTTPS listener script missing"

app_render="$(kubectl kustomize deploy/kubernetes/production/app)"
route_render="$(kubectl kustomize deploy/kubernetes/production/mesh-route)"
combined="${app_render}
${route_render}"

for required in   'name: signalops-gateway-production'   'name: signalops-web-production'   'image: ghcr.io/syncratic-inc/signalops-gateway:production'   'image: ghcr.io/syncratic-inc/signalops-web:production'   'signalops/data/k8s/app/signalops-gateway-runtime-production'   'value: k8s-production'   'signalops.syncratic.io/production-cutover-allowed: "true"'   'hostPath:'   'path: /run/signalops'   'kubernetes.io/metadata.name: signalops-data'   'marketops-postgres-production'   'port: 5432'   'name: signalops-production-route'   'signalops.syncratic.io'   'name: signalops-production-tls'   'name: letsencrypt-prod'
do
  grep -q "$required" <<<"$combined" || fail "render missing required marker: $required"
done

if grep -q 'signalops-staging.syncratic.co' <<<"$combined"; then
  fail "production render must not bind staging hostname"
fi
if grep -q '192.168.2.5/32' <<<"$combined" || grep -q 'port: 1543' <<<"$combined"; then
  fail "production app manifest must target Kubernetes data services, not Docker host database ports"
fi
if grep -q 'signalops/data/k8s/app/signalops-gateway-runtime-staging' <<<"$combined"; then
  fail "production render must not use staging OpenBao runtime path"
fi
if grep -q 'ghcr.io/syncratic-inc/signalops-gateway:staging' <<<"$combined" || grep -q 'ghcr.io/syncratic-inc/signalops-web:staging' <<<"$combined"; then
  fail "production render must not use staging image tags"
fi

kubectl apply --dry-run=server -k deploy/kubernetes/production/app >/dev/null
kubectl apply --dry-run=server -k deploy/kubernetes/production/mesh-route >/dev/null

cat <<EOF
signalops_k8s_production_app_cutover_package_verified
production_app_overlay=verified
production_mesh_route_overlay=verified
openbao_runtime_path=signalops/data/k8s/app/signalops-gateway-runtime-production
images=ghcr.io/syncratic-inc/signalops-gateway:production,ghcr.io/syncratic-inc/signalops-web:production
host=signalops.syncratic.io
server_side_dry_run=passed
production_traffic_moved=false
production_cutover_allowed=false
EOF
