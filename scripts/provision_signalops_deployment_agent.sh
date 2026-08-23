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
source_safe_deploy="$repo_dir/scripts/deploy_signalops_public_production.sh"
source_bridge="$repo_dir/scripts/signalops_deployment_agent_bridge.py"
bridge_template="$repo_dir/deploy/systemd/signalops-deployment-agent-bridge.service.in"
standalone_unit_templates=(
  signalops-storage-monitor.service.in
  signalops-retention-governance.service.in
  signalops-marketops-retention-governance.service.in
)
standalone_timers=(
  signalops-storage-monitor.timer
  signalops-retention-governance.timer
)
agent_dir=/usr/local/lib/signalops-deployment-agent
agent_bin=/usr/local/sbin/signalops-deploy-agent
sudoers_file=/etc/sudoers.d/signalops-deploy-agent

[[ -x "$source_agent" && -x "$source_renderer" && -x "$source_browser_smoke" && -x "$source_safe_deploy" && -x "$source_bridge" && -r "$bridge_template" ]] || {
  printf 'Deployment-agent source files are missing or not executable.\n' >&2
  exit 3
}
for template in "${standalone_unit_templates[@]}"; do
  [[ -r "$repo_dir/deploy/systemd/$template" ]] || {
    printf 'Standalone service template is missing: %s\n' "$template" >&2
    exit 3
  }
done
for timer in "${standalone_timers[@]}"; do
  [[ -r "$repo_dir/deploy/systemd/$timer" ]] || {
    printf 'Standalone timer template is missing: %s\n' "$timer" >&2
    exit 3
  }
done

install -d -m 0750 -o root -g root "$agent_dir"
agent_temporary="$(mktemp "$agent_dir/.signalops-deploy-agent.XXXXXX")"
temporary="$(mktemp /etc/sudoers.d/.signalops-deploy-agent.XXXXXX)"
trap 'rm -f "$temporary" "$agent_temporary"' EXIT
sed "s|@REPOSITORY_DIR@|$repo_dir|g" "$source_agent" > "$agent_temporary"
install -m 0750 -o root -g root "$agent_temporary" "$agent_bin"
install -m 0750 -o root -g root "$source_renderer" "$agent_dir/render_marketops_cutover_env.sh"
install -m 0750 -o root -g root "$source_bridge" "$agent_dir/signalops_deployment_agent_bridge.py"
bridge_rendered="$(mktemp /etc/systemd/system/.signalops-deployment-agent-bridge.XXXXXX)"
sed "s|@AGENT_DIR@|$agent_dir|g" "$bridge_template" > "$bridge_rendered"
install -m 0644 -o root -g root "$bridge_rendered" /etc/systemd/system/signalops-deployment-agent-bridge.service
rm -f "$bridge_rendered"
for template in "${standalone_unit_templates[@]}"; do
  rendered="$(mktemp /etc/systemd/system/.signalops-standalone-unit.XXXXXX)"
  sed "s|@WORKDIR@|$repo_dir|g" "$repo_dir/deploy/systemd/$template" > "$rendered"
  install -m 0644 -o root -g root "$rendered" "/etc/systemd/system/${template%.in}"
  rm -f "$rendered"
done
for timer in "${standalone_timers[@]}"; do
  install -m 0644 -o root -g root "$repo_dir/deploy/systemd/$timer" "/etc/systemd/system/$timer"
done
systemctl daemon-reload
systemctl enable --now signalops-deployment-agent-bridge.service
printf '%s ALL=(root) NOPASSWD: %s\n' "$operator" "$agent_bin" > "$temporary"
visudo -cf "$temporary" >/dev/null
install -m 0440 -o root -g root "$temporary" "$sudoers_file"

printf 'Installed SignalOps deployment-control agent.\n'
printf 'Installed standalone disabled units: signalops-storage-monitor.service, signalops-retention-governance.service, signalops-marketops-retention-governance.service.\n'
printf 'Enabled deployment-agent Unix socket bridge: signalops-deployment-agent-bridge.service.\n'
printf 'Allowed operator: %s\n' "$operator"
printf 'Available actions: render-cutover-env, scheduler-preflight, scheduler-intraday-run, scheduler-intraday-enable, scheduler-intraday-disable, scheduler-fmp-annual-enable, scheduler-fmp-annual-disable, scheduler-run-now:<job_id>, scheduler-status, operations-monitor-install, operations-monitor-run, operations-monitor-enable, operations-monitor-disable, watch-limits-stage, backup-run, restore-rehearsal-run, marketops-recovery-resume, marketops-postclose-systemd-reconcile, marketops-global-market-state-migration, subscriber-subscription-commerce-migration, subscriber-global-intraday-shadow-migration, subscriber-global-intraday-shadow-dry-run, subscriber-global-intraday-shadow-schedule-once, subscriber-global-eod-history-materialize, subscriber-qualified-warm-cohort-reconcile, marketops-fmp-annual-run, fmp-annual-entitlement-preflight, subscriber-pilot-ui-smoke, marketops-saf-projection-refresh, subscription-enforcement-canary, signalops-production-deploy, signalops-production-web-deploy, signalops-production-gateway-deploy, marketops-web-deploy, marketops-gateway-deploy, marketops-writer-cutover, scheduler-run-now:marketops-retention-governance\n'
