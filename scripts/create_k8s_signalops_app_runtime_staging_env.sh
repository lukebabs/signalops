#!/usr/bin/env bash
set -euo pipefail

ENV_FILE="${1:-.env}"
RUNTIME_FILE="${2:-/tmp/signalops-openbao-app-runtime-staging.env}"
DATA_NAMESPACE="${SIGNALOPS_K8S_DATA_NAMESPACE:-signalops-data}"
PRIMARY_SECRET="${SIGNALOPS_K8S_MARKETOPS_POSTGRES_SECRET:-marketops-postgres-staging-auth}"
TEMPORAL_SECRET="${SIGNALOPS_K8S_MARKETOPS_TEMPORAL_SECRET:-marketops-timescaledb-staging-auth}"
PRIMARY_STATEFULSET="${SIGNALOPS_K8S_MARKETOPS_POSTGRES_SERVICE:-marketops-postgres-staging}"
TEMPORAL_STATEFULSET="${SIGNALOPS_K8S_MARKETOPS_TEMPORAL_SERVICE:-marketops-timescaledb-staging}"
PRIMARY_DATABASE="${SIGNALOPS_K8S_MARKETOPS_POSTGRES_DB:-marketops}"
TEMPORAL_DATABASE="${SIGNALOPS_K8S_MARKETOPS_TEMPORAL_DB:-marketops_temporal}"
PRIMARY_USER="${SIGNALOPS_K8S_MARKETOPS_POSTGRES_USER:-signalops}"
TEMPORAL_USER="${SIGNALOPS_K8S_MARKETOPS_TEMPORAL_USER:-signalops}"

fail() {
  echo "signalops_k8s_app_runtime_staging_env_failed: $*" >&2
  exit 1
}

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"
[[ -r "$ENV_FILE" ]] || fail "env file is missing or unreadable: ${ENV_FILE}"

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=./lib/dotenv.sh
source "$repo_dir/scripts/lib/dotenv.sh"
load_dotenv "$ENV_FILE"

required_env=(
  OPENBAO_ADMIN_TOKEN
  SIGNALOPS_NOTIFICATION_ENCRYPTION_KEY
  STRIPE_RESTRICTED_API_KEY
  STRIPE_WEBHOOK_SECRET
  SYNCRATIC_API_BASE_URL
  SYNCRATIC_AUTH_MODE
  SYNCRATIC_TOKEN_URL
  SYNCRATIC_TOKEN_GRANT
  SYNCRATIC_CLIENT_ID
  SYNCRATIC_CLIENT_SECRET
  SYNCRATIC_USERNAME
  SYNCRATIC_PASSWORD
)
for key in "${required_env[@]}"; do
  [[ -n "${!key:-}" ]] || fail "missing required ${key} in ${ENV_FILE} or environment"
done

read_secret_value() {
  local secret_name="$1"
  kubectl get secret "$secret_name" -n "$DATA_NAMESPACE" -o jsonpath='{.data.POSTGRES_PASSWORD}' 2>/dev/null | base64 -d
}

primary_password="$(read_secret_value "$PRIMARY_SECRET")"
temporal_password="$(read_secret_value "$TEMPORAL_SECRET")"
[[ -n "$primary_password" ]] || fail "could not read ${DATA_NAMESPACE}/${PRIMARY_SECRET}"
[[ -n "$temporal_password" ]] || fail "could not read ${DATA_NAMESPACE}/${TEMPORAL_SECRET}"

primary_url="postgres://${PRIMARY_USER}:${primary_password}@${PRIMARY_STATEFULSET}.${DATA_NAMESPACE}.svc.cluster.local:5432/${PRIMARY_DATABASE}?sslmode=disable"
temporal_url="postgres://${TEMPORAL_USER}:${temporal_password}@${TEMPORAL_STATEFULSET}.${DATA_NAMESPACE}.svc.cluster.local:5432/${TEMPORAL_DATABASE}?sslmode=disable"

umask 077
{
  printf 'SIGNALOPS_K8S_STAGING_RUNTIME_NON_PRODUCTION_APPROVED=true\n'
  printf 'OPENBAO_ADMIN_TOKEN=%s\n' "$OPENBAO_ADMIN_TOKEN"
  printf 'SIGNALOPS_DATABASE_URL=%s\n' "$primary_url"
  printf 'SIGNALOPS_TEMPORAL_DATABASE_URL=%s\n' "$temporal_url"
  printf 'SIGNALOPS_MARKETOPS_DATABASE_URL=%s\n' "$primary_url"
  printf 'SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL=%s\n' "$temporal_url"
  printf 'SIGNALOPS_SUBSCRIBER_GATEWAY_DATABASE_URL=%s\n' "$primary_url"
  printf 'SIGNALOPS_NOTIFICATION_ENCRYPTION_KEY=%s\n' "$SIGNALOPS_NOTIFICATION_ENCRYPTION_KEY"
  printf 'STRIPE_RESTRICTED_API_KEY=%s\n' "$STRIPE_RESTRICTED_API_KEY"
  printf 'STRIPE_WEBHOOK_SECRET=%s\n' "$STRIPE_WEBHOOK_SECRET"
  printf 'SYNCRATIC_API_BASE_URL=%s\n' "$SYNCRATIC_API_BASE_URL"
  printf 'SYNCRATIC_AUTH_MODE=%s\n' "$SYNCRATIC_AUTH_MODE"
  printf 'SYNCRATIC_TOKEN_URL=%s\n' "$SYNCRATIC_TOKEN_URL"
  printf 'SYNCRATIC_TOKEN_GRANT=%s\n' "$SYNCRATIC_TOKEN_GRANT"
  printf 'SYNCRATIC_CLIENT_ID=%s\n' "$SYNCRATIC_CLIENT_ID"
  printf 'SYNCRATIC_CLIENT_SECRET=%s\n' "$SYNCRATIC_CLIENT_SECRET"
  printf 'SYNCRATIC_USERNAME=%s\n' "$SYNCRATIC_USERNAME"
  printf 'SYNCRATIC_PASSWORD=%s\n' "$SYNCRATIC_PASSWORD"
  printf 'SYNCRATIC_TOKEN_AUDIENCE=%s\n' "${SYNCRATIC_TOKEN_AUDIENCE:-}"
} >"$RUNTIME_FILE"

cat <<EOF
signalops_k8s_app_runtime_staging_env_created
path=${RUNTIME_FILE}
mode=0600
database=dedicated-marketops-staging
secret_values_printed=false
production_cutover_allowed=false
EOF
