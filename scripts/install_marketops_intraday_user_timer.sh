#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
unit_dir="${XDG_CONFIG_HOME:-$HOME/.config}/systemd/user"
mkdir -p "$unit_dir"
docker compose --profile marketops-intraday build marketops-intraday-monitor
sed "s|@WORKDIR@|$ROOT_DIR|g" "$ROOT_DIR/deploy/systemd/signalops-marketops-intraday.service.in" > "$unit_dir/signalops-marketops-intraday.service"
install -m 0644 "$ROOT_DIR/deploy/systemd/signalops-marketops-intraday.timer" "$unit_dir/signalops-marketops-intraday.timer"
systemctl --user daemon-reload
systemctl --user enable --now signalops-marketops-intraday.timer
systemctl --user list-timers signalops-marketops-intraday.timer --no-pager
