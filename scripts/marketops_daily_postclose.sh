#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

usage() {
  printf '%s\n' \
    'Usage: scripts/marketops_daily_postclose.sh [--date YYYY-MM-DD] [--dry-run|--write] [--plan]' \
    '' \
    'Runs one bounded post-close MarketOps workflow:' \
    '  credential preflight -> equity EOD -> normalization barrier -> options -> 10-symbol cohorts' \
    '' \
    '--write requires MARKETOPS_DAILY_ACKNOWLEDGE_WRITES=true.'
}

session_date=""
write_mode=false
plan_mode=false
while (($# > 0)); do
  case "$1" in
    --date)
      [[ $# -ge 2 ]] || { printf 'missing value for --date\n' >&2; exit 2; }
      session_date="$2"
      shift 2
      ;;
    --dry-run)
      write_mode=false
      shift
      ;;
    --write)
      write_mode=true
      shift
      ;;
    --plan)
      plan_mode=true
      shift
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      printf 'unknown argument: %s\n' "$1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  . ./.env
  set +a
fi

timezone="${MARKETOPS_DAILY_TIMEZONE:-America/New_York}"
minimum_equity_symbols="${MARKETOPS_DAILY_MIN_EQUITY_SYMBOLS:-50}"
normalization_timeout="${MARKETOPS_DAILY_NORMALIZATION_TIMEOUT_SECONDS:-300}"
normalization_poll="${MARKETOPS_DAILY_NORMALIZATION_POLL_SECONDS:-10}"
reconciliation_deadline="${MARKETOPS_DAILY_RECONCILIATION_DEADLINE:-15m}"
reconciliation_backoffs="${MARKETOPS_DAILY_RECONCILIATION_BACKOFFS:-30s,2m}"
reconciliation_attempts="${MARKETOPS_DAILY_RECONCILIATION_MAX_ATTEMPTS:-2}"
reconciliation_poll="${MARKETOPS_DAILY_RECONCILIATION_POLL:-5s}"
skip_complete_equity="${MARKETOPS_DAILY_SKIP_COMPLETE_EQUITY:-true}"
actor="${MARKETOPS_DAILY_ACTOR:-systemd-postclose}"
option_symbols="${MARKETOPS_DAILY_OPTION_SYMBOLS:-NVDA,AAPL,GOOGL,MSFT,AMZN,TSM,SPCX,AVGO,TSLA,META,MU,BRK.B,LLY,JPM,AMD,WMT,ASML,V,JNJ,INTC,XOM,TCEHY,MA,AMAT,ABBV,CSCO,CAT,LRCX,BAC,COST,ORCL,GE,UNH,KO,MS,HD,PG,ARM,HSBC,CVX,NFLX,PLTR,MRK,GS,GEV,PM,RY,BABA,NVS,PANW}"
lock_file="${MARKETOPS_DAILY_LOCK_FILE:-/tmp/signalops-marketops-daily.lock}"

log() {
  printf '%s %s\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" "$*"
}

is_true() {
  case "${1,,}" in
    1|true|yes|on) return 0 ;;
    *) return 1 ;;
  esac
}

require_positive_integer() {
  local name="$1" value="$2"
  [[ "$value" =~ ^[1-9][0-9]*$ ]] || { printf '%s must be a positive integer\n' "$name" >&2; exit 2; }
}

require_positive_integer MARKETOPS_DAILY_MIN_EQUITY_SYMBOLS "$minimum_equity_symbols"
require_positive_integer MARKETOPS_DAILY_NORMALIZATION_TIMEOUT_SECONDS "$normalization_timeout"
require_positive_integer MARKETOPS_DAILY_NORMALIZATION_POLL_SECONDS "$normalization_poll"

