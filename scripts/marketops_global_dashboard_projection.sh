#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/marketops_schedule_database.sh"

session_date="${1:-}"
[[ "$session_date" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]] || {
  printf 'Usage: %s YYYY-MM-DD\n' "$0" >&2
  exit 2
}

project_kind() {
  local kind="$1" algorithm_id="${2:-}" output run_id selected manifest_args materializer_args
  manifest_args=(--execute --evidence-kinds "$kind" --session-date "$session_date" --newest-first --limit 1000 --correlation-id "postclose-global-dashboard-${session_date}")
  materializer_args=(--execute --evidence-kinds "$kind" --limit 1000 --correlation-id "postclose-global-dashboard-${session_date}")
  if [[ -n "$algorithm_id" ]]; then
    manifest_args+=(--algorithm-id "$algorithm_id")
    materializer_args+=(--algorithm-id "$algorithm_id")
  fi
  output="$(marketops_compose --profile subscriber-global-evidence run --rm subscriber-global-marketops-parity-manifest "${manifest_args[@]}")"
  printf '%s\n' "$output"
  run_id="$(printf '%s\n' "$output" | grep '^parity_run_id=' | cut -d' ' -f1 | cut -d= -f2)"
  selected="$(printf '%s\n' "$output" | tr ' ' '\n' | grep '^selected=' | cut -d= -f2)"
  [[ -n "$run_id" && -n "$selected" ]] || {
    printf 'unparseable global projection manifest for %s\n' "$kind" >&2
    exit 3
  }
  [[ "$selected" == 0 ]] && return
  marketops_compose --profile subscriber-global-evidence run --rm subscriber-global-marketops-evidence-materializer --parity-run-id "$run_id" "${materializer_args[@]}"
}

project_kind options_snapshot
project_kind risk_reward
project_kind market_state
project_kind outcome
project_kind valuation signalops.algorithms.eroc_v6

options_source="$(marketops_primary_psql -Atc "SELECT count(*) FROM marketops_options_distribution_daily WHERE tenant_id='tenant-local' AND trade_date=DATE '$session_date' AND window_name='10_trade_days'" | tr -d '[:space:]')"
options_global="$(marketops_primary_psql -Atc "SELECT count(*) FROM subscriber_gateway_global_options_distributions WHERE trade_date=DATE '$session_date' AND window_name='10_trade_days'" | tr -d '[:space:]')"
risk_source="$(marketops_primary_psql -Atc "SELECT count(*) FROM marketops_risk_reward_snapshots WHERE tenant_id='tenant-local' AND session_date=DATE '$session_date'" | tr -d '[:space:]')"
risk_global="$(marketops_primary_psql -Atc "SELECT count(*) FROM subscriber_gateway_global_risk_reward_snapshots WHERE session_date=DATE '$session_date'" | tr -d '[:space:]')"
state_source="$(marketops_primary_psql -Atc "SELECT count(*) FROM marketops_market_states WHERE tenant_id='tenant-local' AND session_date=DATE '$session_date'" | tr -d '[:space:]')"
state_global="$(marketops_primary_psql -Atc "SELECT count(*) FROM subscriber_gateway_global_market_states WHERE session_date=DATE '$session_date'" | tr -d '[:space:]')"
outcome_source="$(marketops_primary_psql -Atc "SELECT count(*) FROM marketops_signal_outcomes WHERE tenant_id='tenant-local' AND source_type='opportunity' AND outcome_status='matured' AND matured_session_date=DATE '$session_date' AND direction IN ('upside','downside') AND directional_hit IS NOT NULL" | tr -d '[:space:]')"
outcome_global="$(marketops_primary_psql -Atc "SELECT count(*) FROM subscriber_gateway_global_signal_assurance_observations WHERE source_type='opportunity' AND matured_session_date=DATE '$session_date'" | tr -d '[:space:]')"
eroc_source="$(marketops_primary_psql -Atc "SELECT count(DISTINCT symbol) FROM marketops_valuation_results WHERE tenant_id='tenant-local' AND algorithm_id='signalops.algorithms.eroc_v6' AND session_date=DATE '$session_date'" | tr -d '[:space:]')"
eroc_global="$(marketops_primary_psql -Atc "SELECT count(DISTINCT symbol) FROM subscriber_gateway_global_eroc_results WHERE session_date=DATE '$session_date'" | tr -d '[:space:]')"

(( options_global >= options_source )) || {
  printf 'global options projection incomplete: source=%s global=%s session=%s\n' "$options_source" "$options_global" "$session_date" >&2
  exit 4
}
(( risk_global >= risk_source )) || {
  printf 'global risk/reward projection incomplete: source=%s global=%s session=%s\n' "$risk_source" "$risk_global" "$session_date" >&2
  exit 5
}
(( state_global >= state_source )) || {
  printf 'global Market State projection incomplete: source=%s global=%s session=%s\n' "$state_source" "$state_global" "$session_date" >&2
  exit 6
}
(( outcome_global >= outcome_source )) || {
  printf 'global SAF outcome projection incomplete: source=%s global=%s session=%s\n' "$outcome_source" "$outcome_global" "$session_date" >&2
  exit 7
}
(( eroc_global >= eroc_source )) || {
  printf 'global EROC projection incomplete: source=%s global=%s session=%s\n' "$eroc_source" "$eroc_global" "$session_date" >&2
  exit 10
}
printf 'global Dashboard evidence projection passed: session=%s options=%s risk_reward=%s market_state=%s saf_outcomes=%s eroc=%s\n' "$session_date" "$options_global" "$risk_global" "$state_global" "$outcome_global" "$eroc_global"
