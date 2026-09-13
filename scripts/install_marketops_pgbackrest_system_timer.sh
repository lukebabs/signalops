#!/usr/bin/env bash
set -euo pipefail

[[ "${EUID}" -eq 0 ]] || { echo "Run this installer as root." >&2; exit 2; }
root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
template_dir="$root_dir/deploy/systemd"
for command in install sed systemctl; do command -v "$command" >/dev/null || { echo "required command not found: $command" >&2; exit 2; }; done
sed "s|@WORKDIR@|$root_dir|g" "$template_dir/signalops-marketops-pgbackrest.service.in" > /etc/systemd/system/signalops-marketops-pgbackrest.service
install -m 0644 "$template_dir/signalops-marketops-pgbackrest.timer" /etc/systemd/system/signalops-marketops-pgbackrest.timer
systemctl daemon-reload
systemctl enable --now signalops-pgbackrest-credentials.timer
systemctl enable --now signalops-marketops-pgbackrest.timer
echo "Enabled dedicated MarketOps pgBackRest recovery-point timer at 02:45 UTC daily."
