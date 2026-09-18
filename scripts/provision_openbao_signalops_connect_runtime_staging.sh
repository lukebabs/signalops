#!/usr/bin/env bash
set -euo pipefail

RUNTIME_ENV_FILE="${1:-${SIGNALOPS_K8S_CONNECT_RUNTIME_ENV_FILE:-/tmp/signalops-openbao-connect-runtime-staging.env}}"
OPENBAO_NAMESPACE="${OPENBAO_NAMESPACE:-openbao}"
OPENBAO_POD="${OPENBAO_POD:-openbao-0}"
OPENBAO_ADDR="${OPENBAO_ADDR:-https://openbao.openbao.svc:8200}"
OPENBAO_KV_MOUNT="${OPENBAO_KV_MOUNT:-signalops}"
OPENBAO_CONNECT_ROLE="${OPENBAO_CONNECT_ROLE:-signalops-connect}"
OPENBAO_DENY_ROLE="${OPENBAO_DENY_ROLE:-signalops-marketops}"
CONNECT_NAMESPACE="${CONNECT_NAMESPACE:-signalops-connect}"
DENY_NAMESPACE="${DENY_NAMESPACE:-signalops-marketops}"
CONNECT_SERVICE_ACCOUNT="${CONNECT_SERVICE_ACCOUNT:-signalops-connect-worker}"
DENY_SERVICE_ACCOUNT="${DENY_SERVICE_ACCOUNT:-signalops-marketops-provider-worker}"
CONNECT_SECRET_PATH="${CONNECT_SECRET_PATH:-k8s/connect/connect-worker-runtime-staging}"

