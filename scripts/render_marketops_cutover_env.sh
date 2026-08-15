#!/usr/bin/env bash
set -euo pipefail

[[ "${EUID}" -eq 0 ]] || {
  printf 'Run this command as root.\n' >&2
  exit 2
}

source_env="${1:-/etc/signalops/marketops-boundary.env}"
output_env="${2:-/etc/signalops/marketops-cutover.env}"
# shellcheck source=lib/marketops_boundary_env.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/lib/marketops_boundary_env.sh"
[[ -r "$source_env" ]] || {
  printf 'MarketOps boundary secret file is not readable: %s\n' "$source_env" >&2
  exit 3
}

load_marketops_boundary_env "$source_env"

output_dir="$(dirname "$output_env")"
install -d -m 0750 -o root -g root "$output_dir"
temp_env="$(mktemp "$output_dir/.marketops-cutover.XXXXXX")"
trap 'rm -f "$temp_env"' EXIT
# Compose also resolves the dedicated database service definitions, which require
# standalone variables in addition to the protected connection URLs.
printf '%s\n' \
  "SIGNALOPS_MARKETOPS_POSTGRES_PASSWORD=${SIGNALOPS_MARKETOPS_POSTGRES_PASSWORD}" \
  "SIGNALOPS_MARKETOPS_TEMPORAL_PASSWORD=${SIGNALOPS_MARKETOPS_TEMPORAL_PASSWORD}" \
  "SIGNALOPS_MARKETOPS_DATABASE_URL=postgres://signalops:${SIGNALOPS_MARKETOPS_POSTGRES_PASSWORD}@marketops-postgres:5432/marketops?sslmode=disable" \
  "SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL=postgres://signalops:${SIGNALOPS_MARKETOPS_TEMPORAL_PASSWORD}@marketops-timescaledb:5432/marketops_temporal?sslmode=disable" \
  > "$temp_env"
install -m 0600 -o root -g root "$temp_env" "$output_env"
printf 'Rendered protected MarketOps cutover environment: %s\n' "$output_env"
