#!/usr/bin/env bash
set -euo pipefail
# This job runs after the New York market close; anchor the default to the current
# New York trading date so post-close does not lag by one session. Explicit
# MARKETOPS_SESSION_DATE remains supported for bounded reconciliation runs.
session_date="${MARKETOPS_SESSION_DATE:-$(TZ=America/New_York date +%F)}"
start_date="${MARKETOPS_POSTCLOSE_WINDOW_START:-$session_date}"
ack="${MARKETOPS_POSTCLOSE_ACKNOWLEDGE_WRITES:-false}"
[[ "$ack" == true ]] || { echo "postclose writer requires MARKETOPS_POSTCLOSE_ACKNOWLEDGE_WRITES=true" >&2; exit 2; }
: "${SIGNALOPS_MARKETOPS_DATABASE_URL:?dedicated MarketOps database required}"
mapfile -t symbols < <(psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -Atc "SELECT ticker FROM marketops_universal_assets WHERE tenant_id='tenant-local' AND is_active ORDER BY rank NULLS LAST,ticker")
((${#symbols[@]} > 0)) || { echo "no active MarketOps symbols" >&2; exit 3; }
option_symbols="$(printf "%s\n" "${symbols[@]:0:50}" | paste -sd, -)"
[[ -n "$option_symbols" ]] || { echo "no options symbols" >&2; exit 4; }
csv_all="$(IFS=,; echo "${symbols[*]}")"
signalops-massive-puller --mode pull --date "$session_date" --symbols "$csv_all" --allow-unseeded-symbols --datasets equity --max-companies "${#symbols[@]}" --max-provider-requests "${#symbols[@]}" --max-events-built "${#symbols[@]}" --max-events-published "${#symbols[@]}" --max-retries 0 --continue-on-error=true --acknowledge-writes --dry-run=false
: "${SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL:?dedicated MarketOps temporal database required}"
option_count=${#symbols[@]}; (( option_count > 50 )) && option_count=50
minimum_normalized=$(( (option_count * 95 + 99) / 100 ))
deadline=$((SECONDS + 900))
while true; do normalized="$(psql "$SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL" -Atc "SELECT count(DISTINCT upper(normalized_payload->>\$q\$symbol\$q\$)) FROM normalized_event_ledger WHERE tenant_id=\$q\$tenant-local\$q\$ AND source_id=\$q\$src-massive\$q\$ AND dataset=\$q\$equity_eod_prices\$q\$ AND observation_time::date=DATE \$q\$${session_date}\$q\$ AND upper(normalized_payload->>\$q\$symbol\$q\$) = ANY(string_to_array(\$q\$${option_symbols}\$q\$, \$q\$,\$q\$));" | tr -d "[:space:]")"; [[ "$normalized" =~ ^[0-9]+$ && "$normalized" -ge "$minimum_normalized" ]] && break; (( SECONDS >= deadline )) && { echo "partial equity normalization accepted normalized=$normalized minimum=$minimum_normalized option_count=$option_count; downstream continues" >&2; break; }; sleep 10; done
if [[ "$normalized" =~ ^[0-9]+$ && "$normalized" -lt "$option_count" ]]; then echo "marketops_k8s_postclose_partial_coverage session=$session_date normalized=$normalized expected=$option_count"; fi
for ((i=0;i<${#symbols[@]};i+=10)); do
  batch=("${symbols[@]:i:10}"); csv=$(IFS=,; echo "${batch[*]}");
  signalops-marketops-intelligence-cohort-runner --tenant-id tenant-local --symbols "$csv" --max-symbols "${#batch[@]}" --session-start "$start_date" --session-end "$session_date" --stages preflight,state_materialization,hypothesis_evaluation,opportunity_build,outcome_materialization,hypothesis_proposal_generation --continue-on-error=true --dry-run=false --acknowledge-writes --run-id "k8s-postclose-${session_date}-$(printf '%03d' "$i")"
done
signalops-marketops-options-coverage-runner --tenant-id tenant-local --symbols "$option_symbols" --max-symbols 50 --session-date "$session_date" --run-id "k8s-postclose-${session_date}-options" --limit 250 --max-pages 2 --max-candidates 500 --min-dte 14 --max-dte 120 --min-moneyness 0.70 --max-moneyness 1.30 --skip-complete=true --continue-on-error=true --max-retries 0 --dry-run=false
signalops-marketops-valuation-runner --tenant-id tenant-local --universe-group all_active --session-date "$session_date" --dry-run=false --fmp-max-requests 300 --refresh-financials
signalops-marketops-tactical-valuation-runner --tenant-id tenant-local --universe-group all_active --session-date "$session_date"
signalops-marketops-eroc-runner --tenant-id tenant-local --universe-group all_active --session-date "$session_date" --dry-run=false
signalops-marketops-eeom-runner --tenant-id tenant-local --session-date "$session_date" --dry-run=false
signalops-marketops-syncratic-intelligence-runner --tenant-id tenant-local --session-date "$session_date"
# Refresh the subscriber-facing global projection after all source algorithms finish.
# This is append-only and runs against the dedicated K8s MarketOps database.
k8s-marketops-global-dashboard-projection "$session_date"
echo "marketops_k8s_postclose_writer_completed session=$session_date symbols=${#symbols[@]}"
