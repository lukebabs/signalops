#!/usr/bin/env bash
set -euo pipefail

ENV_FILE="${1:-.env}"
RUNTIME_FILE="${2:-/tmp/signalops-openbao-connect-runtime-staging.env}"
RUNTIME_SMOKE_NAMESPACE="${SIGNALOPS_K8S_RUNTIME_SMOKE_NAMESPACE:-syncratic-runtime-smoke}"
RUNTIME_SMOKE_SECRET="${SIGNALOPS_K8S_RUNTIME_SMOKE_POSTGRES_SECRET:-syncratic-refactor-smoke-postgres-auth}"
RUNTIME_SMOKE_POSTGRES_POD="${SIGNALOPS_K8S_RUNTIME_SMOKE_POSTGRES_POD:-syncratic-refactor-smoke-syncratic-phase1-postgres-0}"
RUNTIME_SMOKE_POSTGRES_SERVICE="${SIGNALOPS_K8S_RUNTIME_SMOKE_POSTGRES_SERVICE:-syncratic-refactor-smoke-syncratic-phase1-postgres}"
BROKER_SERVICE="${SIGNALOPS_K8S_CONNECT_BROKER_SERVICE:-signalops-redpanda-staging}"
BROKER_NAMESPACE="${SIGNALOPS_K8S_DATA_NAMESPACE:-signalops-data}"

fail() {
  echo "signalops_k8s_connect_runtime_staging_env_failed: $*" >&2
  exit 1
}

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"
[[ -r "$ENV_FILE" ]] || fail "env file is missing or unreadable: ${ENV_FILE}"

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=./lib/dotenv.sh
source "$repo_dir/scripts/lib/dotenv.sh"
load_dotenv "$ENV_FILE"
[[ -n "${OPENBAO_ADMIN_TOKEN:-}" ]] || fail "OPENBAO_ADMIN_TOKEN is required in ${ENV_FILE} or environment"

kubectl get namespace "$RUNTIME_SMOKE_NAMESPACE" >/dev/null || fail "runtime-smoke namespace missing: ${RUNTIME_SMOKE_NAMESPACE}"
kubectl get secret "$RUNTIME_SMOKE_SECRET" -n "$RUNTIME_SMOKE_NAMESPACE" >/dev/null || fail "runtime-smoke Postgres secret missing"
kubectl get pod "$RUNTIME_SMOKE_POSTGRES_POD" -n "$RUNTIME_SMOKE_NAMESPACE" >/dev/null || fail "runtime-smoke Postgres pod missing"
kubectl get service "$BROKER_SERVICE" -n "$BROKER_NAMESPACE" >/dev/null || fail "staging broker service missing: ${BROKER_NAMESPACE}/${BROKER_SERVICE}"

secret_json="$(kubectl get secret "$RUNTIME_SMOKE_SECRET" -n "$RUNTIME_SMOKE_NAMESPACE" -o json)"
export secret_json OPENBAO_ADMIN_TOKEN RUNTIME_SMOKE_NAMESPACE RUNTIME_SMOKE_POSTGRES_SERVICE BROKER_SERVICE BROKER_NAMESPACE
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
broker = f"{os.environ['BROKER_SERVICE']}.{os.environ['BROKER_NAMESPACE']}.svc.cluster.local:9092"
lines = {
    'SIGNALOPS_K8S_CONNECT_RUNTIME_NON_PRODUCTION_APPROVED': 'true',
    'OPENBAO_ADMIN_TOKEN': os.environ['OPENBAO_ADMIN_TOKEN'],
    'SIGNALOPS_ENV': 'kubernetes-staging',
    'SIGNALOPS_DATABASE_URL': url,
    'SIGNALOPS_BROKER_PROVIDER': 'redpanda',
    'SIGNALOPS_BROKER_BROKERS': broker,
    'SIGNALOPS_CONNECT_SHADOW_MODE': 'true',
}
target.write_text('\n'.join(f'{k}={v}' for k, v in lines.items()) + '\n')
target.chmod(0o600)
PYENV

python3 - <<'PYPGENV' >/tmp/signalops-connect-runtime-smoke-pg.env
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
. /tmp/signalops-connect-runtime-smoke-pg.env
set +a
kubectl exec -i -n "$RUNTIME_SMOKE_NAMESPACE" "$RUNTIME_SMOKE_POSTGRES_POD" -- env PGPASSWORD="$PGPASSWORD" psql -v ON_ERROR_STOP=1 -U "$PGUSER" -d "$PGDATABASE" < migrations/000058_cyberops_connect_ingress.up.sql >/dev/null
rm -f /tmp/signalops-connect-runtime-smoke-pg.env

cat <<EOF
signalops_k8s_connect_runtime_staging_env_created
path=${RUNTIME_FILE}
mode=0600
database=runtime-smoke
broker=${BROKER_SERVICE}.${BROKER_NAMESPACE}.svc.cluster.local:9092
connect_tables=true
secret_values_printed=false
production_cutover_allowed=false
EOF
