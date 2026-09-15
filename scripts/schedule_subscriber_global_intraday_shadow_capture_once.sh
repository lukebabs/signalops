#!/usr/bin/env bash
set -euo pipefail

[[ "${EUID}" -eq 0 ]] || { echo "Run this command as root." >&2; exit 2; }
root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
unit=signalops-subscriber-global-intraday-shadow-capture-once
calendar='2026-08-17 09:50:00 America/New_York'

systemd-analyze calendar "$calendar" >/dev/null
systemctl stop "$unit.timer" "$unit.service" 2>/dev/null || true
systemctl reset-failed "$unit.service" 2>/dev/null || true
systemd-run --unit="$unit" --on-calendar="$calendar" --property=Type=oneshot \
  --property=TimeoutStartSec=20min --property=NoNewPrivileges=yes \
  "$root_dir/scripts/run_subscriber_global_intraday_shadow_capture_once.sh"
systemctl show "$unit.timer" --property=NextElapseUSecRealtime --property=Unit --no-pager
echo "subscriber_global_intraday_shadow_capture_one_shot_scheduled"
