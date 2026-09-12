#!/usr/bin/env bash
set -euo pipefail

OPENBAO_NAMESPACE="${OPENBAO_NAMESPACE:-openbao}"
OPENBAO_POD="${OPENBAO_POD:-openbao-0}"
OPENBAO_ADDR="${OPENBAO_ADDR:-https://openbao.openbao.svc:8200}"
OPENBAO_KV_MOUNT="${OPENBAO_KV_MOUNT:-signalops}"
OPENBAO_MARKETOPS_ROLE="${OPENBAO_MARKETOPS_ROLE:-signalops-marketops}"
OPENBAO_DENY_ROLE="${OPENBAO_DENY_ROLE:-signalops-app}"
MARKETOPS_NAMESPACE="${MARKETOPS_NAMESPACE:-signalops-marketops}"
DENY_NAMESPACE="${DENY_NAMESPACE:-signalops-app}"
MARKETOPS_SERVICE_ACCOUNT="${MARKETOPS_SERVICE_ACCOUNT:-signalops-marketops-provider-worker}"
DENY_SERVICE_ACCOUNT="${DENY_SERVICE_ACCOUNT:-signalops-gateway}"
MARKETOPS_SECRET_PATH="${MARKETOPS_SECRET_PATH:-k8s/marketops/marketops-worker-runtime-staging}"
RUNTIME_ENV_FILE="${1:-${SIGNALOPS_K8S_MARKETOPS_RUNTIME_ENV_FILE:-/etc/signalops/openbao-signalops-marketops-staging-runtime.env}}"

fail() {
  echo "openbao_signalops_marketops_runtime_staging_failed: $*" >&2
  exit 1
}

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"
[[ -r "$RUNTIME_ENV_FILE" ]] || fail "runtime env file is missing or unreadable: ${RUNTIME_ENV_FILE}"

set -a
# shellcheck disable=SC1090
source "$RUNTIME_ENV_FILE"
set +a

ADMIN_TOKEN="${BAO_TOKEN:-${VAULT_TOKEN:-${OPENBAO_TOKEN:-${OPENBAO_ADMIN_TOKEN:-}}}}"
[[ -n "$ADMIN_TOKEN" ]] || fail "runtime env file must provide BAO_TOKEN, VAULT_TOKEN, OPENBAO_TOKEN, or OPENBAO_ADMIN_TOKEN"
[[ "${SIGNALOPS_K8S_MARKETOPS_RUNTIME_NON_PRODUCTION_APPROVED:-}" == "true" ]] || fail "set SIGNALOPS_K8S_MARKETOPS_RUNTIME_NON_PRODUCTION_APPROVED=true in the runtime env file"

required_keys=(
  SIGNALOPS_DATABASE_URL
  SIGNALOPS_TEMPORAL_DATABASE_URL
  SIGNALOPS_MARKETOPS_DATABASE_URL
  SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL
  SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL
  SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_TEMPORAL_DATABASE_URL
)
for key in "${required_keys[@]}"; do
  [[ -n "${!key:-}" ]] || fail "missing required runtime key: ${key}"
done

python3 - <<'PYCHECK'
import os
from urllib.parse import urlparse

required_urls = [
    'SIGNALOPS_DATABASE_URL',
    'SIGNALOPS_TEMPORAL_DATABASE_URL',
    'SIGNALOPS_MARKETOPS_DATABASE_URL',
    'SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL',
    'SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL',
    'SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_TEMPORAL_DATABASE_URL',
]
for key in required_urls:
    raw = os.environ.get(key, '').strip()
    parsed = urlparse(raw)
    if parsed.scheme not in {'postgres', 'postgresql'} or not parsed.hostname:
        raise SystemExit(f'invalid PostgreSQL URL for {key}')
    host = parsed.hostname.lower()
    forbidden_exact = {
        'postgres.staging.invalid',
        'timescaledb.staging.invalid',
        'marketops-postgres.staging.invalid',
        'marketops-timescaledb.staging.invalid',
    }
    forbidden_hosts = {
        'localhost',
        '127.0.0.1',
        'postgres',
        'timescaledb',
        'marketops-postgres',
        'marketops-timescaledb',
    }
    forbidden_fragments = (
        'production',
        'prod-',
        '-prod',
        'signalops-postgres-1',
        'signalops-marketops-postgres-1',
        'signalops-marketops-timescaledb-1',
    )
    if host in forbidden_exact or host.endswith('.invalid'):
        raise SystemExit(f'placeholder host is not allowed for {key}')
    if host in forbidden_hosts or any(fragment in host for fragment in forbidden_fragments):
        raise SystemExit(f'production-like host is not allowed for {key}')
    if '.svc' not in host:
        raise SystemExit(f'non-production Kubernetes service DNS name is required for {key}')
PYCHECK

