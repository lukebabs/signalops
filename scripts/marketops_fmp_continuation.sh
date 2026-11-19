#!/usr/bin/env bash
set -euo pipefail
cd "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=marketops_schedule_database.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/marketops_schedule_database.sh"
exec marketops_compose --profile marketops-daily run --rm marketops-valuation-runner \
  --tenant-id tenant-local --universe-group all_active \
  --fmp-max-requests "${MARKETOPS_FMP_CONTINUATION_MAX_REQUESTS:-240}" --refresh-financials
