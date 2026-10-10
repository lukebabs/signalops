#!/usr/bin/env bash
set -euo pipefail

ENV_FILE="${1:-.env}"
RUNTIME_FILE="${2:-/tmp/signalops-openbao-app-runtime-production.env}"
K8S_DATA_NAMESPACE="${K8S_DATA_NAMESPACE:-signalops-data}"
SIGNALOPS_PRODUCTION_POSTGRES_SERVICE="${SIGNALOPS_PRODUCTION_POSTGRES_SERVICE:-signalops-postgres-production}"
SIGNALOPS_PRODUCTION_TIMESCALE_SERVICE="${SIGNALOPS_PRODUCTION_TIMESCALE_SERVICE:-signalops-timescaledb-production}"
MARKETOPS_PRODUCTION_POSTGRES_SERVICE="${MARKETOPS_PRODUCTION_POSTGRES_SERVICE:-marketops-postgres-production}"
MARKETOPS_PRODUCTION_TIMESCALE_SERVICE="${MARKETOPS_PRODUCTION_TIMESCALE_SERVICE:-marketops-timescaledb-production}"

fail() {
  echo "signalops_k8s_app_runtime_production_env_failed: $*" >&2
  exit 1
}

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_dir"

[[ -r "$ENV_FILE" ]] || fail "env file is missing or unreadable: ${ENV_FILE}"

# shellcheck source=./lib/dotenv.sh
source "$repo_dir/scripts/lib/dotenv.sh"
load_dotenv "$ENV_FILE"

