#!/usr/bin/env bash
set -euo pipefail

ENV_FILE="${1:-${SIGNALOPS_K8S_MARKETOPS_RUNTIME_ENV_FILE:-/etc/signalops/openbao-signalops-marketops-staging-runtime.env}}"
NAMESPACE="${SIGNALOPS_K8S_MARKETOPS_NAMESPACE:-signalops-marketops}"
IMAGE="${SIGNALOPS_MARKETOPS_K8S_JOB_RUNNER_IMAGE:-ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner:staging}"
DRY_RUN_JOB_ID="${SIGNALOPS_K8S_MARKETOPS_DRY_RUN_JOB_ID:-${2:-marketops-fmp-annual-financial}}"
JOB_NAME="${SIGNALOPS_K8S_MARKETOPS_DRY_RUN_JOB_NAME:-signalops-marketops-non-provider-dry-run}"
SECRET_PATH="${MARKETOPS_SECRET_PATH:-signalops/data/k8s/marketops/marketops-worker-runtime-staging}"
OPENBAO_ADDR="${OPENBAO_ADDR:-https://openbao.openbao.svc:8200}"
RUN_ID="${SIGNALOPS_K8S_MARKETOPS_DRY_RUN_RUN_ID:-${JOB_NAME}-$(date -u +%Y%m%dT%H%M%SZ)}"
RUNTIME_SMOKE_NAMESPACE="${SIGNALOPS_K8S_RUNTIME_SMOKE_NAMESPACE:-syncratic-runtime-smoke}"
OPENBAO_TLS_SECRET="${SIGNALOPS_OPENBAO_CA_SECRET:-signalops-openbao-ca}"
OPENBAO_CA_CERT="${SIGNALOPS_OPENBAO_CA_CERT:-/vault/tls/ca.crt}"

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
[[ "$RUN_ID" =~ ^[A-Za-z0-9._:-]+$ ]] || fail "run id contains unsupported characters"
case "$DRY_RUN_JOB_ID" in
  marketops-fmp-annual-financial|marketops-operations-monitor|marketops-retention-governance|marketops-task-retry|marketops-warm-eod) ;;
  *) fail "unsupported non-provider dry-run job id: ${DRY_RUN_JOB_ID}" ;;
esac

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
"$repo_dir/scripts/provision_openbao_signalops_marketops_runtime_staging.sh" "$ENV_FILE" >/dev/null

kubectl get namespace "$NAMESPACE" >/dev/null || fail "namespace missing: ${NAMESPACE}"
kubectl get namespace "$RUNTIME_SMOKE_NAMESPACE" >/dev/null || fail "namespace missing: ${RUNTIME_SMOKE_NAMESPACE}"
kubectl delete job "$JOB_NAME" -n "$NAMESPACE" --ignore-not-found --wait=true >/dev/null

job_yaml="$(mktemp -t signalops-marketops-dry-run-job.XXXXXX.yaml)"
policy_yaml="$(mktemp -t signalops-marketops-dry-run-policy.XXXXXX.yaml)"
cleanup() {
  kubectl delete job "$JOB_NAME" -n "$NAMESPACE" --ignore-not-found --wait=false >/dev/null 2>&1 || true
  kubectl delete networkpolicy allow-marketops-k8s-dry-run-runtime-smoke-postgres-egress -n "$NAMESPACE" --ignore-not-found >/dev/null 2>&1 || true
  kubectl delete networkpolicy allow-marketops-k8s-dry-run-to-postgres -n "$RUNTIME_SMOKE_NAMESPACE" --ignore-not-found >/dev/null 2>&1 || true
  rm -f "$job_yaml" "$policy_yaml"
}
trap cleanup EXIT
cat >"$policy_yaml" <<EOF
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-marketops-k8s-dry-run-runtime-smoke-postgres-egress
  namespace: ${NAMESPACE}
  labels:
    app.kubernetes.io/name: signalops-marketops-k8s-job-runner
    app.kubernetes.io/component: non-provider-dry-run
    app.kubernetes.io/part-of: signalops
    signalops.syncratic.io/plane: marketops
    signalops.syncratic.io/stage: staging
    signalops.syncratic.io/production-cutover-allowed: "false"
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: signalops-marketops-k8s-job-runner
      app.kubernetes.io/component: non-provider-dry-run
      app.kubernetes.io/part-of: signalops
      signalops.syncratic.io/production-cutover-allowed: "false"
      signalops.syncratic.io/stage: staging
  policyTypes:
    - Egress
  egress:
    - to:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: ${RUNTIME_SMOKE_NAMESPACE}
          podSelector:
            matchLabels:
              app.kubernetes.io/component: postgres
              app.kubernetes.io/instance: syncratic-refactor-smoke
              app.kubernetes.io/name: syncratic-phase1
      ports:
        - protocol: TCP
          port: 5432
