#!/usr/bin/env bash
set -euo pipefail
cd "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if (($# > 0)); then
  exec docker compose --profile marketops-intraday run --rm marketops-intraday-monitor "$@"
fi

symbols="$(docker compose exec -T postgres psql -U signalops -d signalops -Atc "SELECT string_agg(ticker, ',' ORDER BY universe_priority, rank) FROM (SELECT DISTINCT ON (ticker) ticker, universe_priority, rank FROM marketops_universal_assets WHERE tenant_id='tenant-local' AND is_active ORDER BY ticker, universe_priority, rank) canonical;")"
[[ -n "$symbols" ]] || { echo "active MarketOps universe is empty" >&2; exit 3; }
IFS=',' read -r -a assets <<< "$symbols"
batch_size="${MARKETOPS_INTRADAY_BATCH_SIZE:-50}"
[[ "$batch_size" =~ ^[1-9][0-9]*$ ]] || { echo "MARKETOPS_INTRADAY_BATCH_SIZE must be positive" >&2; exit 2; }
for ((offset=0, batch=1; offset<${#assets[@]}; offset+=batch_size, batch++)); do
  batch_symbols=("${assets[@]:offset:batch_size}")
  batch_csv="$(IFS=,; printf '%s' "${batch_symbols[*]}")"
  docker compose --profile marketops-intraday run --rm marketops-intraday-monitor --universe-group all_active --symbols "$batch_csv" --max-symbols "${#batch_symbols[@]}"
done
