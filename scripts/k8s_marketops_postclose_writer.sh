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
mapfile -t symbols < <(psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -Atc "SELECT ticker FROM marketops_primary_assets WHERE tenant_id='tenant-local' ORDER BY universe_priority,rank NULLS LAST,ticker")
((${#symbols[@]} > 0)) || { echo "no active MarketOps symbols" >&2; exit 3; }
# SRI is platform-global and consumes a fixed ETF source set that is not part
# of the tenant-local stock universe. Capture those 24 ETFs in this same
# bounded provider pull; downstream options and algorithms remain stock-only.
sri_symbols="IBB,IGV,KBE,KRE,OIH,QQQ,RSP,SKYY,SMH,SOXX,SPY,XBI,XLB,XLC,XLE,XLF,XLI,XLK,XLP,XLRE,XLU,XLV,XLY,XOP"
option_symbols="$(printf "%s\n" "${symbols[@]}" | paste -sd, -)"
[[ -n "$option_symbols" ]] || { echo "no options symbols" >&2; exit 4; }
csv_all="$(IFS=,; echo "${symbols[*]}")"
capture_symbols="${csv_all},${sri_symbols}"
capture_count="$(printf '%s\n' "$capture_symbols" | tr ',' '\n' | sort -u | awk 'NF { count++ } END { print count + 0 }')"
signalops-massive-puller --mode pull --date "$session_date" --symbols "$capture_symbols" --allow-unseeded-symbols --datasets equity --max-companies "$capture_count" --max-provider-requests "$capture_count" --max-events-built "$capture_count" --max-events-published "$capture_count" --max-retries 0 --continue-on-error=true --acknowledge-writes --dry-run=false
: "${SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL:?dedicated MarketOps temporal database required}"
option_count=${#symbols[@]}
minimum_normalized=$(( (option_count * 95 + 99) / 100 ))
deadline=$((SECONDS + 900))
while true; do normalized="$(psql "$SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL" -Atc "SELECT count(DISTINCT upper(normalized_payload->>\$q\$symbol\$q\$)) FROM normalized_event_ledger WHERE tenant_id=\$q\$tenant-local\$q\$ AND source_id=\$q\$src-massive\$q\$ AND dataset=\$q\$equity_eod_prices\$q\$ AND observation_time::date=DATE \$q\$${session_date}\$q\$ AND upper(normalized_payload->>\$q\$symbol\$q\$) = ANY(string_to_array(\$q\$${option_symbols}\$q\$, \$q\$,\$q\$));" | tr -d "[:space:]")"; [[ "$normalized" =~ ^[0-9]+$ && "$normalized" -ge "$minimum_normalized" ]] && break; (( SECONDS >= deadline )) && { echo "partial equity normalization accepted normalized=$normalized minimum=$minimum_normalized option_count=$option_count; downstream continues" >&2; break; }; sleep 10; done
if [[ "$normalized" =~ ^[0-9]+$ && "$normalized" -lt "$option_count" ]]; then echo "marketops_k8s_postclose_partial_coverage session=$session_date normalized=$normalized expected=$option_count"; fi

# Reconcile only missing equity EOD rows after the initial pull. This is
# intentionally separate from the provider pull so a transient failure for one
# asset cannot block the rest of the post-close algorithms. The reconciler
# discovers the live primary operational union from all_active, reuses
# any raw events already present, retries missing symbols at most twice, and
# reports an actionable recovery state when a provider gap remains.
reconciliation_degraded=false
if ! signalops-massive-puller --mode reconcile-equity --date "$session_date" --universe-group primary_eod \
  --max-provider-requests "${MARKETOPS_EOD_RECONCILIATION_MAX_PROVIDER_REQUESTS:-0}" \
  --max-attempts "${MARKETOPS_EOD_RECONCILIATION_MAX_ATTEMPTS:-2}" \
  --deadline "${MARKETOPS_EOD_RECONCILIATION_DEADLINE:-15m}" \
  --retry-backoffs "${MARKETOPS_EOD_RECONCILIATION_BACKOFFS:-30s,2m}" \
  --normalization-poll "${MARKETOPS_EOD_RECONCILIATION_POLL:-5s}" \
  --requeue-failed --acknowledge-writes; then
  reconciliation_degraded=true
  echo "marketops_k8s_postclose_eod_reconciliation_degraded session=$session_date; continuing algorithm stages" >&2
fi
# Options coverage is an upstream dependency of state materialization and hypothesis evaluation.
# Run it before the cohort stages so the hypotheses observe the current session's option features.
# The options runner accepts at most 200 symbols per invocation. Process the
# complete operational union in bounded batches so expanding the catalog does
# not silently drop symbols or block the post-close pipeline.
for ((options_offset=0; options_offset<${#symbols[@]}; options_offset+=200)); do
  options_batch=("${symbols[@]:options_offset:200}")
  options_csv="$(IFS=,; echo "${options_batch[*]}")"
  signalops-marketops-options-coverage-runner --tenant-id tenant-local --symbols "$options_csv" --max-symbols "${#options_batch[@]}" --session-date "$session_date" --run-id "k8s-postclose-${session_date}-options-$(printf '%03d' "$options_offset")" --limit 250 --max-pages 2 --max-candidates 500 --min-dte 14 --max-dte 120 --min-moneyness 0.70 --max-moneyness 1.30 --skip-complete=true --continue-on-error=true --max-retries 0 --dry-run=false
done
for ((i=0;i<${#symbols[@]};i+=10)); do
  batch=("${symbols[@]:i:10}"); csv=$(IFS=,; echo "${batch[*]}");
  if ! signalops-marketops-intelligence-cohort-runner --tenant-id tenant-local --symbols "$csv" --max-symbols "${#batch[@]}" --session-start "$start_date" --session-end "$session_date" --stages preflight,state_materialization,hypothesis_evaluation,opportunity_build,outcome_materialization,hypothesis_proposal_generation --continue-on-error=true --dry-run=false --acknowledge-writes --run-id "k8s-postclose-${session_date}-$(printf '%03d' "$i")"; then
    echo "marketops_k8s_postclose_cohort_batch_continued batch=$i session=$session_date reason=duplicate_or_partial_ledger" >&2
  fi
done
signalops-marketops-valuation-runner --tenant-id tenant-local --universe-group primary_eod --session-date "$session_date" --dry-run=false --fmp-max-requests 300 --refresh-financials
signalops-marketops-tactical-valuation-runner --tenant-id tenant-local --universe-group primary_eod --session-date "$session_date"
signalops-marketops-eroc-runner --tenant-id tenant-local --universe-group primary_eod --session-date "$session_date" --dry-run=false
signalops-marketops-eeom-runner --tenant-id tenant-local --session-date "$session_date" --dry-run=false
signalops-marketops-syncratic-intelligence-runner --tenant-id tenant-local --session-date "$session_date"
# Refresh the subscriber-facing global projection after all source algorithms finish.
# This is append-only and runs against the dedicated K8s MarketOps database.
k8s-marketops-global-dashboard-projection "$session_date"
echo "marketops_k8s_postclose_writer_completed session=$session_date symbols=${#symbols[@]}"
if [[ "$reconciliation_degraded" == true ]]; then
  echo "marketops_k8s_postclose_writer_recovery_needed session=$session_date reason=equity_eod_reconciliation_incomplete" >&2
  exit 42
fi
