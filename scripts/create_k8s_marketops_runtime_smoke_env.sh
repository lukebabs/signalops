#!/usr/bin/env bash
set -euo pipefail

ENV_FILE="${1:-.env}"
RUNTIME_FILE="${2:-/tmp/signalops-openbao-marketops-runtime-staging.env}"
RUNTIME_SMOKE_NAMESPACE="${SIGNALOPS_K8S_RUNTIME_SMOKE_NAMESPACE:-syncratic-runtime-smoke}"
RUNTIME_SMOKE_SECRET="${SIGNALOPS_K8S_RUNTIME_SMOKE_POSTGRES_SECRET:-syncratic-refactor-smoke-postgres-auth}"
RUNTIME_SMOKE_POSTGRES_POD="${SIGNALOPS_K8S_RUNTIME_SMOKE_POSTGRES_POD:-syncratic-refactor-smoke-syncratic-phase1-postgres-0}"
RUNTIME_SMOKE_POSTGRES_SERVICE="${SIGNALOPS_K8S_RUNTIME_SMOKE_POSTGRES_SERVICE:-syncratic-refactor-smoke-syncratic-phase1-postgres}"

fail() {
  echo "signalops_k8s_marketops_runtime_smoke_env_failed: $*" >&2
  exit 1
}

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"
[[ -f "$ENV_FILE" ]] || fail "env file not found: ${ENV_FILE}"

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=./lib/dotenv.sh
source "$repo_dir/scripts/lib/dotenv.sh"
load_dotenv "$ENV_FILE"
[[ -n "${OPENBAO_ADMIN_TOKEN:-}" ]] || fail "OPENBAO_ADMIN_TOKEN is required in ${ENV_FILE} or environment"

kubectl get namespace "$RUNTIME_SMOKE_NAMESPACE" >/dev/null || fail "runtime-smoke namespace missing: ${RUNTIME_SMOKE_NAMESPACE}"
kubectl get secret "$RUNTIME_SMOKE_SECRET" -n "$RUNTIME_SMOKE_NAMESPACE" >/dev/null || fail "runtime-smoke Postgres secret missing"
kubectl get pod "$RUNTIME_SMOKE_POSTGRES_POD" -n "$RUNTIME_SMOKE_NAMESPACE" >/dev/null || fail "runtime-smoke Postgres pod missing"

secret_json="$(kubectl get secret "$RUNTIME_SMOKE_SECRET" -n "$RUNTIME_SMOKE_NAMESPACE" -o json)"
export secret_json OPENBAO_ADMIN_TOKEN RUNTIME_SMOKE_NAMESPACE RUNTIME_SMOKE_POSTGRES_SERVICE
python3 - "$RUNTIME_FILE" <<'PYENV'
import base64, json, os, sys
from pathlib import Path
from urllib.parse import urlparse, urlunparse

target = Path(sys.argv[1])
secret = json.loads(os.environ['secret_json'])
raw = base64.b64decode(secret['data']['GATEWAY_MIGRATION_DSN']).decode().strip()
parsed = urlparse(raw)
if parsed.scheme not in {'postgres', 'postgresql'} or not parsed.username or not parsed.hostname:
    raise SystemExit('runtime-smoke GATEWAY_MIGRATION_DSN is not an expected PostgreSQL URL')
svc_host = f"{os.environ['RUNTIME_SMOKE_POSTGRES_SERVICE']}.{os.environ['RUNTIME_SMOKE_NAMESPACE']}.svc.cluster.local"
netloc = parsed.username
if parsed.password is not None:
    netloc += ':' + parsed.password
netloc += '@' + svc_host
if parsed.port:
    netloc += f':{parsed.port}'
