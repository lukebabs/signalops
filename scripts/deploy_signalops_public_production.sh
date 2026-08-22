#!/usr/bin/env bash
set -Eeuo pipefail

usage() {
  cat <<'USAGE'
Usage:
  scripts/deploy_signalops_public_production.sh [--dry-run] [--no-build] [--web-only|--gateway-only]

Safely rebuild/restart the public SignalOps production entrypoints with every
required overlay:

  - compose.yaml
  - compose.marketops-boundary.yaml
  - compose.marketops-read-cutover.yaml
  - compose.traefik.yaml

Defaults:
  services: gateway web
  env file: .env
  cutover env: /etc/signalops/marketops-cutover.env

Environment overrides:
  SIGNALOPS_PRODUCTION_ENV_FILE
  SIGNALOPS_MARKETOPS_CUTOVER_ENV
  SIGNALOPS_COMPOSE_PROJECT
USAGE
}

fail() {
  printf '%s\n' "$*" >&2
  exit 2
}

quote_command() {
  local item
  for item in "$@"; do
    printf '%q ' "$item"
  done
  printf '\n'
}

env_value() {
  local file="$1"
  local key="$2"
  local line value
  line="$(grep -E "^${key}=" "$file" | tail -n 1 || true)"
  [[ -n "$line" ]] || return 1
  value="${line#*=}"
  value="${value%\"}"
  value="${value#\"}"
  value="${value%\'}"
  value="${value#\'}"
  printf '%s\n' "$value"
}

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
runtime_env="${SIGNALOPS_PRODUCTION_ENV_FILE:-$root_dir/.env}"
cutover_env="${SIGNALOPS_MARKETOPS_CUTOVER_ENV:-/etc/signalops/marketops-cutover.env}"
project="${SIGNALOPS_COMPOSE_PROJECT:-signalops}"
dry_run=false
build=true
services=(gateway web)

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run)
      dry_run=true
      ;;
    --no-build)
      build=false
      ;;
    --web-only)
      services=(web)
      ;;
    --gateway-only)
      services=(gateway)
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      fail "Unknown argument: $1"
      ;;
  esac
  shift
done

[[ -r "$runtime_env" ]] || fail "Production env file is not readable: $runtime_env"
for compose_file in \
  "$root_dir/compose.yaml" \
  "$root_dir/compose.marketops-boundary.yaml" \
  "$root_dir/compose.marketops-read-cutover.yaml" \
  "$root_dir/compose.traefik.yaml"
do
  [[ -r "$compose_file" ]] || fail "Required Compose file is not readable: $compose_file"
done

public_host="$(env_value "$runtime_env" SIGNALOPS_PUBLIC_HOST || true)"
traefik_network="$(env_value "$runtime_env" TRAEFIK_NETWORK || true)"
[[ -n "$public_host" ]] || fail "SIGNALOPS_PUBLIC_HOST must be set in $runtime_env"
[[ -n "$traefik_network" ]] || fail "TRAEFIK_NETWORK must be set in $runtime_env"

compose=(
  docker compose
  --env-file "$runtime_env"
  --env-file "$cutover_env"
  -p "$project"
  -f "$root_dir/compose.yaml"
  -f "$root_dir/compose.marketops-boundary.yaml"
  -f "$root_dir/compose.marketops-read-cutover.yaml"
  -f "$root_dir/compose.traefik.yaml"
)

up=("${compose[@]}" up -d)
if [[ "$build" == true ]]; then
  up+=(--build)
fi
up+=(--no-deps "${services[@]}")

if [[ "$dry_run" == true ]]; then
  printf 'Safe production deploy command:\n'
  quote_command "${up[@]}"
  printf 'Post-deploy verification:\n'
  printf '  %s\n' "docker inspect signalops-web-1 --format '{{ index .Config.Labels \"traefik.enable\" }} {{ index .Config.Labels \"traefik.http.routers.signalops.rule\" }}'"
  printf '  %s\n' "curl -fsS http://127.0.0.1:15173/readyz"
  printf '  %s\n' "curl -fsS http://127.0.0.1:18000/readyz"
  printf '  %s\n' "curl -fsS https://$public_host/readyz"
  exit 0
fi

[[ -r "$cutover_env" ]] || fail "MarketOps cutover env is not readable: $cutover_env. Run as root or use the deployment agent."

"${compose[@]}" config --quiet
"${up[@]}"

web_id="$(docker compose -p "$project" ps -q web)"
gateway_id="$(docker compose -p "$project" ps -q gateway)"
[[ -n "$web_id" ]] || fail "signalops web container is not running after deploy"
[[ -n "$gateway_id" ]] || fail "signalops gateway container is not running after deploy"

traefik_enabled="$(docker inspect --format '{{ index .Config.Labels "traefik.enable" }}' "$web_id")"
traefik_rule="$(docker inspect --format '{{ index .Config.Labels "traefik.http.routers.signalops.rule" }}' "$web_id")"
[[ "$traefik_enabled" == "true" ]] || fail "web container is missing traefik.enable=true"
[[ "$traefik_rule" == *"$public_host"* ]] || fail "web container Traefik host rule does not include $public_host"

gateway_env_names="$(docker inspect --format '{{range .Config.Env}}{{println .}}{{end}}' "$gateway_id" | cut -d= -f1)"
grep -qx "SIGNALOPS_MARKETOPS_DATABASE_URL" <<< "$gateway_env_names" || fail "gateway is missing SIGNALOPS_MARKETOPS_DATABASE_URL"
grep -qx "SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL" <<< "$gateway_env_names" || fail "gateway is missing SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL"
if env_value "$runtime_env" STRIPE_WEBHOOK_SECRET >/dev/null 2>&1; then
  grep -qx "STRIPE_WEBHOOK_SECRET" <<< "$gateway_env_names" || fail "gateway is missing STRIPE_WEBHOOK_SECRET"
fi

curl -fsS http://127.0.0.1:15173/readyz >/dev/null
curl -fsS http://127.0.0.1:18000/readyz >/dev/null
curl -fsS "https://$public_host/readyz" >/dev/null

printf '%s\n' "signalops_public_production_deploy_verified"
