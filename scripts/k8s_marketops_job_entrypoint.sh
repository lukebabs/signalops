#!/usr/bin/env bash
set -euo pipefail

job_id="${1:-}"
runtime_env_file="${SIGNALOPS_K8S_RUNTIME_ENV_FILE:-/vault/secrets/marketops-worker-runtime.env}"

fail() {
  echo "signalops_k8s_marketops_job_failed: $*" >&2
  exit 2
}

[[ -n "$job_id" ]] || fail "job id is required"
if [[ -f "$runtime_env_file" ]]; then
  # shellcheck disable=SC1090
  source "$runtime_env_file"
fi

export SIGNALOPS_ENV="${SIGNALOPS_ENV:-kubernetes-staging}"
export SIGNALOPS_MARKETOPS_DATA_BOUNDARY_REQUIRED="${SIGNALOPS_MARKETOPS_DATA_BOUNDARY_REQUIRED:-true}"

case "$job_id" in
  marketops-intraday)
    exec signalops-marketops-intraday-monitor \
      --tenant-id "${MARKETOPS_INTRADAY_TENANT_ID:-tenant-local}" \
      --universe-group "${MARKETOPS_INTRADAY_UNIVERSE_GROUP:-all_active}" \
      --max-symbols "${MARKETOPS_INTRADAY_MAX_SYMBOLS:-200}"
    ;;
  marketops-sri-refresh)
    exec signalops-marketops-sri-runner \
      --tenant-id "${SIGNALOPS_SRI_OUTPUT_TENANT_ID:-platform-global}" \
      --input-tenant-id "${SIGNALOPS_SRI_INPUT_TENANT_ID:-tenant-local}" \
      --as-of "${MARKETOPS_SESSION_DATE:-$(date -u +%F)}"
    ;;
  marketops-sri-holdings-refresh)
    exec signalops-marketops-sri-holdings-runner \
      --tenant-id "${SIGNALOPS_SRI_OUTPUT_TENANT_ID:-platform-global}"
    ;;
  marketops-fmp-annual-financial)
    exec signalops-subscriber-global-annual-financial-task-worker \
      --execute \
      --max-assets "${MARKETOPS_FMP_ANNUAL_MAX_ASSETS:-1000}" \
      --session-date "${MARKETOPS_SESSION_DATE:-}"
    ;;
  marketops-saf-benchmark)
    exec signalops-subscriber-global-saf-benchmark-materializer \
      --execute \
      --max-observations "${MARKETOPS_SAF_BENCHMARK_MAX_OBSERVATIONS:-500}" \
      --calculation-version "${MARKETOPS_SAF_BENCHMARK_CALCULATION_VERSION:-saf_benchmark.k8s_staging}" \
      --correlation-id "${MARKETOPS_SAF_BENCHMARK_CORRELATION_ID:-k8s-staging-cronjob}"
    ;;
  *)
    fail "unsupported MarketOps Kubernetes job id: $job_id"
    ;;
esac
