#!/usr/bin/env bash
set -euo pipefail

ENV_FILE="${1:-.env}"
NAMESPACE="${NAMESPACE:-signalops-orchestration}"
DB_NAMESPACE="${DB_NAMESPACE:-signalops-data}"
DB_POD="${DB_POD:-marketops-postgres-production-0}"
OPENBAO_NAMESPACE="${OPENBAO_NAMESPACE:-openbao}"
OPENBAO_POD="${OPENBAO_POD:-openbao-0}"
OPENBAO_ADDR="${OPENBAO_ADDR:-https://openbao.openbao.svc:8200}"
OPENBAO_PATH="${OPENBAO_PATH:-k8s/orchestration/dagster-marketops-readonly-production}"
OPENBAO_ROLE="${OPENBAO_ROLE:-signalops-dagster-shadow}"
OPENBAO_POLICY="${OPENBAO_POLICY:-signalops-dagster-shadow-read}"
DB_ROLE="${DB_ROLE:-signalops_dagster_readonly}"
DB_NAME="${DB_NAME:-marketops}"
DB_SERVICE="${DB_SERVICE:-marketops-postgres-production.signalops-data.svc.cluster.local}"
DB_PORT="${DB_PORT:-5432}"
DB_URL_KEY="${DB_URL_KEY:-MARKETOPS_DATABASE_URL}"
DB_POD_ENV="${DB_POD_ENV:-/vault/secrets/signalops-data-runtime.env}"
BOUND_SERVICE_ACCOUNT="${BOUND_SERVICE_ACCOUNT:-signalops-dagster-dagster-user-deployments-user-deployments}"

fail() { echo "dagster_marketops_readonly_failed: $*" >&2; exit 1; }
[[ -f "$ENV_FILE" ]] || fail "env file not found: $ENV_FILE"
source "scripts/lib/dotenv.sh"
load_dotenv "$ENV_FILE"
ADMIN_TOKEN="${BAO_TOKEN:-${VAULT_TOKEN:-${OPENBAO_TOKEN:-${OPENBAO_ADMIN_TOKEN:-}}}}"
[[ -n "$ADMIN_TOKEN" ]] || fail "OpenBao admin token is required in env"

kubectl get pod "$DB_POD" -n "$DB_NAMESPACE" >/dev/null || fail "database pod not found"
kubectl get pod "$OPENBAO_POD" -n "$OPENBAO_NAMESPACE" >/dev/null || fail "OpenBao pod not found"
kubectl get serviceaccount "$BOUND_SERVICE_ACCOUNT" -n "$NAMESPACE" >/dev/null || fail "Dagster service account not found"

readonly_password="$(od -An -N32 -tx1 /dev/urandom | tr -d ' \n')"
[[ "${#readonly_password}" -eq 64 ]] || fail "failed to generate database password"
database_url="postgres://${DB_ROLE}:${readonly_password}@${DB_SERVICE}:${DB_PORT}/${DB_NAME}?sslmode=disable"

cat <<SQL | kubectl exec -i -n "$DB_NAMESPACE" "$DB_POD" -c postgres -- sh -c '. /vault/secrets/signalops-data-runtime.env; export PGPASSWORD="$SIGNALOPS_MARKETOPS_POSTGRES_PASSWORD"; exec psql -v ON_ERROR_STOP=1 -U signalops -d '"$DB_NAME"''
DO \$\$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '$DB_ROLE') THEN
    CREATE ROLE $DB_ROLE LOGIN PASSWORD '$readonly_password';
  ELSE
    ALTER ROLE $DB_ROLE LOGIN PASSWORD '$readonly_password';
  END IF;
END
\$\$;
ALTER ROLE $DB_ROLE NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT NOREPLICATION NOBYPASSRLS;
GRANT CONNECT ON DATABASE $DB_NAME TO $DB_ROLE;
GRANT USAGE ON SCHEMA public TO $DB_ROLE;
GRANT SELECT ON TABLE marketops_primary_assets, marketops_feature_observations, subscriber_gateway_global_risk_reward_snapshots TO $DB_ROLE;
SQL

policy_payload="$(cat <<HCL
path "signalops/data/$OPENBAO_PATH" {
  capabilities = ["read"]
}
path "signalops/metadata/$OPENBAO_PATH" {
  capabilities = ["read"]
}
HCL
)"
role_jwt="$(kubectl create token "$BOUND_SERVICE_ACCOUNT" -n "$NAMESPACE" --duration=10m)"
printf '%s\n' "$policy_payload" | kubectl exec -i -n "$OPENBAO_NAMESPACE" "$OPENBAO_POD" -- sh -c 'cat >/tmp/dagster-shadow-policy.hcl'
kubectl exec -n "$OPENBAO_NAMESPACE" "$OPENBAO_POD" -- env \
  BAO_ADDR="$OPENBAO_ADDR" BAO_TOKEN="$ADMIN_TOKEN" \
  OPENBAO_POLICY="$OPENBAO_POLICY" OPENBAO_ROLE="$OPENBAO_ROLE" \
  BOUND_SERVICE_ACCOUNT="$BOUND_SERVICE_ACCOUNT" NAMESPACE="$NAMESPACE" \
  OPENBAO_PATH="$OPENBAO_PATH" DATABASE_URL="$database_url" ROLE_JWT="$role_jwt" \
  sh -c '
    set -euo pipefail
    bao policy write "$OPENBAO_POLICY" /tmp/dagster-shadow-policy.hcl >/dev/null
    bao write "auth/kubernetes/role/$OPENBAO_ROLE" \
      bound_service_account_names="$BOUND_SERVICE_ACCOUNT" \
      bound_service_account_namespaces="$NAMESPACE" \
      policies="$OPENBAO_POLICY" ttl="1h" >/dev/null
    bao kv put "signalops/$OPENBAO_PATH" MARKETOPS_DATABASE_URL="$DATABASE_URL" >/dev/null
    token="$(bao write -field=token auth/kubernetes/login role="$OPENBAO_ROLE" jwt="$ROLE_JWT")"
    BAO_TOKEN="$token" bao kv get "signalops/$OPENBAO_PATH" >/dev/null
    echo dagster_marketops_readonly_openbao_verified
  '

echo "dagster_marketops_readonly_provisioned"
echo "openbao_path=signalops/$OPENBAO_PATH"
echo "openbao_role=$OPENBAO_ROLE"
echo "database_role=$DB_ROLE"
echo "database_service=$DB_SERVICE"
echo "database_name=$DB_NAME"
echo "write_privilege=denied_by_role_attributes"
