#!/usr/bin/env bash
set -euo pipefail

ENV_FILE="${1:-.env}"
RUNTIME_FILE="${2:-/tmp/signalops-openbao-data-runtime-production.env}"

fail() {
  echo "signalops_k8s_data_runtime_production_env_failed: $*" >&2
  exit 1
}

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_dir"
[[ -r "$ENV_FILE" ]] || fail "env file is missing or unreadable: ${ENV_FILE}"
# shellcheck source=./lib/dotenv.sh
source "$repo_dir/scripts/lib/dotenv.sh"
load_dotenv "$ENV_FILE"

[[ -n "${OPENBAO_ADMIN_TOKEN:-}" ]] || fail "OPENBAO_ADMIN_TOKEN is required"
[[ -n "${SIGNALOPS_MARKETOPS_POSTGRES_PASSWORD:-}" ]] || fail "SIGNALOPS_MARKETOPS_POSTGRES_PASSWORD is required"
[[ -n "${SIGNALOPS_MARKETOPS_TEMPORAL_PASSWORD:-}" ]] || fail "SIGNALOPS_MARKETOPS_TEMPORAL_PASSWORD is required"

export SIGNALOPS_DATABASE_URL SIGNALOPS_TEMPORAL_DATABASE_URL SIGNALOPS_MARKETOPS_POSTGRES_PASSWORD SIGNALOPS_MARKETOPS_TEMPORAL_PASSWORD OPENBAO_ADMIN_TOKEN
python3 - "$RUNTIME_FILE" <<'PYENV'
import os, stat, sys
from pathlib import Path
from urllib.parse import urlparse

def password_from_url(name: str, default: str) -> str:
    raw = os.environ.get(name, "")
    if not raw:
        return default
    parsed = urlparse(raw)
    return parsed.password or default

target = Path(sys.argv[1])
values = {
    "SIGNALOPS_K8S_PRODUCTION_DATA_RUNTIME_APPROVED": "false",
    "OPENBAO_ADMIN_TOKEN": os.environ["OPENBAO_ADMIN_TOKEN"],
    "SIGNALOPS_POSTGRES_PASSWORD": password_from_url("SIGNALOPS_DATABASE_URL", "signalops"),
    "SIGNALOPS_TIMESCALE_PASSWORD": password_from_url("SIGNALOPS_TEMPORAL_DATABASE_URL", "signalops"),
    "SIGNALOPS_MARKETOPS_POSTGRES_PASSWORD": os.environ["SIGNALOPS_MARKETOPS_POSTGRES_PASSWORD"],
    "SIGNALOPS_MARKETOPS_TEMPORAL_PASSWORD": os.environ["SIGNALOPS_MARKETOPS_TEMPORAL_PASSWORD"],
}
target.write_text("\n".join(f"{k}={v}" for k, v in values.items()) + "\n")
target.chmod(stat.S_IRUSR | stat.S_IWUSR)
PYENV

cat <<EOF
signalops_k8s_data_runtime_production_env_created
path=${RUNTIME_FILE}
mode=0600
approval_marker=SIGNALOPS_K8S_PRODUCTION_DATA_RUNTIME_APPROVED=false
secret_values_printed=false
production_openbao_write_performed=false
EOF
