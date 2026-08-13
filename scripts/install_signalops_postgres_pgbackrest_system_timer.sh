#!/usr/bin/env bash
set -euo pipefail

[[ "${EUID}" -eq 0 ]] || {
  printf 'Run this installer as root. It installs a system timer that owns protected backup credentials.\n' >&2
  exit 2
}

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
template_dir="$root_dir/deploy/systemd"

for command in docker systemctl sed install; do
  command -v "$command" >/dev/null 2>&1 || {
    printf 'required command not found: %s\n' "$command" >&2
    exit 2
  }
done

sed "s|@WORKDIR@|$root_dir|g" \
  "$template_dir/signalops-pgbackrest-credentials.service.in" \
  > /etc/systemd/system/signalops-pgbackrest-credentials.service
sed "s|@WORKDIR@|$root_dir|g" \
  "$template_dir/signalops-postgres-pgbackrest.service.in" \
  > /etc/systemd/system/signalops-postgres-pgbackrest.service
install -m 0644 "$template_dir/signalops-pgbackrest-credentials.timer" \
  /etc/systemd/system/signalops-pgbackrest-credentials.timer
install -m 0644 "$template_dir/signalops-postgres-pgbackrest.timer" \
  /etc/systemd/system/signalops-postgres-pgbackrest.timer
systemctl daemon-reload
systemctl enable signalops-pgbackrest-credentials.timer signalops-postgres-pgbackrest.timer
systemctl start signalops-pgbackrest-credentials.service
systemctl list-timers 'signalops-*pgbackrest*' --no-pager
