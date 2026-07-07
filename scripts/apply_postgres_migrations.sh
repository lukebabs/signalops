#!/bin/sh
set -eu

DB_URL="${SIGNALOPS_DATABASE_URL:-postgres://signalops:signalops@postgres:5432/signalops?sslmode=disable}"
MIGRATION_DIR="${SIGNALOPS_MIGRATION_DIR:-/workspace/migrations}"

psql "$DB_URL" -v ON_ERROR_STOP=1 -c "CREATE TABLE IF NOT EXISTS schema_migrations (version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())"

for migration in "$MIGRATION_DIR"/*.up.sql; do
  version="$(basename "$migration" .up.sql)"
  applied="$(psql "$DB_URL" -Atc "SELECT 1 FROM schema_migrations WHERE version = '$version'" 2>/dev/null || true)"
  if [ "$applied" = "1" ]; then
    echo "skip $version"
    continue
  fi
  echo "apply $version"
  psql "$DB_URL" -v ON_ERROR_STOP=1 -f "$migration"
  psql "$DB_URL" -v ON_ERROR_STOP=1 -c "INSERT INTO schema_migrations(version) VALUES ('$version') ON CONFLICT DO NOTHING"
done
