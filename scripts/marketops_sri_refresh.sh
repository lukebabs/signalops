#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=marketops_schedule_database.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/marketops_schedule_database.sh"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

usage() {
  printf '%s\n' \
    'Usage: scripts/marketops_sri_refresh.sh [--date YYYY-MM-DD]' \
    '' \
    'Reconciles the dedicated SRI ETF registry to one completed EOD session,' \
    'waits for canonical normalization, then produces SRI snapshots.'
}

session_date=""
while (($# > 0)); do
  case "$1" in
    --date) [[ $# -ge 2 ]] || { printf 'missing value for --date\n' >&2; exit 2; }; session_date="$2"; shift 2 ;;
    --help|-h) usage; exit 0 ;;
    *) printf 'unknown argument: %s\n' "$1" >&2; usage >&2; exit 2 ;;
  esac
done

if [[ -f .env ]]; then
  # shellcheck source=lib/dotenv.sh
  source "$ROOT_DIR/scripts/lib/dotenv.sh"
  load_dotenv "$ROOT_DIR/.env"
fi

timezone="${MARKETOPS_SRI_TIMEZONE:-America/New_York}"
normalization_timeout="${MARKETOPS_SRI_NORMALIZATION_TIMEOUT_SECONDS:-600}"
normalization_poll="${MARKETOPS_SRI_NORMALIZATION_POLL_SECONDS:-5}"
lock_file="${MARKETOPS_SRI_LOCK_FILE:-/tmp/signalops-marketops-sri-refresh.lock}"
symbols="IBB,IGV,KBE,KRE,OIH,QQQ,RSP,SKYY,SMH,SOXX,SPY,XBI,XLB,XLC,XLE,XLF,XLI,XLK,XLP,XLRE,XLU,XLV,XLY,XOP"

if [[ -z "$session_date" ]]; then session_date="$(TZ="$timezone" date '+%F')"; fi
[[ "$session_date" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]] || { printf 'invalid session date: %s\n' "$session_date" >&2; exit 2; }
[[ "$(date -u -d "$session_date" '+%F' 2>/dev/null)" == "$session_date" ]] || { printf 'invalid session date: %s\n' "$session_date" >&2; exit 2; }
(( $(date -u -d "$session_date" '+%u') <= 5 )) || { printf 'session date must be a weekday: %s\n' "$session_date" >&2; exit 2; }
[[ "$normalization_timeout" =~ ^[1-9][0-9]*$ ]] || { printf 'MARKETOPS_SRI_NORMALIZATION_TIMEOUT_SECONDS must be a positive integer\n' >&2; exit 2; }
[[ "$normalization_poll" =~ ^[1-9][0-9]*$ ]] || { printf 'MARKETOPS_SRI_NORMALIZATION_POLL_SECONDS must be a positive integer\n' >&2; exit 2; }

IFS=',' read -r -a symbol_list <<< "$symbols"
expected_symbols="${#symbol_list[@]}"
exec 9>"$lock_file"
flock -n 9 || { printf 'another SRI refresh holds %s\n' "$lock_file" >&2; exit 3; }

printf '%s SRI source reconciliation started session=%s symbols=%s\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" "$session_date" "$expected_symbols"
marketops_compose --profile massive-pull run --rm massive-puller \
  --mode pull --date "$session_date" --symbols "$symbols" --allow-unseeded-symbols \
  --datasets equity --max-companies "$expected_symbols" --max-provider-requests "$expected_symbols" \
  --max-events-built "$expected_symbols" --max-events-published "$expected_symbols" \
  --max-retries 1 --dry-run=false

deadline=$((SECONDS + normalization_timeout))
while true; do
  normalized="$(marketops_temporal_psql -Atc \
    "SELECT count(DISTINCT UPPER(normalized_payload->>'symbol')) FROM normalized_event_ledger WHERE tenant_id='tenant-local' AND source_id='src-massive' AND dataset='equity_eod_prices' AND observation_time::date=DATE '$session_date' AND UPPER(normalized_payload->>'symbol') = ANY(string_to_array('$symbols', ','));" | tr -d '[:space:]')"
  [[ "$normalized" == "$expected_symbols" ]] && break
  if (( SECONDS >= deadline )); then
    printf 'SRI source normalization incomplete: normalized=%s expected=%s session=%s\n' "$normalized" "$expected_symbols" "$session_date" >&2
    exit 4
  fi
  sleep "$normalization_poll"
done

printf '%s SRI source normalization passed session=%s symbols=%s\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" "$session_date" "$normalized"
marketops_compose --profile marketops-daily run --rm marketops-sri-runner \
