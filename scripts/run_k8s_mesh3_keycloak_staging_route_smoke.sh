#!/usr/bin/env bash
set -euo pipefail

APP_NAMESPACE="${SIGNALOPS_K8S_STAGING_NAMESPACE:-signalops-app}"
APP_MANIFEST_DIR="${SIGNALOPS_K8S_STAGING_APP_MANIFEST_DIR:-deploy/kubernetes/staging/app}"
MESH_MANIFEST_DIR="${SIGNALOPS_K8S_MESH2_MANIFEST_DIR:-deploy/kubernetes/staging/mesh-route}"
PYTHON_BIN="${SIGNALOPS_PLAYWRIGHT_PYTHON:-.venv/bin/python}"
DOTENV_PATH="${SIGNALOPS_E2E_ENV_FILE:-.env}"
HOSTNAME="${SIGNALOPS_K8S_SIGNALOPS_STAGING_HOSTNAME:-signalops-staging.syncratic.co}"
GATEWAY_NAMESPACE="${SIGNALOPS_K8S_ISTIO_NAMESPACE:-istio-system}"
GATEWAY_NAME="${SIGNALOPS_K8S_ISTIO_GATEWAY:-public-ingress}"
GATEWAY_SERVICE="${SIGNALOPS_K8S_ISTIO_GATEWAY_SERVICE:-public-ingress-istio}"

fail() {
  echo "signalops_k8s_mesh3_keycloak_route_smoke_failed: $*" >&2
  exit 1
}

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"
[[ -x "$PYTHON_BIN" ]] || fail "Playwright Python runtime not found at $PYTHON_BIN"
"$PYTHON_BIN" -c "import playwright" >/dev/null 2>&1 || fail "Playwright is not installed in $PYTHON_BIN"

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_dir"

# shellcheck source=./lib/dotenv.sh
source "$repo_dir/scripts/lib/dotenv.sh"
load_dotenv "$DOTENV_PATH"
export SIGNALOPS_B2C_WEB="${SIGNALOPS_B2C_WEB:-${SYNCRATIC_QA_CLIENT:-}}"
export SIGNALOPS_B2C_WEB_PASS="${SIGNALOPS_B2C_WEB_PASS:-${SYNCRATIC_QA_PASS:-}}"
: "${SIGNALOPS_B2C_WEB:?SIGNALOPS_B2C_WEB or SYNCRATIC_QA_CLIENT is required}"
: "${SIGNALOPS_B2C_WEB_PASS:?SIGNALOPS_B2C_WEB_PASS or SYNCRATIC_QA_PASS is required}"

cleanup() {
  kubectl scale deployment/signalops-web deployment/signalops-gateway -n "$APP_NAMESPACE" --replicas=0 >/dev/null 2>&1 || true
}
trap cleanup EXIT

scripts/verify_keycloak_oidc_discovery_reachability.sh >/dev/null
scripts/verify_k8s_mesh1_istio_readiness.sh >/dev/null
scripts/verify_k8s_staging_app_manifests.sh "$APP_MANIFEST_DIR" >/dev/null
scripts/verify_k8s_mesh2_signalops_staging_route_manifests.sh "$MESH_MANIFEST_DIR" >/dev/null

kubectl apply -k "$APP_MANIFEST_DIR" >/dev/null
kubectl apply -k "$MESH_MANIFEST_DIR" >/dev/null
app_runtime_env="${SIGNALOPS_K8S_APP_RUNTIME_ENV_FILE:-/tmp/signalops-openbao-app-runtime-staging.env}"
scripts/create_k8s_signalops_app_runtime_staging_env.sh "$DOTENV_PATH" "$app_runtime_env" >/dev/null
scripts/provision_openbao_signalops_app_runtime_staging.sh "$app_runtime_env" >/dev/null
scripts/bootstrap_k8s_staging_enrollment_schema.sh >/dev/null
kubectl wait --for=condition=Ready certificate/signalops-staging-tls -n "$APP_NAMESPACE" --timeout=120s >/dev/null
scripts/provision_k8s_signalops_staging_https_listener.sh >/dev/null
kubectl scale deployment/signalops-web deployment/signalops-gateway -n "$APP_NAMESPACE" --replicas=1 >/dev/null
kubectl rollout status deployment/signalops-gateway -n "$APP_NAMESPACE" --timeout=120s
kubectl rollout status deployment/signalops-web -n "$APP_NAMESPACE" --timeout=120s

route_accepted="$(kubectl get httproute signalops-staging-route -n "$APP_NAMESPACE" -o jsonpath='{.status.parents[?(@.parentRef.sectionName=="signalops-staging-https")].conditions[?(@.type=="Accepted")].status}' 2>/dev/null || true)"
route_refs="$(kubectl get httproute signalops-staging-route -n "$APP_NAMESPACE" -o jsonpath='{.status.parents[?(@.parentRef.sectionName=="signalops-staging-https")].conditions[?(@.type=="ResolvedRefs")].status}' 2>/dev/null || true)"
[[ "$route_accepted" == "True" ]] || fail "HTTPRoute was not accepted by the Istio HTTPS Gateway listener"
[[ "$route_refs" == "True" ]] || fail "HTTPRoute HTTPS backend references were not resolved"

gateway_ip="$(kubectl get gateway "$GATEWAY_NAME" -n "$GATEWAY_NAMESPACE" -o jsonpath='{.status.addresses[0].value}' 2>/dev/null || true)"
if [[ -z "$gateway_ip" ]]; then
  gateway_ip="$(kubectl get service "$GATEWAY_SERVICE" -n "$GATEWAY_NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null || true)"
fi
[[ -n "$gateway_ip" ]] || fail "could not resolve Istio Gateway address"

export SIGNALOPS_K8S_MESH3_BASE_URL="https://${HOSTNAME}"
export SIGNALOPS_K8S_MESH3_HOST_RESOLVER_IP="$gateway_ip"
export SIGNALOPS_E2E_ARTIFACT_DIR="${SIGNALOPS_E2E_ARTIFACT_DIR:-/tmp/signalops-mesh3-auth-e2e-artifacts}"
PYTHONDONTWRITEBYTECODE=1 "$PYTHON_BIN" -m pytest -q python/tests/test_k8s_mesh3_keycloak_staging_route_parity.py

cat <<EOF
signalops_k8s_mesh3_keycloak_route_smoke_verified
namespace=${APP_NAMESPACE}
staging_hostname=${HOSTNAME}
gateway=${GATEWAY_NAMESPACE}/${GATEWAY_NAME}
gateway_ip=${gateway_ip}
https_listener=signalops-staging-https
authenticated_keycloak_redirect=true
provider_polling=false
production_cutover_allowed=false
scaled_back_to_zero=true
EOF
