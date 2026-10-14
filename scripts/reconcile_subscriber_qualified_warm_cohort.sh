#!/usr/bin/env bash
set -euo pipefail

# One controlled expansion of the ranked candidate pool into a full, qualified
# 1,000-member warm-EOD plan. It never enables intraday collection and makes no
# price, options, or FMP request. Massive is used only for bounded reference
# validation of the next ranked candidates, with no retries.
[[ "${EUID}" -eq 0 ]] || {
  printf '%s\n' 'Run this command through the SignalOps deployment agent.' >&2
  exit 2
}

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=lib/marketops_boundary_env.sh
source "$root_dir/scripts/lib/marketops_boundary_env.sh"
runtime_env="${1:-${SIGNALOPS_PRODUCTION_ENV_FILE:-}}"
boundary_env=/etc/signalops/marketops-boundary.env
ranking_input="$root_dir/companies.csv"

[[ -n "$runtime_env" && -r "$runtime_env" ]] || {
  printf '%s\n' 'Provide a readable production Compose environment file as argument 1.' >&2
  exit 2
}
[[ -r "$boundary_env" ]] || {
  printf '%s\n' 'Protected MarketOps boundary secret is not readable.' >&2
  exit 3
}
[[ -r "$ranking_input" ]] || {
  printf '%s\n' 'The reviewed ranked candidate source companies.csv is not present.' >&2
  exit 4
}

# One active reconciliation is permitted. The lock spans provider validation,
# planning, and activation so a detached caller cannot create a duplicate run.
exec 9>/var/lock/signalops-subscriber-qualified-warm-cohort.lock
flock -n 9 || {
  printf '%s\n' 'subscriber_qualified_warm_cohort_reconciliation_already_running' >&2
  exit 3
}
load_marketops_boundary_env "$boundary_env"
export SIGNALOPS_MARKETOPS_DATABASE_URL="postgres://signalops:${SIGNALOPS_MARKETOPS_POSTGRES_PASSWORD}@marketops-postgres:5432/marketops?sslmode=disable"
export SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL="postgres://signalops:${SIGNALOPS_MARKETOPS_TEMPORAL_PASSWORD}@marketops-timescaledb:5432/marketops_temporal?sslmode=disable"
compose=(docker compose --env-file "$runtime_env" -p signalops
  -f "$root_dir/compose.yaml"
  -f "$root_dir/compose.marketops-boundary.yaml"
  -f "$root_dir/compose.marketops-scheduled-cutover.yaml")
correlation_id="subscriber-qualified-warm-$(date -u +%Y%m%dT%H%M%SZ)"

# Retain the first 1,500 ranked candidates. The existing first 1,000 evidence
# is preserved, while 500 next-ranked candidates provide replacement capacity.
"${compose[@]}" --profile subscriber-global-evidence run --rm --no-deps --build \
  subscriber-global-ranking-import \
  --execute --input /input/companies.csv --as-of 2026-08-12 --candidate-limit 1500 \
  --actor subscriber-catalog-reference-sync

# Reference-only qualification of at most the 500 newly ranked candidates. A
# single lookup failure becomes a deferred decision; the worker has no retry.
"${compose[@]}" --profile subscriber-global-evidence run --rm --no-deps --build \
  subscriber-global-catalog-admission \
  --execute --max-assets 500 --request-delay 300ms \
  --actor subscriber-catalog-reference-sync --correlation-id "$correlation_id"

"${compose[@]}" --profile subscriber-global-evidence run --rm --no-deps --build \
  subscriber-global-eod-shadow-planner \
  --execute --capacity 1000 --actor subscriber-global-eod-reconciler \
  --correlation-id "$correlation_id"

plan_id="$("${compose[@]}" exec -T marketops-postgres psql -v ON_ERROR_STOP=1 -U signalops -d marketops -Atc "SELECT plan_run_id FROM subscriber_global_eod_hot_set_plan_runs WHERE correlation_id='${correlation_id}' AND capacity=1000 AND selected_count=1000 ORDER BY planned_at DESC, plan_run_id DESC LIMIT 1;")"
[[ -n "$plan_id" ]] || {
  printf '%s\n' 'Qualified candidate pool did not produce a full 1,000-member plan; no warm activation was changed.' >&2
  exit 5
}

"${compose[@]}" exec -T marketops-postgres psql -v ON_ERROR_STOP=1 -U signalops -d marketops -c "
  INSERT INTO subscriber_global_eod_warm_set_activations
    (activation_id, plan_run_id, activation_state, policy_version, activated_by, correlation_id, rationale, activated_at)
  VALUES
    ('subwarm-' || md5('${plan_id}:subscriber-warm-eod-v2'), '${plan_id}', 'enabled', 'subscriber-warm-eod-v2',
     'subscriber-qualified-warm-cohort-reconciler', '${correlation_id}',
     'fill the 1,000-member qualified US-common-stock warm cohort from ranked replacements', now())
  ON CONFLICT (activation_id) DO NOTHING;"

"${compose[@]}" exec -T marketops-postgres psql -v ON_ERROR_STOP=1 -U signalops -d marketops -c "
  SELECT count(*) AS active_warm_assets, min(source_rank) AS first_source_rank,
         max(source_rank) AS last_source_rank
  FROM subscriber_global_warm_eod_assets;"
printf '%s\n' "subscriber_qualified_warm_cohort_reconciled correlation_id=${correlation_id} plan_id=${plan_id}"
