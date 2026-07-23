#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  . ./.env
  set +a
fi

api_key="${SIGNALOPS_MASSIVE_API_KEY:-}"
if [[ -z "$api_key" ]]; then
  cat >&2 <<'MSG'
Missing explicit Massive API key. Set this in .env or the shell environment:
  SIGNALOPS_MASSIVE_API_KEY

The generic API_KEY variable is intentionally ignored by this smoke to avoid accidental use of unrelated credentials. No provider requests were made.
MSG
  exit 2
fi

observation_date="${MARKETOPS_INGEST_SMOKE_DATE:-${SIGNALOPS_MASSIVE_OBSERVATION_DATE:-$(date -u -d 'yesterday' +%F)}}"
datasets="${MARKETOPS_INGEST_SMOKE_DATASETS:-equity}"
max_companies="${MARKETOPS_INGEST_SMOKE_MAX_COMPANIES:-1}"
options_limit="${MARKETOPS_INGEST_SMOKE_OPTIONS_LIMIT:-5}"
request_delay="${MARKETOPS_INGEST_SMOKE_REQUEST_DELAY:-250ms}"
max_retries="${MARKETOPS_INGEST_SMOKE_MAX_RETRIES:-0}"
retry_backoff="${MARKETOPS_INGEST_SMOKE_RETRY_BACKOFF:-1s}"
max_provider_requests="${MARKETOPS_INGEST_SMOKE_MAX_PROVIDER_REQUESTS:-1}"
max_events_built="${MARKETOPS_INGEST_SMOKE_MAX_EVENTS_BUILT:-1}"
max_events_published="${MARKETOPS_INGEST_SMOKE_MAX_EVENTS_PUBLISHED:-1}"

printf 'MarketOps calibration ingest smoke: date=%s datasets=%s max_companies=%s max_events_published=%s\n' \
  "$observation_date" "$datasets" "$max_companies" "$max_events_published"

if [[ "${MARKETOPS_INGEST_SKIP_PREFLIGHT:-false}" != "true" ]]; then
  MARKETOPS_MASSIVE_PREFLIGHT_DATE="$observation_date" scripts/marketops_massive_credential_preflight.sh
fi

docker compose -f compose.yaml -f compose.traefik.yaml up -d normalizer raw-worker signal-persister

docker compose -f compose.yaml -f compose.traefik.yaml --profile massive-pull run --rm \
  -e SIGNALOPS_MASSIVE_DRY_RUN=false \
  -e SIGNALOPS_MASSIVE_CONTINUE_ON_ERROR=false \
  -e SIGNALOPS_MASSIVE_OBSERVATION_DATE="$observation_date" \
  -e SIGNALOPS_MASSIVE_DATASETS="$datasets" \
  -e SIGNALOPS_MASSIVE_MAX_COMPANIES="$max_companies" \
  -e SIGNALOPS_MASSIVE_OPTIONS_LIMIT="$options_limit" \
  -e SIGNALOPS_MASSIVE_REQUEST_DELAY="$request_delay" \
  -e SIGNALOPS_MASSIVE_MAX_RETRIES="$max_retries" \
  -e SIGNALOPS_MASSIVE_RETRY_BACKOFF="$retry_backoff" \
  -e SIGNALOPS_MASSIVE_MAX_PROVIDER_REQUESTS="$max_provider_requests" \
  -e SIGNALOPS_MASSIVE_MAX_EVENTS_BUILT="$max_events_built" \
  -e SIGNALOPS_MASSIVE_MAX_EVENTS_PUBLISHED="$max_events_published" \
  massive-puller \
  --date "$observation_date" \
  --datasets "$datasets" \
  --max-companies "$max_companies" \
  --options-limit "$options_limit" \
  --request-delay "$request_delay" \
  --max-retries "$max_retries" \
  --retry-backoff "$retry_backoff" \
  --max-provider-requests "$max_provider_requests" \
  --max-events-built "$max_events_built" \
  --max-events-published "$max_events_published" \
  --dry-run=false \
  --continue-on-error=false

printf 'Published bounded Massive raw event(s). Wait a few seconds, then check /v1/marketops/backtest-coverage for normalized coverage.\n'
