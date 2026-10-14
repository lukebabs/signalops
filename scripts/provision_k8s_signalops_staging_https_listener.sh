#!/usr/bin/env bash
set -euo pipefail

GATEWAY_NAMESPACE="${SIGNALOPS_K8S_ISTIO_NAMESPACE:-istio-system}"
GATEWAY_NAME="${SIGNALOPS_K8S_ISTIO_GATEWAY:-public-ingress}"
LISTENER_NAME="${SIGNALOPS_K8S_SIGNALOPS_STAGING_HTTPS_LISTENER:-signalops-staging-https}"
HOSTNAME="${SIGNALOPS_K8S_SIGNALOPS_STAGING_HOSTNAME:-signalops-staging.syncratic.co}"
TLS_SECRET_NAMESPACE="${SIGNALOPS_K8S_STAGING_NAMESPACE:-signalops-app}"
TLS_SECRET_NAME="${SIGNALOPS_K8S_SIGNALOPS_STAGING_TLS_SECRET:-signalops-staging-tls}"

fail() {
  echo "signalops_k8s_staging_https_listener_failed: $*" >&2
  exit 1
}

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"

existing="$(kubectl get gateway "$GATEWAY_NAME" -n "$GATEWAY_NAMESPACE" -o jsonpath='{range .spec.listeners[*]}{.name}{"\n"}{end}' 2>/dev/null || true)"
if printf '%s
' "$existing" | grep -qx "$LISTENER_NAME"; then
  cat <<EOF
signalops_k8s_staging_https_listener_verified
gateway=${GATEWAY_NAMESPACE}/${GATEWAY_NAME}
listener=${LISTENER_NAME}
hostname=${HOSTNAME}
tls_secret=${TLS_SECRET_NAMESPACE}/${TLS_SECRET_NAME}
changed=false
production_cutover_allowed=false
EOF
  exit 0
fi

patch="$(python3 - <<PYJSON
import json
print(json.dumps([{
  "op": "add",
  "path": "/spec/listeners/-",
  "value": {
    "name": "$LISTENER_NAME",
    "hostname": "$HOSTNAME",
    "port": 443,
    "protocol": "HTTPS",
    "tls": {
      "mode": "Terminate",
      "certificateRefs": [{
        "group": "",
        "kind": "Secret",
        "name": "$TLS_SECRET_NAME",
        "namespace": "$TLS_SECRET_NAMESPACE",
      }],
    },
    "allowedRoutes": {
      "namespaces": {
        "from": "Selector",
        "selector": {"matchLabels": {"syncratic.io/public-ingress": "true"}},
      },
    },
  },
}]))
PYJSON
)"

kubectl patch gateway "$GATEWAY_NAME" -n "$GATEWAY_NAMESPACE" --type=json -p="$patch" >/dev/null

cat <<EOF
signalops_k8s_staging_https_listener_verified
gateway=${GATEWAY_NAMESPACE}/${GATEWAY_NAME}
listener=${LISTENER_NAME}
hostname=${HOSTNAME}
tls_secret=${TLS_SECRET_NAMESPACE}/${TLS_SECRET_NAME}
changed=true
production_cutover_allowed=false
EOF
