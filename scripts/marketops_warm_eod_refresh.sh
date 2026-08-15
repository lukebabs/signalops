#!/usr/bin/env bash
set -euo pipefail
# Completed-session acquisition for the centrally governed warm EOD tier. This
# path deliberately does not capture options or intraday quotes.
# shellcheck source=marketops_schedule_database.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/marketops_schedule_database.sh"
# shellcheck source=marketops_coverage_tiers.sh
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/marketops_coverage_tiers.sh"

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"
session_date=""
write_mode=false
while (($# > 0)); do
  case "$1" in
    --date) [[ $# -ge 2 ]] || { printf 'missing value for --date\n' >&2; exit 2; }; session_date="$2"; shift 2 ;;
    --dry-run) write_mode=false; shift ;;
    --write) write_mode=true; shift ;;
    --help|-h) printf '%s\n' 'Usage: scripts/marketops_warm_eod_refresh.sh [--date YYYY-MM-DD] [--dry-run|--write]'; exit 0 ;;
    *) printf 'unknown argument: %s\n' "$1" >&2; exit 2 ;;
  esac
done
if [[ -f .env ]]; then set -a; . ./.env; set +a; fi
timezone="${MARKETOPS_WARM_EOD_TIMEZONE:-America/New_York}"
batch_size="${MARKETOPS_WARM_EOD_BATCH_SIZE:-100}"
normalization_timeout="${MARKETOPS_WARM_EOD_NORMALIZATION_TIMEOUT_SECONDS:-900}"
normalization_poll="${MARKETOPS_WARM_EOD_NORMALIZATION_POLL_SECONDS:-10}"
lock_file="${MARKETOPS_WARM_EOD_LOCK_FILE:-/tmp/signalops-marketops-warm-eod.lock}"
if [[ -z "$session_date" ]]; then
  session_date="$(TZ="$timezone" date '+%F')"
  [[ "$(TZ="$timezone" date '+%H%M%S')" -ge 180000 ]] || session_date="$(TZ="$timezone" date -d "$session_date -1 day" '+%F')"
fi
[[ "$session_date" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ && "$(date -u -d "$session_date" '+%F' 2>/dev/null)" == "$session_date" ]] || { printf 'invalid session date: %s\n' "$session_date" >&2; exit 2; }
(( $(date -u -d "$session_date" '+%u') <= 5 )) || { printf 'session date must be a weekday: %s\n' "$session_date" >&2; exit 2; }
[[ "$batch_size" =~ ^[1-9][0-9]*$ && "$batch_size" -le 250 ]] || { printf 'MARKETOPS_WARM_EOD_BATCH_SIZE must be 1 through 250\n' >&2; exit 2; }
[[ "$normalization_timeout" =~ ^[1-9][0-9]*$ && "$normalization_poll" =~ ^[1-9][0-9]*$ ]] || { printf 'warm EOD normalization settings must be positive integers\n' >&2; exit 2; }
if $write_mode && [[ "${MARKETOPS_WARM_EOD_ACKNOWLEDGE_WRITES:-false}" != "true" ]]; then printf 'warm EOD write mode requires MARKETOPS_WARM_EOD_ACKNOWLEDGE_WRITES=true\n' >&2; exit 2; fi
symbols="$(marketops_warm_eod_symbols)"
[[ -n "$symbols" ]] || { printf 'no enabled warm EOD plan is available\n' >&2; exit 4; }
IFS=',' read -r -a symbol_list <<< "$symbols"
(( ${#symbol_list[@]} <= 1000 )) || { printf 'warm EOD set exceeds the 1000-symbol policy cap\n' >&2; exit 4; }
exec 9>"$lock_file"
flock -n 9 || { printf 'another warm EOD refresh holds %s\n' "$lock_file" >&2; exit 3; }
dry_run=true; $write_mode && dry_run=false
printf '%s warm EOD acquisition started session=%s symbols=%s write=%s\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" "$session_date" "${#symbol_list[@]}" "$write_mode"
for ((offset=0; offset<${#symbol_list[@]}; offset+=batch_size)); do
  batch=("${symbol_list[@]:offset:batch_size}")
  batch_csv="$(IFS=,; printf '%s' "${batch[*]}")"
  marketops_compose --profile massive-pull run --rm massive-puller --mode pull --date "$session_date" --symbols "$batch_csv" --allow-unseeded-symbols --datasets equity --max-companies "${#batch[@]}" --max-provider-requests "${#batch[@]}" --max-events-built "${#batch[@]}" --max-events-published "${#batch[@]}" --request-delay 250ms --max-retries 0 --continue-on-error=true --dry-run="$dry_run"
done
if $dry_run; then printf '%s warm EOD dry-run completed session=%s symbols=%s\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" "$session_date" "${#symbol_list[@]}"; exit 0; fi
deadline=$((SECONDS + normalization_timeout))
while true; do
  normalized="$(marketops_temporal_psql -Atc "SELECT count(DISTINCT upper(normalized_payload->>'symbol')) FROM normalized_event_ledger WHERE tenant_id='tenant-local' AND source_id='src-massive' AND dataset='equity_eod_prices' AND observation_time::date=DATE '$session_date' AND upper(normalized_payload->>'symbol') = ANY(string_to_array('$symbols', ','));" | tr -d '[:space:]')"
  [[ "$normalized" == "${#symbol_list[@]}" ]] && break
  (( SECONDS < deadline )) || { printf 'warm EOD normalization incomplete: normalized=%s expected=%s session=%s\n' "$normalized" "${#symbol_list[@]}" "$session_date" >&2; exit 5; }
  sleep "$normalization_poll"
done
printf '%s warm EOD acquisition completed session=%s symbols=%s\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" "$session_date" "$normalized"
