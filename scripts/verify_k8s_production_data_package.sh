#!/usr/bin/env bash
set -euo pipefail

fail() {
  echo "signalops_k8s_production_data_package_failed: $*" >&2
  exit 1
}

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_dir"

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"
rendered="$(kubectl kustomize deploy/kubernetes/production/data)"

for required in \
  'name: signalops-postgres-production' \
  'name: signalops-timescaledb-production' \
  'name: marketops-postgres-production' \
  'name: marketops-timescaledb-production' \
  'image: ghcr.io/syncratic-inc/signalops-postgres-pgbackrest:production' \
  'image: ghcr.io/syncratic-inc/signalops-marketops-timescaledb-pgbackrest:production' \
  'storageClassName: syncratic-data-retain' \
  'vault.hashicorp.com/role: signalops-data' \
  'signalops/data/k8s/data/signalops-databases-runtime-production' \
  'serviceAccountName: signalops-data-secret-reader' \
  'kind: NetworkPolicy' \
  'production-cutover-allowed: "false"'
do
  grep -q "$required" <<<"$rendered" || fail "render missing required marker: $required"
done

if grep -q 'storageClassName: local-path' <<<"$rendered"; then
  fail "production data overlay must not use local-path storage"
fi
if grep -q 'secretKeyRef' <<<"$rendered"; then
  fail "production data overlay must not use Kubernetes Secret password references; use OpenBao injection"
fi
if grep -q 'type: LoadBalancer' <<<"$rendered" || grep -q 'kind: Ingress' <<<"$rendered"; then
  fail "production data overlay must not expose databases externally"
fi

kubectl apply --dry-run=server -k deploy/kubernetes/production/data >/dev/null

cat <<EOF
signalops_k8s_production_data_package_verified
production_data_overlay=verified
namespace=signalops-data
storage_class=syncratic-data-retain
secret_source=openbao
openbao_path=signalops/data/k8s/data/signalops-databases-runtime-production
services=signalops-postgres-production,signalops-timescaledb-production,marketops-postgres-production,marketops-timescaledb-production
server_side_dry_run=passed
production_traffic_moved=false
production_cutover_allowed=false
EOF
