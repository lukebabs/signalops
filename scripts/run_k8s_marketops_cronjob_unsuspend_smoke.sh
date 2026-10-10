#!/usr/bin/env bash
set -euo pipefail

ENV_FILE="${1:-.env}"
NAMESPACE="${SIGNALOPS_K8S_MARKETOPS_NAMESPACE:-signalops-marketops}"
DATA_NAMESPACE="${SIGNALOPS_K8S_DATA_NAMESPACE:-signalops-data}"
CRONJOB="${SIGNALOPS_K8S_MARKETOPS_CRONJOB_SMOKE_NAME:-marketops-fmp-annual-financial}"
MANIFEST_DIR="${SIGNALOPS_K8S_MARKETOPS_JOBS_MANIFEST_DIR:-deploy/kubernetes/staging/marketops-jobs}"
DATA_GATE_SCRIPT="${SIGNALOPS_K8S_MARKETOPS_DATA_GATE_SCRIPT:-scripts/run_k8s_marketops_dedicated_staging_db_gate.sh}"
IMMUTABLE_IMAGE="${SIGNALOPS_MARKETOPS_K8S_JOB_RUNNER_IMAGE:-ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner:6488950e98df}"
RUN_ID="${SIGNALOPS_K8S_MARKETOPS_CRONJOB_RUN_ID:-k8s-cronjob-unsuspend-$(date -u +%Y%m%dT%H%M%SZ)}"
SELECTOR="signalops.syncratic.io/cronjob-smoke-run=${RUN_ID}"
RESTORE_DONE=false

fail() {
  echo "signalops_k8s_marketops_cronjob_unsuspend_smoke_failed: $*" >&2
  exit 1
}

restore_cronjob() {
  if [[ "$RESTORE_DONE" == "true" ]]; then
    return 0
  fi
  kubectl apply -k "$MANIFEST_DIR" >/dev/null 2>&1 || true
  kubectl patch cronjob "$CRONJOB" -n "$NAMESPACE" --type=merge -p '{"spec":{"suspend":true}}' >/dev/null 2>&1 || true
  RESTORE_DONE=true
}

cleanup() {
  restore_cronjob
  kubectl delete job -n "$NAMESPACE" -l "$SELECTOR" --ignore-not-found --wait=false >/dev/null 2>&1 || true
}
trap cleanup EXIT

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"
command -v python3 >/dev/null 2>&1 || fail "python3 is required"
[[ -r "$ENV_FILE" ]] || fail "env file is missing or unreadable: ${ENV_FILE}"
[[ "$RUN_ID" =~ ^[A-Za-z0-9._:-]+$ ]] || fail "run id contains unsupported characters"

set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a

[[ "${SIGNALOPS_K8S_MARKETOPS_CRONJOB_UNSUSPEND_APPROVED:-true}" == "true" ]] || fail "set SIGNALOPS_K8S_MARKETOPS_CRONJOB_UNSUSPEND_APPROVED=true"

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_dir"

scripts/provision_signalops_openbao_ca_trust.sh >/dev/null
scripts/verify_k8s_marketops_scheduled_jobs_manifests.sh >/dev/null
"$DATA_GATE_SCRIPT" "$ENV_FILE" >/dev/null

kubectl apply -k "$MANIFEST_DIR" >/dev/null
kubectl get cronjob "$CRONJOB" -n "$NAMESPACE" >/dev/null || fail "CronJob missing after apply: ${NAMESPACE}/${CRONJOB}"

before_suspend="$(kubectl get cronjob "$CRONJOB" -n "$NAMESPACE" -o jsonpath='{.spec.suspend}')"
[[ "$before_suspend" == "true" ]] || fail "CronJob must start suspended before smoke; found suspend=${before_suspend}"

patch_file="$(mktemp -t signalops-cronjob-unsuspend.XXXXXX.json)"
python3 - "$patch_file" "$IMMUTABLE_IMAGE" "$RUN_ID" "$CRONJOB" <<'PYJSON'
import json
import sys
path, image, run_id, cronjob = sys.argv[1:]
patch = {
  "spec": {
    "schedule": "* * * * *",
    "suspend": False,
    "concurrencyPolicy": "Forbid",
    "successfulJobsHistoryLimit": 1,
    "failedJobsHistoryLimit": 1,
    "jobTemplate": {
      "spec": {
        "backoffLimit": 0,
        "activeDeadlineSeconds": 900,
        "template": {
          "metadata": {
            "labels": {
              "signalops.syncratic.io/cronjob-smoke": "true",
              "signalops.syncratic.io/cronjob-smoke-run": run_id,
              "signalops.syncratic.io/production-cutover-allowed": "false"
            }
          },
          "spec": {
            "containers": [
              {
                "name": "marketops-job",
                "image": image,
                "imagePullPolicy": "Always",
                "args": [cronjob],
                "env": [
                  {"name": "SIGNALOPS_ENV", "value": "kubernetes-staging"},
                  {"name": "SIGNALOPS_MARKETOPS_DATA_BOUNDARY_REQUIRED", "value": "true"},
                  {"name": "SIGNALOPS_K8S_RUNTIME_ENV_FILE", "value": "/vault/secrets/marketops-worker-runtime.env"},
                  {"name": "MARKETOPS_K8S_DRY_RUN", "value": "true"},
                  {"name": "MARKETOPS_FMP_ANNUAL_MAX_ASSETS", "value": "1"},
                  {"name": "MARKETOPS_K8S_RUN_ID", "value": run_id},
                  {"name": "MARKETOPS_K8S_SCHEDULE_LABEL", "value": "Kubernetes CronJob unsuspend smoke"},
                  {"name": "MARKETOPS_K8S_TIMEZONE", "value": "UTC"},
                  {"name": "MARKETOPS_K8S_RUNNER_ID", "value": "kubernetes-cronjob-smoke"}
                ]
              }
            ]
          }
        }
      }
    }
  }
}
with open(path, 'w', encoding='utf-8') as fh:
    json.dump(patch, fh)
