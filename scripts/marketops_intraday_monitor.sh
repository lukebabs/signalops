#!/usr/bin/env bash
set -euo pipefail
cd "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
exec docker compose --profile marketops-intraday run --rm marketops-intraday-monitor "$@"
