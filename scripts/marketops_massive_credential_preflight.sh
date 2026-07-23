#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  . ./.env
  set +a
fi

api_key="${SIGNALOPS_MASSIVE_API_KEY:-}"
if [[ -z "$api_key" ]]; then
  cat >&2 <<'MSG'
Missing explicit Massive API key. Set this in .env or the shell environment:
  SIGNALOPS_MASSIVE_API_KEY

The generic API_KEY variable is intentionally ignored by this preflight to avoid accidental use of unrelated credentials.
MSG
  exit 2
fi

base_url="${SIGNALOPS_MASSIVE_BASE_URL:-https://api.massive.com}"
symbol="${MARKETOPS_MASSIVE_PREFLIGHT_SYMBOL:-NVDA}"
observation_date="${MARKETOPS_MASSIVE_PREFLIGHT_DATE:-${MARKETOPS_INGEST_SMOKE_DATE:-${SIGNALOPS_MASSIVE_OBSERVATION_DATE:-$(date -u -d 'yesterday' +%F)}}}"
endpoint="${base_url%/}/v2/aggs/ticker/${symbol}/range/1/day/${observation_date}/${observation_date}"

printf 'Massive credential preflight: symbol=%s date=%s endpoint=%s\n' "$symbol" "$observation_date" "${base_url%/}/v2/aggs/ticker/..."

status="$(curl -sS -o /tmp/marketops_massive_preflight_body.json -w '%{http_code}' --get "$endpoint" --data-urlencode "apiKey=$api_key")"
case "$status" in
  2*)
    printf 'Massive credential preflight passed with HTTP %s.\n' "$status"
    ;;
  401|403)
    printf 'Massive credential preflight failed with HTTP %s. The configured key is present but rejected by Massive.\n' "$status" >&2
    exit 3
    ;;
  *)
    printf 'Massive credential preflight failed with HTTP %s. Check provider availability, base URL, entitlement, and request parameters.\n' "$status" >&2
    exit 4
    ;;
esac
