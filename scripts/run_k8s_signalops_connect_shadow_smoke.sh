#!/usr/bin/env bash
set -euo pipefail

ENV_FILE="${1:-.env}"
RUNTIME_ENV="${SIGNALOPS_K8S_CONNECT_RUNTIME_ENV_FILE:-/tmp/signalops-openbao-connect-runtime-staging.env}"
NAMESPACE="${SIGNALOPS_K8S_CONNECT_NAMESPACE:-signalops-connect}"
PERSISTER="${SIGNALOPS_K8S_CONNECT_PERSISTER_DEPLOYMENT:-signalops-connect-persister}"
OUTBOX="${SIGNALOPS_K8S_CONNECT_OUTBOX_DEPLOYMENT:-signalops-connect-outbox}"
TIMEOUT_SECONDS="${SIGNALOPS_K8S_CONNECT_SHADOW_TIMEOUT_SECONDS:-45}"

fail() {
  echo "signalops_k8s_connect_shadow_smoke_failed: $*" >&2
  exit 1
}

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"
repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_dir"

scripts/create_k8s_signalops_connect_runtime_staging_env.sh "$ENV_FILE" "$RUNTIME_ENV" >/dev/null
scripts/provision_openbao_signalops_connect_runtime_staging.sh "$RUNTIME_ENV" >/dev/null
scripts/run_k8s_signalops_connect_broker_smoke.sh >/dev/null

cleanup() {
  kubectl scale deployment "$PERSISTER" -n "$NAMESPACE" --replicas=0 >/dev/null 2>&1 || true
  kubectl scale deployment "$OUTBOX" -n "$NAMESPACE" --replicas=0 >/dev/null 2>&1 || true
}
trap cleanup EXIT

kubectl get deployment "$PERSISTER" -n "$NAMESPACE" >/dev/null || fail "deployment missing: ${NAMESPACE}/${PERSISTER}"
kubectl get deployment "$OUTBOX" -n "$NAMESPACE" >/dev/null || fail "deployment missing: ${NAMESPACE}/${OUTBOX}"

run_worker_shadow() {
  local deployment="$1"
  local workload_id="$2"
  kubectl scale deployment "$deployment" -n "$NAMESPACE" --replicas=1 >/dev/null
  kubectl rollout status "deployment/${deployment}" -n "$NAMESPACE" --timeout=180s >/dev/null
  sleep "$TIMEOUT_SECONDS"
  local pod
  pod="$(kubectl get pod -n "$NAMESPACE" -l app.kubernetes.io/name=signalops-connect-k8s-worker,signalops.syncratic.io/workload-id="$workload_id" -o jsonpath='{.items[0].metadata.name}')"
  [[ -n "$pod" ]] || fail "expected shadow pod was not found for ${workload_id}"
  local phase
  phase="$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.status.phase}')"
  [[ "$phase" == "Running" ]] || fail "${workload_id} pod phase=${phase}"
  if kubectl logs -n "$NAMESPACE" "$pod" --tail=120 | grep -Eiq 'failed|panic|fatal|permission denied'; then
    kubectl logs -n "$NAMESPACE" "$pod" --tail=120 >&2 || true
    fail "${workload_id} emitted failure log marker"
  fi
  kubectl scale deployment "$deployment" -n "$NAMESPACE" --replicas=0 >/dev/null
  kubectl rollout status "deployment/${deployment}" -n "$NAMESPACE" --timeout=120s >/dev/null
}

run_worker_shadow "$PERSISTER" connect-persister
run_worker_shadow "$OUTBOX" connect-outbox

cleanup
trap - EXIT

cat <<EOF
signalops_k8s_connect_shadow_smoke_verified
namespace=${NAMESPACE}
persister=${PERSISTER}
outbox=${OUTBOX}
replicas_scaled_to_one=sequential
replicas_restored_to_zero=true
runtime=openbao_non_production
broker=signalops-redpanda-staging.signalops-data.svc.cluster.local:9092
provider_polling=false
production_cutover_allowed=false
EOF
