#!/usr/bin/env bash
set -euo pipefail
root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$root_dir/scripts/lib/marketops_trading_calendar.sh"

assert_trading() {
  local date_value="$1"
  if ! marketops_is_trading_day America/New_York "$date_value"; then
    printf 'expected trading day: %s, got %s
' "$date_value" "$(marketops_non_trading_reason America/New_York "$date_value")" >&2
    exit 1
  fi
}

assert_non_trading() {
  local date_value="$1" expected="$2" actual
  actual="$(marketops_non_trading_reason America/New_York "$date_value")"
  if [[ "$actual" != "$expected" ]]; then
    printf 'expected %s for %s, got %s
' "$expected" "$date_value" "$actual" >&2
    exit 1
  fi
  if marketops_is_trading_day America/New_York "$date_value"; then
    printf 'expected non-trading day: %s
' "$date_value" >&2
    exit 1
  fi
}

assert_trading 2026-08-21
assert_non_trading 2026-08-22 weekend
assert_non_trading 2026-09-07 market_holiday
assert_non_trading 2026-11-26 market_holiday
assert_non_trading 2027-07-05 market_holiday

marketops_weekend_permitted_job marketops-fmp-annual-financial
! marketops_weekend_permitted_job marketops-daily-postclose

printf '%s
' marketops_trading_calendar_tests_passed
