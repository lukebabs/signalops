#!/usr/bin/env bash
set -euo pipefail

[[ "${EUID}" -eq 0 ]] || {
  printf 'Run this command as root.\n' >&2
  exit 2
}

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
runtime_env="${1:-}"
run_as_user="${2:-${SUDO_USER:-adminalien}}"
template="$root_dir/deploy/systemd/signalops-marketops-boundary-schedule@.service.in"
unit=/etc/systemd/system/signalops-marketops-boundary-schedule@.service

[[ -n "$runtime_env" && -r "$runtime_env" ]] || {
  printf 'Provide a readable protected production Compose environment file as argument 1.\n' >&2
  exit 2
}
[[ "$run_as_user" =~ ^[a-z_][a-z0-9_-]*$ ]] || {
  printf 'Invalid service user: %s\n' "$run_as_user" >&2
  exit 2
}
id "$run_as_user" >/dev/null
[[ -r /etc/signalops/marketops-cutover.env ]] || {
  printf 'Dedicated MarketOps cutover environment is not readable. Render it first.\n' >&2
  exit 3
}
[[ -r "$template" ]] || {
  printf 'Missing unit template: %s\n' "$template" >&2
  exit 3
}

temporary="$(mktemp /etc/systemd/system/.signalops-marketops-boundary.XXXXXX)"
trap 'rm -f "$temporary"' EXIT
sed \
  -e "s|@WORKDIR@|$root_dir|g" \
  -e "s|@RUNTIME_ENV@|$runtime_env|g" \
  -e "s|@RUN_AS_USER@|$run_as_user|g" \
  "$template" > "$temporary"
install -m 0644 -o root -g root "$temporary" "$unit"
systemctl daemon-reload

printf 'Installed disabled dedicated scheduler dispatcher: %s\n' "$unit"
printf 'No timer was enabled. Use a separately approved one-job smoke command, for example:\n'
printf '  sudo systemctl start signalops-marketops-boundary-schedule@marketops-intraday.service\n'
