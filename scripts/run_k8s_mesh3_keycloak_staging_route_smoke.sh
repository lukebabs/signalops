#!/usr/bin/env bash
set -euo pipefail

APP_NAMESPACE="${SIGNALOPS_K8S_STAGING_NAMESPACE:-signalops-app}"
APP_MANIFEST_DIR="${SIGNALOPS_K8S_STAGING_APP_MANIFEST_DIR:-deploy/kubernetes/staging/app}"
MESH_MANIFEST_DIR="${SIGNALOPS_K8S_MESH2_MANIFEST_DIR:-deploy/kubernetes/staging/mesh-route}"
PYTHON_BIN="${SIGNALOPS_PLAYWRIGHT_PYTHON:-.venv/bin/python}"
LOCAL_PORT="${SIGNALOPS_K8S_MESH3_LOCAL_PORT:-}"
DOTENV_PATH="${SIGNALOPS_E2E_ENV_FILE:-.env}"

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

if [[ -z "$LOCAL_PORT" ]]; then
  LOCAL_PORT="$($PYTHON_BIN - <<'PYPORT'
import socket
for port in range(18280, 18320):
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        try:
            sock.bind(("127.0.0.1", port))
        except OSError:
            continue
        print(port)
        raise SystemExit(0)
raise SystemExit(1)
PYPORT
)" || fail "no free local mesh auth smoke port found in 18280-18319"
fi
[[ "$LOCAL_PORT" =~ ^[0-9]+$ ]] || fail "SIGNALOPS_K8S_MESH3_LOCAL_PORT must be numeric"

port_forward_log="$(mktemp -t signalops-k8s-mesh3-port-forward.XXXXXX.log)"
port_forward_pid=""

cleanup() {
  if [[ -n "$port_forward_pid" ]]; then
    kill "$port_forward_pid" >/dev/null 2>&1 || true
    wait "$port_forward_pid" >/dev/null 2>&1 || true
  fi
  kubectl scale deployment/signalops-web deployment/signalops-gateway -n "$APP_NAMESPACE" --replicas=0 >/dev/null 2>&1 || true
  rm -f "$port_forward_log"
}
trap cleanup EXIT

scripts/verify_keycloak_oidc_discovery_reachability.sh >/dev/null
scripts/verify_k8s_mesh1_istio_readiness.sh >/dev/null
scripts/verify_k8s_staging_app_manifests.sh "$APP_MANIFEST_DIR" >/dev/null
scripts/verify_k8s_mesh2_signalops_staging_route_manifests.sh "$MESH_MANIFEST_DIR" >/dev/null

kubectl apply -k "$APP_MANIFEST_DIR" >/dev/null
kubectl apply -k "$MESH_MANIFEST_DIR" >/dev/null
kubectl scale deployment/signalops-web deployment/signalops-gateway -n "$APP_NAMESPACE" --replicas=1 >/dev/null
kubectl rollout status deployment/signalops-gateway -n "$APP_NAMESPACE" --timeout=120s
kubectl rollout status deployment/signalops-web -n "$APP_NAMESPACE" --timeout=120s

route_accepted="$(kubectl get httproute signalops-staging-route -n "$APP_NAMESPACE" -o jsonpath='{.status.parents[0].conditions[?(@.type=="Accepted")].status}' 2>/dev/null || true)"
route_refs="$(kubectl get httproute signalops-staging-route -n "$APP_NAMESPACE" -o jsonpath='{.status.parents[0].conditions[?(@.type=="ResolvedRefs")].status}' 2>/dev/null || true)"
[[ "$route_accepted" == "True" ]] || fail "HTTPRoute was not accepted by the Istio Gateway"
[[ "$route_refs" == "True" ]] || fail "HTTPRoute backend references were not resolved"

kubectl port-forward --address 127.0.0.1 -n istio-system service/public-ingress-istio "${LOCAL_PORT}:80" >"$port_forward_log" 2>&1 &
port_forward_pid="$!"

for _ in $(seq 1 30); do
  if grep -q "Forwarding from 127.0.0.1:${LOCAL_PORT}" "$port_forward_log"; then
    break
  fi
  if ! kill -0 "$port_forward_pid" >/dev/null 2>&1; then
    sed -n '1,120p' "$port_forward_log" >&2 || true
    fail "kubectl port-forward exited before it became ready"
  fi
  sleep 1
done

if ! grep -q "Forwarding from 127.0.0.1:${LOCAL_PORT}" "$port_forward_log"; then
  sed -n '1,120p' "$port_forward_log" >&2 || true
  fail "kubectl port-forward did not become ready"
fi

export SIGNALOPS_K8S_MESH3_LOCAL_PORT="$LOCAL_PORT"
export SIGNALOPS_E2E_ARTIFACT_DIR="${SIGNALOPS_E2E_ARTIFACT_DIR:-/tmp/signalops-mesh3-auth-e2e-artifacts}"
PYTHONDONTWRITEBYTECODE=1 "$PYTHON_BIN" -m pytest -q python/tests/test_k8s_mesh3_keycloak_staging_route_parity.py

cat <<EOF
signalops_k8s_mesh3_keycloak_route_smoke_verified
namespace=${APP_NAMESPACE}
staging_hostname=signalops-staging.syncratic.co
port_forward=istio-system/public-ingress-istio:${LOCAL_PORT}->80
authenticated_keycloak_redirect=true
provider_polling=false
production_cutover_allowed=false
scaled_back_to_zero=true
EOF
