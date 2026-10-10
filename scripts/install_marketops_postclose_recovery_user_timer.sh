#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
template_dir="$ROOT_DIR/deploy/systemd"
config_base="${XDG_CONFIG_HOME:-${HOME}/.config}"
unit_dir="$config_base/systemd/user"
service="signalops-marketops-postclose-recovery"

for command in systemctl sed install; do
  command -v "$command" >/dev/null 2>&1 || { printf 'required command not found: %s\n' "$command" >&2; exit 2; }
done

mkdir -p "$unit_dir" "$ROOT_DIR/runtime/scheduled-jobs" "$ROOT_DIR/runtime/postclose-recovery"
sed "s|@WORKDIR@|$ROOT_DIR|g" "$template_dir/$service.service.in" > "$unit_dir/$service.service"
install -m 0644 "$template_dir/$service.timer" "$unit_dir/$service.timer"
systemctl --user daemon-reload
systemctl --user enable --now "$service.timer"
systemctl --user list-timers "$service.timer" --no-pager
printf 'Installed %s.timer. For unattended execution after logout, enable user lingering.\n' "$service"