fail() {
  echo "openbao_signalops_connect_runtime_staging_failed: $*" >&2
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
[[ "${SIGNALOPS_K8S_CONNECT_RUNTIME_NON_PRODUCTION_APPROVED:-}" == "true" ]] || fail "set SIGNALOPS_K8S_CONNECT_RUNTIME_NON_PRODUCTION_APPROVED=true in the runtime env file"

required_keys=(SIGNALOPS_DATABASE_URL SIGNALOPS_BROKER_BROKERS SIGNALOPS_ENV SIGNALOPS_CONNECT_SHADOW_MODE)
for key in "${required_keys[@]}"; do
  [[ -n "${!key:-}" ]] || fail "missing required runtime key: ${key}"
done

python3 - <<'PYCHECK'
import os
from urllib.parse import urlparse
raw = os.environ.get('SIGNALOPS_DATABASE_URL', '').strip()
parsed = urlparse(raw)
if parsed.scheme not in {'postgres', 'postgresql'} or not parsed.hostname:
    raise SystemExit('invalid PostgreSQL URL for SIGNALOPS_DATABASE_URL')
host = parsed.hostname.lower()
if '.svc' not in host or any(x in host for x in ('production', 'prod-', '-prod', 'signalops-postgres-1')):
    raise SystemExit('non-production Kubernetes .svc PostgreSQL URL is required')
brokers = [x.strip() for x in os.environ.get('SIGNALOPS_BROKER_BROKERS', '').split(',') if x.strip()]
if not brokers:
    raise SystemExit('at least one broker is required')
for broker in brokers:
    broker_host = broker.rsplit(':', 1)[0].lower()
    if '.svc' not in broker_host or any(x in broker_host for x in ('production', 'prod-', '-prod', 'redpanda:')):
        raise SystemExit('non-production Kubernetes .svc broker endpoint is required')
if os.environ.get('SIGNALOPS_CONNECT_SHADOW_MODE') != 'true':
    raise SystemExit('SIGNALOPS_CONNECT_SHADOW_MODE must remain true')
PYCHECK

kubectl get pod "$OPENBAO_POD" -n "$OPENBAO_NAMESPACE" >/dev/null || fail "OpenBao pod not found: ${OPENBAO_NAMESPACE}/${OPENBAO_POD}"
kubectl get serviceaccount "$CONNECT_SERVICE_ACCOUNT" -n "$CONNECT_NAMESPACE" >/dev/null || fail "connect service account missing: ${CONNECT_NAMESPACE}/${CONNECT_SERVICE_ACCOUNT}"
kubectl get serviceaccount "$DENY_SERVICE_ACCOUNT" -n "$DENY_NAMESPACE" >/dev/null || fail "deny-proof service account missing: ${DENY_NAMESPACE}/${DENY_SERVICE_ACCOUNT}"

connect_jwt="$(kubectl create token "$CONNECT_SERVICE_ACCOUNT" -n "$CONNECT_NAMESPACE" --duration=10m)"
deny_jwt="$(kubectl create token "$DENY_SERVICE_ACCOUNT" -n "$DENY_NAMESPACE" --duration=10m)"
[[ -n "$connect_jwt" && -n "$deny_jwt" ]] || fail "failed to create bounded Kubernetes service-account tokens"

tmp_payload="$(mktemp)"
trap 'rm -f "$tmp_payload"' EXIT

export OPENBAO_ADDR ADMIN_TOKEN OPENBAO_KV_MOUNT CONNECT_SECRET_PATH OPENBAO_CONNECT_ROLE OPENBAO_DENY_ROLE connect_jwt deny_jwt
python3 - "$tmp_payload" <<'PYBUILD'
from pathlib import Path
import os
import shlex
import sys

keys = ['SIGNALOPS_DATABASE_URL','SIGNALOPS_BROKER_PROVIDER','SIGNALOPS_BROKER_BROKERS','SIGNALOPS_ENV','SIGNALOPS_CONNECT_SHADOW_MODE']
lines = [
    'set -euo pipefail',
    f'export BAO_ADDR={shlex.quote(os.environ["OPENBAO_ADDR"])}',
    f'export BAO_TOKEN={shlex.quote(os.environ["ADMIN_TOKEN"])}',
    'export BAO_CACERT="${BAO_CACERT:-/openbao/ca/ca.crt}"',
    'bao status >/dev/null',
    'bao kv put ' + shlex.quote(os.environ['OPENBAO_KV_MOUNT'] + '/' + os.environ['CONNECT_SECRET_PATH']) + ' \\',
]
present = [k for k in keys if os.environ.get(k)]
for i, key in enumerate(present):
    lines.append(f'  {key}={shlex.quote(os.environ[key])}' + (' \\' if i < len(present)-1 else ''))
lines.extend([
    'unset BAO_TOKEN',
    f'connect_token="$(bao write -field=token auth/kubernetes/login role={shlex.quote(os.environ["OPENBAO_CONNECT_ROLE"])} jwt={shlex.quote(os.environ["connect_jwt"])})"',
    f'deny_token="$(bao write -field=token auth/kubernetes/login role={shlex.quote(os.environ["OPENBAO_DENY_ROLE"])} jwt={shlex.quote(os.environ["deny_jwt"])})"',
    '[[ -n "$connect_token" && -n "$deny_token" ]]',
    f'BAO_TOKEN="$connect_token" bao kv get {shlex.quote(os.environ["OPENBAO_KV_MOUNT"] + "/" + os.environ["CONNECT_SECRET_PATH"])} >/dev/null',
    f'if BAO_TOKEN="$deny_token" bao kv get {shlex.quote(os.environ["OPENBAO_KV_MOUNT"] + "/" + os.environ["CONNECT_SECRET_PATH"])} >/tmp/signalops-openbao-connect-deny-out 2>/tmp/signalops-openbao-connect-deny-err; then echo "cross-plane denial failed" >&2; exit 1; fi',
])
Path(sys.argv[1]).write_text('\n'.join(lines) + '\n')
PYBUILD

kubectl exec -i -n "$OPENBAO_NAMESPACE" "$OPENBAO_POD" -- sh < "$tmp_payload" >/dev/null

cat <<EOF
openbao_signalops_connect_runtime_staging_verified
mount=${OPENBAO_KV_MOUNT}
connect_role=${OPENBAO_CONNECT_ROLE}
connect_namespace=${CONNECT_NAMESPACE}
connect_service_account=${CONNECT_SERVICE_ACCOUNT}
secret_path=${OPENBAO_KV_MOUNT}/${CONNECT_SECRET_PATH}
deny_role=${OPENBAO_DENY_ROLE}
deny_namespace=${DENY_NAMESPACE}
cross_plane_denied=true
secret_values=non_production_runtime_supplied
production_cutover_allowed=false
EOF