---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-marketops-k8s-dry-run-to-postgres
  namespace: ${RUNTIME_SMOKE_NAMESPACE}
  labels:
    app.kubernetes.io/name: signalops-marketops-k8s-job-runner
    app.kubernetes.io/component: non-provider-dry-run
    app.kubernetes.io/part-of: signalops
    signalops.syncratic.io/plane: marketops
    signalops.syncratic.io/stage: staging
    signalops.syncratic.io/production-cutover-allowed: "false"
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/component: postgres
      app.kubernetes.io/instance: syncratic-refactor-smoke
      app.kubernetes.io/name: syncratic-phase1
  policyTypes:
    - Ingress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: ${NAMESPACE}
          podSelector:
            matchLabels:
              app.kubernetes.io/name: signalops-marketops-k8s-job-runner
              app.kubernetes.io/component: non-provider-dry-run
              app.kubernetes.io/part-of: signalops
              signalops.syncratic.io/production-cutover-allowed: "false"
              signalops.syncratic.io/stage: staging
      ports:
        - protocol: TCP
          port: 5432
EOF
kubectl apply -f "$policy_yaml" >/dev/null

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
        vault.hashicorp.com/service: "${OPENBAO_ADDR}"
        vault.hashicorp.com/tls-secret: "${OPENBAO_TLS_SECRET}"
        vault.hashicorp.com/ca-cert: "${OPENBAO_CA_CERT}"
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
            - ${DRY_RUN_JOB_ID}
          env:
            - name: BAO_ADDR
              value: "${OPENBAO_ADDR}"
            - name: SIGNALOPS_K8S_RUNTIME_ENV_FILE
              value: /vault/secrets/marketops-worker-runtime.env
            - name: MARKETOPS_K8S_DRY_RUN
              value: "true"
            - name: MARKETOPS_FMP_ANNUAL_MAX_ASSETS
              value: "1"
            - name: MARKETOPS_K8S_RUN_ID
              value: "${RUN_ID}"
            - name: MARKETOPS_K8S_SCHEDULE_LABEL
              value: "Kubernetes one-shot non-provider dry-run"
            - name: MARKETOPS_K8S_TIMEZONE
              value: "UTC"
            - name: MARKETOPS_K8S_RUNNER_ID
              value: "kubernetes"
          resources:
            requests:
              cpu: 50m
              memory: 64Mi
            limits:
              cpu: 200m
              memory: 256Mi
EOF
kubectl apply -f "$job_yaml" >/dev/null
pod=""
for _ in $(seq 1 120); do
  pod="$(kubectl get pods -n "$NAMESPACE" -l job-name="$JOB_NAME" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)"
  [[ -n "$pod" ]] || { sleep 1; continue; }
  exit_code="$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.status.containerStatuses[?(@.name=="marketops-job")].state.terminated.exitCode}' 2>/dev/null || true)"
  waiting_reason="$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.status.containerStatuses[?(@.name=="marketops-job")].state.waiting.reason}' 2>/dev/null || true)"
  case "$exit_code" in
    0) break ;;
    '') ;;
    *) kubectl logs -n "$NAMESPACE" "$pod" -c marketops-job >&2 || true; fail "dry-run worker exited ${exit_code}" ;;
  esac
  [[ "$waiting_reason" == "ImagePullBackOff" || "$waiting_reason" == "ErrImagePull" || "$waiting_reason" == "CreateContainerConfigError" ]] && {
    kubectl describe pod "$pod" -n "$NAMESPACE" >&2 || true
    fail "dry-run worker waiting reason ${waiting_reason}"
  }
  sleep 1
