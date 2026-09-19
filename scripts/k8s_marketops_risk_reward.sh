#!/usr/bin/env bash
set -euo pipefail

: "${SIGNALOPS_MARKETOPS_DATABASE_URL:?SIGNALOPS_MARKETOPS_DATABASE_URL is required}"
session_date="${MARKETOPS_SESSION_DATE:-$(date -u +%F)}"
next_date="$(date -u -d "$session_date + 1 day" +%F)"
active_symbols="$(psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -v ON_ERROR_STOP=1 -Atc "SELECT string_agg(ticker, ',' ORDER BY universe_priority, rank) FROM (SELECT DISTINCT ON (ticker) ticker, universe_priority, rank FROM marketops_universal_assets WHERE tenant_id='tenant-local' AND is_active ORDER BY ticker, universe_priority, rank) canonical;")"
[[ -n "$active_symbols" ]] || { echo "active MarketOps universe is empty" >&2; exit 2; }
IFS=',' read -r -a symbols <<< "$active_symbols"
for symbol in "${symbols[@]}"; do
  signalops-algorithm-runner \
    --execution-request-id "marketops-risk-reward-${session_date}-${symbol}" \
    --tenant-id tenant-local \
    --algorithm-id signalops.algorithms.risk_reward_temporal_v1 \
    --algorithm-version risk_reward_temporal.v1 \
    --requested-by kubernetes-marketops \
    --correlation-id "marketops-risk-reward-${session_date}" \
    --dataset marketops_feature_vectors_daily \
    --feature risk_reward_technical_score \
    --symbols "$symbol" \
    --window-start "${session_date}T00:00:00Z" \
    --window-end "${next_date}T00:00:00Z" \
    --max-records 500 \
    --batch-size 500 \
    --min-samples 2 \
    --z-threshold 3.0
done
result_count="$(psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -v ON_ERROR_STOP=1 -Atc "SELECT count(DISTINCT result_payload->>'symbol') FROM algorithm_results WHERE tenant_id='tenant-local' AND algorithm_id='signalops.algorithms.risk_reward_temporal_v1' AND correlation_id='marketops-risk-reward-${session_date}';")"
snapshot_count="$(psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -v ON_ERROR_STOP=1 -Atc "SELECT count(DISTINCT symbol) FROM marketops_risk_reward_snapshots WHERE tenant_id='tenant-local' AND session_date=DATE '${session_date}';")"
expected_count="$(printf '%s' "$active_symbols" | awk -F',' '{print NF}')"
if [[ ! "$result_count" =~ ^[0-9]+$ || ! "$snapshot_count" =~ ^[0-9]+$ || "$result_count" -eq 0 || "$snapshot_count" -eq 0 ]]; then
  echo "marketops_k8s_risk_reward_incomplete session=${session_date} expected=${expected_count} results=${result_count} snapshots=${snapshot_count}" >&2
  exit 4
fi
if [[ "$result_count" -lt "$expected_count" || "$snapshot_count" -lt "$expected_count" ]]; then
  echo "marketops_k8s_risk_reward_degraded session=${session_date} expected=${expected_count} results=${result_count} snapshots=${snapshot_count}" >&2
else
  echo "marketops_k8s_risk_reward_executed session=${session_date} results=${result_count} snapshots=${snapshot_count}"
fi
