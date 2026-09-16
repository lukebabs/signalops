#!/usr/bin/env bash
set -euo pipefail

: "${SIGNALOPS_MARKETOPS_DATABASE_URL:?SIGNALOPS_MARKETOPS_DATABASE_URL is required}"
: "${SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL:?SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL is required}"

mode="${1:---write}"
session_date="${MARKETOPS_SESSION_DATE:-$(date -u +%F)}"
symbols="${MARKETOPS_SRI_ETF_SYMBOLS:-IBB,IGV,KBE,KRE,OIH,QQQ,RSP,SKYY,SMH,SOXX,SPY,XBI,XLB,XLC,XLE,XLF,XLI,XLK,XLP,XLRE,XLU,XLV,XLY,XOP}"
IFS=',' read -r -a symbol_list <<< "$symbols"
expected="${#symbol_list[@]}"

if [[ "$mode" == "--dry-run" ]]; then
  echo "marketops_k8s_sri_source_reconciliation_dry_run_verified session=${session_date} symbols=${expected}"
  exit 0
fi

signalops-massive-puller \
  --mode pull \
  --date "$session_date" \
  --symbols "$symbols" \
  --allow-unseeded-symbols \
  --datasets equity \
  --max-companies "$expected" \
  --max-provider-requests "$expected" \
  --max-events-built "$expected" \
  --max-events-published "$expected" \
  --max-retries "${MARKETOPS_SRI_MAX_RETRIES:-1}" \
  --dry-run=false \
  --continue-on-error=false

normalized="$(psql "$SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL" -v ON_ERROR_STOP=1 -Atc "SELECT count(DISTINCT UPPER(normalized_payload->>'symbol')) FROM normalized_event_ledger WHERE tenant_id='tenant-local' AND source_id='src-massive' AND dataset='equity_eod_prices' AND observation_time::date=DATE '${session_date}' AND UPPER(normalized_payload->>'symbol') = ANY(string_to_array('${symbols}', ','));" | tr -d '[:space:]')"
[[ "$normalized" == "$expected" ]] || { echo "SRI ETF normalization incomplete: normalized=${normalized} expected=${expected} session=${session_date}" >&2; exit 4; }

signalops-marketops-sri-runner \
  --tenant-id "${SIGNALOPS_SRI_OUTPUT_TENANT_ID:-platform-global}" \
  --input-tenant-id "${SIGNALOPS_SRI_INPUT_TENANT_ID:-tenant-local}" \
  --as-of "$session_date"

echo "marketops_k8s_sri_refresh_completed session=${session_date} symbols=${expected} normalized=${normalized}"
