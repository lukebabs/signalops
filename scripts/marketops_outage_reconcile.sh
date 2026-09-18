#!/usr/bin/env bash
set -euo pipefail

# Reconcile one missed completed MarketOps trading session after host outage.
# This is intentionally bounded and date-specific. It does not accept ranges,
# weekends, holidays, or future dates.

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"
# shellcheck source=marketops_schedule_database.sh
source "$ROOT_DIR/scripts/marketops_schedule_database.sh"
# shellcheck source=lib/marketops_trading_calendar.sh
source "$ROOT_DIR/scripts/lib/marketops_trading_calendar.sh"

usage() {
  printf '%s\n' 'Usage: scripts/marketops_outage_reconcile.sh --date YYYY-MM-DD --write'
}

session_date=""
write_mode=false
while (($# > 0)); do
  case "$1" in
    --date)
      [[ $# -ge 2 ]] || { printf 'missing value for --date\n' >&2; exit 2; }
      session_date="$2"
      shift 2
      ;;
    --write)
      write_mode=true
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

[[ "$session_date" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]] || { usage >&2; exit 2; }
[[ "$(date -u -d "$session_date" '+%F' 2>/dev/null)" == "$session_date" ]] || { printf 'invalid session date: %s\n' "$session_date" >&2; exit 2; }
$write_mode || { printf 'outage reconciliation requires --write\n' >&2; exit 2; }

timezone="${MARKETOPS_DAILY_TIMEZONE:-America/New_York}"
now_session_date="$(TZ="$timezone" date '+%F')"
if [[ "$session_date" > "$now_session_date" ]]; then
  printf 'session date is in the future: %s\n' "$session_date" >&2
  exit 2
fi
marketops_is_trading_day "$timezone" "$session_date" || { printf 'session date must be a trading day: %s\n' "$session_date" >&2; exit 2; }

# Prevent accidental same-session catch-up before the market has actually closed.
if [[ "$session_date" == "$now_session_date" && "$(TZ="$timezone" date '+%H%M%S')" -lt 163000 ]]; then
  printf 'same-session outage reconciliation is blocked before 16:30:00 %s\n' "$timezone" >&2
  exit 2
fi

printf '%s outage reconciliation started session=%s\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" "$session_date"

lock_dir="${MARKETOPS_OUTAGE_RECONCILE_LOCK_DIR:-/run/signalops/marketops-outage-reconcile}"
mkdir -p "$lock_dir"
export MARKETOPS_WARM_EOD_LOCK_FILE="$lock_dir/warm-eod.lock"
export MARKETOPS_DAILY_LOCK_FILE="$lock_dir/daily-postclose.lock"
export MARKETOPS_POSTCLOSE_RECOVERY_LOCK_FILE="$lock_dir/postclose-recovery.lock"
export MARKETOPS_SRI_LOCK_FILE="$lock_dir/sri-refresh.lock"
export MARKETOPS_SRI_HOLDINGS_LOCK_FILE="$lock_dir/sri-holdings.lock"

# Outage catch-up runs against completed historical sessions. Keep the same
# provider-gap tolerance as the normal warm EOD path, but avoid waiting the full
# scheduled-job normalization window when the only remaining gap is a bounded
# provider no-bar set.
export MARKETOPS_WARM_EOD_NORMALIZATION_TIMEOUT_SECONDS="${MARKETOPS_OUTAGE_WARM_EOD_NORMALIZATION_TIMEOUT_SECONDS:-120}"

export MARKETOPS_WARM_EOD_ACKNOWLEDGE_WRITES=true
bash "$ROOT_DIR/scripts/marketops_scheduled_job.sh" marketops-warm-eod "Outage catch-up for completed trading day" "$timezone" \
  "$ROOT_DIR/scripts/marketops_warm_eod_refresh.sh" --date "$session_date" --write

export MARKETOPS_DAILY_ACKNOWLEDGE_WRITES=true
bash "$ROOT_DIR/scripts/marketops_scheduled_job.sh" marketops-daily-postclose "Outage catch-up for completed trading day" "$timezone" \
  "$ROOT_DIR/scripts/marketops_daily_postclose.sh" --date "$session_date" --write

bash "$ROOT_DIR/scripts/marketops_scheduled_job.sh" marketops-postclose-recovery "Outage catch-up completion guard" "$timezone" \
  "$ROOT_DIR/scripts/marketops_postclose_recovery.sh" --date "$session_date"

bash "$ROOT_DIR/scripts/marketops_scheduled_job.sh" marketops-sri-refresh "Outage catch-up SRI refresh" "$timezone" \
  "$ROOT_DIR/scripts/marketops_sri_refresh.sh" --date "$session_date"

# Holdings are a current issuer-composition snapshot, not session-date historical evidence.
# Refresh once as part of the outage closeout so the SRI view has current composition data.
bash "$ROOT_DIR/scripts/marketops_scheduled_job.sh" marketops-sri-holdings-refresh "Outage catch-up SRI holdings refresh" "$timezone" \
  "$ROOT_DIR/scripts/marketops_sri_holdings_refresh.sh"

bash "$ROOT_DIR/scripts/marketops_global_dashboard_projection.sh" "$session_date"

marketops_compose --profile subscriber-global-evidence run --rm --build subscriber-global-saf-benchmark-materializer \
  --execute --calculation-version saf_benchmark.v5 --max-observations 500 \
  --correlation-id "saf-benchmark-outage-reconcile-$session_date"

printf '%s outage reconciliation completed session=%s\n' "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" "$session_date"
