#!/usr/bin/env bash
set -euo pipefail
# shellcheck source=marketops_schedule_database.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/marketops_schedule_database.sh"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

usage() {
  printf '%s\n' \
    'Usage: scripts/marketops_sri_refresh.sh [--date YYYY-MM-DD] [--normalized-only]' \
    '' \
    'Reconciles the dedicated SRI ETF registry to one completed EOD session,' \
    'waits for canonical normalization, then produces SRI snapshots.' \
    '' \
    '--normalized-only never calls a provider. It requires the canonical' \
    'dedicated temporal ledger to already contain the full SRI ETF source set.'
}

session_date=""
normalized_only=false
while (($# > 0)); do
  case "$1" in
    --date) [[ $# -ge 2 ]] || { printf 'missing value for --date\n' >&2; exit 2; }; session_date="$2"; shift 2 ;;
    --normalized-only) normalized_only=true; shift ;;
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

normalized_symbol_count() {
  local count
  count="$(marketops_temporal_psql -Atc \
    "SELECT count(DISTINCT UPPER(normalized_payload->>'symbol')) FROM normalized_event_ledger WHERE tenant_id='tenant-local' AND source_id='src-massive' AND dataset='equity_eod_prices' AND observation_time::date=DATE '$session_date' AND UPPER(normalized_payload->>'symbol') = ANY(string_to_array('$symbols', ','));" | tr -d '[:space:]')"
  [[ "$count" =~ ^[0-9]+$ ]] || {
    printf 'invalid SRI normalized source count: %s\n' "$count" >&2
    exit 5
  }
  printf '%s\n' "$count"
}

normalized="$(normalized_symbol_count)"
if [[ "$normalized" == "$expected_symbols" ]]; then
  printf '%s SRI source reconciliation reused canonical normalization session=%s symbols=%s\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" "$session_date" "$normalized"
elif $normalized_only; then
  printf 'SRI canonical normalization incomplete: normalized=%s expected=%s session=%s; provider pull intentionally disabled\n' "$normalized" "$expected_symbols" "$session_date" >&2
  exit 4
else
printf '%s SRI source reconciliation started session=%s symbols=%s\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" "$session_date" "$expected_symbols"
marketops_compose --profile massive-pull run --rm massive-puller \
  --mode pull --date "$session_date" --symbols "$symbols" --allow-unseeded-symbols \
  --datasets equity --max-companies "$expected_symbols" --max-provider-requests "$expected_symbols" \
  --max-events-built "$expected_symbols" --max-events-published "$expected_symbols" \
  --max-retries 1 --dry-run=false

deadline=$((SECONDS + normalization_timeout))
while true; do
  normalized="$(normalized_symbol_count)"
  [[ "$normalized" == "$expected_symbols" ]] && break
  if (( SECONDS >= deadline )); then
    printf 'SRI source normalization incomplete: normalized=%s expected=%s session=%s\n' "$normalized" "$expected_symbols" "$session_date" >&2
    exit 4
  fi
  sleep "$normalization_poll"
done
fi

printf '%s SRI source normalization passed session=%s symbols=%s\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" "$session_date" "$normalized"
marketops_compose --profile marketops-daily run --rm marketops-sri-runner \
  --tenant-id "${SIGNALOPS_SRI_OUTPUT_TENANT_ID:-platform-global}" \
  --input-tenant-id "${SIGNALOPS_SRI_INPUT_TENANT_ID:-tenant-local}" \
  --as-of "$session_date"

output_tenant="${SIGNALOPS_SRI_OUTPUT_TENANT_ID:-platform-global}"
expected_segments="$(marketops_primary_psql -Atc \
  "SELECT count(*) FROM sri_segments WHERE tenant_id='$output_tenant' AND active AND segment_type <> 'benchmark';" | tr -d '[:space:]')"
materialized_segments="$(marketops_primary_psql -Atc \
  "SELECT count(DISTINCT segment_id) FROM sri_segment_snapshots WHERE tenant_id='$output_tenant' AND session_date=DATE '$session_date';" | tr -d '[:space:]')"
[[ "$expected_segments" =~ ^[1-9][0-9]*$ && "$materialized_segments" =~ ^[0-9]+$ ]] || {
  printf 'invalid SRI output coverage: expected=%s materialized=%s\n' "$expected_segments" "$materialized_segments" >&2
  exit 6
}
if [[ "$materialized_segments" != "$expected_segments" ]]; then
  printf 'SRI output materialization incomplete: materialized=%s expected=%s tenant=%s session=%s\n' "$materialized_segments" "$expected_segments" "$output_tenant" "$session_date" >&2
  exit 6
fi
printf '%s SRI output materialization passed tenant=%s session=%s segments=%s\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" "$output_tenant" "$session_date" "$materialized_segments"
