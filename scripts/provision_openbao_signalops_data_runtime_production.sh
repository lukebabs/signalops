#!/usr/bin/env bash
set -euo pipefail

OPENBAO_NAMESPACE="${OPENBAO_NAMESPACE:-openbao}"
OPENBAO_POD="${OPENBAO_POD:-openbao-0}"
OPENBAO_ADDR="${OPENBAO_ADDR:-https://openbao.openbao.svc:8200}"
OPENBAO_KV_MOUNT="${OPENBAO_KV_MOUNT:-signalops}"
OPENBAO_DATA_ROLE="${OPENBAO_DATA_ROLE:-signalops-data}"
OPENBAO_DENY_ROLE="${OPENBAO_DENY_ROLE:-signalops-app}"
DATA_NAMESPACE="${DATA_NAMESPACE:-signalops-data}"
DENY_NAMESPACE="${DENY_NAMESPACE:-signalops-app}"
DATA_SERVICE_ACCOUNT="${DATA_SERVICE_ACCOUNT:-signalops-data-secret-reader}"
DENY_SERVICE_ACCOUNT="${DENY_SERVICE_ACCOUNT:-signalops-app-secret-reader}"
DATA_SECRET_PATH="${DATA_SECRET_PATH:-k8s/data/signalops-databases-runtime-production}"
RUNTIME_ENV_FILE="${1:-${SIGNALOPS_K8S_PRODUCTION_DATA_RUNTIME_ENV_FILE:-/etc/signalops/openbao-signalops-data-production-runtime.env}}"

fail() {
  echo "openbao_signalops_data_runtime_production_failed: $*" >&2
  exit 1
}

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"
[[ -r "$RUNTIME_ENV_FILE" ]] || fail "runtime env file is missing or unreadable: ${RUNTIME_ENV_FILE}"
set -a
# shellcheck disable=SC1090
source "$RUNTIME_ENV_FILE"
set +a

ADMIN_TOKEN="${BAO_TOKEN:-${VAULT_TOKEN:-${OPENBAO_TOKEN:-${OPENBAO_ADMIN_TOKEN:-}}}}"
[[ -n "$ADMIN_TOKEN" ]] || fail "runtime env file must provide OpenBao admin token"
[[ "${SIGNALOPS_K8S_PRODUCTION_DATA_RUNTIME_APPROVED:-}" == "true" ]] || fail "set SIGNALOPS_K8S_PRODUCTION_DATA_RUNTIME_APPROVED=true only after named production data-secret migration approval"
for key in SIGNALOPS_POSTGRES_PASSWORD SIGNALOPS_TIMESCALE_PASSWORD SIGNALOPS_MARKETOPS_POSTGRES_PASSWORD SIGNALOPS_MARKETOPS_TEMPORAL_PASSWORD; do
  [[ -n "${!key:-}" ]] || fail "missing required runtime key: ${key}"
done

kubectl get pod "$OPENBAO_POD" -n "$OPENBAO_NAMESPACE" >/dev/null || fail "OpenBao pod not found: ${OPENBAO_NAMESPACE}/${OPENBAO_POD}"
kubectl get serviceaccount "$DATA_SERVICE_ACCOUNT" -n "$DATA_NAMESPACE" >/dev/null || fail "data service account missing: ${DATA_NAMESPACE}/${DATA_SERVICE_ACCOUNT}"
kubectl get serviceaccount "$DENY_SERVICE_ACCOUNT" -n "$DENY_NAMESPACE" >/dev/null || fail "deny-proof service account missing: ${DENY_NAMESPACE}/${DENY_SERVICE_ACCOUNT}"

data_jwt="$(kubectl create token "$DATA_SERVICE_ACCOUNT" -n "$DATA_NAMESPACE" --duration=10m)"
deny_jwt="$(kubectl create token "$DENY_SERVICE_ACCOUNT" -n "$DENY_NAMESPACE" --duration=10m)"
[[ -n "$data_jwt" && -n "$deny_jwt" ]] || fail "failed to create bounded Kubernetes service-account tokens"

tmp_payload="$(mktemp)"
trap 'rm -f "$tmp_payload"' EXIT
export OPENBAO_ADDR ADMIN_TOKEN OPENBAO_KV_MOUNT DATA_SECRET_PATH OPENBAO_DATA_ROLE OPENBAO_DENY_ROLE data_jwt deny_jwt
python3 - "$tmp_payload" <<'PYBUILD'
from pathlib import Path
import os, shlex, sys
keys = ["SIGNALOPS_POSTGRES_PASSWORD", "SIGNALOPS_TIMESCALE_PASSWORD", "SIGNALOPS_MARKETOPS_POSTGRES_PASSWORD", "SIGNALOPS_MARKETOPS_TEMPORAL_PASSWORD"]
lines = [
    "set -euo pipefail",
    f"export BAO_ADDR={shlex.quote(os.environ['OPENBAO_ADDR'])}",
    f"export BAO_TOKEN={shlex.quote(os.environ['ADMIN_TOKEN'])}",
    'export BAO_CACERT="${BAO_CACERT:-/openbao/ca/ca.crt}"',
    "bao status >/dev/null",
    "bao kv put " + shlex.quote(os.environ["OPENBAO_KV_MOUNT"] + "/" + os.environ["DATA_SECRET_PATH"]) + " \\",
]
for index, key in enumerate(keys):
    lines.append(f"  {key}={shlex.quote(os.environ[key])}{' \\\\' if index < len(keys)-1 else ''}")
lines.extend([
    "unset BAO_TOKEN",
    f"data_token=\"$(bao write -field=token auth/kubernetes/login role={shlex.quote(os.environ['OPENBAO_DATA_ROLE'])} jwt={shlex.quote(os.environ['data_jwt'])})\"",
    f"deny_token=\"$(bao write -field=token auth/kubernetes/login role={shlex.quote(os.environ['OPENBAO_DENY_ROLE'])} jwt={shlex.quote(os.environ['deny_jwt'])})\"",
    '[[ -n "$data_token" && -n "$deny_token" ]]',
    f"BAO_TOKEN=\"$data_token\" bao kv get {shlex.quote(os.environ['OPENBAO_KV_MOUNT'] + '/' + os.environ['DATA_SECRET_PATH'])} >/dev/null",
    f"if BAO_TOKEN=\"$deny_token\" bao kv get {shlex.quote(os.environ['OPENBAO_KV_MOUNT'] + '/' + os.environ['DATA_SECRET_PATH'])} >/tmp/signalops-openbao-data-prod-deny-out 2>/tmp/signalops-openbao-data-prod-deny-err; then echo \"cross-plane denial failed\" >&2; exit 1; fi",
])
Path(sys.argv[1]).write_text("\n".join(lines)+"\n")
PYBUILD
kubectl exec -i -n "$OPENBAO_NAMESPACE" "$OPENBAO_POD" -- sh < "$tmp_payload" >/dev/null

cat <<EOF
openbao_signalops_data_runtime_production_verified
mount=${OPENBAO_KV_MOUNT}
data_role=${OPENBAO_DATA_ROLE}
data_namespace=${DATA_NAMESPACE}
data_service_account=${DATA_SERVICE_ACCOUNT}
secret_path=${OPENBAO_KV_MOUNT}/${DATA_SECRET_PATH}
deny_role=${OPENBAO_DENY_ROLE}
deny_namespace=${DENY_NAMESPACE}
cross_plane_denied=true
secret_values=production_data_runtime_supplied
production_traffic_moved=false
EOF
