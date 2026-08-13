#!/usr/bin/env bash
set -euo pipefail

[[ "${EUID}" -eq 0 ]] || {
  printf 'Run this provisioning command as root.\n' >&2
  exit 2
}

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
handoff_path="${1:-/tmp/signalops-marketops-boundary-secrets.env}"
secret_dir=/etc/signalops
secret_env="$secret_dir/marketops-boundary.env"

[[ -f "$handoff_path" ]] || {
  printf 'MarketOps boundary secret handoff not found: %s\n' "$handoff_path" >&2
  exit 3
}

for command in install rm; do
  command -v "$command" >/dev/null 2>&1 || {
    printf 'required command not found: %s\n' "$command" >&2
    exit 3
  }
done

install -d -m 0750 -o root -g root "$secret_dir"
install -m 0600 -o root -g root "$handoff_path" "$secret_env"
rm -f "$handoff_path"

set -a
# shellcheck disable=SC1090
. "$secret_env"
set +a
MARKETOPS_BOUNDARY_ACKNOWLEDGE_WRITES=true \
  "$root_dir/scripts/bootstrap_marketops_database_boundary.sh"
