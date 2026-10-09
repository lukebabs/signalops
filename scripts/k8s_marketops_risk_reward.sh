#!/usr/bin/env bash
set -euo pipefail

: "${SIGNALOPS_MARKETOPS_DATABASE_URL:?SIGNALOPS_MARKETOPS_DATABASE_URL is required}"
session_date="${MARKETOPS_SESSION_DATE:-$(date -u +%F)}"
next_date="$(date -u -d "$session_date + 1 day" +%F)"
active_symbols="$(psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -v ON_ERROR_STOP=1 -Atc "SELECT string_agg(ticker, ',' ORDER BY universe_priority, rank) FROM (SELECT DISTINCT ON (ticker) ticker, universe_priority, rank FROM marketops_universal_assets WHERE tenant_id='tenant-local' AND is_active ORDER BY ticker, universe_priority, rank) canonical;")"
[[ -n "$active_symbols" ]] || { echo "active MarketOps universe is empty" >&2; exit 2; }
IFS=',' read -r -a symbols <<< "$active_symbols"
expected_count="$(printf '%s' "$active_symbols" | awk -F',' '{print NF}')"
required_features="range_position_252d,rsi_14,return_5d,volume_ratio_10d,distance_sma_50_pct,distance_sma_200_pct,sma_50_slope_20d_pct,atr_14_pct"
wait_seconds="${MARKETOPS_RISK_REWARD_WAIT_SECONDS:-1200}"
deadline=$((SECONDS + wait_seconds))
# The post-close writer persists feature observations in serialized batches.
# Wait briefly for the cohort to settle, then process every symbol with a
# complete technical vector. Assets without the required inputs are explicitly
# skipped and reported as degraded; they must not block the usable cohort.
ready_grace_seconds="${MARKETOPS_RISK_REWARD_GRACE_SECONDS:-120}"
ready_since=""
while true; do
  complete_count="$(psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -v ON_ERROR_STOP=1 -Atc "SELECT count(*) FROM (SELECT o.symbol FROM marketops_feature_observations o WHERE o.tenant_id='tenant-local' AND o.app_id='marketops' AND o.session_date=DATE '${session_date}' AND o.feature_key = ANY(string_to_array('${required_features}', ',')) GROUP BY o.symbol HAVING count(DISTINCT o.feature_key) = 8) complete;")"
  if [[ "$complete_count" =~ ^[0-9]+$ && "$complete_count" -ge "$expected_count" ]]; then
    echo "marketops_k8s_risk_reward_dependency_ready session=${session_date} symbols=${complete_count}/${expected_count}"
    break
  fi
  if [[ "$complete_count" =~ ^[0-9]+$ && "$complete_count" -gt 0 ]]; then
    [[ -n "$ready_since" ]] || ready_since=$SECONDS
    if (( SECONDS - ready_since >= ready_grace_seconds )); then
      echo "marketops_k8s_risk_reward_dependency_partial session=${session_date} symbols=${complete_count}/${expected_count}; skipping incomplete assets" >&2
      break
    fi
  fi
  if (( SECONDS >= deadline )); then
    echo "marketops_k8s_risk_reward_dependency_incomplete session=${session_date} symbols=${complete_count:-0}/${expected_count} wait_seconds=${wait_seconds}" >&2
    exit 42
  fi
  sleep 10
done
mapfile -t ready_symbols < <(psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -v ON_ERROR_STOP=1 -Atc "SELECT o.symbol FROM marketops_feature_observations o WHERE o.tenant_id='tenant-local' AND o.app_id='marketops' AND o.session_date=DATE '${session_date}' AND o.feature_key = ANY(string_to_array('${required_features}', ',')) GROUP BY o.symbol HAVING count(DISTINCT o.feature_key) = 8 ORDER BY o.symbol;")
for symbol in "${ready_symbols[@]}"; do
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
if [[ ! "$result_count" =~ ^[0-9]+$ || ! "$snapshot_count" =~ ^[0-9]+$ || "$result_count" -eq 0 || "$snapshot_count" -eq 0 ]]; then
  echo "marketops_k8s_risk_reward_incomplete session=${session_date} expected=${expected_count} results=${result_count} snapshots=${snapshot_count}" >&2
  exit 4
fi
ready_count="${#ready_symbols[@]}"
if [[ "$result_count" -lt "$ready_count" || "$snapshot_count" -lt "$ready_count" ]]; then
  echo "marketops_k8s_risk_reward_degraded session=${session_date} expected=${expected_count} results=${result_count} snapshots=${snapshot_count}" >&2
  exit 42
else
  echo "marketops_k8s_risk_reward_executed session=${session_date} results=${result_count} snapshots=${snapshot_count} ready=${ready_count} active=${expected_count}"
  (( ready_count == expected_count )) || exit 42
fi
