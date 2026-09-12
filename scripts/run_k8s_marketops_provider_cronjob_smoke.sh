#!/usr/bin/env bash
set -euo pipefail

ENV_FILE="${1:-.env}"
NAMESPACE="${SIGNALOPS_K8S_MARKETOPS_NAMESPACE:-signalops-marketops}"
DATA_NAMESPACE="${SIGNALOPS_K8S_DATA_NAMESPACE:-signalops-data}"
CRONJOB="${SIGNALOPS_K8S_MARKETOPS_PROVIDER_CRONJOB_SMOKE_NAME:-marketops-fmp-annual-financial}"
MANIFEST_DIR="${SIGNALOPS_K8S_MARKETOPS_JOBS_MANIFEST_DIR:-deploy/kubernetes/staging/marketops-jobs}"
IMMUTABLE_IMAGE="${SIGNALOPS_MARKETOPS_K8S_JOB_RUNNER_IMAGE:?set SIGNALOPS_MARKETOPS_K8S_JOB_RUNNER_IMAGE to the immutable provider-smoke image tag}"
RUN_ID="${SIGNALOPS_K8S_MARKETOPS_PROVIDER_CRONJOB_RUN_ID:-k8s-provider-cronjob-${CRONJOB}-$(date -u +%Y%m%dT%H%M%SZ)}"
SELECTOR="signalops.syncratic.io/provider-cronjob-smoke-run=${RUN_ID}"
PRIMARY_SECRET="marketops-postgres-staging-auth"
TEMPORAL_SECRET="marketops-timescaledb-staging-auth"
PRIMARY_STATEFULSET="marketops-postgres-staging"
TEMPORAL_STATEFULSET="marketops-timescaledb-staging"
PRIMARY_POD="${PRIMARY_STATEFULSET}-0"
TEMPORAL_POD="${TEMPORAL_STATEFULSET}-0"
PRIMARY_DATABASE="marketops"
TEMPORAL_DATABASE="marketops_temporal"
PRIMARY_USER="signalops"
TEMPORAL_USER="signalops"
RUNTIME_ENV="/tmp/signalops-openbao-marketops-provider-cronjob-runtime.env"
RESTORE_DONE=false

fail() {
  echo "signalops_k8s_marketops_provider_cronjob_smoke_failed: $*" >&2
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
  rm -f "$RUNTIME_ENV"
}
trap cleanup EXIT

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"
command -v python3 >/dev/null 2>&1 || fail "python3 is required"
[[ -r "$ENV_FILE" ]] || fail "env file is missing or unreadable: ${ENV_FILE}"
[[ "$RUN_ID" =~ ^[A-Za-z0-9._:-]+$ ]] || fail "run id contains unsupported characters"
[[ "$CRONJOB" == "marketops-fmp-annual-financial" ]] || fail "provider smoke is currently constrained to marketops-fmp-annual-financial"

set -a
# shellcheck disable=SC1090
source "$ENV_FILE"
set +a

[[ "${SIGNALOPS_K8S_MARKETOPS_PROVIDER_CRONJOB_APPROVED:-true}" == "true" ]] || fail "set SIGNALOPS_K8S_MARKETOPS_PROVIDER_CRONJOB_APPROVED=true"
[[ -n "${OPENBAO_ADMIN_TOKEN:-}" ]] || fail "OPENBAO_ADMIN_TOKEN is required in ${ENV_FILE}"
[[ -n "${SIGNALOPS_FMP_API_KEY:-}" ]] || fail "SIGNALOPS_FMP_API_KEY is required in ${ENV_FILE}"
SIGNALOPS_FMP_BASE_URL="${SIGNALOPS_FMP_BASE_URL:-https://financialmodelingprep.com}"

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_dir"

scripts/provision_signalops_openbao_ca_trust.sh >/dev/null
scripts/verify_k8s_marketops_scheduled_jobs_manifests.sh >/dev/null
kubectl apply -k "$repo_dir/deploy/kubernetes/staging/marketops-data" >/dev/null
kubectl rollout status statefulset/"$PRIMARY_STATEFULSET" -n "$DATA_NAMESPACE" --timeout=180s >/dev/null
kubectl rollout status statefulset/"$TEMPORAL_STATEFULSET" -n "$DATA_NAMESPACE" --timeout=180s >/dev/null

