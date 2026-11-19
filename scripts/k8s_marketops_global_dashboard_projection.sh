#!/usr/bin/env bash
set -euo pipefail
session_date="${1:-}"
[[ "$session_date" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}$ ]] || { echo "usage: $0 YYYY-MM-DD" >&2; exit 2; }
: "${SIGNALOPS_MARKETOPS_DATABASE_URL:?dedicated MarketOps database required}"
export SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL="$SIGNALOPS_MARKETOPS_DATABASE_URL"
correlation="postclose-global-dashboard-${session_date}"

# The projection is the final publish boundary.  Require a persisted
# dependency watermark for the current immutable asset-list revision so a
# partially materialized upstream run cannot publish a misleading dashboard.
read -r watermark_status watermark_expected watermark_ready watermark_projected watermark_revision watermark_reason <<<"$(psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -AtF ' ' -c "
  SELECT w.status, w.expected_assets, w.ready_assets, w.projected_assets, w.asset_list_revision, replace(w.reason, ' ', '_')
  FROM marketops_dependency_watermarks w
  JOIN marketops_asset_lists l ON l.list_id=w.asset_list_id
  WHERE w.tenant_id='tenant-local'
    AND w.session_date=DATE '$session_date'
    AND w.asset_list_id='marketops-primary-tenant-local'
    AND w.asset_list_revision=l.current_revision
  ORDER BY w.observed_at DESC LIMIT 1;")"
if [[ "$watermark_status" == blocked || "$watermark_status" == "" ]]; then
  echo "marketops_global_dashboard_projection_blocked_by_dependency_watermark session=$session_date status=${watermark_status:-missing} expected=${watermark_expected:-0} ready=${watermark_ready:-0} projected=${watermark_projected:-0} revision=${watermark_revision:-unknown}" >&2
  exit 12
fi
if [[ "$watermark_status" == degraded ]]; then
  echo "marketops_global_dashboard_projection_dependency_watermark_degraded session=$session_date expected=$watermark_expected ready=$watermark_ready reason=${watermark_reason:-unavailable_assets}" >&2
else
  echo "marketops_global_dashboard_projection_dependency_watermark_ready session=$session_date expected=$watermark_expected revision=$watermark_revision"
fi
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
read -r list_revision list_fingerprint expected_assets risk_global risk_eligible <<<"$(psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -AtF ' ' -c "
  SELECT l.current_revision, r.membership_fingerprint,
         (SELECT count(*) FROM marketops_primary_assets WHERE tenant_id='tenant-local'),
         count(s.session_date), count(s.session_date) FILTER (WHERE eligible AND quality_state='usable')
  FROM marketops_asset_lists l
  JOIN marketops_asset_list_revisions r ON r.list_id=l.list_id AND r.revision=l.current_revision
  LEFT JOIN subscriber_gateway_global_risk_reward_snapshots s ON s.session_date=DATE '$session_date'
  WHERE l.list_id='marketops-primary-tenant-local'
  GROUP BY l.current_revision, r.membership_fingerprint")"
if [[ ! "$expected_assets" =~ ^[0-9]+$ || ! "$risk_global" =~ ^[0-9]+$ || ! "$risk_eligible" =~ ^[0-9]+$ || "$risk_global" -ne "$expected_assets" ]]; then
  echo "global risk/reward projection incomplete: session=$session_date expected=$expected_assets rows=$risk_global usable=$risk_eligible" >&2
  exit 4
fi
psql "$SIGNALOPS_MARKETOPS_DATABASE_URL" -v ON_ERROR_STOP=1 -c "
  SELECT marketops_record_dependency_watermark(
    'dagster-projection:$session_date:$list_revision', 'tenant-local', DATE '$session_date',
    'marketops-primary-tenant-local', $list_revision, '$list_fingerprint',
    $expected_assets, $risk_eligible, $risk_eligible, 4,
    CASE WHEN $risk_eligible = $expected_assets THEN 'ready' ELSE 'degraded' END,
    CASE WHEN $risk_eligible = $expected_assets THEN '' ELSE 'unavailable_assets_excluded_from_usable_cohort' END,
    '$correlation');" >/dev/null
echo "k8s_global_dashboard_projection_verified session=$session_date risk_reward=$risk_global usable=$risk_eligible unavailable=$((expected_assets-risk_eligible)) asset_list=$expected_assets"