PYJSON
kubectl patch cronjob "$CRONJOB" -n "$NAMESPACE" --type=merge --patch-file "$patch_file" >/dev/null
rm -f "$patch_file"

job=""
for _ in $(seq 1 150); do
  job="$(kubectl get jobs -n "$NAMESPACE" -l "$SELECTOR" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)"
  [[ -n "$job" ]] && break
  sleep 1
done
[[ -n "$job" ]] || fail "CronJob did not create a smoke Job within timeout"

# Resuspend immediately after the controller creates exactly one smoke Job.
kubectl patch cronjob "$CRONJOB" -n "$NAMESPACE" --type=merge -p '{"spec":{"suspend":true}}' >/dev/null

pod=""
for _ in $(seq 1 120); do
  pod="$(kubectl get pods -n "$NAMESPACE" -l job-name="$job" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)"
  [[ -n "$pod" ]] || { sleep 1; continue; }
  exit_code="$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.status.containerStatuses[?(@.name=="marketops-job")].state.terminated.exitCode}' 2>/dev/null || true)"
  waiting_reason="$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.status.containerStatuses[?(@.name=="marketops-job")].state.waiting.reason}' 2>/dev/null || true)"
  case "$exit_code" in
    0) break ;;
    '') ;;
    *) kubectl logs -n "$NAMESPACE" "$pod" -c marketops-job >&2 || true; fail "CronJob smoke worker exited ${exit_code}" ;;
  esac
  [[ "$waiting_reason" == "ImagePullBackOff" || "$waiting_reason" == "ErrImagePull" || "$waiting_reason" == "CreateContainerConfigError" ]] && {
    kubectl describe pod "$pod" -n "$NAMESPACE" >&2 || true
    fail "CronJob smoke pod waiting reason ${waiting_reason}"
  }
  sleep 1
done
[[ -n "$pod" ]] || fail "CronJob smoke pod was not created"
exit_code="$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.status.containerStatuses[?(@.name=="marketops-job")].state.terminated.exitCode}' 2>/dev/null || true)"
[[ "$exit_code" == "0" ]] || {
  kubectl describe pod "$pod" -n "$NAMESPACE" >&2 || true
  kubectl logs -n "$NAMESPACE" "$pod" -c vault-agent-init --tail=80 >&2 || true
  kubectl logs -n "$NAMESPACE" "$pod" -c marketops-job --tail=80 >&2 || true
  fail "CronJob smoke worker did not terminate successfully; exit_code=${exit_code:-missing}"
}

logs="$(kubectl logs -n "$NAMESPACE" "$pod" -c marketops-job)"
[[ "$logs" == *"dry_run"* || "$logs" == *"--dry-run"* || "$logs" == *"DRY"* ]] || fail "dry-run evidence marker missing from CronJob smoke logs"

pg_password="$(kubectl get secret marketops-postgres-staging-auth -n "$DATA_NAMESPACE" -o jsonpath='{.data.POSTGRES_PASSWORD}' | base64 -d)"
status_line="$(kubectl exec -n "$DATA_NAMESPACE" marketops-postgres-staging-0 -- env PGPASSWORD="$pg_password" psql -h 127.0.0.1 -U signalops -d marketops -Atc "SELECT status || '|' || runner || '|' || COALESCE(exit_code::text,'') || '|' || COALESCE((detail->>'dry_run'),'') FROM marketops_scheduled_job_runs WHERE run_id='${RUN_ID}'")"
[[ "$status_line" == "succeeded|kubernetes-cronjob-smoke|0|true" ]] || fail "DB-backed scheduler status parity mismatch: ${status_line:-empty}"

job_count="$(kubectl get jobs -n "$NAMESPACE" -l "$SELECTOR" --no-headers 2>/dev/null | wc -l | tr -d ' ')"
[[ "$job_count" == "1" ]] || fail "expected exactly one smoke Job, found ${job_count}"

restore_cronjob
restored_suspend="$(kubectl get cronjob "$CRONJOB" -n "$NAMESPACE" -o jsonpath='{.spec.suspend}')"
[[ "$restored_suspend" == "true" ]] || fail "CronJob did not restore to suspended state"

kubectl delete job "$job" -n "$NAMESPACE" --wait=true >/dev/null
remaining_jobs="$(kubectl get jobs -n "$NAMESPACE" -l "$SELECTOR" --no-headers 2>/dev/null | wc -l | tr -d ' ')"
[[ "$remaining_jobs" == "0" ]] || fail "smoke Job cleanup incomplete: ${remaining_jobs} remain"

cat <<EOF
signalops_k8s_marketops_cronjob_unsuspend_smoke_verified
namespace=${NAMESPACE}
cronjob=${CRONJOB}
image=${IMMUTABLE_IMAGE}
run_id=${RUN_ID}
job_created=true
jobs_created=1
resuspended=true
scheduler_status_parity=verified
provider_polling=false
production_cutover_allowed=false
EOF
