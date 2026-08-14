#!/usr/bin/env bash
set -euo pipefail

[[ "${EUID}" -eq 0 ]] || {
  printf 'Run this command as root.\n' >&2
  exit 2
}

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
boundary_env=/etc/signalops/marketops-boundary.env
cutover_env=/etc/signalops/marketops-cutover.env
[[ -r "$boundary_env" ]] || {
  printf 'Protected MarketOps boundary secret is not readable: %s\n' "$boundary_env" >&2
  exit 3
}

# Compose interpolates every service definition, including the dedicated-store
# definitions not selected by this gateway-only invocation. Exporting these
# values only within this root process satisfies interpolation without exposing
# them in a shell history, repository file, or container environment.
set -a
# shellcheck disable=SC1090
. "$boundary_env"
set +a

"$root_dir/scripts/render_marketops_cutover_env.sh" "$boundary_env" "$cutover_env"

exec docker compose -p signalops \
  -f "$root_dir/compose.yaml" \
  -f "$root_dir/compose.marketops-boundary.yaml" \
  -f "$root_dir/compose.marketops-read-cutover.yaml" \
  up -d --build --no-deps gateway
