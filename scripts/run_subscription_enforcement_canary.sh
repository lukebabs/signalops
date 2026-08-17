#!/usr/bin/env bash
# Root-only, self-restoring production canary for subscription enforcement.
# It changes one gateway environment variable for a single browser/API proof.
# The production dotenv, schedules, data, tenant contracts and user roles stay untouched.
set -Eeuo pipefail

[[ "${EUID}" -eq 0 ]] || { printf '%s\n' 'Run this command through the SignalOps deployment agent.' >&2; exit 2; }

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
operator="${1:-${SUDO_USER:-}}"
[[ "$operator" =~ ^[a-z_][a-z0-9_-]*$ ]] || { printf '%s\n' 'subscription_canary_operator_invalid' >&2; exit 2; }
runtime_env="$repo_dir/.env"
boundary_env=/etc/signalops/marketops-boundary.env
cutover_env=/etc/signalops/marketops-cutover.env
canary_env="$(mktemp /tmp/signalops-subscription-canary.XXXXXX.env)"
source "$repo_dir/scripts/lib/marketops_boundary_env.sh"

base_compose=(docker compose --env-file "$runtime_env" --env-file "$cutover_env" -p signalops -f "$repo_dir/compose.yaml" -f "$repo_dir/compose.marketops-boundary.yaml" -f "$repo_dir/compose.marketops-read-cutover.yaml")
canary_compose=(docker compose --env-file "$runtime_env" --env-file "$cutover_env" --env-file "$canary_env" -p signalops -f "$repo_dir/compose.yaml" -f "$repo_dir/compose.marketops-boundary.yaml" -f "$repo_dir/compose.marketops-read-cutover.yaml")

wait_for_gateway() {
  local attempt
  for attempt in {1..30}; do
    [[ "1000 4 24 27 30 46 101 988 1000docker inspect -f '{{.State.Running}}' signalops-gateway-1 2>/dev/null || true)" == true ]] && curl --fail --silent --show-error --max-time 2 http://127.0.0.1:18000/readyz >/dev/null && return 0
    [[ "$(docker inspect -f '{{.State.Running}}' signalops-gateway-1 2>/dev/null || true)" == true ]] && curl --fail --silent --show-error --max-time 2 http://127.0.0.1:18000/readyz >/dev/null && return 0
  done
  docker logs --tail 80 signalops-gateway-1 >&2 || true
  return 1
}

prepare_gateway_credentials() {
  load_marketops_boundary_env "$boundary_env"
  "$repo_dir/scripts/render_marketops_cutover_env.sh" "$boundary_env" "$cutover_env"
  local subscriber_gateway_password
  subscriber_gateway_password="$(grep -E "^SIGNALOPS_SUBSCRIBER_GATEWAY_PASSWORD=" "$cutover_env" | cut -d= -f2-)"
  [[ "$subscriber_gateway_password" =~ ^[A-Fa-f0-9]{64}$ ]] || {
    printf '%s\n' 'subscription_canary_gateway_secret_invalid' >&2
    return 1
  }
  "${base_compose[@]}" exec -T marketops-postgres psql -v ON_ERROR_STOP=1 -U signalops -d marketops \
    -c "DO \$\$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'signalops_subscriber_gateway_runtime') THEN CREATE ROLE signalops_subscriber_gateway_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS; END IF; END \$\$;" \
    -c "ALTER ROLE signalops_subscriber_gateway_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS PASSWORD '${subscriber_gateway_password}';" \
    -c "GRANT signalops_subscriber_gateway TO signalops_subscriber_gateway_runtime;"
}

restore() {
  local original_status=$?
  local restore_status=0
  set +e
  prepare_gateway_credentials || restore_status=1
  "${base_compose[@]}" up -d --build --no-deps gateway || restore_status=1
  wait_for_gateway || restore_status=1
  docker exec signalops-gateway-1 sh -c 'test "${SIGNALOPS_SUBSCRIPTIONS_ENABLED:-false}" != true' || restore_status=1
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

# Only the flag is in the final override file. Dedicated-boundary credentials
# remain solely in the freshly rendered protected cutover environment.
printf '%s\n' 'SIGNALOPS_SUBSCRIPTIONS_ENABLED=true' > "$canary_env"
prepare_gateway_credentials
"${canary_compose[@]}" up -d --build --no-deps gateway
wait_for_gateway
docker exec signalops-gateway-1 sh -c 'test "${SIGNALOPS_SUBSCRIPTIONS_ENABLED:-false}" = true'
printf '%s\n' 'subscription_enforcement_canary_enabled'

operator_home="$(getent passwd "$operator" | cut -d: -f6)"
[[ -n "$operator_home" && -d "$operator_home" ]] || { printf '%s\n' 'subscription_canary_operator_home_missing' >&2; exit 2; }
runuser -u "$operator" -- env "HOME=$operator_home" "PLAYWRIGHT_BROWSERS_PATH=$operator_home/.cache/ms-playwright" \
  "$repo_dir/scripts/run_subscription_enforcement_canary_ui.sh"
printf '%s\n' 'subscription_enforcement_canary_verified'