primary_password="$(kubectl get secret "$PRIMARY_SECRET" -n "$DATA_NAMESPACE" -o jsonpath='{.data.POSTGRES_PASSWORD}' | base64 -d)"
temporal_password="$(kubectl get secret "$TEMPORAL_SECRET" -n "$DATA_NAMESPACE" -o jsonpath='{.data.POSTGRES_PASSWORD}' | base64 -d)"
[[ -n "$primary_password" ]] || fail "primary staging password could not be read"
[[ -n "$temporal_password" ]] || fail "temporal staging password could not be read"

kubectl exec -i -n "$DATA_NAMESPACE" "$PRIMARY_POD" -- env PGPASSWORD="$primary_password" psql -U "$PRIMARY_USER" -d "$PRIMARY_DATABASE" -v ON_ERROR_STOP=1 >/dev/null <<'SQL'
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'signalops_subscriber_global_eod') THEN
    CREATE ROLE signalops_subscriber_global_eod;
  END IF;
END
$$;

CREATE TABLE IF NOT EXISTS public.subscriber_global_warm_eod_assets (
  global_asset_id text PRIMARY KEY,
  canonical_symbol text NOT NULL,
  asset_name text,
  market_status text NOT NULL DEFAULT 'active',
  coverage_status text NOT NULL DEFAULT 'active',
  as_of_date date NOT NULL DEFAULT CURRENT_DATE,
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.subscriber_global_assets (
  global_asset_id text PRIMARY KEY,
  canonical_symbol text NOT NULL,
  asset_name text,
  priority integer NOT NULL DEFAULT 1,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.subscriber_global_marketops_evidence_runs (
  evidence_run_id text PRIMARY KEY,
  evidence_kind text NOT NULL,
  algorithm_id text NOT NULL,
  algorithm_version text NOT NULL,
  execution_mode text NOT NULL,
  source_scope text NOT NULL,
  session_start_date date NOT NULL,
  session_end_date date NOT NULL,
  input_manifest_fingerprint text NOT NULL,
  validation_contract_ref text NOT NULL,
  immutable_baseline_ref text NOT NULL,
  provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  recorded_by text NOT NULL,
  correlation_id text NOT NULL,
  recorded_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.subscriber_global_marketops_evidence_records (
  global_evidence_id text PRIMARY KEY,
  evidence_run_id text NOT NULL REFERENCES public.subscriber_global_marketops_evidence_runs(evidence_run_id),
  global_asset_id text NOT NULL,
  session_date date NOT NULL,
  evidence_kind text NOT NULL,
  algorithm_id text NOT NULL,
  algorithm_version text NOT NULL,
  quality_state text NOT NULL,
  source_system text NOT NULL,
  source_event_id text NOT NULL,
  source_run_id text NOT NULL,
  evidence_fingerprint text NOT NULL,
  validation_contract_ref text NOT NULL,
  immutable_baseline_ref text NOT NULL,
  payload jsonb NOT NULL DEFAULT '{}'::jsonb,
  provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  observed_at timestamptz NOT NULL DEFAULT now(),
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (global_asset_id, session_date, evidence_kind, algorithm_id, algorithm_version, evidence_fingerprint)
);


CREATE TABLE IF NOT EXISTS public.subscriber_global_annual_financial_workflows (
  workflow_id text PRIMARY KEY,
  session_date date NOT NULL UNIQUE,
  status text NOT NULL CHECK (status IN ('queued','running','succeeded','degraded','failed')),
  schedule_job_id text NOT NULL DEFAULT 'marketops-fmp-annual-financial',
  coverage jsonb NOT NULL DEFAULT '{}'::jsonb,
  failure_class text NOT NULL DEFAULT '',
  error_message text NOT NULL DEFAULT '',
  started_at timestamptz,
  completed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS public.subscriber_global_annual_financial_tasks (
  task_id text PRIMARY KEY,
  workflow_id text NOT NULL REFERENCES public.subscriber_global_annual_financial_workflows(workflow_id) ON DELETE CASCADE,
  global_asset_id text NOT NULL REFERENCES public.subscriber_global_assets(global_asset_id) ON DELETE RESTRICT,
  symbol text NOT NULL,
  status text NOT NULL CHECK (status IN ('queued','running','retry_scheduled','succeeded','skipped_no_data','blocked_entitlement','deferred_quota','failed_terminal')),
  attempt_count integer NOT NULL DEFAULT 0 CHECK (attempt_count >= 0),
  max_attempts integer NOT NULL DEFAULT 3 CHECK (max_attempts BETWEEN 1 AND 10),
  next_attempt_at timestamptz NOT NULL DEFAULT now(),
  lease_expires_at timestamptz,
  failure_class text NOT NULL DEFAULT '',
  provider_status integer,
  error_message text NOT NULL DEFAULT '',
  result jsonb NOT NULL DEFAULT '{}'::jsonb,
  completed_at timestamptz,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (workflow_id, global_asset_id)
);

CREATE TABLE IF NOT EXISTS public.marketops_scheduled_job_statuses (
  job_id text PRIMARY KEY,
  schedule text,
  timezone text,
  status text NOT NULL,
  reason text,
  runner text NOT NULL,
  run_id text,
  started_at timestamptz,
  completed_at timestamptz,
  exit_code integer,
  updated_at timestamptz NOT NULL DEFAULT now(),
  detail jsonb NOT NULL DEFAULT '{}'::jsonb
);

CREATE TABLE IF NOT EXISTS public.marketops_scheduled_job_runs (
  run_id text PRIMARY KEY,
  job_id text NOT NULL,
  schedule text,
  timezone text,
  status text NOT NULL,
  reason text,
  runner text NOT NULL,
  started_at timestamptz NOT NULL DEFAULT now(),
  finished_at timestamptz,
  completed_at timestamptz,
  exit_code integer,
  updated_at timestamptz NOT NULL DEFAULT now(),
  detail jsonb NOT NULL DEFAULT '{}'::jsonb
);

INSERT INTO public.subscriber_global_assets (
  global_asset_id, canonical_symbol, asset_name, priority, updated_at
) VALUES (
  'k8s-staging-aapl', 'AAPL', 'Apple Inc.', 1, now()
) ON CONFLICT (global_asset_id) DO UPDATE
SET canonical_symbol = EXCLUDED.canonical_symbol,
    asset_name = EXCLUDED.asset_name,
    priority = EXCLUDED.priority,
    updated_at = EXCLUDED.updated_at;

INSERT INTO public.subscriber_global_warm_eod_assets (
  global_asset_id, canonical_symbol, asset_name, market_status, coverage_status, as_of_date, updated_at
) VALUES (
  'k8s-staging-aapl', 'AAPL', 'Apple Inc.', 'active', 'active', CURRENT_DATE, now()
) ON CONFLICT (global_asset_id) DO UPDATE
SET canonical_symbol = EXCLUDED.canonical_symbol,
    asset_name = EXCLUDED.asset_name,
    market_status = EXCLUDED.market_status,
    coverage_status = EXCLUDED.coverage_status,
    as_of_date = EXCLUDED.as_of_date,
    updated_at = EXCLUDED.updated_at;

GRANT USAGE ON SCHEMA public TO signalops_subscriber_global_eod;
GRANT SELECT ON public.subscriber_global_warm_eod_assets TO signalops_subscriber_global_eod;
GRANT SELECT ON public.subscriber_global_assets TO signalops_subscriber_global_eod;
GRANT SELECT, INSERT, UPDATE ON public.subscriber_global_annual_financial_workflows TO signalops_subscriber_global_eod;
GRANT SELECT, INSERT, UPDATE ON public.subscriber_global_annual_financial_tasks TO signalops_subscriber_global_eod;
GRANT SELECT, INSERT, UPDATE ON public.subscriber_global_marketops_evidence_runs TO signalops_subscriber_global_eod;
GRANT SELECT, INSERT, UPDATE ON public.subscriber_global_marketops_evidence_records TO signalops_subscriber_global_eod;
GRANT SELECT, INSERT, UPDATE ON public.marketops_scheduled_job_statuses TO signalops_subscriber_global_eod;
GRANT SELECT, INSERT, UPDATE ON public.marketops_scheduled_job_runs TO signalops_subscriber_global_eod;
SQL

kubectl exec -n "$DATA_NAMESPACE" "$TEMPORAL_POD" -- env PGPASSWORD="$temporal_password" psql -U "$TEMPORAL_USER" -d "$TEMPORAL_DATABASE" -v ON_ERROR_STOP=1 -c 'SELECT 1' >/dev/null

primary_url="postgres://${PRIMARY_USER}:${primary_password}@${PRIMARY_STATEFULSET}.${DATA_NAMESPACE}.svc.cluster.local:5432/${PRIMARY_DATABASE}?sslmode=disable"
temporal_url="postgres://${TEMPORAL_USER}:${temporal_password}@${TEMPORAL_STATEFULSET}.${DATA_NAMESPACE}.svc.cluster.local:5432/${TEMPORAL_DATABASE}?sslmode=disable"

umask 077
{
  printf 'SIGNALOPS_K8S_MARKETOPS_RUNTIME_NON_PRODUCTION_APPROVED=true\n'
  printf 'SIGNALOPS_K8S_MARKETOPS_PROVIDER_CRONJOB_APPROVED=true\n'
  printf 'OPENBAO_ADMIN_TOKEN=%s\n' "$OPENBAO_ADMIN_TOKEN"
  printf 'SIGNALOPS_ENV=kubernetes-staging\n'
  printf 'SIGNALOPS_MARKETOPS_DATA_BOUNDARY_REQUIRED=true\n'
  printf 'SIGNALOPS_K8S_STATUS_RECORDING_REQUIRED=true\n'
  printf 'SIGNALOPS_DATABASE_URL=%s\n' "$primary_url"
  printf 'SIGNALOPS_MARKETOPS_DATABASE_URL=%s\n' "$primary_url"
  printf 'SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL=%s\n' "$primary_url"
  printf 'SIGNALOPS_K8S_STATUS_DATABASE_URL=%s\n' "$primary_url"
  printf 'SIGNALOPS_TEMPORAL_DATABASE_URL=%s\n' "$temporal_url"
  printf 'SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL=%s\n' "$temporal_url"
  printf 'SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_TEMPORAL_DATABASE_URL=%s\n' "$temporal_url"
  printf 'SIGNALOPS_FMP_BASE_URL=%s\n' "$SIGNALOPS_FMP_BASE_URL"
  printf 'SIGNALOPS_FMP_API_KEY=%s\n' "$SIGNALOPS_FMP_API_KEY"
} >"$RUNTIME_ENV"

scripts/provision_openbao_signalops_marketops_runtime_staging.sh "$RUNTIME_ENV" >/dev/null
kubectl apply -k "$MANIFEST_DIR" >/dev/null
kubectl get cronjob "$CRONJOB" -n "$NAMESPACE" >/dev/null || fail "CronJob missing after apply: ${NAMESPACE}/${CRONJOB}"

before_suspend="$(kubectl get cronjob "$CRONJOB" -n "$NAMESPACE" -o jsonpath='{.spec.suspend}')"
[[ "$before_suspend" == "true" ]] || fail "CronJob must start suspended before smoke; found suspend=${before_suspend}"

patch_file="$(mktemp -t signalops-provider-cronjob-smoke.XXXXXX.json)"
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
              "signalops.syncratic.io/provider-cronjob-smoke": "true",
              "signalops.syncratic.io/provider-cronjob-smoke-run": run_id,
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
                  {"name": "MARKETOPS_K8S_DRY_RUN", "value": "false"},
                  {"name": "MARKETOPS_FMP_ANNUAL_MAX_ASSETS", "value": "1"},
                  {"name": "MARKETOPS_FMP_ANNUAL_MAX_RETRIES", "value": "0"},
                  {"name": "MARKETOPS_K8S_RUN_ID", "value": run_id},
                  {"name": "MARKETOPS_K8S_SCHEDULE_LABEL", "value": "Kubernetes provider-enabled one-asset smoke"},
                  {"name": "MARKETOPS_K8S_TIMEZONE", "value": "UTC"},
                  {"name": "MARKETOPS_K8S_RUNNER_ID", "value": "kubernetes-provider-cronjob-smoke"},
                  {"name": "MARKETOPS_FMP_ANNUAL_CORRELATION_ID", "value": run_id}
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
[[ -n "$job" ]] || fail "CronJob did not create a provider smoke Job within timeout"

kubectl patch cronjob "$CRONJOB" -n "$NAMESPACE" --type=merge -p '{"spec":{"suspend":true}}' >/dev/null

pod=""
for _ in $(seq 1 180); do
  pod="$(kubectl get pods -n "$NAMESPACE" -l job-name="$job" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)"
  [[ -n "$pod" ]] || { sleep 1; continue; }
  exit_code="$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.status.containerStatuses[?(@.name=="marketops-job")].state.terminated.exitCode}' 2>/dev/null || true)"
  waiting_reason="$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.status.containerStatuses[?(@.name=="marketops-job")].state.waiting.reason}' 2>/dev/null || true)"
  case "$exit_code" in
    0) break ;;
    '') ;;
    *) kubectl logs -n "$NAMESPACE" "$pod" -c marketops-job >&2 || true; fail "provider smoke worker exited ${exit_code}" ;;
  esac
  [[ "$waiting_reason" == "ImagePullBackOff" || "$waiting_reason" == "ErrImagePull" || "$waiting_reason" == "CreateContainerConfigError" ]] && {
    kubectl describe pod "$pod" -n "$NAMESPACE" >&2 || true
    fail "provider smoke pod waiting reason ${waiting_reason}"
  }
  sleep 1
done
[[ -n "$pod" ]] || fail "provider smoke pod was not created"
exit_code="$(kubectl get pod "$pod" -n "$NAMESPACE" -o jsonpath='{.status.containerStatuses[?(@.name=="marketops-job")].state.terminated.exitCode}' 2>/dev/null || true)"
[[ "$exit_code" == "0" ]] || {
  kubectl describe pod "$pod" -n "$NAMESPACE" >&2 || true
  kubectl logs -n "$NAMESPACE" "$pod" -c vault-agent-init --tail=80 >&2 || true
  kubectl logs -n "$NAMESPACE" "$pod" -c marketops-job --tail=120 >&2 || true
  fail "provider smoke worker did not terminate successfully; exit_code=${exit_code:-missing}"
}

logs="$(kubectl logs -n "$NAMESPACE" "$pod" -c marketops-job)"
echo "$logs" | grep -q 'warm_assets=1' || fail "one-asset boundary missing from worker logs"
echo "$logs" | grep -q 'fmp_calls=1' || fail "expected exactly one FMP provider call"
echo "$logs" | grep -q 'correlation_id=' || fail "provider evidence correlation missing from worker logs"
if echo "$logs" | grep -q 'dry_run=true'; then
  fail "provider smoke unexpectedly ran in dry-run mode"
fi

status_line="$(kubectl exec -n "$DATA_NAMESPACE" "$PRIMARY_POD" -- env PGPASSWORD="$primary_password" psql -h 127.0.0.1 -U "$PRIMARY_USER" -d "$PRIMARY_DATABASE" -Atc "SELECT status || '|' || runner || '|' || COALESCE(exit_code::text,'') || '|' || COALESCE((detail->>'dry_run'),'') FROM marketops_scheduled_job_runs WHERE run_id='${RUN_ID}'")"
[[ "$status_line" == "succeeded|kubernetes-provider-cronjob-smoke|0|false" ]] || fail "DB-backed scheduler status parity mismatch: ${status_line:-empty}"

record_count="$(kubectl exec -n "$DATA_NAMESPACE" "$PRIMARY_POD" -- env PGPASSWORD="$primary_password" psql -h 127.0.0.1 -U "$PRIMARY_USER" -d "$PRIMARY_DATABASE" -Atc "SELECT count(*) FROM subscriber_global_marketops_evidence_records rec JOIN subscriber_global_marketops_evidence_runs run ON run.evidence_run_id=rec.evidence_run_id WHERE run.correlation_id='${RUN_ID}' AND run.evidence_kind='fundamental_annual' AND rec.global_asset_id='k8s-staging-aapl'")"
[[ "$record_count" =~ ^[0-9]+$ ]] || fail "could not read evidence record count"
[[ "$record_count" -ge 1 ]] || fail "provider evidence record was not appended"

job_count="$(kubectl get jobs -n "$NAMESPACE" -l "$SELECTOR" --no-headers 2>/dev/null | wc -l | tr -d ' ')"
[[ "$job_count" == "1" ]] || fail "expected exactly one provider smoke Job, found ${job_count}"

restore_cronjob
restored_suspend="$(kubectl get cronjob "$CRONJOB" -n "$NAMESPACE" -o jsonpath='{.spec.suspend}')"
[[ "$restored_suspend" == "true" ]] || fail "CronJob did not restore to suspended state"

kubectl delete job "$job" -n "$NAMESPACE" --wait=true >/dev/null
remaining_jobs="$(kubectl get jobs -n "$NAMESPACE" -l "$SELECTOR" --no-headers 2>/dev/null | wc -l | tr -d ' ')"
[[ "$remaining_jobs" == "0" ]] || fail "provider smoke Job cleanup incomplete: ${remaining_jobs} remain"

cat <<EOF
signalops_k8s_marketops_provider_cronjob_smoke_verified
namespace=${NAMESPACE}
cronjob=${CRONJOB}
image=${IMMUTABLE_IMAGE}
run_id=${RUN_ID}
job_created=true
jobs_created=1
resuspended=true
max_assets=1
max_retries=0
fmp_calls=1
scheduler_status_parity=verified
provider_polling=true
production_cutover_allowed=false
EOF
