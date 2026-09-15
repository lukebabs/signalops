#!/usr/bin/env bash
set -euo pipefail

OPENBAO_NAMESPACE="${OPENBAO_NAMESPACE:-openbao}"
OPENBAO_POD="${OPENBAO_POD:-openbao-0}"
OPENBAO_ADDR="${OPENBAO_ADDR:-https://openbao.openbao.svc:8200}"
OPENBAO_KV_MOUNT="${OPENBAO_KV_MOUNT:-signalops}"
OPENBAO_APP_ROLE="${OPENBAO_APP_ROLE:-signalops-app}"
OPENBAO_DENY_ROLE="${OPENBAO_DENY_ROLE:-signalops-marketops}"
APP_NAMESPACE="${APP_NAMESPACE:-signalops-app}"
DENY_NAMESPACE="${DENY_NAMESPACE:-signalops-marketops}"
APP_BOUND_SERVICE_ACCOUNTS="${APP_BOUND_SERVICE_ACCOUNTS:-signalops-gateway,signalops-web,signalops-app-secret-reader}"
DENY_BOUND_SERVICE_ACCOUNTS="${DENY_BOUND_SERVICE_ACCOUNTS:-signalops-marketops-secret-reader}"
APP_SECRET_PATH="${APP_SECRET_PATH:-k8s/app/signalops-gateway-runtime-staging}"
APP_POLICY_NAME="${APP_POLICY_NAME:-signalops-app-staging-read}"
DENY_POLICY_NAME="${DENY_POLICY_NAME:-signalops-marketops-staging-deny-proof}"
TOKEN_TTL="${TOKEN_TTL:-1h}"
ADMIN_TOKEN="${BAO_TOKEN:-${VAULT_TOKEN:-${OPENBAO_TOKEN:-}}}"

fail() {
  echo "openbao_signalops_app_staging_failed: $*" >&2
  exit 1
}

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"
[[ -n "$ADMIN_TOKEN" ]] || fail "set BAO_TOKEN, VAULT_TOKEN, or OPENBAO_TOKEN in the environment; do not pass it as an argument"

kubectl get pod "$OPENBAO_POD" -n "$OPENBAO_NAMESPACE" >/dev/null || fail "OpenBao pod not found: ${OPENBAO_NAMESPACE}/${OPENBAO_POD}"
kubectl get namespace "$APP_NAMESPACE" >/dev/null || fail "namespace missing: ${APP_NAMESPACE}"
kubectl get namespace "$DENY_NAMESPACE" >/dev/null || fail "namespace missing: ${DENY_NAMESPACE}"
IFS=',' read -r -a app_sas <<< "$APP_BOUND_SERVICE_ACCOUNTS"
for sa in "${app_sas[@]}"; do
  kubectl get serviceaccount "$sa" -n "$APP_NAMESPACE" >/dev/null || fail "app service account missing: ${APP_NAMESPACE}/${sa}"
done
IFS=',' read -r -a deny_sas <<< "$DENY_BOUND_SERVICE_ACCOUNTS"
for sa in "${deny_sas[@]}"; do
  kubectl get serviceaccount "$sa" -n "$DENY_NAMESPACE" >/dev/null || fail "deny-proof service account missing: ${DENY_NAMESPACE}/${sa}"
done

app_jwt="$(kubectl create token "${app_sas[0]}" -n "$APP_NAMESPACE" --duration=10m)"
deny_jwt="$(kubectl create token "${deny_sas[0]}" -n "$DENY_NAMESPACE" --duration=10m)"
[[ -n "$app_jwt" ]] || fail "failed to create app service account token"
[[ -n "$deny_jwt" ]] || fail "failed to create deny-proof service account token"

tmp_payload="$(mktemp)"
trap 'rm -f "$tmp_payload"' EXIT
export OPENBAO_ADDR ADMIN_TOKEN OPENBAO_KV_MOUNT APP_POLICY_NAME DENY_POLICY_NAME
export OPENBAO_APP_ROLE OPENBAO_DENY_ROLE APP_BOUND_SERVICE_ACCOUNTS DENY_BOUND_SERVICE_ACCOUNTS
export APP_NAMESPACE DENY_NAMESPACE TOKEN_TTL APP_SECRET_PATH app_jwt deny_jwt

