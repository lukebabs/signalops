#!/usr/bin/env bash
set -euo pipefail

OPENBAO_NAMESPACE="${OPENBAO_NAMESPACE:-openbao}"
MIN_SERVER_PODS="${OPENBAO_MIN_SERVER_PODS:-3}"
ALLOW_SINGLE_NODE_STAGING="false"

fail() {
  echo "openbao_ha_readiness_failed: $*" >&2
  exit 1
}

usage() {
  cat <<EOF
Usage: $0 [--allow-single-node-staging]

Verifies the OpenBao Kubernetes surface expected by SignalOps.

Default mode enforces the production HA gate: at least ${MIN_SERVER_PODS} ready
non-injector OpenBao server pods.

--allow-single-node-staging allows exactly the documented non-production staging
exception: at least one ready server pod is accepted, and the output is clearly
marked single_node_staging_exception. This must not be used to approve
production workload cutover.
EOF
}

for arg in "$@"; do
  case "$arg" in
    --allow-single-node-staging)
      ALLOW_SINGLE_NODE_STAGING="true"
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      usage >&2
      fail "unknown argument: $arg"
      ;;
  esac
done

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
  if [[ "$ALLOW_SINGLE_NODE_STAGING" == "true" && "$server_ready_count" -ge 1 ]]; then
    readiness_mode="single_node_staging_exception"
  else
    fail "ready OpenBao server pods ${server_ready_count}/${MIN_SERVER_PODS}; HA gate requires at least ${MIN_SERVER_PODS} ready non-injector server pods"
  fi
else
  readiness_mode="production_ha"
fi

webhook_prefix="$(kubectl get mutatingwebhookconfiguration openbao-agent-injector-cfg -o jsonpath='{.webhooks[0].name}')"
if [[ "$webhook_prefix" != *"vault.hashicorp.com"* ]]; then
  fail "unexpected injector webhook name: ${webhook_prefix}; verify annotation prefix before workload conversion"
fi

cat <<EOF
openbao_readiness_surface_verified
mode=${readiness_mode}
production_cutover_allowed=false
namespace=${OPENBAO_NAMESPACE}
server_ready_pods=${server_ready_count}
required_production_server_pods=${MIN_SERVER_PODS}
injector_available_replicas=${injector_available}
services=openbao,openbao-active,openbao-standby,openbao-agent-injector-svc
webhook=${webhook_prefix}
annotation_prefix=vault.hashicorp.com
note=This verifies the Kubernetes OpenBao/injector surface only. Single-node staging mode is not production HA. OpenBao seal state, storage snapshots, audit devices, policies, and Kubernetes auth roles still require authenticated OpenBao validation.
EOF
