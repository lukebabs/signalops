#!/usr/bin/env bash
set -euo pipefail

ENV_FILE="${1:-.env}"
NAMESPACE="${SIGNALOPS_K8S_DATA_NAMESPACE:-signalops-data}"
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
RUN_ID="${SIGNALOPS_K8S_MARKETOPS_DRY_RUN_RUN_ID:-k8s-dedicated-db-parity-$(date -u +%Y%m%dT%H%M%SZ)}"
RUNTIME_ENV="/tmp/signalops-openbao-marketops-dedicated-staging-runtime.env"
GATE_OUTPUT="/tmp/signalops-k8s-marketops-dedicated-staging-db-gate.out"

fail() {
  echo "signalops_k8s_marketops_dedicated_staging_db_gate_failed: $*" >&2
  exit 1
}

cleanup() {
  rm -f "$RUNTIME_ENV" "$GATE_OUTPUT"
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

[[ -n "${OPENBAO_ADMIN_TOKEN:-}" ]] || fail "OPENBAO_ADMIN_TOKEN is required in ${ENV_FILE}"

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

ensure_secret() {
  local name="$1"
  if kubectl get secret "$name" -n "$NAMESPACE" >/dev/null 2>&1; then
    return
  fi
  local password
  password="$(python3 - <<'PY'
import secrets
print(secrets.token_urlsafe(32))
PY
)"
  kubectl create secret generic "$name" -n "$NAMESPACE" \
    --from-literal=POSTGRES_PASSWORD="$password" >/dev/null
}

secret_value() {
  local name="$1"
  kubectl get secret "$name" -n "$NAMESPACE" -o jsonpath='{.data.POSTGRES_PASSWORD}' | base64 -d
}

kubectl get namespace "$NAMESPACE" >/dev/null || fail "namespace missing: ${NAMESPACE}"
ensure_secret "$PRIMARY_SECRET"
ensure_secret "$TEMPORAL_SECRET"

kubectl apply -k "$repo_dir/deploy/kubernetes/staging/marketops-data" >/dev/null
kubectl rollout status statefulset/"$PRIMARY_STATEFULSET" -n "$NAMESPACE" --timeout=180s >/dev/null
kubectl rollout status statefulset/"$TEMPORAL_STATEFULSET" -n "$NAMESPACE" --timeout=180s >/dev/null

primary_password="$(secret_value "$PRIMARY_SECRET")"
temporal_password="$(secret_value "$TEMPORAL_SECRET")"
[[ -n "$primary_password" ]] || fail "primary staging password could not be read"
[[ -n "$temporal_password" ]] || fail "temporal staging password could not be read"

kubectl exec -i -n "$NAMESPACE" "$PRIMARY_POD" -- env PGPASSWORD="$primary_password" psql -U "$PRIMARY_USER" -d "$PRIMARY_DATABASE" -v ON_ERROR_STOP=1 >/dev/null <<'SQL'
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

ALTER TABLE public.marketops_scheduled_job_statuses
  ADD COLUMN IF NOT EXISTS schedule text,
  ADD COLUMN IF NOT EXISTS timezone text,
  ADD COLUMN IF NOT EXISTS reason text,
  ADD COLUMN IF NOT EXISTS started_at timestamptz,
  ADD COLUMN IF NOT EXISTS completed_at timestamptz,
  ADD COLUMN IF NOT EXISTS exit_code integer,
  ADD COLUMN IF NOT EXISTS updated_at timestamptz NOT NULL DEFAULT now(),
  ADD COLUMN IF NOT EXISTS detail jsonb NOT NULL DEFAULT '{}'::jsonb,
  ADD COLUMN IF NOT EXISTS runner text;

ALTER TABLE public.marketops_scheduled_job_runs
  ADD COLUMN IF NOT EXISTS schedule text,
  ADD COLUMN IF NOT EXISTS timezone text,
  ADD COLUMN IF NOT EXISTS reason text,
  ADD COLUMN IF NOT EXISTS started_at timestamptz NOT NULL DEFAULT now(),
  ADD COLUMN IF NOT EXISTS finished_at timestamptz,
  ADD COLUMN IF NOT EXISTS completed_at timestamptz,
  ADD COLUMN IF NOT EXISTS exit_code integer,
  ADD COLUMN IF NOT EXISTS updated_at timestamptz NOT NULL DEFAULT now(),
  ADD COLUMN IF NOT EXISTS detail jsonb NOT NULL DEFAULT '{}'::jsonb,
  ADD COLUMN IF NOT EXISTS runner text;

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
GRANT SELECT, INSERT, UPDATE ON public.marketops_scheduled_job_statuses TO signalops_subscriber_global_eod;
GRANT SELECT, INSERT, UPDATE ON public.marketops_scheduled_job_runs TO signalops_subscriber_global_eod;
SQL

kubectl exec -n "$NAMESPACE" "$TEMPORAL_POD" -- env PGPASSWORD="$temporal_password" psql -U "$TEMPORAL_USER" -d "$TEMPORAL_DATABASE" -v ON_ERROR_STOP=1 -c 'SELECT 1' >/dev/null

primary_url="postgres://${PRIMARY_USER}:${primary_password}@${PRIMARY_STATEFULSET}.${NAMESPACE}.svc.cluster.local:5432/${PRIMARY_DATABASE}?sslmode=disable"
temporal_url="postgres://${TEMPORAL_USER}:${temporal_password}@${TEMPORAL_STATEFULSET}.${NAMESPACE}.svc.cluster.local:5432/${TEMPORAL_DATABASE}?sslmode=disable"

umask 077
{
  printf 'SIGNALOPS_K8S_MARKETOPS_RUNTIME_NON_PRODUCTION_APPROVED=true\n'
  printf 'SIGNALOPS_K8S_MARKETOPS_DRY_RUN_APPROVED=true\n'
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
} >"$RUNTIME_ENV"

SIGNALOPS_MARKETOPS_K8S_JOB_RUNNER_IMAGE="${SIGNALOPS_MARKETOPS_K8S_JOB_RUNNER_IMAGE:-ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner:staging}" \
SIGNALOPS_K8S_RUNTIME_SMOKE_NAMESPACE="$NAMESPACE" \
SIGNALOPS_K8S_RUNTIME_SMOKE_POSTGRES_POD="$PRIMARY_POD" \
SIGNALOPS_K8S_MARKETOPS_DRY_RUN_RUN_ID="$RUN_ID" \
SIGNALOPS_K8S_STATUS_PARITY_VERIFY=true \
  "$repo_dir/scripts/run_k8s_marketops_non_provider_dry_run_job.sh" "$RUNTIME_ENV" >"$GATE_OUTPUT"

grep -q 'scheduler_status_parity=verified' "$GATE_OUTPUT" || fail "scheduler status parity marker missing"
grep -q 'provider_polling=false' "$GATE_OUTPUT" || fail "provider polling guard marker missing"

cat "$GATE_OUTPUT"
cat <<EOF
signalops_k8s_marketops_dedicated_staging_db_gate_verified
namespace=${NAMESPACE}
primary_service=${PRIMARY_STATEFULSET}
temporal_service=${TEMPORAL_STATEFULSET}
run_id=${RUN_ID}
scheduler_status_parity=verified
provider_polling=false
production_cutover_allowed=false
EOF
