#!/usr/bin/env bash
set -euo pipefail

ENV_FILE="${1:-.env}"
OPENBAO_NAMESPACE="${OPENBAO_NAMESPACE:-openbao}"
OPENBAO_POD="${OPENBAO_POD:-openbao-0}"
OPENBAO_ADDR="${OPENBAO_ADDR:-https://openbao.openbao.svc:8200}"
OPENBAO_KV_MOUNT="${OPENBAO_KV_MOUNT:-signalops}"
OPENBAO_CONNECT_ROLE="${OPENBAO_CONNECT_ROLE:-signalops-connect}"
OPENBAO_DENY_ROLE="${OPENBAO_DENY_ROLE:-signalops-marketops}"
CONNECT_NAMESPACE="${CONNECT_NAMESPACE:-signalops-connect}"
DENY_NAMESPACE="${DENY_NAMESPACE:-signalops-marketops}"
CONNECT_BOUND_SERVICE_ACCOUNTS="${CONNECT_BOUND_SERVICE_ACCOUNTS:-signalops-connect-worker,signalops-connect-outbox,signalops-connect-secret-reader}"
DENY_SERVICE_ACCOUNT="${DENY_SERVICE_ACCOUNT:-signalops-marketops-provider-worker}"
CONNECT_SECRET_PATH="${CONNECT_SECRET_PATH:-k8s/connect/connect-worker-runtime-staging}"
CONNECT_POLICY_NAME="${CONNECT_POLICY_NAME:-signalops-connect-staging-read}"
DENY_POLICY_NAME="${DENY_POLICY_NAME:-signalops-marketops-staging-read}"
TOKEN_TTL="${TOKEN_TTL:-1h}"

fail() {
  echo "openbao_signalops_connect_staging_failed: $*" >&2
  exit 1
}

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"
[[ -f "$ENV_FILE" ]] || fail "env file not found: ${ENV_FILE}"

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_dir"
# shellcheck source=./lib/dotenv.sh
source "$repo_dir/scripts/lib/dotenv.sh"
load_dotenv "$ENV_FILE"

ADMIN_TOKEN="${BAO_TOKEN:-${VAULT_TOKEN:-${OPENBAO_TOKEN:-${OPENBAO_ADMIN_TOKEN:-}}}}"
[[ -n "$ADMIN_TOKEN" ]] || fail "set BAO_TOKEN, VAULT_TOKEN, OPENBAO_TOKEN, or OPENBAO_ADMIN_TOKEN in env file/environment"

kubectl get pod "$OPENBAO_POD" -n "$OPENBAO_NAMESPACE" >/dev/null || fail "OpenBao pod not found: ${OPENBAO_NAMESPACE}/${OPENBAO_POD}"
kubectl get namespace "$CONNECT_NAMESPACE" >/dev/null || fail "namespace missing: ${CONNECT_NAMESPACE}"
kubectl get namespace "$DENY_NAMESPACE" >/dev/null || fail "namespace missing: ${DENY_NAMESPACE}"
IFS=',' read -r -a connect_sas <<< "$CONNECT_BOUND_SERVICE_ACCOUNTS"
for sa in "${connect_sas[@]}"; do
  kubectl get serviceaccount "$sa" -n "$CONNECT_NAMESPACE" >/dev/null || fail "connect service account missing: ${CONNECT_NAMESPACE}/${sa}"
done
kubectl get serviceaccount "$DENY_SERVICE_ACCOUNT" -n "$DENY_NAMESPACE" >/dev/null || fail "deny-proof service account missing: ${DENY_NAMESPACE}/${DENY_SERVICE_ACCOUNT}"

connect_jwt="$(kubectl create token "${connect_sas[0]}" -n "$CONNECT_NAMESPACE" --duration=10m)"
deny_jwt="$(kubectl create token "$DENY_SERVICE_ACCOUNT" -n "$DENY_NAMESPACE" --duration=10m)"
[[ -n "$connect_jwt" && -n "$deny_jwt" ]] || fail "failed to create bounded Kubernetes service-account tokens"

