#!/usr/bin/env bash
set -euo pipefail

worker_id="${1:-}"
runtime_env_file="${SIGNALOPS_K8S_RUNTIME_ENV_FILE:-/vault/secrets/connect-worker-runtime.env}"

fail() {
  echo "signalops_k8s_connect_worker_failed: $*" >&2
  exit 2
}

[[ -n "$worker_id" ]] || fail "worker id is required"
if [[ -f "$runtime_env_file" ]]; then
  # shellcheck disable=SC1090
  source "$runtime_env_file"
fi

export SIGNALOPS_ENV="${SIGNALOPS_ENV:-kubernetes-staging}"
export SIGNALOPS_CONNECT_SHADOW_MODE="${SIGNALOPS_CONNECT_SHADOW_MODE:-true}"

case "$worker_id" in
  connect-persister)
    exec signalops-cyberops-connect-persister
    ;;
  connect-outbox)
    exec signalops-cyberops-connect-outbox
    ;;
  *)
    fail "unsupported Signal-Connect worker id: $worker_id"
    ;;
esac
