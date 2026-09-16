#!/usr/bin/env bash
set -euo pipefail
session_date="${MARKETOPS_SESSION_DATE:-$(date -u -d yesterday +%F)}"
start_date="${MARKETOPS_POSTCLOSE_WINDOW_START:-$session_date}"
ack="${MARKETOPS_POSTCLOSE_ACKNOWLEDGE_WRITES:-false}"
[[ "$ack" == true ]] || { echo "postclose writer requires MARKETOPS_POSTCLOSE_ACKNOWLEDGE_WRITES=true" >&2; exit 2; }
: "${SIGNALOPS_MARKETOPS_DATABASE_URL:?dedicated MarketOps database required}"
mapfile -t symbols < <(psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -Atc "SELECT ticker FROM marketops_universal_assets WHERE tenant_id='tenant-local' AND universe_group='all_active' AND is_active ORDER BY rank NULLS LAST,ticker")
((${#symbols[@]} > 0)) || { echo "no active MarketOps symbols" >&2; exit 3; }
for ((i=0;i<${#symbols[@]};i+=10)); do
  batch=("${symbols[@]:i:10}"); csv=$(IFS=,; echo "${batch[*]}");
  signalops-marketops-intelligence-cohort-runner --tenant-id tenant-local --symbols "$csv" --max-symbols "${#batch[@]}" --session-start "$start_date" --session-end "$session_date" --stages preflight,state_materialization,hypothesis_evaluation,opportunity_build,outcome_materialization,hypothesis_proposal_generation --continue-on-error=true --dry-run=false --acknowledge-writes --run-id "k8s-postclose-${session_date}-$(printf '%03d' "$i")"
done
signalops-marketops-valuation-runner --tenant-id tenant-local --universe-group all_active --session-date "$session_date" --dry-run=false --fmp-max-requests 300 --refresh-financials
signalops-marketops-tactical-valuation-runner --tenant-id tenant-local --universe-group all_active --session-date "$session_date"
signalops-marketops-eroc-runner --tenant-id tenant-local --universe-group all_active --session-date "$session_date" --dry-run=false
signalops-marketops-eeom-runner --tenant-id tenant-local --session-date "$session_date" --dry-run=false
signalops-marketops-syncratic-intelligence-runner --tenant-id tenant-local --session-date "$session_date"
echo "marketops_k8s_postclose_writer_completed session=$session_date symbols=${#symbols[@]}"
