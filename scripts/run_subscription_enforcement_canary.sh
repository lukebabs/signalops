#!/usr/bin/env bash
# Root-only, self-restoring production canary for subscription enforcement.
# It changes one gateway environment variable for the duration of a single
# browser/API proof. It never writes the production dotenv or touches jobs.
set -Eeuo pipefail

[[ "${EUID}" -eq 0 ]] || { printf '%s\n' 'Run this command through the SignalOps deployment agent.' >&2; exit 2; }

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
operator="${1:-${SUDO_USER:-}}"
[[ "$operator" =~ ^[a-z_][a-z0-9_-]*$ ]] || { printf '%s\n' 'subscription_canary_operator_invalid' >&2; exit 2; }
runtime_env="$repo_dir/.env"
cutover_env=/etc/signalops/marketops-cutover.env
canary_env="$(mktemp /tmp/signalops-subscription-canary.XXXXXX.env)"
base_compose=(docker compose --env-file "$cutover_env" --env-file "$runtime_env" -p signalops -f "$repo_dir/compose.yaml" -f "$repo_dir/compose.marketops-boundary.yaml" -f "$repo_dir/compose.marketops-read-cutover.yaml")

restore() {
  local original_status=$?
  local restore_status=0
  set +e
  "${base_compose[@]}" up -d --build --no-deps gateway
  [[ $? -eq 0 ]] || restore_status=1
  docker exec signalops-gateway-1 sh -c 'test "${SIGNALOPS_SUBSCRIPTIONS_ENABLED:-false}" != true'
  [[ $? -eq 0 ]] || restore_status=1
  rm -f "$canary_env"
  if [[ "$restore_status" -eq 0 ]]; then
    printf '%s\n' 'subscription_enforcement_canary_restored'
  else
    printf '%s\n' 'subscription_enforcement_canary_restore_failed' >&2
  fi
  if [[ "$original_status" -ne 0 || "$restore_status" -ne 0 ]]; then
    exit 1
  fi
}
trap restore EXIT

cp "$runtime_env" "$canary_env"
sed -i '/^SIGNALOPS_SUBSCRIPTIONS_ENABLED=/d' "$canary_env"
printf '%s\n' 'SIGNALOPS_SUBSCRIPTIONS_ENABLED=true' >> "$canary_env"

# The isolated canary env is last, therefore it is the only true value.
canary_compose=(docker compose --env-file "$cutover_env" --env-file "$canary_env" -p signalops -f "$repo_dir/compose.yaml" -f "$repo_dir/compose.marketops-boundary.yaml" -f "$repo_dir/compose.marketops-read-cutover.yaml")
"${canary_compose[@]}" up -d --build --no-deps gateway
docker exec signalops-gateway-1 sh -c 'test "${SIGNALOPS_SUBSCRIPTIONS_ENABLED:-false}" = true'
printf '%s\n' 'subscription_enforcement_canary_enabled'

operator_home="$(getent passwd "$operator" | cut -d: -f6)"
[[ -n "$operator_home" && -d "$operator_home" ]] || { printf '%s\n' 'subscription_canary_operator_home_missing' >&2; exit 2; }
runuser -u "$operator" -- env "HOME=$operator_home" "PLAYWRIGHT_BROWSERS_PATH=$operator_home/.cache/ms-playwright" \
  "$repo_dir/scripts/run_subscription_enforcement_canary_ui.sh"
printf '%s\n' 'subscription_enforcement_canary_verified'
