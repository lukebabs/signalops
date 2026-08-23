#!/usr/bin/env bash
# Calendar guard for scheduler dispatch. It is intentionally conservative:
# listed maintenance jobs may run on weekends; all other jobs are skipped.
#
# The holiday table is a static NYSE-style market-closure list for the current
# operational planning range. Update it annually from the official exchange
# calendar before enabling a new production year.

marketops_weekend_permitted_job() {
  case "${1:-}" in
    marketops-operations-monitor|signalops-storage-monitor|signalops-retention-governance|marketops-retention-governance|marketops-fmp-annual-financial)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

marketops_is_weekend() {
  local timezone="${1:-America/New_York}" weekday date_value="${2:-}"
  if [[ -n "$date_value" ]]; then
    weekday="$(TZ="$timezone" date -d "$date_value" '+%u')"
  else
    weekday="$(TZ="$timezone" date '+%u')"
  fi
  [[ "$weekday" == "6" || "$weekday" == "7" ]]
}

marketops_is_market_holiday() {
  local date_value="${1:-}"
  [[ "$date_value" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]] || return 1
  case "$date_value" in
    2026-01-01|2026-01-19|2026-02-16|2026-04-03|2026-05-25|2026-06-19|2026-07-03|2026-09-07|2026-11-26|2026-12-25|    2027-01-01|2027-01-18|2027-02-15|2027-03-26|2027-05-31|2027-06-18|2027-07-05|2027-09-06|2027-11-25|2027-12-24)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

marketops_today() {
  local timezone="${1:-America/New_York}"
  TZ="$timezone" date '+%F'
}

marketops_is_trading_day() {
  local timezone="${1:-America/New_York}" date_value="${2:-}"
  if [[ -z "$date_value" ]]; then
    date_value="$(marketops_today "$timezone")"
  fi
  ! marketops_is_weekend "$timezone" "$date_value" && ! marketops_is_market_holiday "$date_value"
}

marketops_non_trading_reason() {
  local timezone="${1:-America/New_York}" date_value="${2:-}"
  if [[ -z "$date_value" ]]; then
    date_value="$(marketops_today "$timezone")"
  fi
  if marketops_is_weekend "$timezone" "$date_value"; then
    printf '%s
' weekend
    return 0
  fi
  if marketops_is_market_holiday "$date_value"; then
    printf '%s
' market_holiday
    return 0
  fi
  printf '%s
' trading_day
}