now_session_date="$(TZ="$timezone" date '+%F')"
local_clock="$(TZ="$timezone" date '+%H%M%S')"
if [[ -z "$session_date" ]]; then
  session_date="$now_session_date"
  if [[ "$local_clock" -lt 180000 ]]; then
    session_date="$(TZ="$timezone" date -d "$now_session_date -1 day" '+%F')"
    while (( $(TZ="$timezone" date -d "$session_date" '+%u') > 5 )); do
      session_date="$(TZ="$timezone" date -d "$session_date -1 day" '+%F')"
    done
  fi
fi
[[ "$session_date" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]] || { printf 'invalid session date: %s\n' "$session_date" >&2; exit 2; }
[[ "$(date -u -d "$session_date" '+%F' 2>/dev/null)" == "$session_date" ]] || { printf 'invalid session date: %s\n' "$session_date" >&2; exit 2; }
weekday="$(date -u -d "$session_date" '+%u')"
((weekday <= 5)) || { printf 'session date must be a weekday: %s\n' "$session_date" >&2; exit 2; }
if ! $plan_mode && [[ "$session_date" > "$now_session_date" ]]; then
  printf 'session date is in the future: %s\n' "$session_date" >&2
  exit 2
fi

if $write_mode && ! $plan_mode; then
  is_true "${MARKETOPS_DAILY_ACKNOWLEDGE_WRITES:-false}" || {
    printf 'write mode requires MARKETOPS_DAILY_ACKNOWLEDGE_WRITES=true\n' >&2
    exit 2
  }
  if [[ "$session_date" == "$now_session_date" ]]; then
    local_time="$(TZ="$timezone" date '+%H%M%S')"
    [[ "$local_time" -ge 180000 ]] || {
      printf 'same-session writes are blocked before 18:00:00 %s\n' "$timezone" >&2
      exit 2
    }
  fi
fi

