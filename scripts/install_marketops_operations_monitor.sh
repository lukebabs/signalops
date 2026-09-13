#!/usr/bin/env bash
set -euo pipefail

[[ "${EUID}" -eq 0 ]] || { printf 'Run this command as root.\n' >&2; exit 2; }
root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
template="$root_dir/deploy/systemd/signalops-marketops-operations-monitor.service.in"
service=/etc/systemd/system/signalops-marketops-operations-monitor.service
timer=/etc/systemd/system/signalops-marketops-operations-monitor.timer

[[ -r "$template" && -r "$root_dir/deploy/systemd/$(basename "$timer")" ]] || { printf 'Operations monitor unit templates are missing.\n' >&2; exit 3; }
sed "s|@WORKDIR@|$root_dir|g" "$template" > "$service"
install -m 0644 "$root_dir/deploy/systemd/$(basename "$timer")" "$timer"
install -d -m 0750 -o root -g root /var/lib/signalops/marketops-operations
systemctl daemon-reload
if [[ "${1:-}" == "--enable" ]]; then
  systemctl enable --now signalops-marketops-operations-monitor.timer
  printf 'Enabled hourly dedicated MarketOps operations monitor.\n'
else
  systemctl disable --now signalops-marketops-operations-monitor.timer >/dev/null 2>&1 || true
  printf 'Installed disabled dedicated MarketOps operations monitor. Use --enable only after a successful manual run.\n'
fi
