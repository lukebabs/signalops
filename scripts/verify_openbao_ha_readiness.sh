#!/usr/bin/env bash
set -euo pipefail

OPENBAO_NAMESPACE="${OPENBAO_NAMESPACE:-openbao}"
MIN_SERVER_PODS="${OPENBAO_MIN_SERVER_PODS:-3}"

fail() {
  echo "openbao_ha_readiness_failed: $*" >&2
  exit 1
}

info() {
  echo "$*"
}

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"

kubectl get namespace "$OPENBAO_NAMESPACE" >/dev/null || fail "namespace ${OPENBAO_NAMESPACE} not found"

for svc in openbao openbao-active openbao-standby openbao-agent-injector-svc; do
  kubectl get service "$svc" -n "$OPENBAO_NAMESPACE" >/dev/null || fail "service ${OPENBAO_NAMESPACE}/${svc} not found"
done

kubectl get mutatingwebhookconfiguration openbao-agent-injector-cfg >/dev/null || fail "openbao-agent-injector-cfg webhook not found"

kubectl get deployment openbao-agent-injector -n "$OPENBAO_NAMESPACE" >/dev/null || fail "openbao-agent-injector deployment not found"
injector_available="$(kubectl get deployment openbao-agent-injector -n "$OPENBAO_NAMESPACE" -o jsonpath='{.status.availableReplicas}')"
if [[ -z "${injector_available}" || "${injector_available}" -lt 1 ]]; then
  fail "openbao-agent-injector has no available replicas"
fi

pod_lines="$(kubectl get pods -n "$OPENBAO_NAMESPACE" --no-headers || true)"
[[ -n "$pod_lines" ]] || fail "no pods found in ${OPENBAO_NAMESPACE}"

server_ready_count="$(printf '%s\n' "$pod_lines" | awk '!/injector/ && $2 ~ /^[0-9]+\/[0-9]+$/ { split($2,a,"/"); if (a[1] == a[2] && $3 == "Running") c++ } END { print c+0 }')"
if [[ "$server_ready_count" -lt "$MIN_SERVER_PODS" ]]; then
  fail "ready OpenBao server pods ${server_ready_count}/${MIN_SERVER_PODS}; HA gate requires at least ${MIN_SERVER_PODS} ready non-injector server pods"
fi

webhook_prefix="$(kubectl get mutatingwebhookconfiguration openbao-agent-injector-cfg -o jsonpath='{.webhooks[0].name}')"
if [[ "$webhook_prefix" != *"vault.hashicorp.com"* ]]; then
  fail "unexpected injector webhook name: ${webhook_prefix}; verify annotation prefix before workload conversion"
fi

cat <<EOF
openbao_ha_readiness_surface_verified
namespace=${OPENBAO_NAMESPACE}
server_ready_pods=${server_ready_count}
injector_available_replicas=${injector_available}
services=openbao,openbao-active,openbao-standby,openbao-agent-injector-svc
webhook=${webhook_prefix}
annotation_prefix=vault.hashicorp.com
note=This verifies the Kubernetes HA/injector surface only. OpenBao seal state, storage snapshots, audit devices, policies, and Kubernetes auth roles still require authenticated OpenBao validation.
EOF
