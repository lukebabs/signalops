#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${SIGNALOPS_K8S_STAGING_NAMESPACE:-signalops-app}"
MANIFEST_DIR="${SIGNALOPS_K8S_STAGING_APP_MANIFEST_DIR:-deploy/kubernetes/staging/app}"
LOCAL_PORT="${SIGNALOPS_K8S_STAGING_WEB_PORT:-}"
PYTHON_BIN="${SIGNALOPS_PLAYWRIGHT_PYTHON:-.venv/bin/python}"

fail() {
  echo "signalops_k8s_staging_app_parity_smoke_failed: $*" >&2
  exit 1
}

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"
[[ -x "$PYTHON_BIN" ]] || fail "Playwright Python runtime not found at $PYTHON_BIN"
if [[ -z "$LOCAL_PORT" ]]; then
  LOCAL_PORT="$($PYTHON_BIN - <<'PY'
import socket

for port in range(18080, 18120):
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        try:
            sock.bind(("127.0.0.1", port))
        except OSError:
            continue
        print(port)
        raise SystemExit(0)
raise SystemExit(1)
PY
)" || fail "no free local staging port found in 18080-18119"
fi
[[ "$LOCAL_PORT" =~ ^[0-9]+$ ]] || fail "SIGNALOPS_K8S_STAGING_WEB_PORT must be numeric"

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_dir"

port_forward_log="$(mktemp -t signalops-k8s-app-parity-port-forward.XXXXXX.log)"
port_forward_pid=""

cleanup() {
  if [[ -n "$port_forward_pid" ]]; then
    kill "$port_forward_pid" >/dev/null 2>&1 || true
    wait "$port_forward_pid" >/dev/null 2>&1 || true
  fi
  kubectl scale deployment/signalops-web deployment/signalops-gateway -n "$NAMESPACE" --replicas=0 >/dev/null 2>&1 || true
  rm -f "$port_forward_log"
}
trap cleanup EXIT

"$PYTHON_BIN" -c "import playwright" >/dev/null 2>&1 || fail "Playwright is not installed in $PYTHON_BIN"

scripts/verify_k8s_staging_app_manifests.sh
kubectl apply -k "$MANIFEST_DIR" >/dev/null
kubectl scale deployment/signalops-web deployment/signalops-gateway -n "$NAMESPACE" --replicas=1 >/dev/null
kubectl rollout status deployment/signalops-gateway -n "$NAMESPACE" --timeout=120s
kubectl rollout status deployment/signalops-web -n "$NAMESPACE" --timeout=120s

kubectl port-forward --address 127.0.0.1 -n "$NAMESPACE" service/signalops-web "${LOCAL_PORT}:8080" >"$port_forward_log" 2>&1 &
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

if ! kill -0 "$port_forward_pid" >/dev/null 2>&1; then
  sed -n '1,120p' "$port_forward_log" >&2 || true
  fail "kubectl port-forward exited before the Playwright smoke could run"
fi

SIGNALOPS_K8S_STAGING_BASE_URL="http://127.0.0.1:${LOCAL_PORT}" \
  "$PYTHON_BIN" -m pytest -q python/tests/test_k8s_staging_app_parity.py

cat <<EOF
signalops_k8s_staging_app_parity_smoke_verified
namespace=${NAMESPACE}
base_url=http://127.0.0.1:${LOCAL_PORT}
web_ready=true
gateway_proxy_healthz=true
gateway_proxy_readyz=true
authenticated_keycloak_redirect=false
provider_polling=false
production_cutover_allowed=false
scaled_back_to_zero=true
EOF
