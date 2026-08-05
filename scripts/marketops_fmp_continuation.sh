#!/usr/bin/env bash
set -euo pipefail
cd "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
exec docker compose --profile marketops-daily run --rm marketops-valuation-runner \
  --tenant-id tenant-local --universe-group all_workflow_ready \
  --fmp-max-requests "${MARKETOPS_FMP_CONTINUATION_MAX_REQUESTS:-240}" --refresh-financials
