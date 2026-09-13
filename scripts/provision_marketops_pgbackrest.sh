#!/usr/bin/env bash
set -euo pipefail

[[ "${EUID}" -eq 0 ]] || { echo "Run this provisioning command as root." >&2; exit 2; }
root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source_env=/etc/signalops/pgbackrest-source.env
config_path=/etc/signalops/marketops-pgbackrest/pgbackrest.conf
boundary_env=/etc/signalops/marketops-boundary.env
[[ -r "$source_env" && -r "$boundary_env" ]] || { echo "Existing root-owned pgBackRest source/configuration is required." >&2; exit 3; }
grep -qx "SIGNALOPS_MARKETOPS_PGBACKREST_CONFIG_PATH=$config_path" "$source_env" || printf "SIGNALOPS_MARKETOPS_PGBACKREST_CONFIG_PATH=%s\n" "$config_path" >> "$source_env"
set -a
. "$source_env"
. "$boundary_env"
set +a
"$root_dir/scripts/refresh_signalops_pgbackrest_credentials.sh"
compose=(docker compose -p signalops --env-file "$boundary_env" -f "$root_dir/compose.yaml" -f "$root_dir/compose.marketops-boundary.yaml" -f "$root_dir/compose.marketops-pgbackrest.yaml")
SIGNALOPS_PGBACKREST_CONFIG_PATH="$config_path" "${compose[@]}" up -d --build --wait marketops-postgres marketops-timescaledb
for target in "marketops-postgres marketops-primary marketops" "marketops-timescaledb marketops-temporal marketops_temporal"; do
  read -r service stanza database <<<"$target"
  SIGNALOPS_PGBACKREST_CONFIG_PATH="$config_path" "${compose[@]}" exec -T --user postgres "$service" pgbackrest --stanza="$stanza" stanza-create
  SIGNALOPS_PGBACKREST_CONFIG_PATH="$config_path" "${compose[@]}" exec -T "$service" psql -U signalops -d "$database" -v ON_ERROR_STOP=1 -c "ALTER SYSTEM SET archive_mode = 'on';" -c "ALTER SYSTEM SET archive_command = 'pgbackrest --stanza=$stanza archive-push %p';" -c "ALTER SYSTEM SET archive_timeout = '15min';"
done
SIGNALOPS_PGBACKREST_CONFIG_PATH="$config_path" "${compose[@]}" restart marketops-postgres marketops-timescaledb
SIGNALOPS_PGBACKREST_CONFIG_PATH="$config_path" "${compose[@]}" up -d --wait marketops-postgres marketops-timescaledb
SIGNALOPS_PGBACKREST_CONFIG_PATH="$config_path" "$root_dir/scripts/marketops_pgbackrest_backup.sh" check
SIGNALOPS_PGBACKREST_CONFIG_PATH="$config_path" "$root_dir/scripts/marketops_pgbackrest_backup.sh" full
SIGNALOPS_PGBACKREST_CONFIG_PATH="$config_path" "${compose[@]}" exec -T --user postgres marketops-postgres pgbackrest --stanza=marketops-primary info --output=json
SIGNALOPS_PGBACKREST_CONFIG_PATH="$config_path" "${compose[@]}" exec -T --user postgres marketops-timescaledb pgbackrest --stanza=marketops-temporal info --output=json
"$root_dir/scripts/install_marketops_pgbackrest_system_timer.sh"
echo "Dedicated MarketOps pgBackRest provisioning completed. The shared backup timer remains disabled."