tmp_payload="$(mktemp)"
trap 'rm -f "$tmp_payload"' EXIT
export OPENBAO_ADDR ADMIN_TOKEN OPENBAO_KV_MOUNT CONNECT_POLICY_NAME DENY_POLICY_NAME
export OPENBAO_CONNECT_ROLE OPENBAO_DENY_ROLE CONNECT_BOUND_SERVICE_ACCOUNTS CONNECT_NAMESPACE DENY_NAMESPACE DENY_SERVICE_ACCOUNT TOKEN_TTL CONNECT_SECRET_PATH connect_jwt deny_jwt

python3 - "$tmp_payload" <<'PYBUILD'
from pathlib import Path
import os
import sys

payload = """set -euo pipefail
export BAO_ADDR=\"__OPENBAO_ADDR__\"
export BAO_TOKEN=\"__ADMIN_TOKEN__\"
export BAO_CACERT=\"${BAO_CACERT:-/openbao/ca/ca.crt}\"

bao status >/dev/null

if ! bao secrets list -format=json | grep -q '\"__OPENBAO_KV_MOUNT__/\"'; then
  bao secrets enable -path=\"__OPENBAO_KV_MOUNT__\" -version=2 kv >/dev/null
fi

if ! bao auth list -format=json | grep -q '\"kubernetes/\"'; then
  bao auth enable kubernetes >/dev/null
fi

k8s_ca=\"$(cat /var/run/secrets/kubernetes.io/serviceaccount/ca.crt)\"
k8s_jwt=\"$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)\"
bao write auth/kubernetes/config \\
  token_reviewer_jwt=\"$k8s_jwt\" \\
  kubernetes_host=\"https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT}\" \\
  kubernetes_ca_cert=\"$k8s_ca\" >/dev/null

cat >/tmp/signalops-connect-policy.hcl <<'EOP'
path \"__OPENBAO_KV_MOUNT__/data/k8s/connect/*\" {
  capabilities = [\"read\"]
}
path \"__OPENBAO_KV_MOUNT__/metadata/k8s/connect/*\" {
  capabilities = [\"list\", \"read\"]
}
EOP
bao policy write \"__CONNECT_POLICY_NAME__\" /tmp/signalops-connect-policy.hcl >/dev/null

cat >/tmp/signalops-marketops-policy.hcl <<'EOP'
path \"__OPENBAO_KV_MOUNT__/data/k8s/marketops/*\" {
  capabilities = [\"read\"]
}
path \"__OPENBAO_KV_MOUNT__/metadata/k8s/marketops/*\" {
  capabilities = [\"list\", \"read\"]
}
EOP
bao policy write \"__DENY_POLICY_NAME__\" /tmp/signalops-marketops-policy.hcl >/dev/null

bao write \"auth/kubernetes/role/__OPENBAO_CONNECT_ROLE__\" \\
  bound_service_account_names=\"__CONNECT_BOUND_SERVICE_ACCOUNTS__\" \\
  bound_service_account_namespaces=\"__CONNECT_NAMESPACE__\" \\
  policies=\"__CONNECT_POLICY_NAME__\" \\
  ttl=\"__TOKEN_TTL__\" >/dev/null

bao write \"auth/kubernetes/role/__OPENBAO_DENY_ROLE__\" \\
  bound_service_account_names=\"__DENY_SERVICE_ACCOUNT__\" \\
  bound_service_account_namespaces=\"__DENY_NAMESPACE__\" \\
  policies=\"__DENY_POLICY_NAME__\" \\
  ttl=\"__TOKEN_TTL__\" >/dev/null

bao kv put \"__OPENBAO_KV_MOUNT__/__CONNECT_SECRET_PATH__\" \\
  SIGNALOPS_DATABASE_URL=\"postgres://staging_user:staging_password@postgres.staging.invalid:5432/signalops?sslmode=require\" \\
  SIGNALOPS_BROKER_BROKERS=\"redpanda.staging.invalid:9092\" \\
  SIGNALOPS_ENV=\"kubernetes-staging\" \\
  SIGNALOPS_CONNECT_SHADOW_MODE=\"true\" >/dev/null

unset BAO_TOKEN
connect_token=\"$(bao write -field=token auth/kubernetes/login role=\"__OPENBAO_CONNECT_ROLE__\" jwt=\"__CONNECT_JWT__\")\"
deny_token=\"$(bao write -field=token auth/kubernetes/login role=\"__OPENBAO_DENY_ROLE__\" jwt=\"__DENY_JWT__\")\"
[[ -n \"$connect_token\" && -n \"$deny_token\" ]] || { echo \"bounded OpenBao token issue failed\" >&2; exit 1; }

BAO_TOKEN=\"$connect_token\" bao kv get \"__OPENBAO_KV_MOUNT__/__CONNECT_SECRET_PATH__\" >/dev/null
if BAO_TOKEN=\"$deny_token\" bao kv get \"__OPENBAO_KV_MOUNT__/__CONNECT_SECRET_PATH__\" >/tmp/deny_out 2>/tmp/deny_err; then
  echo \"cross-plane denial failed: marketops role read connect secret path\" >&2
  exit 1
fi

cat <<EOF
openbao_signalops_connect_staging_verified
mount=__OPENBAO_KV_MOUNT__
connect_role=__OPENBAO_CONNECT_ROLE__
connect_namespace=__CONNECT_NAMESPACE__
connect_service_accounts=__CONNECT_BOUND_SERVICE_ACCOUNTS__
secret_path=__OPENBAO_KV_MOUNT__/__CONNECT_SECRET_PATH__
deny_role=__OPENBAO_DENY_ROLE__
deny_namespace=__DENY_NAMESPACE__
cross_plane_denied=true
secret_values=placeholder_only
production_cutover_allowed=false
EOF
"""
repl = {
    '__OPENBAO_ADDR__': os.environ['OPENBAO_ADDR'],
    '__ADMIN_TOKEN__': os.environ['ADMIN_TOKEN'],
    '__OPENBAO_KV_MOUNT__': os.environ['OPENBAO_KV_MOUNT'],
    '__CONNECT_POLICY_NAME__': os.environ['CONNECT_POLICY_NAME'],
    '__DENY_POLICY_NAME__': os.environ['DENY_POLICY_NAME'],
    '__OPENBAO_CONNECT_ROLE__': os.environ['OPENBAO_CONNECT_ROLE'],
    '__OPENBAO_DENY_ROLE__': os.environ['OPENBAO_DENY_ROLE'],
    '__CONNECT_BOUND_SERVICE_ACCOUNTS__': os.environ['CONNECT_BOUND_SERVICE_ACCOUNTS'],
    '__CONNECT_NAMESPACE__': os.environ['CONNECT_NAMESPACE'],
    '__DENY_SERVICE_ACCOUNT__': os.environ['DENY_SERVICE_ACCOUNT'],
    '__DENY_NAMESPACE__': os.environ['DENY_NAMESPACE'],
    '__TOKEN_TTL__': os.environ['TOKEN_TTL'],
    '__CONNECT_SECRET_PATH__': os.environ['CONNECT_SECRET_PATH'],
    '__CONNECT_JWT__': os.environ['connect_jwt'],
    '__DENY_JWT__': os.environ['deny_jwt'],
}
for key, value in repl.items():
    payload = payload.replace(key, value)
Path(sys.argv[1]).write_text(payload)
PYBUILD

kubectl exec -i -n "$OPENBAO_NAMESPACE" "$OPENBAO_POD" -- sh < "$tmp_payload"
