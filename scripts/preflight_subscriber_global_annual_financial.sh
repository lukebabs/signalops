#!/usr/bin/env bash
set -euo pipefail

# One-symbol, non-writing entitlement check for the centrally governed annual
# FMP capture. This is intentionally not a scheduler or a provider-write path.

[[ "${EUID}" -eq 0 ]] || {
  printf '%s\n' 'Run this command through the SignalOps deployment agent.' >&2
  exit 2
}

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=lib/marketops_boundary_env.sh
source "$root_dir/scripts/lib/marketops_boundary_env.sh"
runtime_env="${1:-${SIGNALOPS_PRODUCTION_ENV_FILE:-}}"
boundary_env=/etc/signalops/marketops-boundary.env

[[ -n "$runtime_env" && -r "$runtime_env" ]] || {
  printf '%s\n' 'Provide a readable production Compose environment file as argument 1.' >&2
  exit 2
}
[[ -r "$boundary_env" ]] || {
  printf '%s\n' 'Protected MarketOps boundary secret is not readable.' >&2
  exit 3
}

load_marketops_boundary_env "$boundary_env"
# The root-only boundary secret is alphanumeric by contract, so this literal
# Docker-network URL needs no shell evaluation or URL escaping. It is exported
# only to the one-shot Compose worker and is never printed.
export SIGNALOPS_MARKETOPS_DATABASE_URL="postgres://signalops:${SIGNALOPS_MARKETOPS_POSTGRES_PASSWORD}@marketops-postgres:5432/marketops?sslmode=disable"
compose=(docker compose --env-file "$runtime_env" -p signalops
  -f "$root_dir/compose.yaml"
  -f "$root_dir/compose.marketops-boundary.yaml"
  -f "$root_dir/compose.marketops-writer-cutover.yaml")

"${compose[@]}" --profile subscriber-global-evidence run --rm --no-deps --build \
  subscriber-global-annual-financial-refresh \
  --dry-run --max-assets 1 --request-interval 250ms \
  --correlation-id fmp-annual-entitlement-preflight
