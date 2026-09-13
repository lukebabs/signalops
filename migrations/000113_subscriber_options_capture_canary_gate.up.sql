-- S6 one-asset Options-capture control plane. This is disabled by constraint:
-- it freezes one selected shadow member but cannot authorize provider work.

CREATE TABLE subscriber_options_capture_canary_plans (
  capture_plan_id text PRIMARY KEY,
  snapshot_run_id text NOT NULL REFERENCES subscriber_options_demand_snapshot_runs(snapshot_run_id) ON DELETE RESTRICT,
  capture_version text NOT NULL,
  session_date date NOT NULL,
  max_provider_requests integer NOT NULL CHECK (max_provider_requests = 1),
  provider_execution_enabled boolean NOT NULL DEFAULT false CHECK (provider_execution_enabled = false),
  scheduled_execution_enabled boolean NOT NULL DEFAULT false CHECK (scheduled_execution_enabled = false),
  kill_switch_engaged boolean NOT NULL DEFAULT true CHECK (kill_switch_engaged = true),
  execution_state text NOT NULL CHECK (execution_state = 'disabled'),
  expected_worker_identity text NOT NULL CHECK (expected_worker_identity = 'subscriber-options-capture'),
  expected_provider text NOT NULL CHECK (expected_provider = 'massive'),
  control_provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  correlation_id text NOT NULL,
  planned_by text NOT NULL CHECK (planned_by = 'subscriber-options-capture'),
  planned_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (snapshot_run_id)
);

CREATE TABLE subscriber_options_capture_canary_members (
  capture_plan_id text NOT NULL REFERENCES subscriber_options_capture_canary_plans(capture_plan_id) ON DELETE RESTRICT,
  global_asset_id text NOT NULL REFERENCES subscriber_global_assets(global_asset_id) ON DELETE RESTRICT,
  expected_symbol text NOT NULL,
  request_ordinal integer NOT NULL CHECK (request_ordinal = 1),
  expected_readiness_policy text NOT NULL,
  expected_baseline_provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  PRIMARY KEY (capture_plan_id, global_asset_id),
  UNIQUE (capture_plan_id, request_ordinal)
);

CREATE FUNCTION subscriber_options_capture_canary_member_guard()
RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
  frozen_symbol text;
BEGIN
  SELECT asset.canonical_symbol INTO frozen_symbol
  FROM subscriber_options_capture_canary_plans plan
  JOIN subscriber_options_demand_snapshot_members member ON member.snapshot_run_id=plan.snapshot_run_id
  JOIN subscriber_global_assets asset ON asset.global_asset_id=member.global_asset_id
  WHERE plan.capture_plan_id=NEW.capture_plan_id AND member.global_asset_id=NEW.global_asset_id
    AND member.selection_state='selected' AND member.priority=1;
  IF NOT FOUND OR frozen_symbol <> NEW.expected_symbol THEN
    RAISE EXCEPTION 'Options capture member must match selected priority-one shadow member';
  END IF;
  RETURN NEW;
END;
$$;
CREATE TRIGGER trg_subscriber_options_capture_canary_member_guard
BEFORE INSERT ON subscriber_options_capture_canary_members
FOR EACH ROW EXECUTE FUNCTION subscriber_options_capture_canary_member_guard();

CREATE TABLE subscriber_options_capture_canary_evidence_events (
  evidence_event_id text PRIMARY KEY,
  capture_plan_id text NOT NULL,
  global_asset_id text NOT NULL,
  evidence_kind text NOT NULL CHECK (evidence_kind IN ('capture_planned', 'provider_request_started', 'provider_response_received', 'capture_normalized', 'capture_failed')),
  event_ordinal integer NOT NULL CHECK (event_ordinal > 0),
  payload jsonb NOT NULL DEFAULT '{}'::jsonb,
  provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  recorded_by text NOT NULL,
  recorded_at timestamptz NOT NULL,
  FOREIGN KEY (capture_plan_id, global_asset_id) REFERENCES subscriber_options_capture_canary_members(capture_plan_id, global_asset_id) ON DELETE RESTRICT,
  UNIQUE (capture_plan_id, global_asset_id, evidence_kind)
);

CREATE FUNCTION subscriber_options_capture_canary_evidence_disabled_guard()
RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE state text;
BEGIN
  SELECT execution_state INTO state FROM subscriber_options_capture_canary_plans WHERE capture_plan_id=NEW.capture_plan_id;
  IF state <> 'disabled' OR (NEW.evidence_kind <> 'capture_planned') THEN
    RAISE EXCEPTION 'Options capture canary is disabled; only capture_planned evidence is permitted';
  END IF;
  RETURN NEW;
END;
$$;
CREATE TRIGGER trg_subscriber_options_capture_canary_evidence_disabled_guard
BEFORE INSERT ON subscriber_options_capture_canary_evidence_events
FOR EACH ROW EXECUTE FUNCTION subscriber_options_capture_canary_evidence_disabled_guard();

ALTER TABLE subscriber_options_capture_canary_plans OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_options_capture_canary_members OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_options_capture_canary_evidence_events OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_options_capture_canary_plans, subscriber_options_capture_canary_members, subscriber_options_capture_canary_evidence_events FROM PUBLIC;
GRANT SELECT ON subscriber_options_demand_snapshot_runs, subscriber_options_demand_snapshot_members, subscriber_global_assets TO signalops_subscriber_options_capture;
GRANT SELECT, INSERT ON subscriber_options_capture_canary_plans, subscriber_options_capture_canary_members, subscriber_options_capture_canary_evidence_events TO signalops_subscriber_options_capture;
REVOKE ALL ON FUNCTION subscriber_options_capture_canary_member_guard(), subscriber_options_capture_canary_evidence_disabled_guard() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION subscriber_options_capture_canary_member_guard(), subscriber_options_capture_canary_evidence_disabled_guard() TO signalops_subscriber_options_capture;
