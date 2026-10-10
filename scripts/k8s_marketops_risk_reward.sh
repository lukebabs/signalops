#!/usr/bin/env bash
set -euo pipefail

: "${SIGNALOPS_MARKETOPS_DATABASE_URL:?SIGNALOPS_MARKETOPS_DATABASE_URL is required}"
session_date="${MARKETOPS_SESSION_DATE:-$(date -u +%F)}"
next_date="$(date -u -d "$session_date + 1 day" +%F)"
active_symbols="$(psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -v ON_ERROR_STOP=1 -Atc "SELECT string_agg(ticker, ',' ORDER BY universe_priority, rank NULLS LAST, ticker) FROM marketops_primary_assets WHERE tenant_id='tenant-local';")"
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
psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -v ON_ERROR_STOP=1 -v session_date="$session_date" <<'SQL'
WITH primary_assets AS (
  SELECT ticker FROM marketops_primary_assets WHERE tenant_id='tenant-local'
), coverage AS (
  SELECT p.ticker,
         count(DISTINCT o.feature_key) FILTER (
           WHERE o.quality_state IN ('usable','usable_with_warning')
             AND o.numeric_value IS NOT NULL
             AND o.feature_key IN ('range_position_252d','rsi_14','return_5d','volume_ratio_10d',
                                   'distance_sma_50_pct','distance_sma_200_pct',
                                   'sma_50_slope_20d_pct','atr_14_pct')
         )::integer AS usable_count
  FROM primary_assets p
  LEFT JOIN marketops_feature_observations o
    ON o.tenant_id='tenant-local' AND o.app_id='marketops'
   AND o.symbol=p.ticker AND o.session_date=:'session_date'::date
  GROUP BY p.ticker
), missing AS (
  SELECT c.* FROM coverage c
  LEFT JOIN marketops_risk_reward_snapshots existing
    ON existing.tenant_id='tenant-local' AND existing.symbol=c.ticker
   AND existing.session_date=:'session_date'::date
  WHERE existing.snapshot_id IS NULL
)
INSERT INTO marketops_risk_reward_snapshots (
  snapshot_id,tenant_id,algorithm_result_id,execution_request_id,symbol,
  session_date,observed_at,technical_score,technical_direction,risk_level,
  confidence,usable_input_count,required_input_count,eligible,result_payload,input_snapshot
)
SELECT 'rr-unavailable-'||md5('tenant-local:'||ticker||':'||:'session_date'),
  'tenant-local',
  'rr-unavailable-result-'||md5('tenant-local:'||ticker||':'||:'session_date'),
  'marketops-risk-reward-'||:'session_date'||'-'||ticker,
  ticker,:'session_date'::date,:'session_date'::timestamptz+interval '20 hours',
  0,'neutral','unavailable',0,usable_count,8,false,
  jsonb_build_object('status','unavailable','reason','required technical inputs unavailable',
                     'session_date',:'session_date','source','marketops_primary_assets'),
  jsonb_build_object('usable_input_count',usable_count,'required_input_count',8)
FROM missing
ON CONFLICT (tenant_id,algorithm_result_id) DO UPDATE SET
  usable_input_count=EXCLUDED.usable_input_count,
  result_payload=EXCLUDED.result_payload,
  input_snapshot=EXCLUDED.input_snapshot,
  created_at=now();
SQL
result_count="$(psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -v ON_ERROR_STOP=1 -Atc "SELECT count(DISTINCT result_payload->>'symbol') FROM algorithm_results WHERE tenant_id='tenant-local' AND algorithm_id='signalops.algorithms.risk_reward_temporal_v1' AND correlation_id='marketops-risk-reward-${session_date}';")"
snapshot_count="$(psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -v ON_ERROR_STOP=1 -Atc "SELECT count(DISTINCT symbol) FROM marketops_risk_reward_snapshots WHERE tenant_id='tenant-local' AND session_date=DATE '${session_date}';")"
if [[ ! "$result_count" =~ ^[0-9]+$ || ! "$snapshot_count" =~ ^[0-9]+$ || "$result_count" -eq 0 || "$snapshot_count" -eq 0 ]]; then
  echo "marketops_k8s_risk_reward_incomplete session=${session_date} expected=${expected_count} results=${result_count} snapshots=${snapshot_count}" >&2
  exit 4
fi
ready_count="${#ready_symbols[@]}"
if [[ "$result_count" -lt "$ready_count" || "$snapshot_count" -lt "$expected_count" ]]; then
  echo "marketops_k8s_risk_reward_degraded session=${session_date} expected=${expected_count} results=${result_count} snapshots=${snapshot_count}" >&2
  exit 42
else
  echo "marketops_k8s_risk_reward_executed session=${session_date} results=${result_count} snapshots=${snapshot_count} ready=${ready_count} unavailable=$((expected_count - ready_count)) active=${expected_count}"
fi
