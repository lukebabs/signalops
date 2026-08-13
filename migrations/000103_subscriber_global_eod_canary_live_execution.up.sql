-- Subscriber Project S4: narrowly scoped, explicitly authorized live execution.
-- Nothing is scheduled. Every authorization is limited to one frozen two-symbol
-- gate and every provider intent must be persisted before the external call.

CREATE TABLE subscriber_global_eod_canary_live_authorizations (
  authorization_id text PRIMARY KEY,
  execution_plan_id text NOT NULL UNIQUE REFERENCES subscriber_global_eod_canary_execution_plans(execution_plan_id) ON DELETE RESTRICT,
  authorization_version text NOT NULL,
  authorized_worker_identity text NOT NULL CHECK (authorized_worker_identity = 'subscriber-global-eod-reconciler'),
  authorized_provider text NOT NULL CHECK (authorized_provider = 'massive'),
  session_date date NOT NULL,
  provider_request_budget integer NOT NULL CHECK (provider_request_budget = 2),
  scheduler_execution_enabled boolean NOT NULL DEFAULT false CHECK (scheduler_execution_enabled = false),
  authorization_state text NOT NULL CHECK (authorization_state = 'authorized'),
  authorization_provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  authorized_by text NOT NULL,
  authorized_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE subscriber_global_eod_canary_live_authorization_members (
  authorization_id text NOT NULL REFERENCES subscriber_global_eod_canary_live_authorizations(authorization_id) ON DELETE RESTRICT,
  global_asset_id text NOT NULL,
  request_ordinal integer NOT NULL CHECK (request_ordinal > 0 AND request_ordinal <= 2),
  expected_symbol text NOT NULL,
  PRIMARY KEY (authorization_id, global_asset_id),
  UNIQUE (authorization_id, request_ordinal)
);

CREATE FUNCTION subscriber_global_eod_canary_live_authorization_member_guard()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  frozen_symbol text;
  frozen_ordinal integer;
BEGIN
  SELECT member.expected_symbol, member.request_ordinal
    INTO frozen_symbol, frozen_ordinal
  FROM subscriber_global_eod_canary_live_authorizations auth
  JOIN subscriber_global_eod_canary_execution_members member ON member.execution_plan_id=auth.execution_plan_id
  WHERE auth.authorization_id=NEW.authorization_id
    AND member.global_asset_id=NEW.global_asset_id;
  IF NOT FOUND OR frozen_ordinal <> NEW.request_ordinal OR frozen_symbol <> NEW.expected_symbol THEN
    RAISE EXCEPTION 'live authorization member must match the frozen execution plan';
  END IF;
  RETURN NEW;
END;
$$;
CREATE TRIGGER trg_subscriber_global_eod_canary_live_authorization_member_guard
BEFORE INSERT ON subscriber_global_eod_canary_live_authorization_members
FOR EACH ROW EXECUTE FUNCTION subscriber_global_eod_canary_live_authorization_member_guard();

CREATE TABLE subscriber_global_eod_canary_live_runs (
  live_run_id text PRIMARY KEY,
  authorization_id text NOT NULL UNIQUE REFERENCES subscriber_global_eod_canary_live_authorizations(authorization_id) ON DELETE RESTRICT,
  worker_identity text NOT NULL CHECK (worker_identity = 'subscriber-global-eod-reconciler'),
  provider_request_budget integer NOT NULL CHECK (provider_request_budget = 2),
  session_date date NOT NULL,
  correlation_id text NOT NULL,
  started_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE subscriber_global_eod_canary_live_run_members (
  live_run_id text NOT NULL REFERENCES subscriber_global_eod_canary_live_runs(live_run_id) ON DELETE RESTRICT,
  global_asset_id text NOT NULL,
  request_ordinal integer NOT NULL CHECK (request_ordinal > 0 AND request_ordinal <= 2),
  expected_symbol text NOT NULL,
  PRIMARY KEY (live_run_id, global_asset_id),
  UNIQUE (live_run_id, request_ordinal)
);

CREATE TABLE subscriber_global_eod_canary_live_evidence_events (
  evidence_event_id text PRIMARY KEY,
  live_run_id text NOT NULL REFERENCES subscriber_global_eod_canary_live_runs(live_run_id) ON DELETE RESTRICT,
  global_asset_id text NOT NULL,
  evidence_kind text NOT NULL CHECK (evidence_kind IN ('provider_request_started', 'provider_response_received', 'normalization_completed', 'provider_request_failed')),
  event_ordinal integer NOT NULL CHECK (event_ordinal > 0),
  payload jsonb NOT NULL DEFAULT '{}'::jsonb,
  provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  recorded_at timestamptz NOT NULL,
  PRIMARY KEY (evidence_event_id),
  FOREIGN KEY (live_run_id, global_asset_id)
    REFERENCES subscriber_global_eod_canary_live_run_members(live_run_id, global_asset_id) ON DELETE RESTRICT,
  UNIQUE (live_run_id, global_asset_id, evidence_kind)
);

-- The intended call must exist before a provider call. An authorization has
-- exactly two frozen members, and each has one immutable call intent.
CREATE UNIQUE INDEX idx_subscriber_global_eod_canary_live_provider_intent_once
  ON subscriber_global_eod_canary_live_evidence_events(live_run_id, global_asset_id)
  WHERE evidence_kind = 'provider_request_started';

CREATE TABLE subscriber_global_eod_canary_baseline_results (
  live_run_id text NOT NULL REFERENCES subscriber_global_eod_canary_live_runs(live_run_id) ON DELETE RESTRICT,
  global_asset_id text NOT NULL,
  session_date date NOT NULL,
  symbol text NOT NULL,
  provider_event_id text NOT NULL DEFAULT '',
  normalized_payload jsonb NOT NULL,
  normalized_fingerprint text NOT NULL,
  algorithm_version text NOT NULL,
  quality_state text NOT NULL CHECK (quality_state IN ('usable', 'invalid')),
  provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  calculated_at timestamptz NOT NULL,
  PRIMARY KEY (live_run_id, global_asset_id),
  FOREIGN KEY (live_run_id, global_asset_id)
    REFERENCES subscriber_global_eod_canary_live_run_members(live_run_id, global_asset_id) ON DELETE RESTRICT
);

CREATE TABLE subscriber_global_eod_canary_parity_reports (
  parity_report_id text PRIMARY KEY,
  live_run_id text NOT NULL REFERENCES subscriber_global_eod_canary_live_runs(live_run_id) ON DELETE RESTRICT,
  global_asset_id text NOT NULL,
  comparison_tenant_id text NOT NULL CHECK (comparison_tenant_id = 'tenant-local'),
  comparison_event_id text NOT NULL DEFAULT '',
  global_fingerprint text NOT NULL,
  comparison_fingerprint text NOT NULL DEFAULT '',
  parity_status text NOT NULL CHECK (parity_status IN ('matched', 'mismatched', 'missing')),
  mismatch_reason text NOT NULL DEFAULT '',
  provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  compared_at timestamptz NOT NULL,
  UNIQUE (live_run_id, global_asset_id),
  FOREIGN KEY (live_run_id, global_asset_id)
    REFERENCES subscriber_global_eod_canary_baseline_results(live_run_id, global_asset_id) ON DELETE RESTRICT
);

ALTER TABLE subscriber_global_eod_canary_live_authorizations OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_global_eod_canary_live_authorization_members OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_global_eod_canary_live_runs OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_global_eod_canary_live_run_members OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_global_eod_canary_live_evidence_events OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_global_eod_canary_baseline_results OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_global_eod_canary_parity_reports OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_global_eod_canary_live_authorizations, subscriber_global_eod_canary_live_authorization_members, subscriber_global_eod_canary_live_runs, subscriber_global_eod_canary_live_run_members, subscriber_global_eod_canary_live_evidence_events, subscriber_global_eod_canary_baseline_results, subscriber_global_eod_canary_parity_reports FROM PUBLIC;
GRANT SELECT, INSERT ON subscriber_global_eod_canary_live_runs, subscriber_global_eod_canary_live_run_members, subscriber_global_eod_canary_live_evidence_events, subscriber_global_eod_canary_baseline_results, subscriber_global_eod_canary_parity_reports TO signalops_subscriber_global_eod;
GRANT SELECT ON subscriber_global_eod_canary_live_authorizations, subscriber_global_eod_canary_live_authorization_members TO signalops_subscriber_global_eod;
REVOKE ALL ON FUNCTION subscriber_global_eod_canary_live_authorization_member_guard() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION subscriber_global_eod_canary_live_authorization_member_guard() TO signalops_subscriber_migrator;
