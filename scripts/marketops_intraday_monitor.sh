#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=marketops_schedule_database.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/marketops_schedule_database.sh"
# shellcheck source=marketops_coverage_tiers.sh
cd "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if (($# > 0)); then
  exec marketops_compose --profile marketops-intraday run --rm marketops-intraday-monitor "$@"
fi

# The production golden rule is the governed primary cohort. Watchlist hotness
# controls provider cadence elsewhere, but must not remove a primary asset from
# the canonical monitor/reconciliation path.
symbols="$(marketops_primary_psql -Atc "SELECT string_agg(ticker, ',' ORDER BY universe_priority, rank NULLS LAST, ticker) FROM marketops_primary_assets WHERE tenant_id='tenant-local'")"
[[ -n "$symbols" ]] || { echo "primary MarketOps cohort is empty"; exit 0; }
IFS="," read -r -a assets <<< "$symbols"
batch_size="${MARKETOPS_INTRADAY_BATCH_SIZE:-50}"
[[ "$batch_size" =~ ^[1-9][0-9]*$ ]] || { echo "MARKETOPS_INTRADAY_BATCH_SIZE must be positive" >&2; exit 2; }
for ((offset=0, batch=1; offset<${#assets[@]}; offset+=batch_size, batch++)); do
  batch_symbols=("${assets[@]:offset:batch_size}")
  batch_csv="$(IFS=,; printf "%s" "${batch_symbols[*]}")"
  marketops_compose --profile marketops-intraday run --rm marketops-intraday-monitor --universe-group primary_eod --symbols "$batch_csv" --max-symbols "${#batch_symbols[@]}"
done
