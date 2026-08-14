#!/usr/bin/env bash
set -euo pipefail

[[ "${EUID}" -eq 0 ]] || {
  printf 'Run this command as root.\n' >&2
  exit 2
}

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
boundary_env=/etc/signalops/marketops-boundary.env
cutover_env=/etc/signalops/marketops-cutover.env
runtime_env="${1:-${SIGNALOPS_PRODUCTION_ENV_FILE:-}}"
[[ -r "$boundary_env" ]] || {
  printf 'Protected MarketOps boundary secret is not readable: %s\n' "$boundary_env" >&2
  exit 3
}
[[ -n "$runtime_env" && -r "$runtime_env" ]] || {
  printf 'Provide a readable protected production Compose environment file as argument 1.\n' >&2
  exit 2
}

for timer in \
  signalops-marketops-daily.timer \
  signalops-marketops-sri-refresh.timer \
  signalops-marketops-sri-holdings-refresh.timer \
  signalops-marketops-intraday.timer \
  signalops-marketops-fmp-continuation.timer \
  signalops-marketops-task-retry.timer \
  signalops-marketops-postclose-recovery.timer; do
  if systemctl is-active --quiet "$timer"; then
    printf 'Refusing writer cutover while scheduled MarketOps timer is active: %s\n' "$timer" >&2
    exit 4
  fi
done

set -a
# shellcheck disable=SC1090
. "$boundary_env"
set +a
"$root_dir/scripts/render_marketops_cutover_env.sh" "$boundary_env" "$cutover_env"

base=(docker compose --env-file "$runtime_env" -p signalops -f "$root_dir/compose.yaml" -f "$root_dir/compose.marketops-boundary.yaml")
continuous_writers=(normalizer signal-persister marketops-signal-assurance-registrar)
restoration_required=false
restore_shared() {
  if "$restoration_required"; then
    printf 'Writer cutover did not complete; restoring the continuous writers with shared-store configuration.\n' >&2
    "${base[@]}" up -d --build --no-deps "${continuous_writers[@]}" || true
  fi
}
trap 'status=$?; restore_shared; exit "$status"' EXIT

"$root_dir/scripts/preflight_marketops_writer_cutover.sh"
"${base[@]}" stop "${continuous_writers[@]}"
restoration_required=true
"$root_dir/scripts/preflight_marketops_writer_cutover.sh"

"${base[@]}" \
  -f "$root_dir/compose.marketops-read-cutover.yaml" \
  -f "$root_dir/compose.marketops-writer-cutover.yaml" \
  up -d --build --no-deps "${continuous_writers[@]}"
restoration_required=false
trap - EXIT
printf 'Continuous MarketOps writers are now routed to the dedicated boundary. Scheduled timers remain inactive.\n'