kubectl get pod "$OPENBAO_POD" -n "$OPENBAO_NAMESPACE" >/dev/null || fail "OpenBao pod not found: ${OPENBAO_NAMESPACE}/${OPENBAO_POD}"
kubectl get serviceaccount "$MARKETOPS_SERVICE_ACCOUNT" -n "$MARKETOPS_NAMESPACE" >/dev/null || fail "marketops service account missing: ${MARKETOPS_NAMESPACE}/${MARKETOPS_SERVICE_ACCOUNT}"
kubectl get serviceaccount "$DENY_SERVICE_ACCOUNT" -n "$DENY_NAMESPACE" >/dev/null || fail "deny-proof service account missing: ${DENY_NAMESPACE}/${DENY_SERVICE_ACCOUNT}"

marketops_jwt="$(kubectl create token "$MARKETOPS_SERVICE_ACCOUNT" -n "$MARKETOPS_NAMESPACE" --duration=10m)"
deny_jwt="$(kubectl create token "$DENY_SERVICE_ACCOUNT" -n "$DENY_NAMESPACE" --duration=10m)"
[[ -n "$marketops_jwt" && -n "$deny_jwt" ]] || fail "failed to create bounded Kubernetes service-account tokens"

tmp_payload="$(mktemp)"
trap 'rm -f "$tmp_payload"' EXIT

export OPENBAO_ADDR ADMIN_TOKEN OPENBAO_KV_MOUNT MARKETOPS_SECRET_PATH OPENBAO_MARKETOPS_ROLE OPENBAO_DENY_ROLE marketops_jwt deny_jwt
python3 - "$tmp_payload" <<'PYBUILD'
from pathlib import Path
import os
import shlex
import sys

keys = [
    'SIGNALOPS_DATABASE_URL',
    'SIGNALOPS_TEMPORAL_DATABASE_URL',
    'SIGNALOPS_MARKETOPS_DATABASE_URL',
    'SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL',
    'SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL',
    'SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_TEMPORAL_DATABASE_URL',
    'SIGNALOPS_MASSIVE_BASE_URL',
    'SIGNALOPS_MASSIVE_API_KEY',
    'SIGNALOPS_FMP_BASE_URL',
    'SIGNALOPS_FMP_API_KEY',
    'SIGNALOPS_ENV',
    'SIGNALOPS_MARKETOPS_DATA_BOUNDARY_REQUIRED',
]
lines = [
    'set -euo pipefail',
    f'export BAO_ADDR={shlex.quote(os.environ["OPENBAO_ADDR"])}',
    f'export BAO_TOKEN={shlex.quote(os.environ["ADMIN_TOKEN"])}',
    'export BAO_CACERT="${BAO_CACERT:-/openbao/ca/ca.crt}"',
    'bao status >/dev/null',
    'bao kv put ' + shlex.quote(os.environ['OPENBAO_KV_MOUNT'] + '/' + os.environ['MARKETOPS_SECRET_PATH']) + ' \\',
]
present_keys = [key for key in keys if os.environ.get(key)]
for index, key in enumerate(present_keys):
    suffix = ' \\' if index < len(present_keys) - 1 else ''
    lines.append(f'  {key}={shlex.quote(os.environ[key])}{suffix}')
lines.extend([
    'unset BAO_TOKEN',
    f'marketops_token="$(bao write -field=token auth/kubernetes/login role={shlex.quote(os.environ["OPENBAO_MARKETOPS_ROLE"])} jwt={shlex.quote(os.environ["marketops_jwt"])})"',
    f'deny_token="$(bao write -field=token auth/kubernetes/login role={shlex.quote(os.environ["OPENBAO_DENY_ROLE"])} jwt={shlex.quote(os.environ["deny_jwt"])})"',
    '[[ -n "$marketops_token" && -n "$deny_token" ]]',
    f'BAO_TOKEN="$marketops_token" bao kv get {shlex.quote(os.environ["OPENBAO_KV_MOUNT"] + "/" + os.environ["MARKETOPS_SECRET_PATH"])} >/dev/null',
    f'if BAO_TOKEN="$deny_token" bao kv get {shlex.quote(os.environ["OPENBAO_KV_MOUNT"] + "/" + os.environ["MARKETOPS_SECRET_PATH"])} >/tmp/signalops-openbao-deny-out 2>/tmp/signalops-openbao-deny-err; then echo "cross-plane denial failed" >&2; exit 1; fi',
])
Path(sys.argv[1]).write_text('\n'.join(lines) + '\n')
PYBUILD

kubectl exec -i -n "$OPENBAO_NAMESPACE" "$OPENBAO_POD" -- sh < "$tmp_payload" >/dev/null

cat <<EOF
openbao_signalops_marketops_runtime_staging_verified
mount=${OPENBAO_KV_MOUNT}
marketops_role=${OPENBAO_MARKETOPS_ROLE}
marketops_namespace=${MARKETOPS_NAMESPACE}
marketops_service_account=${MARKETOPS_SERVICE_ACCOUNT}
secret_path=${OPENBAO_KV_MOUNT}/${MARKETOPS_SECRET_PATH}
deny_role=${OPENBAO_DENY_ROLE}
deny_namespace=${DENY_NAMESPACE}
cross_plane_denied=true
secret_values=non_production_runtime_supplied
production_cutover_allowed=false
EOF
