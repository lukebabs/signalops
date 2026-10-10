#!/usr/bin/env bash

marketops_runtime_lock_file() {
  local lock_name="$1"
  local lock_dir="${MARKETOPS_LOCK_DIR:-/run/signalops/marketops-locks}"
  if ! mkdir -p "$lock_dir" 2>/dev/null; then
    lock_dir="${TMPDIR:-/tmp}/signalops-marketops-locks-${USER:-operator}"
    mkdir -p "$lock_dir"
  fi
  printf '%s/%s.lock\n' "$lock_dir" "$lock_name"
}
