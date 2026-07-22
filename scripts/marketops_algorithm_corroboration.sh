#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

session_date=""
while (($# > 0)); do
  case "$1" in
    --date) session_date="$2"; shift 2 ;;
    *) printf 'Usage: %s --date YYYY-MM-DD\n' "$0" >&2; exit 2 ;;
  esac
done
[[ "$session_date" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]] || { printf 'valid --date is required\n' >&2; exit 2; }

symbols="$(docker compose exec -T postgres psql -U signalops -d signalops -Atc "SELECT string_agg(ticker, ',' ORDER BY rank) FROM marketops_asset_universe WHERE tenant_id='tenant-local' AND universe_group='top50_megacap' AND is_active;")"
[[ -n "$symbols" ]] || { printf 'active universe is empty\n' >&2; exit 3; }
IFS=',' read -r -a symbol_list <<< "$symbols"
run_prefix="daily-evidence-${session_date//-/}"
window_start="${session_date}T00:00:00Z"
window_end="$(date -u -d "${session_date} +1 day" '+%Y-%m-%d')T00:00:00Z"
failures=0
for symbol in "${symbol_list[@]}"; do
  if ! docker compose --profile marketops-daily run --rm marketops-options-feature-materializer --tenant-id tenant-local --symbol "$symbol" --window 10_trade_days --limit 1 --run-id "${run_prefix}-features-${symbol}"; then
    printf 'feature materialization failed: %s\n' "$symbol" >&2
    failures=$((failures + 1))
  fi
done
for spec in 'equity_eod_prices daily_return_pct equity' 'options_distribution_daily call_put_open_interest_ratio options'; do
  read -r dataset feature suffix <<< "$spec"
  if ! docker compose --profile marketops-daily run --rm algorithm-runner --execution-request-id "${run_prefix}-algorithm-${suffix}" --tenant-id tenant-local --algorithm-id signalops.algorithms.zscore_anomaly_v1 --correlation-id "$run_prefix" --dataset "$dataset" --feature "$feature" --symbols "$symbols" --window-start "$window_start" --window-end "$window_end" --max-records 50 --batch-size 50 --min-samples 3 --z-threshold 3.0; then
    printf 'algorithm execution failed: %s\n' "$dataset" >&2
    failures=$((failures + 1))
  fi
done
# Platform algorithms score each asset's ordered history, not a same-session cross section.
history_start="$(date -u -d "${session_date} -45 days" '+%Y-%m-%d')T00:00:00Z"
for platform_algorithm in signalops.algorithms.river_anomaly_v1 signalops.algorithms.ruptures_change_point_v1 signalops.algorithms.statsmodels_forecast_v1; do
  algorithm_suffix="${platform_algorithm#signalops.algorithms.}"
  for symbol in "${symbol_list[@]}"; do
    if ! docker compose --profile marketops-daily run --rm algorithm-runner --execution-request-id "${run_prefix}-platform-${algorithm_suffix}-${symbol}" --tenant-id tenant-local --algorithm-id "$platform_algorithm" --correlation-id "$run_prefix" --dataset equity_eod_prices --feature open_close_move_pct --symbols "" --window-start "" --window-end "$window_end" --max-records 60 --batch-size 60 --min-samples 6 --z-threshold 3.0; then
      printf 'platform algorithm execution failed: %s %s\n' "$platform_algorithm" "$symbol" >&2
      failures=$((failures + 1))
    fi
  done
done

# Research-only multi-feature technical posture; put/call is corroboration only.
risk_reward_start="$(date -u -d "${session_date} - 400 days" '+%Y-%m-%d')T00:00:00Z"
for symbol in "${symbol_list[@]}"; do
  if ! docker compose --profile marketops-daily run --rm algorithm-runner --execution-request-id "${run_prefix}-risk-reward-${symbol}" --tenant-id tenant-local --algorithm-id signalops.algorithms.risk_reward_temporal_v1 --correlation-id "$run_prefix" --dataset marketops_feature_vectors_daily --feature risk_reward_technical_score --symbols "$symbol" --window-start "$risk_reward_start" --window-end "$window_end" --max-records 500 --batch-size 500 --min-samples 2 --z-threshold 3.0; then
    printf 'risk/reward execution failed: %s\n' "$symbol" >&2
    failures=$((failures + 1))
  fi
done
if ! docker compose --profile marketops-daily run --rm marketops-algorithm-adjudicator --tenant-id tenant-local --correlation-id "$run_prefix"; then
  printf 'algorithm adjudication failed: %s\n' "$run_prefix" >&2
  failures=$((failures + 1))
fi

printf 'algorithm_corroboration_failures=%d\n' "$failures"
exit "$failures"
