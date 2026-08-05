#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"
if [[ -f .env ]]; then set -a; . ./.env; set +a; fi
session_date="${1:-$(TZ=America/New_York date -d 'yesterday' +%F)}"
symbols="$(docker compose exec -T postgres psql -U signalops -d signalops -At -c "SELECT symbol FROM marketops_task_items WHERE tenant_id='tenant-local' AND session_date='$session_date' AND task_type='tactical_posture' AND status='retry_scheduled' AND next_attempt_at <= now() ORDER BY symbol" | paste -sd, -)"
[[ -n "$symbols" ]] || { echo "no due tactical retries for $session_date"; exit 0; }
docker compose --profile marketops-daily run --rm marketops-tactical-valuation-runner --tenant-id tenant-local --universe-group all_active --session-date "$session_date" --symbols "$symbols" --max-retries 2
