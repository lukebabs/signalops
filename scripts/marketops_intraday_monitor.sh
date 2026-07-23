#!/usr/bin/env bash
set -euo pipefail
cd "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if (($# > 0)); then
  exec docker compose --profile marketops-intraday run --rm marketops-intraday-monitor "$@"
fi
docker compose --profile marketops-intraday run --rm marketops-intraday-monitor --universe-group top50_megacap
docker compose --profile marketops-intraday run --rm marketops-intraday-monitor --universe-group analyst_watchlist
