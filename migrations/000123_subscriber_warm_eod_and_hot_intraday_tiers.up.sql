-- Operational coverage tiers. Historic S2 "hot set" names remain for
-- compatibility; their purpose is the warm EOD baseline, up to 1,000 eligible
-- US common stocks. Intraday work is an independent, deduplicated demand tier.
CREATE TABLE subscriber_global_eod_warm_set_activations (
  activation_id text PRIMARY KEY,
  plan_run_id text NOT NULL REFERENCES subscriber_global_eod_hot_set_plan_runs(plan_run_id) ON DELETE RESTRICT,
  activation_state text NOT NULL CHECK (activation_state IN ('enabled', 'disabled')),
  policy_version text NOT NULL,
  activated_by text NOT NULL,
  correlation_id text NOT NULL DEFAULT '',
  rationale text NOT NULL DEFAULT '',
  activated_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_subscriber_global_eod_warm_set_activations_current ON subscriber_global_eod_warm_set_activations (activation_state, activated_at DESC);

CREATE VIEW subscriber_global_warm_eod_assets WITH (security_barrier = true) AS
WITH selected_activation AS (
  SELECT activation.plan_run_id, plan.capacity, plan.selected_count, plan.planner_version, activation.policy_version, activation.activated_at
  FROM subscriber_global_eod_warm_set_activations activation
  JOIN subscriber_global_eod_hot_set_plan_runs plan ON plan.plan_run_id=activation.plan_run_id
  WHERE activation.activation_state='enabled'
  ORDER BY activation.activated_at DESC, activation.activation_id DESC LIMIT 1
)
SELECT asset.global_asset_id, asset.canonical_symbol, member.priority, member.source_rank,
  selected.capacity, selected.selected_count, selected.planner_version, selected.policy_version, selected.activated_at
FROM selected_activation selected
JOIN subscriber_global_eod_hot_set_plan_members member ON member.plan_run_id=selected.plan_run_id
JOIN subscriber_global_assets asset ON asset.global_asset_id=member.global_asset_id
WHERE asset.eligibility_status='eligible'
ORDER BY member.priority, asset.global_asset_id;

-- Aggregate demand only: no tenant, subject, or list identity is exposed.
CREATE VIEW subscriber_global_hot_intraday_assets WITH (security_barrier = true) AS
WITH selected_lists AS (
  SELECT preference.tenant_id, preference.subject, list.list_id, preference.updated_at
  FROM subscriber_watchlist_context_preferences preference
  JOIN subscriber_watchlists list ON list.tenant_id=preference.tenant_id
    AND ((preference.selection_mode='list' AND preference.list_id=list.list_id) OR preference.selection_mode='all')
    AND (list.list_kind='tenant_default' OR (list.list_kind='private' AND list.owner_subject=preference.subject))
)
SELECT asset.global_asset_id, asset.canonical_symbol,
  count(DISTINCT selected.tenant_id || E'\\x1f' || selected.subject)::bigint AS watcher_count,
  min(membership.added_at) AS first_selected_at, max(selected.updated_at) AS last_selected_at
FROM selected_lists selected
JOIN subscriber_watchlist_memberships membership ON membership.tenant_id=selected.tenant_id AND membership.list_id=selected.list_id
JOIN subscriber_global_assets asset ON asset.global_asset_id=membership.global_asset_id
WHERE asset.eligibility_status='eligible'
GROUP BY asset.global_asset_id, asset.canonical_symbol;

ALTER TABLE subscriber_global_eod_warm_set_activations OWNER TO signalops_subscriber_migrator;
ALTER VIEW subscriber_global_warm_eod_assets OWNER TO signalops_subscriber_migrator;
ALTER VIEW subscriber_global_hot_intraday_assets OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_global_eod_warm_set_activations FROM PUBLIC;
REVOKE ALL ON subscriber_global_warm_eod_assets, subscriber_global_hot_intraday_assets FROM PUBLIC;
GRANT SELECT, INSERT ON subscriber_global_eod_warm_set_activations TO signalops_subscriber_global_eod;
GRANT SELECT ON subscriber_global_warm_eod_assets, subscriber_global_hot_intraday_assets TO signalops_subscriber_global_eod;


-- The user-directed warm policy takes effect from the latest already-governed
-- S2 plan. This is an auditable policy transition, not a per-run approval;
-- provider work begins only when the separately deployed scheduler invokes the
-- warm-EOD runner.
INSERT INTO subscriber_global_eod_warm_set_activations
  (activation_id, plan_run_id, activation_state, policy_version, activated_by, correlation_id, rationale, activated_at)
SELECT
  'subwarm-' || md5(plan.plan_run_id || ':subscriber-warm-eod-v1'),
  plan.plan_run_id, 'enabled', 'subscriber-warm-eod-v1',
  'subscriber-warm-eod-policy', 'subscriber-warm-eod-policy-20260815',
  'automatic activation of the latest governed warm-EOD plan', now()
FROM subscriber_global_eod_hot_set_plan_runs plan
WHERE plan.selected_count > 0
ORDER BY plan.planned_at DESC, plan.plan_run_id DESC
LIMIT 1
ON CONFLICT (activation_id) DO NOTHING;
