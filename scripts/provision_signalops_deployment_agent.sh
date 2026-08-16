#!/usr/bin/env bash
set -euo pipefail

[[ "${EUID}" -eq 0 ]] || {
  printf 'Run this command as root.\n' >&2
  exit 2
}

operator="${1:-${SUDO_USER:-}}"
[[ "$operator" =~ ^[a-z_][a-z0-9_-]*$ ]] || {
  printf 'Provide a valid operator user name.\n' >&2
  exit 2
}
id "$operator" >/dev/null

repo_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source_agent="$repo_dir/deploy/deployment-agent/signalops-deploy-agent"
source_renderer="$repo_dir/scripts/render_marketops_cutover_env.sh"
source_browser_smoke="$repo_dir/scripts/run_subscriber_pilot_ui_smoke.sh"
agent_dir=/usr/local/lib/signalops-deployment-agent
agent_bin=/usr/local/sbin/signalops-deploy-agent
sudoers_file=/etc/sudoers.d/signalops-deploy-agent

[[ -x "$source_agent" && -x "$source_renderer" && -x "$source_browser_smoke" ]] || {
  printf 'Deployment-agent source files are missing or not executable.\n' >&2
  exit 3
}

install -d -m 0750 -o root -g root "$agent_dir"
agent_temporary="$(mktemp "$agent_dir/.signalops-deploy-agent.XXXXXX")"
temporary="$(mktemp /etc/sudoers.d/.signalops-deploy-agent.XXXXXX)"
trap 'rm -f "$temporary" "$agent_temporary"' EXIT
sed "s|@REPOSITORY_DIR@|$repo_dir|g" "$source_agent" > "$agent_temporary"
install -m 0750 -o root -g root "$agent_temporary" "$agent_bin"
install -m 0750 -o root -g root "$source_renderer" "$agent_dir/render_marketops_cutover_env.sh"
printf '%s ALL=(root) NOPASSWD: %s\n' "$operator" "$agent_bin" > "$temporary"
visudo -cf "$temporary" >/dev/null
install -m 0440 -o root -g root "$temporary" "$sudoers_file"

printf 'Installed SignalOps deployment-control agent.\n'
printf 'Allowed operator: %s\n' "$operator"
printf 'Available actions: render-cutover-env, scheduler-preflight, scheduler-intraday-run, scheduler-intraday-enable, scheduler-intraday-disable, scheduler-status, operations-monitor-install, operations-monitor-run, operations-monitor-enable, operations-monitor-disable, watch-limits-stage, backup-run, restore-rehearsal-run, marketops-recovery-resume, marketops-global-market-state-migration, subscriber-global-eod-history-materialize, subscriber-qualified-warm-cohort-reconcile, marketops-fmp-annual-run, fmp-annual-entitlement-preflight, subscriber-pilot-ui-smoke, marketops-web-deploy\n'
