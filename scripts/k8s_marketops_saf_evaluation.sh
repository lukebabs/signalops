#!/usr/bin/env bash
set -euo pipefail

session_date="${MARKETOPS_SESSION_DATE:-$(TZ=America/New_York date +%F)}"
lookback_days="${MARKETOPS_OUTCOME_LOOKBACK_DAYS:-45}"
: "${SIGNALOPS_MARKETOPS_DATABASE_URL:?dedicated MarketOps database required}"
mapfile -t symbols < <(psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -Atc "SELECT ticker FROM marketops_universal_assets WHERE tenant_id='tenant-local' AND is_active ORDER BY rank NULLS LAST,ticker")
start_date="$(date -d "$session_date - ${lookback_days} days" +%F)"
for ((i=0; i<${#symbols[@]}; i+=10)); do
  batch=("${symbols[@]:i:10}")
  csv=$(IFS=,; echo "${batch[*]}")
  signalops-marketops-intelligence-cohort-runner --tenant-id tenant-local --symbols "$csv" --max-symbols "${#batch[@]}" --session-start "$start_date" --session-end "$session_date" --stages outcome_materialization --continue-on-error=true --dry-run=false --acknowledge-writes --run-id "k8s-saf-outcomes-${session_date}-$(printf '%03d' "$i")"
done
signalops-marketops-signal-assurance-worker --tenant-id tenant-local --as-of "$session_date" --mode RESEARCH --run-id saf-research-materializations-v1
k8s-marketops-global-dashboard-projection "$session_date"
echo "marketops_k8s_saf_evaluation_completed session=$session_date symbols=${#symbols[@]} lookback_days=$lookback_days"
