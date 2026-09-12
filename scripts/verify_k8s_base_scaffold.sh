#!/usr/bin/env bash
set -euo pipefail

namespaces=(
  signalops-app
  signalops-connect
  signalops-cyberops
  signalops-data
  signalops-identity
  signalops-marketops
  signalops-observability
)

expected_service_accounts=29
expected_network_policies=15
expected_policy_configmaps=2

fail() {
  echo "signalops_k8s_base_scaffold_failed: $*" >&2
  exit 1
}

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"

for ns in "${namespaces[@]}"; do
  kubectl get namespace "$ns" >/dev/null || fail "namespace missing: $ns"
done

service_account_count=0
network_policy_count=0
policy_configmap_count=0
pod_count=0

for ns in "${namespaces[@]}"; do
  service_account_count=$((service_account_count + $(kubectl get serviceaccount -n "$ns" --no-headers 2>/dev/null | awk '$1 != "default" { c++ } END { print c+0 }')))
  network_policy_count=$((network_policy_count + $(kubectl get networkpolicy -n "$ns" --no-headers 2>/dev/null | wc -l | tr -d ' ')))
  policy_configmap_count=$((policy_configmap_count + $(kubectl get configmap -n "$ns" --no-headers 2>/dev/null | awk '$1 ~ /^signalops-/ { c++ } END { print c+0 }')))
  pod_count=$((pod_count + $(kubectl get pods -n "$ns" --no-headers 2>/dev/null | wc -l | tr -d ' ')))
done

if [[ "$service_account_count" -ne "$expected_service_accounts" ]]; then
  fail "service account count ${service_account_count}/${expected_service_accounts}"
fi

if [[ "$network_policy_count" -ne "$expected_network_policies" ]]; then
  fail "network policy count ${network_policy_count}/${expected_network_policies}"
fi

if [[ "$policy_configmap_count" -ne "$expected_policy_configmaps" ]]; then
  fail "policy ConfigMap count ${policy_configmap_count}/${expected_policy_configmaps}"
fi

if [[ "$pod_count" -ne 0 ]]; then
  fail "expected zero SignalOps workload pods in K8S-1 scaffold, found ${pod_count}"
fi

cat <<EOF
signalops_k8s_base_scaffold_verified
namespaces=${#namespaces[@]}
service_accounts=${service_account_count}
network_policies=${network_policy_count}
policy_configmaps=${policy_configmap_count}
workload_pods=${pod_count}
production_cutover_allowed=false
EOF
