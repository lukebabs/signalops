#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=marketops_schedule_database.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/marketops_schedule_database.sh"
# shellcheck source=marketops_coverage_tiers.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/marketops_coverage_tiers.sh"
cd "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if (($# > 0)); then
  exec marketops_compose --profile marketops-intraday run --rm marketops-intraday-monitor "$@"
fi

symbols="$(marketops_hot_intraday_symbols)"
[[ -n "$symbols" ]] || { echo "no explicitly selected watchlist assets require intraday monitoring"; exit 0; }
IFS="," read -r -a assets <<< "$symbols"
batch_size="${MARKETOPS_INTRADAY_BATCH_SIZE:-50}"
[[ "$batch_size" =~ ^[1-9][0-9]*$ ]] || { echo "MARKETOPS_INTRADAY_BATCH_SIZE must be positive" >&2; exit 2; }
for ((offset=0, batch=1; offset<${#assets[@]}; offset+=batch_size, batch++)); do
  batch_symbols=("${assets[@]:offset:batch_size}")
  batch_csv="$(IFS=,; printf "%s" "${batch_symbols[*]}")"
  marketops_compose --profile marketops-intraday run --rm marketops-intraday-monitor --universe-group all_active --symbols "$batch_csv" --max-symbols "${#batch_symbols[@]}"
done
