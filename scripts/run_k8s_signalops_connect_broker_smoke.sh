#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${SIGNALOPS_K8S_DATA_NAMESPACE:-signalops-data}"
DEPLOYMENT="${SIGNALOPS_K8S_CONNECT_BROKER_DEPLOYMENT:-signalops-redpanda-staging}"
ENVIRONMENT="${SIGNALOPS_K8S_CONNECT_BROKER_ENVIRONMENT:-kubernetes-staging}"
PARTITIONS="${SIGNALOPS_K8S_CONNECT_BROKER_TOPIC_PARTITIONS:-1}"
REPLICAS="${SIGNALOPS_K8S_CONNECT_BROKER_TOPIC_REPLICAS:-1}"

fail() {
  echo "signalops_k8s_connect_broker_smoke_failed: $*" >&2
  exit 1
}

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"

kubectl get deployment "$DEPLOYMENT" -n "$NAMESPACE" >/dev/null || fail "broker deployment missing: ${NAMESPACE}/${DEPLOYMENT}"
kubectl rollout status "deployment/${DEPLOYMENT}" -n "$NAMESPACE" --timeout=120s >/dev/null

topics=(
  "signalops.${ENVIRONMENT}.raw.v1"
  "signalops.${ENVIRONMENT}.normalized.v1"
  "signalops.${ENVIRONMENT}.signal.v1"
  "signalops.${ENVIRONMENT}.artifact.v1"
  "signalops.${ENVIRONMENT}.graph_mutation.v1"
  "signalops.${ENVIRONMENT}.insight_candidate.v1"
  "signalops.${ENVIRONMENT}.retry.algorithm.v1"
  "signalops.${ENVIRONMENT}.dlq.algorithm.v1"
  "signalops.${ENVIRONMENT}.connect-accepted-raw.v1"
  "signalops.${ENVIRONMENT}.marketops.signal.assurance.eligible.v1"
  "signalops.${ENVIRONMENT}.signal_assertion.v1"
  "signalops.${ENVIRONMENT}.cyberops-allowed-service-state.v1"
)

for topic in "${topics[@]}"; do
  kubectl exec -n "$NAMESPACE" "deploy/${DEPLOYMENT}" -- \
    rpk topic create "$topic" --partitions "$PARTITIONS" --replicas "$REPLICAS" --if-not-exists >/dev/null
done

kubectl exec -n "$NAMESPACE" "deploy/${DEPLOYMENT}" -- \
  rpk topic alter-config "signalops.${ENVIRONMENT}.cyberops-allowed-service-state.v1" --set cleanup.policy=compact >/dev/null

topic_list="$(kubectl exec -n "$NAMESPACE" "deploy/${DEPLOYMENT}" -- rpk topic list)"
for topic in "${topics[@]}"; do
  grep -q "$topic" <<<"$topic_list" || fail "topic missing after bootstrap: ${topic}"
done

cat <<EOF
signalops_k8s_connect_broker_smoke_verified
namespace=${NAMESPACE}
deployment=${DEPLOYMENT}
environment=${ENVIRONMENT}
topics=${#topics[@]}
partitions=${PARTITIONS}
replicas=${REPLICAS}
external_exposure=false
provider_polling=false
production_cutover_allowed=false
EOF