required_source_keys=(
  OPENBAO_ADMIN_TOKEN
  SIGNALOPS_MARKETOPS_POSTGRES_PASSWORD
  SIGNALOPS_MARKETOPS_TEMPORAL_PASSWORD
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

for key in "${required_source_keys[@]}"; do
  [[ -n "${!key:-}" ]] || fail "missing required source key: ${key}"
done

signalops_database_url="${SIGNALOPS_DATABASE_URL:-postgres://signalops:signalops@postgres:5432/signalops?sslmode=disable}"
signalops_temporal_database_url="${SIGNALOPS_TEMPORAL_DATABASE_URL:-postgres://signalops:signalops@timescaledb:5432/signalops?sslmode=disable}"
subscriber_gateway_database_url="${SIGNALOPS_SUBSCRIBER_GATEWAY_DATABASE_URL:-postgres://signalops_subscriber_gateway_runtime:${SIGNALOPS_SUBSCRIBER_GATEWAY_PASSWORD:-}@postgres:5432/signalops?sslmode=disable}"

export K8S_DATA_NAMESPACE SIGNALOPS_PRODUCTION_POSTGRES_SERVICE SIGNALOPS_PRODUCTION_TIMESCALE_SERVICE MARKETOPS_PRODUCTION_POSTGRES_SERVICE MARKETOPS_PRODUCTION_TIMESCALE_SERVICE
export signalops_database_url signalops_temporal_database_url subscriber_gateway_database_url
python3 - "$RUNTIME_FILE" <<'PYENV'
import os
import stat
import sys
from pathlib import Path
from urllib.parse import urlparse, urlunparse

target = Path(sys.argv[1])
data_namespace = os.environ["K8S_DATA_NAMESPACE"]

def service_host(service: str) -> str:
    return f"{service}.{data_namespace}.svc.cluster.local"

def require_url(name: str, raw: str) -> str:
    parsed = urlparse(raw)
    if parsed.scheme not in {"postgres", "postgresql"} or not parsed.hostname:
        raise SystemExit(f"invalid PostgreSQL URL for {name}")
    return raw

def rewrite_host(raw: str, host: str, database: str | None = None) -> str:
    parsed = urlparse(raw)
    if parsed.scheme not in {"postgres", "postgresql"} or not parsed.username:
        raise SystemExit("invalid PostgreSQL URL source")
    userinfo = parsed.username
    if parsed.password is not None:
        userinfo += ":" + parsed.password
    path = parsed.path or ""
    if database:
        path = "/" + database
    return urlunparse((parsed.scheme, f"{userinfo}@{host}:5432", path, parsed.params, parsed.query or "sslmode=disable", parsed.fragment))

shared = require_url("SIGNALOPS_DATABASE_URL", os.environ["signalops_database_url"])
temporal = require_url("SIGNALOPS_TEMPORAL_DATABASE_URL", os.environ["signalops_temporal_database_url"])
subscriber = require_url("SIGNALOPS_SUBSCRIBER_GATEWAY_DATABASE_URL", os.environ["subscriber_gateway_database_url"])

marketops_primary = f"postgres://signalops:{os.environ['SIGNALOPS_MARKETOPS_POSTGRES_PASSWORD']}@{service_host(os.environ['MARKETOPS_PRODUCTION_POSTGRES_SERVICE'])}:5432/marketops?sslmode=disable"
marketops_temporal = f"postgres://signalops:{os.environ['SIGNALOPS_MARKETOPS_TEMPORAL_PASSWORD']}@{service_host(os.environ['MARKETOPS_PRODUCTION_TIMESCALE_SERVICE'])}:5432/marketops_temporal?sslmode=disable"

# Compose-local hostnames are not routable from Kubernetes pods. Rewrite the
# known Compose database endpoints to intended production Kubernetes service DNS.
# Docker-hosted production databases remain an explicit fallback through a
# caller-supplied runtime env file, not the generated default.
rewrites = {
    "SIGNALOPS_DATABASE_URL": (shared, service_host(os.environ["SIGNALOPS_PRODUCTION_POSTGRES_SERVICE"]), "signalops"),
    "SIGNALOPS_TEMPORAL_DATABASE_URL": (temporal, service_host(os.environ["SIGNALOPS_PRODUCTION_TIMESCALE_SERVICE"]), "signalops"),
    "SIGNALOPS_SUBSCRIBER_GATEWAY_DATABASE_URL": (subscriber, service_host(os.environ["SIGNALOPS_PRODUCTION_POSTGRES_SERVICE"]), "signalops"),
}
rewritten = {}
for key, (raw, host_target, dbname) in rewrites.items():
    parsed = urlparse(raw)
    host = (parsed.hostname or "").lower()
    if host in {"postgres", "timescaledb", "localhost", "127.0.0.1"}:
        rewritten[key] = rewrite_host(raw, host_target, dbname)
    else:
        rewritten[key] = raw

values = {
    "SIGNALOPS_K8S_PRODUCTION_RUNTIME_APPROVED": "false",
    "OPENBAO_ADMIN_TOKEN": os.environ["OPENBAO_ADMIN_TOKEN"],
    "SIGNALOPS_ENV": "k8s-production",
    "SIGNALOPS_DATABASE_URL": rewritten["SIGNALOPS_DATABASE_URL"],
    "SIGNALOPS_TEMPORAL_DATABASE_URL": rewritten["SIGNALOPS_TEMPORAL_DATABASE_URL"],
    "SIGNALOPS_MARKETOPS_DATABASE_URL": marketops_primary,
    "SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL": marketops_temporal,
    "SIGNALOPS_SUBSCRIBER_GATEWAY_DATABASE_URL": rewritten["SIGNALOPS_SUBSCRIBER_GATEWAY_DATABASE_URL"],
    "SIGNALOPS_NOTIFICATION_ENCRYPTION_KEY": os.environ["SIGNALOPS_NOTIFICATION_ENCRYPTION_KEY"],
    "STRIPE_RESTRICTED_API_KEY": os.environ["STRIPE_RESTRICTED_API_KEY"],
    "STRIPE_WEBHOOK_SECRET": os.environ["STRIPE_WEBHOOK_SECRET"],
    "STRIPE_API_KEY": os.environ.get("STRIPE_API_KEY") or os.environ["STRIPE_RESTRICTED_API_KEY"],
    "SIGNALOPS_STRIPE_CHECKOUT_SUCCESS_URL": os.environ.get("SIGNALOPS_STRIPE_CHECKOUT_SUCCESS_URL", "https://signalops.syncratic.io/marketops/subscription/return?session_id={CHECKOUT_SESSION_ID}"),
    "SIGNALOPS_STRIPE_CHECKOUT_CANCEL_URL": os.environ.get("SIGNALOPS_STRIPE_CHECKOUT_CANCEL_URL", "https://signalops.syncratic.io/marketops/pricing"),
    "SIGNALOPS_STRIPE_PORTAL_RETURN_URL": os.environ.get("SIGNALOPS_STRIPE_PORTAL_RETURN_URL", "https://signalops.syncratic.io/marketops/pricing"),
    "SYNCRATIC_API_BASE_URL": os.environ["SYNCRATIC_API_BASE_URL"],
    "SYNCRATIC_AUTH_MODE": os.environ["SYNCRATIC_AUTH_MODE"],
    "SYNCRATIC_TOKEN_URL": os.environ["SYNCRATIC_TOKEN_URL"],
    "SYNCRATIC_TOKEN_GRANT": os.environ["SYNCRATIC_TOKEN_GRANT"],
    "SYNCRATIC_CLIENT_ID": os.environ["SYNCRATIC_CLIENT_ID"],
    "SYNCRATIC_CLIENT_SECRET": os.environ["SYNCRATIC_CLIENT_SECRET"],
    "SYNCRATIC_USERNAME": os.environ["SYNCRATIC_USERNAME"],
    "SYNCRATIC_PASSWORD": os.environ["SYNCRATIC_PASSWORD"],
    "SYNCRATIC_TOKEN_AUDIENCE": os.environ.get("SYNCRATIC_TOKEN_AUDIENCE", ""),
}
target.write_text("\n".join(f"{k}={v}" for k, v in values.items()) + "\n")
target.chmod(stat.S_IRUSR | stat.S_IWUSR)
PYENV

cat <<EOF
signalops_k8s_app_runtime_production_env_created
path=${RUNTIME_FILE}
mode=0600
database_host=kubernetes_service_dns
data_namespace=${K8S_DATA_NAMESPACE}
approval_marker=SIGNALOPS_K8S_PRODUCTION_RUNTIME_APPROVED=false
secret_values_printed=false
production_openbao_write_performed=false
EOF
