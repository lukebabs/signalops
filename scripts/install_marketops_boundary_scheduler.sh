#!/usr/bin/env bash
set -euo pipefail

[[ "${EUID}" -eq 0 ]] || {
  printf 'Run this command as root.\n' >&2
  exit 2
}

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
runtime_env="${1:-}"
run_as_user="${2:-${SUDO_USER:-adminalien}}"
enable_sri="${3:-}"
template="$root_dir/deploy/systemd/signalops-marketops-boundary-schedule@.service.in"
unit=/etc/systemd/system/signalops-marketops-boundary-schedule@.service
sri_timers=(
  signalops-marketops-boundary-sri-refresh.timer
  signalops-marketops-boundary-sri-holdings-refresh.timer
)

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
if [[ -n "$enable_sri" && "$enable_sri" != "--enable-sri" ]]; then
  printf 'Optional argument 3 must be --enable-sri.\n' >&2
  exit 2
fi
for timer in "${sri_timers[@]}"; do
  [[ -r "$root_dir/deploy/systemd/$timer" ]] || {
    printf 'Missing SRI timer template: %s\n' "$timer" >&2
    exit 3
  }
done

temporary="$(mktemp /etc/systemd/system/.signalops-marketops-boundary.XXXXXX)"
trap 'rm -f "$temporary"' EXIT
sed \
  -e "s|@WORKDIR@|$root_dir|g" \
  -e "s|@RUNTIME_ENV@|$runtime_env|g" \
  -e "s|@RUN_AS_USER@|$run_as_user|g" \
  "$template" > "$temporary"
install -m 0644 -o root -g root "$temporary" "$unit"
for timer in "${sri_timers[@]}"; do
  install -m 0644 -o root -g root "$root_dir/deploy/systemd/$timer" "/etc/systemd/system/$timer"
done
systemctl daemon-reload

printf 'Installed dedicated scheduler dispatcher: %s\n' "$unit"
if [[ "$enable_sri" == "--enable-sri" ]]; then
  systemctl enable --now "${sri_timers[@]}"
  printf 'Enabled controlled SRI refresh timers: weekdays 20:07 and 20:20 America/New_York.\n'
  exit 0
fi
printf 'No timer was enabled. Use --enable-sri only with recorded approval.\n'
