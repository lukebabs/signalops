#!/usr/bin/env bash
set -Eeuo pipefail

fail() {
  printf '%s\n' "$*" >&2
  exit 2
}

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root_dir"

required_files=(
  compose.yaml
  compose.marketops-boundary.yaml
  compose.marketops-read-cutover.yaml
  compose.marketops-writer-cutover.yaml
  compose.marketops-pgbackrest.yaml
  compose.traefik.yaml
)

compose_file_value="${COMPOSE_FILE:-}"
if [[ -z "$compose_file_value" && -r .env ]]; then
  compose_file_value="$(grep -E '^COMPOSE_FILE=' .env | tail -n 1 | cut -d= -f2- || true)"
fi
[[ -n "$compose_file_value" ]] || fail "COMPOSE_FILE is not set; plain docker compose would use compose.yaml only"

for compose_file in "${required_files[@]}"; do
  [[ ":$compose_file_value:" == *":$compose_file:"* ]] || fail "COMPOSE_FILE is missing $compose_file"
  [[ -r "$compose_file" ]] || fail "required compose file is not readable: $compose_file"
done

for key in \
  SIGNALOPS_MARKETOPS_POSTGRES_PASSWORD \
  SIGNALOPS_MARKETOPS_TEMPORAL_PASSWORD \
  SIGNALOPS_SUBSCRIBER_GATEWAY_PASSWORD
do
  if [[ -z "${!key:-}" && -r .env ]]; then
    value="$(grep -E "^${key}=" .env | tail -n 1 | cut -d= -f2- || true)"
    [[ -n "$value" ]] || fail "$key is missing or empty; plain docker compose cannot render MarketOps production topology"
  elif [[ -z "${!key:-}" ]]; then
    fail "$key is missing and .env is not readable"
  fi
done

docker compose config --quiet

services="$(docker compose config --services)"
for service in gateway web marketops-postgres marketops-timescaledb normalizer signal-persister marketops-signal-assurance-registrar marketops-signal-assurance-outbox; do
  grep -qx "$service" <<< "$services" || fail "plain docker compose graph is missing $service"
done

rendered="$(mktemp)"
trap 'rm -f "$rendered"' EXIT
docker compose config > "$rendered"

grep -q 'traefik.enable: "true"' "$rendered" || fail "web service is missing Traefik labels in plain compose render"
grep -q 'SIGNALOPS_MARKETOPS_DATA_BOUNDARY_REQUIRED: "true"' "$rendered" || fail "plain compose render is missing MarketOps boundary-required flag"
grep -q 'SIGNALOPS_MARKETOPS_DATABASE_URL:' "$rendered" || fail "plain compose render is missing dedicated MarketOps primary URL"
grep -q 'SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL:' "$rendered" || fail "plain compose render is missing dedicated MarketOps temporal URL"

printf '%s\n' signalops_compose_authority_verified