done
[[ -n "$pod" ]] || fail "dry-run pod was not created"
exit_code="$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.status.containerStatuses[?(@.name=="marketops-job")].state.terminated.exitCode}' 2>/dev/null || true)"
[[ "$exit_code" == "0" ]] || {
  kubectl describe pod "$pod" -n "$NAMESPACE" >&2 || true
  kubectl logs -n "$NAMESPACE" "$pod" -c vault-agent-init --tail=80 >&2 || true
  kubectl logs -n "$NAMESPACE" "$pod" -c marketops-job --tail=80 >&2 || true
  fail "dry-run worker did not terminate successfully; exit_code=${exit_code:-missing}"
}
logs="$(kubectl logs -n "$NAMESPACE" "$pod" -c marketops-job)"
[[ "$logs" == *"dry_run"* || "$logs" == *"--dry-run"* || "$logs" == *"DRY"* ]] || fail "dry-run evidence marker missing from job logs"

if [[ "${SIGNALOPS_K8S_STATUS_PARITY_VERIFY:-true}" == "true" ]]; then
  status_sql="SELECT status || '|' || runner || '|' || COALESCE(exit_code::text,'') || '|' || COALESCE((detail->>'dry_run'),'') FROM marketops_scheduled_job_runs WHERE run_id='${RUN_ID}'"
  if command -v psql >/dev/null 2>&1 && psql --version >/dev/null 2>&1; then
    status_line="$(psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -Atc "$status_sql")"
  else
    pg_pod="${SIGNALOPS_K8S_RUNTIME_SMOKE_POSTGRES_POD:-syncratic-refactor-smoke-syncratic-phase1-postgres-0}"
    pg_fields="$(python3 - <<'PYDB'
import os
from urllib.parse import urlparse, unquote
url = os.environ.get('SIGNALOPS_MARKETOPS_DATABASE_URL', '')
parsed = urlparse(url)
if not parsed.scheme.startswith('postgres') or not parsed.username or parsed.password is None or not parsed.path.strip('/'):
    raise SystemExit('invalid database URL for runtime-smoke verification')
print('|'.join([unquote(parsed.username), unquote(parsed.password), parsed.path.strip('/')]))
PYDB
)" || fail "could not parse runtime-smoke database URL for status verification"
    IFS='|' read -r pg_user pg_password pg_database <<<"$pg_fields"
    status_line="$(kubectl exec -n "$RUNTIME_SMOKE_NAMESPACE" "$pg_pod" -- env PGPASSWORD="$pg_password" psql -h 127.0.0.1 -U "$pg_user" -d "$pg_database" -Atc "$status_sql")"
  fi
  [[ "$status_line" == "succeeded|kubernetes|0|true" ]] || fail "DB-backed scheduler status parity mismatch: ${status_line:-empty}"
fi

kubectl delete job "$JOB_NAME" -n "$NAMESPACE" --wait=true >/dev/null
kubectl delete networkpolicy allow-marketops-k8s-dry-run-runtime-smoke-postgres-egress -n "$NAMESPACE" --ignore-not-found >/dev/null
kubectl delete networkpolicy allow-marketops-k8s-dry-run-to-postgres -n "$RUNTIME_SMOKE_NAMESPACE" --ignore-not-found >/dev/null
trap - EXIT
rm -f "$job_yaml" "$policy_yaml"

cat <<EOF
signalops_k8s_marketops_non_provider_dry_run_job_verified
namespace=${NAMESPACE}
image=${IMAGE}
job=${JOB_NAME}
job_id=${DRY_RUN_JOB_ID}
dry_run=true
max_assets=1
run_id=${RUN_ID}
scheduler_status_parity=verified
provider_polling=false
production_cutover_allowed=false
EOF