python3 - "$tmp_payload" <<'PYBUILD'
from pathlib import Path
import os
import sys
payload = '''set -euo pipefail
export BAO_ADDR="__OPENBAO_ADDR__"
export BAO_TOKEN="__ADMIN_TOKEN__"
export BAO_CACERT="${BAO_CACERT:-/openbao/ca/ca.crt}"

bao status >/dev/null

if ! bao secrets list -format=json | grep -q '"__OPENBAO_KV_MOUNT__/"'; then
  bao secrets enable -path="__OPENBAO_KV_MOUNT__" -version=2 kv >/dev/null
fi

if ! bao auth list -format=json | grep -q '"kubernetes/"'; then
  bao auth enable kubernetes >/dev/null
fi

k8s_ca="$(cat /var/run/secrets/kubernetes.io/serviceaccount/ca.crt)"
k8s_jwt="$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)"
bao write auth/kubernetes/config \
  token_reviewer_jwt="$k8s_jwt" \
  kubernetes_host="https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT}" \
  kubernetes_ca_cert="$k8s_ca" >/dev/null

cat >/tmp/signalops-app-policy.hcl <<'EOP'
path "__OPENBAO_KV_MOUNT__/data/k8s/app/*" {
  capabilities = ["read"]
}
path "__OPENBAO_KV_MOUNT__/metadata/k8s/app/*" {
  capabilities = ["list", "read"]
}
EOP
bao policy write "__APP_POLICY_NAME__" /tmp/signalops-app-policy.hcl >/dev/null

cat >/tmp/signalops-deny-proof-policy.hcl <<'EOP'
path "__OPENBAO_KV_MOUNT__/data/k8s/marketops/*" {
  capabilities = ["read"]
}
EOP
bao policy write "__DENY_POLICY_NAME__" /tmp/signalops-deny-proof-policy.hcl >/dev/null

bao write "auth/kubernetes/role/__OPENBAO_APP_ROLE__" \
  bound_service_account_names="__APP_BOUND_SERVICE_ACCOUNTS__" \
  bound_service_account_namespaces="__APP_NAMESPACE__" \
  policies="__APP_POLICY_NAME__" \
  ttl="__TOKEN_TTL__" >/dev/null

bao write "auth/kubernetes/role/__OPENBAO_DENY_ROLE__" \
  bound_service_account_names="__DENY_BOUND_SERVICE_ACCOUNTS__" \
  bound_service_account_namespaces="__DENY_NAMESPACE__" \
  policies="__DENY_POLICY_NAME__" \
  ttl="__TOKEN_TTL__" >/dev/null

bao kv put "__OPENBAO_KV_MOUNT__/__APP_SECRET_PATH__" \
  SIGNALOPS_DATABASE_URL="postgres://staging_user:staging_password@postgres.staging.invalid:5432/signalops?sslmode=require" \
  SIGNALOPS_TEMPORAL_DATABASE_URL="postgres://staging_user:staging_password@timescaledb.staging.invalid:5432/signalops_temporal?sslmode=require" \
  SIGNALOPS_MARKETOPS_DATABASE_URL="postgres://staging_user:staging_password@marketops-postgres.staging.invalid:5432/marketops?sslmode=require" \
  SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL="postgres://staging_user:staging_password@marketops-timescaledb.staging.invalid:5432/marketops_temporal?sslmode=require" \
  SIGNALOPS_SUBSCRIBER_GATEWAY_DATABASE_URL="postgres://staging_user:staging_password@marketops-postgres.staging.invalid:5432/marketops?sslmode=require" \
  SIGNALOPS_NOTIFICATION_ENCRYPTION_KEY="staging-placeholder-not-production" \
  STRIPE_RESTRICTED_API_KEY="sk_test_staging_placeholder_not_live" \
  STRIPE_WEBHOOK_SECRET="whsec_staging_placeholder_not_live" \
  SYNCRATIC_API_BASE_URL="https://staging.invalid" \
  SYNCRATIC_CLIENT_SECRET="staging-placeholder-not-production" >/dev/null

unset BAO_TOKEN
app_token="$(bao write -field=token auth/kubernetes/login role="__OPENBAO_APP_ROLE__" jwt="__APP_JWT__")"
deny_token="$(bao write -field=token auth/kubernetes/login role="__OPENBAO_DENY_ROLE__" jwt="__DENY_JWT__")"
[[ -n "$app_token" ]] || { echo "app auth token not issued" >&2; exit 1; }
[[ -n "$deny_token" ]] || { echo "deny auth token not issued" >&2; exit 1; }

BAO_TOKEN="$app_token" bao kv get "__OPENBAO_KV_MOUNT__/__APP_SECRET_PATH__" >/dev/null

if BAO_TOKEN="$deny_token" bao kv get "__OPENBAO_KV_MOUNT__/__APP_SECRET_PATH__" >/tmp/deny_out 2>/tmp/deny_err; then
  echo "cross-plane denial failed: deny role read app secret path" >&2
  exit 1
fi

cat <<EOF
openbao_signalops_app_staging_verified
mount=__OPENBAO_KV_MOUNT__
app_role=__OPENBAO_APP_ROLE__
app_namespace=__APP_NAMESPACE__
app_service_accounts=__APP_BOUND_SERVICE_ACCOUNTS__
secret_path=__OPENBAO_KV_MOUNT__/__APP_SECRET_PATH__
deny_role=__OPENBAO_DENY_ROLE__
deny_namespace=__DENY_NAMESPACE__
cross_plane_denied=true
secret_values=placeholder_only
production_cutover_allowed=false
EOF
'''
repl = {
    '__OPENBAO_ADDR__': os.environ['OPENBAO_ADDR'],
    '__ADMIN_TOKEN__': os.environ['ADMIN_TOKEN'],
    '__OPENBAO_KV_MOUNT__': os.environ['OPENBAO_KV_MOUNT'],
    '__APP_POLICY_NAME__': os.environ['APP_POLICY_NAME'],
    '__DENY_POLICY_NAME__': os.environ['DENY_POLICY_NAME'],
    '__OPENBAO_APP_ROLE__': os.environ['OPENBAO_APP_ROLE'],
    '__OPENBAO_DENY_ROLE__': os.environ['OPENBAO_DENY_ROLE'],
    '__APP_BOUND_SERVICE_ACCOUNTS__': os.environ['APP_BOUND_SERVICE_ACCOUNTS'],
    '__DENY_BOUND_SERVICE_ACCOUNTS__': os.environ['DENY_BOUND_SERVICE_ACCOUNTS'],
    '__APP_NAMESPACE__': os.environ['APP_NAMESPACE'],
    '__DENY_NAMESPACE__': os.environ['DENY_NAMESPACE'],
    '__TOKEN_TTL__': os.environ['TOKEN_TTL'],
    '__APP_SECRET_PATH__': os.environ['APP_SECRET_PATH'],
    '__APP_JWT__': os.environ['app_jwt'],
    '__DENY_JWT__': os.environ['deny_jwt'],
}
for key, value in repl.items():
    payload = payload.replace(key, value)
Path(sys.argv[1]).write_text(payload)
PYBUILD

kubectl exec -i -n "$OPENBAO_NAMESPACE" "$OPENBAO_POD" -- sh < "$tmp_payload"