option_symbols="${option_symbols//[[:space:]]/}"
IFS=',' read -r -a symbols <<< "$option_symbols"
(( ${#symbols[@]} > 0 && ${#symbols[@]} <= 50 )) || { printf 'option symbol count must be between 1 and 50\n' >&2; exit 2; }
for symbol in "${symbols[@]}"; do
  [[ "$symbol" =~ ^[A-Z0-9.]+$ ]] || { printf 'invalid option symbol: %s\n' "$symbol" >&2; exit 2; }
done

exec 9>"$lock_file"
flock -n 9 || { printf 'another post-close workflow holds %s\n' "$lock_file" >&2; exit 3; }

dry_run=true
$write_mode && dry_run=false
run_prefix="daily-evidence-${session_date//-/}"

equity_command=(docker compose --profile massive-pull run --rm massive-puller
  --datasets equity
  --date "$session_date"
  --max-companies 50
  --max-provider-requests 50
  --max-events-built 50
  --max-events-published 50
  --request-delay 250ms
  --max-retries 0
  --continue-on-error=true
  --dry-run="$dry_run")

reconciliation_command=(docker compose --profile massive-pull run --rm massive-puller
  --mode reconcile-equity
  --date "$session_date"
  --universe-group top50_megacap
  --max-provider-requests 100
  --max-attempts "$reconciliation_attempts"
  --deadline "$reconciliation_deadline"
  --retry-backoffs "$reconciliation_backoffs"
  --normalization-poll "$reconciliation_poll"
  --requeue-failed
  --dry-run="$dry_run")
$write_mode && reconciliation_command+=(--acknowledge-writes)

options_command=(docker compose --profile marketops-daily run --rm marketops-options-coverage-runner
  --tenant-id tenant-local
  --symbols "$option_symbols"
  --max-symbols "${#symbols[@]}"
  --session-date "$session_date"
  --run-id "${run_prefix}-options"
  --limit 250
  --max-pages 2
  --max-candidates 500
  --min-dte 14
  --max-dte 120
  --min-moneyness 0.70
  --max-moneyness 1.30
  --skip-complete=true
  --continue-on-error=true
  --max-retries 0
  --dry-run="$dry_run")

print_command() {
  printf '  '
  printf '%q ' "$@"
  printf '\n'
}

if $plan_mode; then
  log "plan session=$session_date timezone=$timezone write=$write_mode equity_threshold=$minimum_equity_symbols option_symbols=${#symbols[@]}"
  print_command "${equity_command[@]}"
  print_command "${reconciliation_command[@]}"
  print_command "${options_command[@]}"
  for ((offset=0, batch=1; offset<${#symbols[@]}; offset+=10, batch++)); do
    batch_symbols=("${symbols[@]:offset:10}")
    batch_csv="$(IFS=,; printf '%s' "${batch_symbols[*]}")"
    cohort=(docker compose --profile marketops-daily run --rm
      -e "SIGNALOPS_ACTOR=$actor"
      marketops-intelligence-cohort-runner
      --tenant-id tenant-local
      --symbols "$batch_csv"
      --max-symbols "${#batch_symbols[@]}"
      --session-start "$session_date"
      --session-end "$session_date"
      --stages "preflight,state_materialization,hypothesis_evaluation,opportunity_build,outcome_materialization,hypothesis_proposal_generation"
      --continue-on-error=true
      --run-id "$(printf '%s-cohort-%02d' "$run_prefix" "$batch")"
      --dry-run="$dry_run")
    $write_mode && cohort+=(--acknowledge-writes)
    print_command "${cohort[@]}"
  done
  exit 0
fi

for command in docker flock date; do
  command -v "$command" >/dev/null || { printf 'required command not found: %s\n' "$command" >&2; exit 2; }
done

running_services="$(docker compose ps --status running --services)"
for service in redpanda postgres timescaledb normalizer; do
  grep -qx "$service" <<< "$running_services" || { printf 'required service is not running: %s\n' "$service" >&2; exit 4; }
done
# The database universe is authoritative at execution time. This prevents a
# stale environment variable from omitting a newly listed asset from options
# capture or its downstream intelligence cohort.
active_universe_symbols="$(docker compose exec -T postgres psql -U signalops -d signalops -Atc \
  "SELECT string_agg(ticker, ',' ORDER BY universe_priority, rank) FROM marketops_universal_assets WHERE tenant_id='tenant-local' AND is_active;")"
[[ -n "$active_universe_symbols" ]] || { printf 'active equity universe is empty\n' >&2; exit 4; }
IFS=',' read -r -a active_universe_array <<< "$active_universe_symbols"
(( ${#active_universe_array[@]} >= minimum_equity_symbols )) || {
  printf 'active equity universe contains %d assets; expected at least %d\n' "${#active_universe_array[@]}" "$minimum_equity_symbols" >&2
  exit 4
}
for symbol in "${active_universe_array[@]}"; do
  [[ "$symbol" =~ ^[A-Z0-9.]+$ ]] || { printf 'invalid active universe symbol: %s\n' "$symbol" >&2; exit 4; }
done

minimum_equity_symbols="${#active_universe_array[@]}"
# Keep equity pulling, reconciliation, options polling, and ten-symbol cohorts aligned
# with the same active-universe snapshot for this scheduled run.
option_symbols="$active_universe_symbols"
symbols=("${active_universe_array[@]}")
watchlist_reconciliation_command=(docker compose --profile massive-pull run --rm massive-puller
  --mode reconcile-equity
  --date "$session_date"
  --universe-group analyst_watchlist
  --max-provider-requests 100
  --max-attempts "$reconciliation_attempts"
  --deadline "$reconciliation_deadline"
  --retry-backoffs "$reconciliation_backoffs"
  --normalization-poll "$reconciliation_poll"
  --requeue-failed
  --dry-run="$dry_run")
$write_mode && watchlist_reconciliation_command+=(--acknowledge-writes)
# Pull the same combined universe that later stages consume. Explicit symbols
# include provider-validated analyst-watchlist assets beyond the static seed.
equity_command=(docker compose --profile massive-pull run --rm massive-puller
  --datasets equity
  --date "$session_date"
  --symbols "$active_universe_symbols"
  --allow-unseeded-symbols
  --max-companies "${#active_universe_array[@]}"
  --max-provider-requests "${#active_universe_array[@]}"
  --max-events-built "${#active_universe_array[@]}"
  --max-events-published "${#active_universe_array[@]}"
  --request-delay 250ms
  --max-retries 0
  --continue-on-error=true
  --dry-run="$dry_run")
options_command=(docker compose --profile marketops-daily run --rm marketops-options-coverage-runner
  --tenant-id tenant-local
  --symbols "$option_symbols"
  --max-symbols "${#symbols[@]}"
  --session-date "$session_date"
  --run-id "${run_prefix}-options"
  --limit 250
  --max-pages 2
  --max-candidates 500
  --min-dte 14
  --max-dte 120
  --min-moneyness 0.70
  --max-moneyness 1.30
  --skip-complete=true
  --continue-on-error=true
  --max-retries 0
  --dry-run="$dry_run")

coverage_count() {
  local active_symbols
  active_symbols="$(docker compose exec -T postgres psql -U signalops -d signalops -Atc \
    "SELECT string_agg(ticker, ',' ORDER BY universe_priority, rank) FROM marketops_universal_assets WHERE tenant_id='tenant-local' AND is_active;")"
  [[ -n "$active_symbols" ]] || { printf 'active equity universe is empty\n' >&2; return 1; }
  docker compose exec -T timescaledb psql -U signalops -d signalops_temporal -Atc \
    "SELECT count(DISTINCT normalized_payload->>'symbol') FROM normalized_event_ledger WHERE tenant_id='tenant-local' AND source_id='src-massive' AND dataset='equity_eod_prices' AND normalized_payload->>'observation_date' = '$session_date' AND normalized_payload->>'symbol' = ANY(string_to_array('$active_symbols', ','));" | tr -d '[:space:]'
}

log "start session=$session_date timezone=$timezone write=$write_mode option_symbols=${#symbols[@]}"
MARKETOPS_MASSIVE_PREFLIGHT_DATE="$session_date" MARKETOPS_MASSIVE_PREFLIGHT_SYMBOL=GS \
  ./scripts/marketops_massive_credential_preflight.sh

current_coverage="$(coverage_count)"
if $write_mode && is_true "$skip_complete_equity" && (( current_coverage >= minimum_equity_symbols )); then
  log "equity stage skipped existing_coverage=$current_coverage threshold=$minimum_equity_symbols"
else
  log "equity stage started dry_run=$dry_run"
  "${equity_command[@]}"
fi

current_coverage="$(coverage_count)"
if (( current_coverage < minimum_equity_symbols )); then
  log "equity reconciliation started coverage=$current_coverage threshold=$minimum_equity_symbols dry_run=$dry_run"
  "${reconciliation_command[@]}"
  "${watchlist_reconciliation_command[@]}"
else
  log "equity reconciliation skipped coverage=$current_coverage threshold=$minimum_equity_symbols"
fi

deadline=$((SECONDS + normalization_timeout))
while true; do
  current_coverage="$(coverage_count)"
  if (( current_coverage >= minimum_equity_symbols )); then
    log "normalization barrier passed coverage=$current_coverage threshold=$minimum_equity_symbols"
    break
  fi
  if $dry_run; then
    log "dry-run stopped after equity: persisted coverage=$current_coverage is below threshold=$minimum_equity_symbols"
    exit 0
  fi
  if (( SECONDS >= deadline )); then
    printf 'normalization barrier timed out: coverage=%s threshold=%s\n' "$current_coverage" "$minimum_equity_symbols" >&2
    exit 5
  fi
  sleep "$normalization_poll"
done

log "options stage started dry_run=$dry_run"
"${options_command[@]}"

if $write_mode; then
  capture_count="$(docker compose exec -T postgres psql -U signalops -d signalops -Atc \
    "SELECT count(DISTINCT symbol) FROM marketops_options_capture_sessions WHERE tenant_id='tenant-local' AND session_date = DATE '$session_date' AND symbol = ANY (string_to_array('$option_symbols', ','));" | tr -d '[:space:]')"
  (( capture_count == ${#symbols[@]} )) || {
    printf 'options capture barrier failed: captures=%s expected=%s\n' "$capture_count" "${#symbols[@]}" >&2
    exit 6
  }
  log "options capture barrier passed captures=$capture_count"
fi

if $write_mode; then
  log "algorithm corroboration deferred until current-session cohorts have materialized features"
else
  log "algorithm corroboration skipped dry_run=true (algorithm runner has no non-mutating mode)"
fi

for ((offset=0, batch=1; offset<${#symbols[@]}; offset+=10, batch++)); do
  batch_symbols=("${symbols[@]:offset:10}")
  batch_csv="$(IFS=,; printf '%s' "${batch_symbols[*]}")"
  batch_run_id="$(printf '%s-cohort-%02d' "$run_prefix" "$batch")"
  if $write_mode; then
    existing_status="$(docker compose exec -T postgres psql -U signalops -d signalops -Atc \
      "SELECT status FROM marketops_intelligence_cohort_runs WHERE tenant_id='tenant-local' AND run_id='$batch_run_id';" | tr -d '[:space:]')"
    if [[ "$existing_status" == "succeeded" ]]; then
      log "cohort batch=$batch skipped run_id=$batch_run_id status=succeeded"
      continue
    fi
    [[ -z "$existing_status" ]] || {
      printf 'cohort run already exists with non-success status: run_id=%s status=%s\n' "$batch_run_id" "$existing_status" >&2
      exit 7
    }
  fi
  cohort=(docker compose --profile marketops-daily run --rm
    -e "SIGNALOPS_ACTOR=$actor"
    marketops-intelligence-cohort-runner
    --tenant-id tenant-local
    --symbols "$batch_csv"
    --max-symbols "${#batch_symbols[@]}"
    --session-start "$session_date"
    --session-end "$session_date"
    --stages "preflight,state_materialization,hypothesis_evaluation,opportunity_build,outcome_materialization,hypothesis_proposal_generation"
    --continue-on-error=true
    --run-id "$batch_run_id"
    --dry-run="$dry_run")
  $write_mode && cohort+=(--acknowledge-writes)
  log "cohort batch=$batch symbols=$batch_csv dry_run=$dry_run"
  "${cohort[@]}"
done

if $write_mode; then
  if ! ./scripts/marketops_algorithm_corroboration.sh --date "$session_date"; then
    printf "algorithm corroboration reported failures; universal completion gate will enforce required coverage\n" >&2
  fi
  ./scripts/marketops_universal_completion_gate.sh "$session_date" "$active_universe_symbols" "${#symbols[@]}" || exit 8
  docker compose --profile marketops-daily run --rm marketops-syncratic-intelligence-runner --tenant-id tenant-local --session-date "$session_date"
  summary="$(docker compose exec -T postgres psql -U signalops -d signalops -Atc \
    "SELECT 'captures=' || count(DISTINCT symbol) FROM marketops_options_capture_sessions WHERE tenant_id='tenant-local' AND session_date=DATE '$session_date' UNION ALL SELECT 'cohort_results=' || count(*) FROM marketops_intelligence_cohort_symbol_results WHERE tenant_id='tenant-local' AND run_id LIKE '${run_prefix}-cohort-%' UNION ALL SELECT 'algorithm_results=' || count(*) FROM algorithm_results WHERE tenant_id='tenant-local' AND correlation_id='$run_prefix';")"
  log "completed session=$session_date ${summary//$'\n'/ }"
else
  log "completed dry-run session=$session_date"
fi
