#!/usr/bin/env bash
# Read-only reconciliation gate for the dedicated MarketOps writer cutover.
# subscriber_* remains on the central shared control plane and is excluded.
set -euo pipefail

mode="${1:---pre-writer}"
case "$mode" in
  --pre-writer|--dedicated-authoritative) ;;
  *)
    printf 'Usage: %s [--pre-writer|--dedicated-authoritative]\n' "$0" >&2
    exit 2
    ;;
esac

SRC_P=signalops-postgres-1
SRC_PD=signalops
DST_P=signalops-marketops-postgres-1
DST_PD=marketops
SRC_T=signalops-timescaledb-1
SRC_TD=signalops_temporal
DST_T=signalops-marketops-timescaledb-1
DST_TD=marketops_temporal
failures=0

query() {
  docker exec "$1" psql -X -qAt -U signalops -d "$2" -c "$3"
}
tables() {
  query "$1" "$2" "SELECT tablename FROM pg_catalog.pg_tables WHERE schemaname='public' AND tablename !~ '^subscriber_' AND (tablename ~ '^(marketops_|sri_|algorithm_|catalog_|platform_primitive_)' OR tablename IN ('signal_ledger','normalized_event_ledger','signal_assurance_assertions','signal_assurance_outbox','signal_assurance_validation_contracts','signal_assurance_baselines')) ORDER BY tablename;"
}
compare() {
  local kind="$1" table="$2" sc tc result
  if [[ "$table" == "signal_ledger" || "$table" == "normalized_event_ledger" ]]; then
    sc="$(query "$3" "$4" "SELECT count(*) FROM public.$table WHERE app_id='marketops';")"
    tc="$(query "$5" "$6" "SELECT count(*) FROM public.$table WHERE app_id='marketops';")"
    table="$table (app_id=marketops)"
  else
    sc="$(query "$3" "$4" "SELECT count(*) FROM public.$table;")"
    tc="$(query "$5" "$6" "SELECT count(*) FROM public.$table;")"
  fi
  if [[ "$mode" == "--pre-writer" ]]; then
    [[ "$sc" == "$tc" ]] && result=OK || result=MISMATCH
  elif (( tc >= sc )); then
    [[ "$sc" == "$tc" ]] && result=OK || result=TARGET_AHEAD
  else
    result=MISMATCH
  fi
  printf '%-9s %-58s source=%-10s target=%-10s %s\n' "$kind" "$table" "$sc" "$tc" "$result"
  [[ "$result" != MISMATCH ]] || failures=$((failures+1))
}

echo "MarketOps writer-cutover preflight (read-only, mode=$mode)"
echo 'Primary reconciliation (subscriber_* excluded)'
while IFS= read -r t; do [[ -n "$t" ]] && compare primary "$t" "$SRC_P" "$SRC_PD" "$DST_P" "$DST_PD"; done < <(tables "$DST_P" "$DST_PD")
echo 'Temporal reconciliation (subscriber_* excluded)'
while IFS= read -r t; do [[ -n "$t" ]] && compare temporal "$t" "$SRC_T" "$SRC_TD" "$DST_T" "$DST_TD"; done < <(tables "$DST_T" "$DST_TD")

echo 'Boundary checks'
for check in \
  "$DST_P|$DST_PD|normalized_event_ledger|primary normalized events" \
  "$DST_P|$DST_PD|signal_ledger|primary signals" \
  "$DST_T|$DST_TD|normalized_event_ledger|temporal normalized events" \
  "$DST_T|$DST_TD|signal_ledger|temporal signals"; do
  IFS='|' read -r c d t label <<< "$check"
  n="$(query "$c" "$d" "SELECT count(*) FROM public.$t WHERE app_id <> 'marketops';")"
  printf '%-9s %s=%s' boundary "$label" "$n"
  if [[ "$n" == "0" ]]; then printf ' OK\n'; else printf ' MISMATCH\n'; failures=$((failures+1)); fi
done

if [[ "$failures" -gt 0 ]]; then
  echo "Preflight failed: $failures mismatch(es). No cutover action is authorized."
  exit 4
fi
if [[ "$mode" == "--pre-writer" ]]; then
  echo 'Preflight passed: core MarketOps source and dedicated-boundary counts agree.'
else
  echo 'Preflight passed: dedicated MarketOps counts meet or exceed the former shared source, with no cross-workload ledger rows.'
fi
