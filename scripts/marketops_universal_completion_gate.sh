#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=marketops_schedule_database.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/marketops_schedule_database.sh"

session_date="${1:?session date is required}"
symbols="${2:?active symbols are required}"
expected="${3:?expected asset count is required}"

compact() { tr -d "[:space:]"; }
eod="$(marketops_temporal_psql -Atc "SELECT count(DISTINCT normalized_payload->>'symbol') FROM normalized_event_ledger WHERE tenant_id='tenant-local' AND source_id='src-massive' AND dataset='equity_eod_prices' AND normalized_payload->>'observation_date'='${session_date}' AND normalized_payload->>'symbol' = ANY(string_to_array('${symbols}',','));" | compact)"
options="$(marketops_primary_psql -Atc "SELECT count(DISTINCT symbol) FROM marketops_options_capture_sessions WHERE tenant_id='tenant-local' AND session_date=DATE '${session_date}' AND symbol = ANY(string_to_array('${symbols}',','));" | compact)"
features="$(marketops_primary_psql -Atc "SELECT count(DISTINCT symbol) FROM marketops_feature_observations WHERE tenant_id='tenant-local' AND session_date=DATE '${session_date}' AND symbol = ANY(string_to_array('${symbols}',','));" | compact)"
risk_reward="$(marketops_primary_psql -Atc "SELECT count(DISTINCT result_payload->>'symbol') FROM algorithm_results WHERE tenant_id='tenant-local' AND algorithm_id='signalops.algorithms.risk_reward_temporal_v1' AND (result_payload->>'observation_time')::date=DATE '${session_date}' AND result_payload->>'symbol' = ANY(string_to_array('${symbols}',','));" | compact)"
snapshots="$(marketops_primary_psql -Atc "SELECT count(DISTINCT symbol) FROM marketops_risk_reward_snapshots WHERE tenant_id='tenant-local' AND session_date=DATE '${session_date}' AND symbol = ANY(string_to_array('${symbols}',','));" | compact)"

printf "universal completion gate session=%s expected=%s eod=%s options=%s features=%s risk_reward=%s snapshots=%s\\n" "$session_date" "$expected" "$eod" "$options" "$features" "$risk_reward" "$snapshots"
[[ "$eod" == "$expected" && "$options" == "$expected" && "$features" == "$expected" && "$risk_reward" == "$expected" && "$snapshots" == "$expected" ]]
