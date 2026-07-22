#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
template_dir="$ROOT_DIR/deploy/systemd"
config_base="${XDG_CONFIG_HOME:-${HOME}/.config}"
unit_dir="$config_base/systemd/user"

for command in docker systemctl sed install; do
  command -v "$command" >/dev/null || { printf 'required command not found: %s\n' "$command" >&2; exit 2; }
done

docker compose --profile massive-pull --profile marketops-daily build \
  massive-puller marketops-options-coverage-runner marketops-options-feature-materializer algorithm-runner marketops-intelligence-cohort-runner

mkdir -p "$unit_dir"
sed "s|@WORKDIR@|$ROOT_DIR|g" "$template_dir/signalops-marketops-daily.service.in" > "$unit_dir/signalops-marketops-daily.service"
install -m 0644 "$template_dir/signalops-marketops-daily.timer" "$unit_dir/signalops-marketops-daily.timer"

systemctl --user daemon-reload
systemctl --user enable --now signalops-marketops-daily.timer
systemctl --user list-timers signalops-marketops-daily.timer --no-pager

printf '%s\n' \
  "Installed user timer in $unit_dir." \
  'For unattended execution after logout, an administrator must run:' \
  "  loginctl enable-linger $(id -un)"
