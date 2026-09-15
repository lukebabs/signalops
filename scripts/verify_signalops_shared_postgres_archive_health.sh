#!/usr/bin/env bash
set -euo pipefail

fail() {
  echo "signalops_shared_postgres_archive_health_failed: $*" >&2
  exit 1
}

container="${SIGNALOPS_SHARED_POSTGRES_CONTAINER:-signalops-postgres-1}"
database="${SIGNALOPS_SHARED_POSTGRES_DATABASE:-signalops}"
user="${SIGNALOPS_SHARED_POSTGRES_USER:-signalops}"

command -v docker >/dev/null 2>&1 || fail "docker is required"

container_id="$(docker ps -q -f "name=^/${container}$" || true)"
[[ -n "$container_id" ]] || fail "container is not running: $container"

image="$(docker inspect --format '{{.Config.Image}}' "$container_id")"
archive_mode="$(docker exec "$container" psql -U "$user" -d "$database" -Atc "SHOW archive_mode;" 2>/dev/null || true)"
archive_command="$(docker exec "$container" psql -U "$user" -d "$database" -Atc "SHOW archive_command;" 2>/dev/null || true)"
wal_size="$(docker exec "$container" sh -c "du -sh /var/lib/postgresql/data/pg_wal 2>/dev/null | awk '{print \$1}'" 2>/dev/null || true)"
wal_entries="$(docker exec "$container" sh -c "find /var/lib/postgresql/data/pg_wal -maxdepth 1 -type f 2>/dev/null | wc -l" 2>/dev/null || true)"

pgbackrest_available=false
if docker exec --user postgres "$container" pgbackrest version >/dev/null 2>&1; then
  pgbackrest_available=true
fi

status="ok"
reason="archive_disabled_or_pgbackrest_available"
if [[ "$archive_mode" == "on" && "$archive_command" == *"pgbackrest"* && "$pgbackrest_available" != "true" ]]; then
  status="blocked"
  reason="archive_command_requires_pgbackrest_but_live_container_lacks_pgbackrest"
fi

cat <<EOF
signalops_shared_postgres_archive_health_report
status=${status}
container=${container}
image=${image}
archive_mode=${archive_mode:-unknown}
archive_command_kind=$(if [[ "$archive_command" == *"pgbackrest"* ]]; then echo pgbackrest; elif [[ -n "$archive_command" ]]; then echo other; else echo empty; fi)
pgbackrest_available=${pgbackrest_available}
wal_size=${wal_size:-unknown}
wal_entries=${wal_entries:-unknown}
reason=${reason}
EOF

[[ "$status" == "ok" ]] || exit 1
