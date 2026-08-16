#!/usr/bin/env bash
# Calendar guard for scheduler dispatch. It is intentionally conservative:
# listed maintenance jobs may run on weekends; all other jobs are skipped.

marketops_weekend_permitted_job() {
  case "${1:-}" in
    marketops-operations-monitor|signalops-storage-monitor|signalops-retention-governance|marketops-fmp-annual-financial)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

marketops_is_weekend() {
  local timezone="${1:-America/New_York}" weekday
  weekday="$(TZ="$timezone" date '+%u')"
  [[ "$weekday" == "6" || "$weekday" == "7" ]]
}
