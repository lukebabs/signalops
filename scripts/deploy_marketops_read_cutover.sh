#!/usr/bin/env bash
set -euo pipefail

[[ "${EUID}" -eq 0 ]] || {
  printf 'Run this command as root.\n' >&2
  exit 2
}

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=lib/marketops_boundary_env.sh
source "$root_dir/scripts/lib/marketops_boundary_env.sh"
boundary_env=/etc/signalops/marketops-boundary.env
cutover_env=/etc/signalops/marketops-cutover.env
runtime_env="${1:-${SIGNALOPS_PRODUCTION_ENV_FILE:-}}"
[[ -r "$boundary_env" ]] || {
  printf 'Protected MarketOps boundary secret is not readable: %s\n' "$boundary_env" >&2
  exit 3
}
[[ -n "$runtime_env" ]] || {
  printf 'Provide the protected production Compose environment file as argument 1.\n' >&2
  exit 2
}
[[ -r "$runtime_env" ]] || {
  printf 'Production Compose environment file is not readable: %s\n' "$runtime_env" >&2
  exit 3
}

# Compose interpolates every service definition, including the dedicated-store
# definitions not selected by this gateway-only invocation. Parse only the two
# required credentials as literal data; do not execute the secret file.
load_marketops_boundary_env "$boundary_env"

"$root_dir/scripts/render_marketops_cutover_env.sh" "$boundary_env" "$cutover_env"
# The subscriber login is distinct from the primary application login. It is
# provisioned with only the logical gateway role, using a secret generated in
# the root-owned cutover environment for this deployment.
subscriber_gateway_password="$(grep -E "^SIGNALOPS_SUBSCRIBER_GATEWAY_PASSWORD=" "$cutover_env" | cut -d= -f2-)"
[[ "$subscriber_gateway_password" =~ ^[A-Fa-f0-9]{64}$ ]] || {
  printf "%s\n" "marketops_read_cutover_subscriber_gateway_secret_invalid" >&2
  exit 4
}


compose=(docker compose --env-file "$runtime_env" --env-file "$cutover_env" -p signalops -f "$root_dir/compose.yaml" -f "$root_dir/compose.marketops-boundary.yaml" -f "$root_dir/compose.marketops-read-cutover.yaml")

"${compose[@]}" exec -T marketops-postgres psql -v ON_ERROR_STOP=1 -U signalops -d marketops \
  -c "DO \$\$ BEGIN IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'signalops_subscriber_gateway_runtime') THEN CREATE ROLE signalops_subscriber_gateway_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS; END IF; END \$\$;" \
  -c "ALTER ROLE signalops_subscriber_gateway_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS PASSWORD '${subscriber_gateway_password}';" \
  -c "GRANT signalops_subscriber_gateway TO signalops_subscriber_gateway_runtime;"
"${compose[@]}" up -d --build --no-deps gateway
gateway_id="$("${compose[@]}" ps -q gateway)"
[[ -n "$gateway_id" ]] || { printf "%s\n" "marketops_read_cutover_gateway_missing" >&2; exit 4; }
gateway_env_names="$(docker inspect --format "{{range .Config.Env}}{{println .}}{{end}}" "$gateway_id" | cut -d= -f1)"
grep -qx "SIGNALOPS_MARKETOPS_DATABASE_URL" <<< "$gateway_env_names" || { printf "%s\n" "marketops_read_cutover_primary_env_missing" >&2; exit 4; }
grep -qx "SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL" <<< "$gateway_env_names" || { printf "%s\n" "marketops_read_cutover_temporal_env_missing" >&2; exit 4; }
gateway_subscriber_url="$(docker inspect --format '{{range .Config.Env}}{{println .}}{{end}}' "$gateway_id" | grep -E "^SIGNALOPS_SUBSCRIBER_GATEWAY_DATABASE_URL=" | cut -d= -f2-)"
[[ "$gateway_subscriber_url" == *'@marketops-postgres:5432/marketops?sslmode=disable' ]] || { printf "%s\n" "marketops_read_cutover_subscriber_gateway_boundary_missing" >&2; exit 4; }
docker logs "$gateway_id" 2>&1 | grep -Fq "MarketOps gateway reads are routed to the dedicated data boundary" || { printf "%s\n" "marketops_read_cutover_startup_evidence_missing" >&2; exit 4; }
printf "%s\n" "marketops_read_cutover_gateway_verified"

# The installed deployment agent invokes this root-only launcher. Run the
# browser contract as the repository owner so its Playwright installation and
# protected local QA configuration remain outside the root service account.
qa_operator="${SIGNALOPS_DEPLOYMENT_QA_USER:-$(stat -c %U "$root_dir")}"
[[ "$qa_operator" =~ ^[a-z_][a-z0-9_-]*$ ]] || {
  printf "%s\n" "subscriber_pilot_ui_smoke_operator_invalid" >&2
  exit 5
}
qa_home="$(getent passwd "$qa_operator" | cut -d: -f6)"
[[ -n "$qa_home" && -d "$qa_home" ]] || {
  printf "%s\n" "subscriber_pilot_ui_smoke_home_missing" >&2
  exit 5
}
runuser -u "$qa_operator" -- env "HOME=$qa_home" "PLAYWRIGHT_BROWSERS_PATH=$qa_home/.cache/ms-playwright" \
  "$root_dir/scripts/run_subscriber_pilot_ui_smoke.sh"
printf "%s\n" "subscriber_pilot_ui_smoke_verified"
