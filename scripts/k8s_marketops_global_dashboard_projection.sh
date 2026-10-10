#!/usr/bin/env bash
set -euo pipefail
session_date="${1:-}"
[[ "$session_date" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]] || { echo "usage: $0 YYYY-MM-DD" >&2; exit 2; }
: "${SIGNALOPS_MARKETOPS_DATABASE_URL:?dedicated MarketOps database required}"
export SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL="$SIGNALOPS_MARKETOPS_DATABASE_URL"
correlation="postclose-global-dashboard-${session_date}"
project_kind() {
  local kind="$1" algorithm="${2:-}" output run_id selected
  local -a manifest=(--execute --evidence-kinds "$kind" --session-date "$session_date" --newest-first --limit 50000 --correlation-id "$correlation")
  local -a materializer=(--execute --evidence-kinds "$kind" --limit 50000 --correlation-id "$correlation")
  [[ -z "$algorithm" ]] || { manifest+=(--algorithm-id "$algorithm"); materializer+=(--algorithm-id "$algorithm"); }
  output="$(signalops-subscriber-global-marketops-parity-manifest "${manifest[@]}")"
  printf '%s\n' "$output"
  run_id="$(printf '%s\n' "$output" | sed -n 's/^parity_run_id=\([^ ]*\).*/\1/p')"
  selected="$(printf '%s\n' "$output" | sed -n 's/.* selected=\([0-9]*\).*/\1/p')"
  [[ -n "$run_id" && -n "$selected" ]] || { echo "unparseable projection manifest for $kind" >&2; exit 3; }
  [[ "$selected" == 0 ]] || signalops-subscriber-global-marketops-evidence-materializer --parity-run-id "$run_id" "${materializer[@]}"
}
project_kind options_snapshot
project_kind risk_reward
project_kind market_state
project_kind outcome
project_kind valuation signalops.algorithms.eroc_v6
project_kind valuation signalops.algorithms.valuation_composite_v3
project_kind valuation signalops.algorithms.distressed_opportunity_scoring_v3
project_kind eeom
risk_global="$(psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -Atc "SELECT count(*) FROM subscriber_gateway_global_risk_reward_snapshots WHERE session_date=DATE '$session_date'")"
[[ "$risk_global" =~ ^[0-9]+$ && "$risk_global" -gt 0 ]] || { echo "global risk/reward projection incomplete: session=$session_date rows=$risk_global" >&2; exit 4; }
echo "k8s_global_dashboard_projection_verified session=$session_date risk_reward=$risk_global"
