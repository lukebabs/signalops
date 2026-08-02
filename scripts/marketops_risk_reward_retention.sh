#!/usr/bin/env bash
set -euo pipefail

retention_date="${1:-$(date -u -d '365 days ago' '+%Y-%m-%d')}"
[[ "$retention_date" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]] || { printf 'valid retention date is required\n' >&2; exit 2; }

docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U signalops -d signalops \
  -v retention_date="$retention_date" -c "
DELETE FROM marketops_risk_reward_snapshots
WHERE tenant_id='tenant-local' AND session_date < DATE '$retention_date';

DELETE FROM algorithm_results
WHERE tenant_id='tenant-local'
  AND algorithm_id='signalops.algorithms.risk_reward_temporal_v1'
  AND COALESCE(result_payload->>'observation_time','') <> ''
  AND (result_payload->>'observation_time')::timestamptz::date < DATE '$retention_date';"
