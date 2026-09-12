#!/usr/bin/env bash
set -euo pipefail

ENV_FILE="${1:-${SIGNALOPS_K8S_MARKETOPS_RUNTIME_ENV_FILE:-/etc/signalops/openbao-signalops-marketops-staging-runtime.env}}"
NAMESPACE="${SIGNALOPS_K8S_MARKETOPS_NAMESPACE:-signalops-marketops}"
IMAGE="${SIGNALOPS_MARKETOPS_K8S_JOB_RUNNER_IMAGE:-ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner:staging}"
JOB_NAME="${SIGNALOPS_K8S_MARKETOPS_DRY_RUN_JOB_NAME:-signalops-marketops-non-provider-dry-run}"
SECRET_PATH="${MARKETOPS_SECRET_PATH:-signalops/data/k8s/marketops/marketops-worker-runtime-staging}"
OPENBAO_ADDR="${OPENBAO_ADDR:-https://openbao.openbao.svc:8200}"

fail() {
  echo "signalops_k8s_marketops_non_provider_dry_run_job_failed: $*" >&2
  exit 1
}

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"
[[ -r "$ENV_FILE" ]] || fail "runtime env file is missing or unreadable: ${ENV_FILE}"

set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a

[[ "${SIGNALOPS_K8S_MARKETOPS_DRY_RUN_APPROVED:-}" == "true" ]] || fail "set SIGNALOPS_K8S_MARKETOPS_DRY_RUN_APPROVED=true in the runtime env file"

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
"$repo_dir/scripts/provision_openbao_signalops_marketops_runtime_staging.sh" "$ENV_FILE" >/dev/null

kubectl get namespace "$NAMESPACE" >/dev/null || fail "namespace missing: ${NAMESPACE}"
kubectl delete job "$JOB_NAME" -n "$NAMESPACE" --ignore-not-found --wait=true >/dev/null

job_yaml="$(mktemp -t signalops-marketops-dry-run-job.XXXXXX.yaml)"
trap 'kubectl delete job "$JOB_NAME" -n "$NAMESPACE" --ignore-not-found --wait=false >/dev/null 2>&1 || true; rm -f "$job_yaml"' EXIT
cat >"$job_yaml" <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: ${JOB_NAME}
  namespace: ${NAMESPACE}
  labels:
    app.kubernetes.io/name: signalops-marketops-k8s-job-runner
    app.kubernetes.io/component: non-provider-dry-run
    app.kubernetes.io/part-of: signalops
    signalops.syncratic.io/plane: marketops
    signalops.syncratic.io/stage: staging
    signalops.syncratic.io/production-cutover-allowed: "false"
spec:
  backoffLimit: 0
  ttlSecondsAfterFinished: 300
  template:
    metadata:
      annotations:
        vault.hashicorp.com/agent-inject: "true"
        vault.hashicorp.com/role: "signalops-marketops"
        vault.hashicorp.com/agent-inject-secret-marketops-worker-runtime.env: "${SECRET_PATH}"
        vault.hashicorp.com/agent-inject-template-marketops-worker-runtime.env: |
          {{- with secret "${SECRET_PATH}" -}}
          {{- range \$k, \$v := .Data.data }}
          export {{ \$k }}="{{ \$v }}"
          {{- end }}
          {{- end }}
      labels:
        app.kubernetes.io/name: signalops-marketops-k8s-job-runner
        app.kubernetes.io/component: non-provider-dry-run
        app.kubernetes.io/part-of: signalops
        signalops.syncratic.io/plane: marketops
        signalops.syncratic.io/stage: staging
        signalops.syncratic.io/production-cutover-allowed: "false"
    spec:
      restartPolicy: Never
      serviceAccountName: signalops-marketops-provider-worker
      imagePullSecrets:
        - name: signalops-ghcr-pull
      containers:
        - name: marketops-job
          image: ${IMAGE}
          imagePullPolicy: Always
          args:
            - marketops-fmp-annual-financial
          env:
            - name: BAO_ADDR
              value: "${OPENBAO_ADDR}"
            - name: SIGNALOPS_K8S_RUNTIME_ENV_FILE
              value: /vault/secrets/marketops-worker-runtime.env
            - name: MARKETOPS_K8S_DRY_RUN
              value: "true"
            - name: MARKETOPS_FMP_ANNUAL_MAX_ASSETS
              value: "1"
          resources:
            requests:
              cpu: 50m
              memory: 64Mi
            limits:
              cpu: 200m
              memory: 256Mi
EOF
kubectl apply -f "$job_yaml" >/dev/null
kubectl wait --for=condition=complete "job/${JOB_NAME}" -n "$NAMESPACE" --timeout=180s >/dev/null || {
  kubectl describe job "$JOB_NAME" -n "$NAMESPACE" >&2 || true
  pods="$(kubectl get pods -n "$NAMESPACE" -l job-name="$JOB_NAME" -o name 2>/dev/null || true)"
  if [[ -n "$pods" ]]; then
    kubectl logs -n "$NAMESPACE" $pods --all-containers=true >&2 || true
  fi
  fail "dry-run job did not complete"
}

pods="$(kubectl get pods -n "$NAMESPACE" -l job-name="$JOB_NAME" -o name)"
logs="$(kubectl logs -n "$NAMESPACE" $pods --all-containers=true)"
[[ "$logs" == *"dry_run"* || "$logs" == *"--dry-run"* || "$logs" == *"DRY"* ]] || fail "dry-run evidence marker missing from job logs"

kubectl delete job "$JOB_NAME" -n "$NAMESPACE" --wait=true >/dev/null
trap - EXIT
rm -f "$job_yaml"

cat <<EOF
signalops_k8s_marketops_non_provider_dry_run_job_verified
namespace=${NAMESPACE}
image=${IMAGE}
job=${JOB_NAME}
job_id=marketops-fmp-annual-financial
dry_run=true
max_assets=1
provider_polling=false
production_cutover_allowed=false
EOF
