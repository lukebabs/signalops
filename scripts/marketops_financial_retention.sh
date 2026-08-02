#!/usr/bin/env bash
set -euo pipefail

retention_date="${1:-$(date -u -d '7 years ago' '+%Y-%m-%d')}"
[[ "$retention_date" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]] || { printf 'valid retention date is required\n' >&2; exit 2; }

docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U signalops -d signalops -v retention_date="$retention_date" -c "
DELETE FROM marketops_financial_snapshots WHERE tenant_id='tenant-local' AND evaluation_date < DATE '$retention_date';
DELETE FROM marketops_financial_statements s WHERE tenant_id='tenant-local' AND s.fiscal_period_end < DATE '$retention_date' AND NOT EXISTS (SELECT 1 FROM marketops_financial_snapshots f WHERE s.statement_id = ANY(f.statement_ids));"
