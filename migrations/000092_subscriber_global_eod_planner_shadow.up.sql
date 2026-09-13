-- Subscriber Project S2: governed eligibility and EOD-planner shadow.
-- No table is directly available to the browser gateway and no runner is enabled.

CREATE TABLE subscriber_global_asset_eligibility_decisions (
  decision_id text PRIMARY KEY,
  global_asset_id text NOT NULL REFERENCES subscriber_global_assets(global_asset_id) ON DELETE RESTRICT,
  decision text NOT NULL CHECK (decision IN ('eligible', 'ineligible', 'deferred')),
  policy_version text NOT NULL,
  reason_code text NOT NULL,
  provider_reference_at timestamptz,
  evidence jsonb NOT NULL DEFAULT '{}'::jsonb,
  provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  decided_by text NOT NULL,
  decided_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_subscriber_global_asset_eligibility_decisions_asset_time
  ON subscriber_global_asset_eligibility_decisions (global_asset_id, decided_at DESC);

CREATE TABLE subscriber_global_eod_hot_set_plan_runs (
  plan_run_id text PRIMARY KEY,
  planner_version text NOT NULL,
  execution_mode text NOT NULL CHECK (execution_mode IN ('shadow', 'enabled')),
  capacity integer NOT NULL CHECK (capacity > 0 AND capacity <= 1000),
  candidate_count integer NOT NULL DEFAULT 0 CHECK (candidate_count >= 0),
  eligible_count integer NOT NULL DEFAULT 0 CHECK (eligible_count >= 0),
  selected_count integer NOT NULL DEFAULT 0 CHECK (selected_count >= 0 AND selected_count <= capacity),
  excluded_count integer NOT NULL DEFAULT 0 CHECK (excluded_count >= 0),
  report jsonb NOT NULL DEFAULT '{}'::jsonb,
  planned_by text NOT NULL,
  correlation_id text NOT NULL DEFAULT '',
  planned_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_subscriber_global_eod_hot_set_plan_runs_mode_time
  ON subscriber_global_eod_hot_set_plan_runs (execution_mode, planned_at DESC);

CREATE TABLE subscriber_global_eod_hot_set_plan_members (
  plan_run_id text NOT NULL REFERENCES subscriber_global_eod_hot_set_plan_runs(plan_run_id) ON DELETE RESTRICT,
  global_asset_id text NOT NULL REFERENCES subscriber_global_assets(global_asset_id) ON DELETE RESTRICT,
  priority integer NOT NULL CHECK (priority > 0),
  source_rank integer,
  selection_reason text NOT NULL,
  PRIMARY KEY (plan_run_id, global_asset_id),
  UNIQUE (plan_run_id, priority)
);
CREATE INDEX idx_subscriber_global_eod_hot_set_plan_members_asset
  ON subscriber_global_eod_hot_set_plan_members (global_asset_id, plan_run_id DESC);

CREATE TABLE subscriber_global_coverage_activation_requests (
  activation_request_id text PRIMARY KEY,
  global_asset_id text NOT NULL REFERENCES subscriber_global_assets(global_asset_id) ON DELETE RESTRICT,
  request_key text NOT NULL UNIQUE,
  request_state text NOT NULL CHECK (request_state IN ('queued', 'warming_up', 'active', 'deferred', 'rejected', 'cancelled')),
  request_reason text NOT NULL,
  requester_kind text NOT NULL CHECK (requester_kind IN ('operator', 'tenant_default_list', 'user_private_list')),
  requester_tenant_id text NOT NULL DEFAULT '',
  requester_subject text NOT NULL DEFAULT '',
  requester_list_id text NOT NULL DEFAULT '',
  policy_version text NOT NULL,
  provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  requested_at timestamptz NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX idx_subscriber_global_coverage_activation_one_open
  ON subscriber_global_coverage_activation_requests (global_asset_id)
  WHERE request_state IN ('queued', 'warming_up');
CREATE INDEX idx_subscriber_global_coverage_activation_queue
  ON subscriber_global_coverage_activation_requests (request_state, requested_at, global_asset_id);

ALTER TABLE subscriber_global_asset_eligibility_decisions OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_global_eod_hot_set_plan_runs OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_global_eod_hot_set_plan_members OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_global_coverage_activation_requests OWNER TO signalops_subscriber_migrator;

REVOKE ALL ON subscriber_global_asset_eligibility_decisions, subscriber_global_eod_hot_set_plan_runs, subscriber_global_eod_hot_set_plan_members, subscriber_global_coverage_activation_requests FROM PUBLIC;

GRANT SELECT, INSERT, UPDATE ON subscriber_global_asset_eligibility_decisions TO signalops_subscriber_catalog_sync;
GRANT SELECT, INSERT, UPDATE ON subscriber_global_eod_hot_set_plan_runs, subscriber_global_eod_hot_set_plan_members, subscriber_global_coverage_activation_requests TO signalops_subscriber_global_eod;
