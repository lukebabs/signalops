#!/usr/bin/env bash
set -euo pipefail

[[ "${EUID}" -eq 0 ]] || {
  printf 'Run this command as root.\n' >&2
  exit 2
}

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=lib/marketops_boundary_env.sh
source "$root_dir/scripts/lib/marketops_boundary_env.sh"
boundary_env=/etc/signalops/marketops-boundary.env
cutover_env=/etc/signalops/marketops-cutover.env
runtime_env="${1:-${SIGNALOPS_PRODUCTION_ENV_FILE:-}}"
[[ -r "$boundary_env" ]] || {
  printf 'Protected MarketOps boundary secret is not readable: %s\n' "$boundary_env" >&2
  exit 3
}
[[ -n "$runtime_env" ]] || {
  printf 'Provide the protected production Compose environment file as argument 1.\n' >&2
  exit 2
}
[[ -r "$runtime_env" ]] || {
  printf 'Production Compose environment file is not readable: %s\n' "$runtime_env" >&2
  exit 3
}

# Compose interpolates every service definition, including the dedicated-store
# definitions not selected by this gateway-only invocation. Parse only the two
# required credentials as literal data; do not execute the secret file.
load_marketops_boundary_env "$boundary_env"

"$root_dir/scripts/render_marketops_cutover_env.sh" "$boundary_env" "$cutover_env"

compose=(docker compose --env-file "$runtime_env" -p signalops -f "$root_dir/compose.yaml" -f "$root_dir/compose.marketops-boundary.yaml" -f "$root_dir/compose.marketops-read-cutover.yaml")
"${compose[@]}" up -d --build --no-deps gateway
gateway_id="$("${compose[@]}" ps -q gateway)"
[[ -n "$gateway_id" ]] || { printf "%s\n" "marketops_read_cutover_gateway_missing" >&2; exit 4; }
gateway_env_names="$(docker inspect --format "{{range .Config.Env}}{{println .}}{{end}}" "$gateway_id" | cut -d= -f1)"
grep -qx "SIGNALOPS_MARKETOPS_DATABASE_URL" <<< "$gateway_env_names" || { printf "%s\n" "marketops_read_cutover_primary_env_missing" >&2; exit 4; }
grep -qx "SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL" <<< "$gateway_env_names" || { printf "%s\n" "marketops_read_cutover_temporal_env_missing" >&2; exit 4; }
docker logs "$gateway_id" 2>&1 | grep -Fq "MarketOps gateway reads are routed to the dedicated data boundary" || { printf "%s\n" "marketops_read_cutover_startup_evidence_missing" >&2; exit 4; }
printf "%s\n" "marketops_read_cutover_gateway_verified"
