#!/usr/bin/env bash
set -euo pipefail

[[ "${EUID}" -eq 0 ]] || {
  printf 'Run this command as root.\n' >&2
  exit 2
}

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
runtime_env="${1:-}"
run_as_user="${2:-${SUDO_USER:-adminalien}}"
enable_flags=("${@:3}")
template="$root_dir/deploy/systemd/signalops-marketops-boundary-schedule@.service.in"
unit=/etc/systemd/system/signalops-marketops-boundary-schedule@.service
sri_timers=(
  signalops-marketops-boundary-sri-refresh.timer
  signalops-marketops-boundary-sri-holdings-refresh.timer
)
market_data_timers=(
  signalops-marketops-boundary-intraday.timer
  signalops-marketops-boundary-daily-postclose.timer
  signalops-marketops-boundary-postclose-recovery.timer
)
warm_eod_timers=(
  signalops-marketops-boundary-warm-eod.timer
)
annual_fmp_timers=(signalops-marketops-boundary-fmp-annual-financial.timer)
enable_sri=false
enable_market_data=false
enable_warm_eod=false
enable_annual_fmp=false

[[ -n "$runtime_env" && -r "$runtime_env" ]] || {
  printf 'Provide a readable protected production Compose environment file as argument 1.\n' >&2
  exit 2
}
runtime_env="$(cd "$(dirname "$runtime_env")" && pwd)/$(basename "$runtime_env")"
[[ "$run_as_user" =~ ^[a-z_][a-z0-9_-]*$ ]] || {
  printf 'Invalid service user: %s\n' "$run_as_user" >&2
  exit 2
}
id "$run_as_user" >/dev/null
[[ -r /etc/signalops/marketops-cutover.env ]] || {
  printf 'Dedicated MarketOps cutover environment is not readable. Render it first.\n' >&2
  exit 3
}
[[ -r "$template" ]] || {
  printf 'Missing unit template: %s\n' "$template" >&2
  exit 3
}
for flag in "${enable_flags[@]}"; do
  case "$flag" in
    --enable-sri) enable_sri=true ;;
    --enable-market-data) enable_market_data=true ;;
    --enable-warm-eod) enable_warm_eod=true ;;
    --enable-fmp-annual) enable_annual_fmp=true ;;
    "") ;;
    *)
      printf "Optional flags must be --enable-sri, --enable-market-data, --enable-warm-eod, and/or --enable-fmp-annual.\n" >&2
      exit 2
      ;;
  esac
done
for timer in "${sri_timers[@]}" "${market_data_timers[@]}" "${warm_eod_timers[@]}" "${annual_fmp_timers[@]}"; do
  [[ -r "$root_dir/deploy/systemd/$timer" ]] || {
    printf 'Missing dedicated scheduler timer template: %s\n' "$timer" >&2
    exit 3
  }
done

temporary="$(mktemp /etc/systemd/system/.signalops-marketops-boundary.XXXXXX)"
trap 'rm -f "$temporary"' EXIT
sed \
  -e "s|@WORKDIR@|$root_dir|g" \
  -e "s|@RUNTIME_ENV@|$runtime_env|g" \
  -e "s|@RUN_AS_USER@|$run_as_user|g" \
  "$template" > "$temporary"
install -m 0644 -o root -g root "$temporary" "$unit"
for timer in "${sri_timers[@]}" "${market_data_timers[@]}" "${warm_eod_timers[@]}" "${annual_fmp_timers[@]}"; do
  install -m 0644 -o root -g root "$root_dir/deploy/systemd/$timer" "/etc/systemd/system/$timer"
done
systemctl daemon-reload

printf 'Installed dedicated scheduler dispatcher: %s\n' "$unit"
if "$enable_sri"; then
  systemctl enable --now "${sri_timers[@]}"
  printf "Enabled controlled SRI refresh timers: weekdays 20:07 and 20:20 America/New_York.\n"
fi
if "$enable_market_data"; then
  systemctl enable --now "${market_data_timers[@]}"
  printf "Enabled controlled MarketOps timers: intraday every 15 minutes 09:30-20:00, post-close 18:01:55, and bounded recovery from 18:30 America/New_York.\n"
fi
if "$enable_warm_eod"; then
  systemctl enable --now "${warm_eod_timers[@]}"
  printf "Enabled centrally governed warm EOD acquisition: weekdays 18:00 America/New_York.\n"
fi
if "$enable_annual_fmp"; then systemctl enable --now "${annual_fmp_timers[@]}"; printf "Enabled governed FMP annual financial capture: Saturdays 02:30 America/New_York.\n"; fi
if ! "$enable_sri" && ! "$enable_market_data" && ! "$enable_warm_eod" && ! "$enable_annual_fmp"; then
  printf "No timer was enabled. Use --enable-sri, --enable-market-data, --enable-warm-eod, and/or --enable-fmp-annual only with recorded approval.\n"
fi
