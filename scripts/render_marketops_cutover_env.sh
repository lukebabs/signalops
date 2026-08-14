#!/usr/bin/env bash
set -euo pipefail

[[ "${EUID}" -eq 0 ]] || {
  printf 'Run this command as root.\n' >&2
  exit 2
}

source_env="${1:-/etc/signalops/marketops-boundary.env}"
output_env="${2:-/etc/signalops/marketops-cutover.env}"
[[ -r "$source_env" ]] || {
  printf 'MarketOps boundary secret file is not readable: %s\n' "$source_env" >&2
  exit 3
}

set -a
# shellcheck disable=SC1090
. "$source_env"
set +a
for name in SIGNALOPS_MARKETOPS_POSTGRES_PASSWORD SIGNALOPS_MARKETOPS_TEMPORAL_PASSWORD; do
  [[ "${!name:-}" =~ ^[A-Za-z0-9]{32,}$ ]] || {
    printf '%s must be a URL-safe 32-character minimum secret\n' "$name" >&2
    exit 2
  }
done

output_dir="$(dirname "$output_env")"
install -d -m 0750 -o root -g root "$output_dir"
temp_env="$(mktemp "$output_dir/.marketops-cutover.XXXXXX")"
trap 'rm -f "$temp_env"' EXIT
printf '%s\n' \
  "SIGNALOPS_MARKETOPS_DATABASE_URL=postgres://signalops:${SIGNALOPS_MARKETOPS_POSTGRES_PASSWORD}@marketops-postgres:5432/marketops?sslmode=disable" \
  "SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL=postgres://signalops:${SIGNALOPS_MARKETOPS_TEMPORAL_PASSWORD}@marketops-timescaledb:5432/marketops_temporal?sslmode=disable" \
  > "$temp_env"
install -m 0600 -o root -g root "$temp_env" "$output_env"
printf 'Rendered protected MarketOps cutover environment: %s\n' "$output_env"
