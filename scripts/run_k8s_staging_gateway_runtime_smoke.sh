#!/usr/bin/env bash
set -euo pipefail

RUNTIME_ENV_FILE="${1:-${SIGNALOPS_K8S_STAGING_RUNTIME_ENV_FILE:-/etc/signalops/openbao-signalops-app-staging-runtime.env}}"
NAMESPACE="${SIGNALOPS_K8S_STAGING_NAMESPACE:-signalops-app}"
MANIFEST_DIR="${SIGNALOPS_K8S_STAGING_APP_MANIFEST_DIR:-deploy/kubernetes/staging/app}"

fail() {
  echo "signalops_k8s_staging_gateway_runtime_smoke_failed: $*" >&2
  exit 1
}

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"

cleanup() {
  kubectl scale deployment/signalops-web deployment/signalops-gateway -n "$NAMESPACE" --replicas=0 >/dev/null 2>&1 || true
}
trap cleanup EXIT

scripts/provision_openbao_signalops_app_runtime_staging.sh "$RUNTIME_ENV_FILE"
scripts/verify_k8s_staging_app_manifests.sh
kubectl apply -k "$MANIFEST_DIR" >/dev/null
kubectl scale deployment/signalops-web deployment/signalops-gateway -n "$NAMESPACE" --replicas=0 >/dev/null
kubectl scale deployment/signalops-gateway -n "$NAMESPACE" --replicas=1 >/dev/null
kubectl rollout status deployment/signalops-gateway -n "$NAMESPACE" --timeout=120s
kubectl exec -n "$NAMESPACE" deployment/signalops-gateway -c gateway -- python3 - <<'PYSMOKE' >/dev/null
from urllib.request import urlopen
for path in ('/healthz', '/readyz'):
    with urlopen('http://127.0.0.1:8080' + path, timeout=3) as response:
        if response.status != 200:
            raise SystemExit(f'{path} returned {response.status}')
PYSMOKE
cleanup
cat <<EOF
signalops_k8s_staging_gateway_runtime_smoke_verified
namespace=${NAMESPACE}
gateway_ready=true
healthz=true
readyz=true
provider_polling=false
production_cutover_allowed=false
scaled_back_to_zero=true
EOF
