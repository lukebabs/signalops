#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
template_dir="$ROOT_DIR/deploy/systemd"
config_base="${XDG_CONFIG_HOME:-${HOME}/.config}"
unit_dir="$config_base/systemd/user"
for command in docker systemctl sed install; do command -v "$command" >/dev/null || { printf 'required command not found: %s\n' "$command" >&2; exit 2; }; done
docker compose --profile massive-pull --profile marketops-daily build massive-puller marketops-options-coverage-runner marketops-options-feature-materializer algorithm-runner marketops-intelligence-cohort-runner marketops-valuation-runner marketops-tactical-valuation-runner marketops-eroc-runner marketops-eeom-runner marketops-intraday-monitor marketops-asset-backfill-worker storage-monitor retention-governor cyberops-daily-feature-materializer administration-notification-recorder
mkdir -p "$unit_dir" "$ROOT_DIR/runtime/scheduled-jobs"
for service in signalops-marketops-daily signalops-marketops-intraday signalops-marketops-fmp-continuation signalops-marketops-task-retry signalops-storage-monitor signalops-retention-governance; do sed "s|@WORKDIR@|$ROOT_DIR|g" "$template_dir/$service.service.in" > "$unit_dir/$service.service"; install -m 0644 "$template_dir/$service.timer" "$unit_dir/$service.timer"; done
systemctl --user daemon-reload
systemctl --user enable --now signalops-marketops-daily.timer signalops-marketops-intraday.timer signalops-marketops-fmp-continuation.timer signalops-marketops-task-retry.timer signalops-storage-monitor.timer signalops-retention-governance.timer
systemctl --user list-timers 'signalops-*' --no-pager
printf '%s\n' "Installed user timers in $unit_dir." 'For unattended execution after logout, an administrator must run:' "  loginctl enable-linger $(id -un)"
