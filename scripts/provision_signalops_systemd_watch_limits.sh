#!/usr/bin/env bash
set -euo pipefail

[[ "${EUID}" -eq 0 ]] || { printf 'Run this command as root.\n' >&2; exit 2; }

install -d -m 0755 /etc/systemd/system.conf.d
cat > /etc/systemd/system.conf.d/70-signalops-watch-limits.conf <<'EOF'
[Manager]
DefaultLimitNOFILE=1048576
EOF
cat > /etc/sysctl.d/70-signalops-inotify.conf <<'EOF'
fs.inotify.max_user_instances = 2048
fs.inotify.max_user_watches = 524288
fs.inotify.max_queued_events = 32768
EOF
sysctl -p /etc/sysctl.d/70-signalops-inotify.conf
systemctl daemon-reexec
printf 'Installed persistent SignalOps inotify/watch limits and re-executed systemd. Verify the scheduler warnings are absent.\n'