url = urlunparse((parsed.scheme, netloc, parsed.path or '/postgres', parsed.params, parsed.query or 'sslmode=disable', parsed.fragment))
lines = {
    'SIGNALOPS_K8S_MARKETOPS_RUNTIME_NON_PRODUCTION_APPROVED': 'true',
    'SIGNALOPS_K8S_MARKETOPS_DRY_RUN_APPROVED': 'true',
    'OPENBAO_ADMIN_TOKEN': os.environ['OPENBAO_ADMIN_TOKEN'],
    'SIGNALOPS_ENV': 'kubernetes-staging',
    'SIGNALOPS_MARKETOPS_DATA_BOUNDARY_REQUIRED': 'true',
    'SIGNALOPS_K8S_STATUS_RECORDING_REQUIRED': 'true',
    'SIGNALOPS_DATABASE_URL': url,
    'SIGNALOPS_TEMPORAL_DATABASE_URL': url,
    'SIGNALOPS_MARKETOPS_DATABASE_URL': url,
    'SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL': url,
    'SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL': url,
    'SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_TEMPORAL_DATABASE_URL': url,
}
target.write_text('\n'.join(f'{k}={v}' for k, v in lines.items()) + '\n')
target.chmod(0o600)
PYENV

python3 - <<'PYPGENV' >/tmp/signalops-runtime-smoke-pg.env
import base64, json, os
from urllib.parse import urlparse
secret=json.loads(os.environ['secret_json'])
raw=base64.b64decode(secret['data']['GATEWAY_MIGRATION_DSN']).decode().strip()
p=urlparse(raw)
print('PGUSER=' + (p.username or 'postgres'))
print('PGPASSWORD=' + (p.password or ''))
print('PGDATABASE=' + ((p.path or '/postgres').lstrip('/') or 'postgres'))
PYPGENV
set -a
. /tmp/signalops-runtime-smoke-pg.env
set +a
kubectl exec -i -n "$RUNTIME_SMOKE_NAMESPACE" "$RUNTIME_SMOKE_POSTGRES_POD" -- env PGPASSWORD="$PGPASSWORD" psql -v ON_ERROR_STOP=1 -U "$PGUSER" -d "$PGDATABASE" <<'SQL' >/dev/null
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'signalops_subscriber_global_eod') THEN
    CREATE ROLE signalops_subscriber_global_eod;
  END IF;
END $$;
CREATE TABLE IF NOT EXISTS public.subscriber_global_warm_eod_assets (
  global_asset_id text PRIMARY KEY,
  canonical_symbol text NOT NULL,
  priority integer NOT NULL DEFAULT 1
);
CREATE TABLE IF NOT EXISTS public.marketops_scheduled_job_statuses (
  job_id text PRIMARY KEY,
  schedule text NOT NULL,
  timezone text NOT NULL,
  status text NOT NULL CHECK (status IN ('pending','running','succeeded','failed','skipped','recovery_needed','recovering','degraded')),
  reason text NOT NULL DEFAULT '',
  started_at timestamptz,
  completed_at timestamptz,
  exit_code integer,
  detail jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(detail) = 'object'),
  runner text NOT NULL DEFAULT '',
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS public.marketops_scheduled_job_runs (
  run_id text PRIMARY KEY,
  job_id text NOT NULL,
  schedule text NOT NULL,
  timezone text NOT NULL,
  status text NOT NULL CHECK (status IN ('running','succeeded','failed','skipped','recovery_needed','recovering','degraded')),
  reason text NOT NULL DEFAULT '',
  started_at timestamptz NOT NULL,
  completed_at timestamptz,
  exit_code integer,
  detail jsonb NOT NULL DEFAULT '{}'::jsonb CHECK (jsonb_typeof(detail) = 'object'),
  runner text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);
INSERT INTO public.subscriber_global_warm_eod_assets(global_asset_id, canonical_symbol, priority)
VALUES ('k8s-staging-smoke-aapl', 'AAPL', 1)
ON CONFLICT (global_asset_id) DO UPDATE SET canonical_symbol = EXCLUDED.canonical_symbol, priority = EXCLUDED.priority;
GRANT USAGE ON SCHEMA public TO signalops_subscriber_global_eod;
GRANT SELECT ON public.subscriber_global_warm_eod_assets TO signalops_subscriber_global_eod;
GRANT SELECT, INSERT, UPDATE ON public.marketops_scheduled_job_statuses, public.marketops_scheduled_job_runs TO signalops_subscriber_global_eod;
SQL
rm -f /tmp/signalops-runtime-smoke-pg.env

cat <<EOF
signalops_k8s_marketops_runtime_smoke_env_created
path=${RUNTIME_FILE}
mode=0600
database=runtime-smoke
status_tables=true
secret_values_printed=false
EOF
