-- Subscriber Project S4: append-only execution gate for a separately approved
-- two-symbol canary. This migration does not enable provider collection. The
-- only permitted plan state is disabled and the kill switch is engaged.

CREATE TABLE subscriber_global_eod_canary_execution_plans (
  execution_plan_id text PRIMARY KEY,
  canary_run_id text NOT NULL REFERENCES subscriber_global_eod_canary_runs(canary_run_id) ON DELETE RESTRICT,
  execution_version text NOT NULL,
  expected_worker_identity text NOT NULL CHECK (expected_worker_identity = 'subscriber-global-eod-reconciler'),
  session_date date NOT NULL,
  max_provider_requests integer NOT NULL CHECK (max_provider_requests > 0 AND max_provider_requests <= 2),
  provider_execution_enabled boolean NOT NULL DEFAULT false CHECK (provider_execution_enabled = false),
  scheduled_execution_enabled boolean NOT NULL DEFAULT false CHECK (scheduled_execution_enabled = false),
  kill_switch_engaged boolean NOT NULL DEFAULT true CHECK (kill_switch_engaged = true),
  execution_state text NOT NULL DEFAULT 'disabled' CHECK (execution_state = 'disabled'),
  correlation_id text NOT NULL DEFAULT '',
  control_provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  planned_by text NOT NULL,
  planned_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (canary_run_id)
);

CREATE TABLE subscriber_global_eod_canary_execution_members (
  execution_plan_id text NOT NULL REFERENCES subscriber_global_eod_canary_execution_plans(execution_plan_id) ON DELETE RESTRICT,
  global_asset_id text NOT NULL REFERENCES subscriber_global_assets(global_asset_id) ON DELETE RESTRICT,
  request_ordinal integer NOT NULL CHECK (request_ordinal > 0 AND request_ordinal <= 2),
  expected_symbol text NOT NULL,
  expected_algorithm_version text NOT NULL,
  expected_validation_contract_ref text NOT NULL,
  expected_baseline_provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  PRIMARY KEY (execution_plan_id, global_asset_id),
  UNIQUE (execution_plan_id, request_ordinal)
);

CREATE FUNCTION subscriber_global_eod_canary_execution_member_guard()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  frozen_symbol text;
  frozen_priority integer;
BEGIN
  SELECT asset.canonical_symbol, member.priority
    INTO frozen_symbol, frozen_priority
  FROM subscriber_global_eod_canary_execution_plans plan
  JOIN subscriber_global_eod_canary_members member ON member.canary_run_id=plan.canary_run_id
  JOIN subscriber_global_assets asset ON asset.global_asset_id=member.global_asset_id
  WHERE plan.execution_plan_id=NEW.execution_plan_id
    AND member.global_asset_id=NEW.global_asset_id;
  IF NOT FOUND OR frozen_priority <> NEW.request_ordinal OR frozen_symbol <> NEW.expected_symbol THEN
    RAISE EXCEPTION 'canary execution member must match the frozen canary symbol and priority';
  END IF;
  RETURN NEW;
END;
$$;

CREATE TRIGGER trg_subscriber_global_eod_canary_execution_member_guard
BEFORE INSERT ON subscriber_global_eod_canary_execution_members
FOR EACH ROW EXECUTE FUNCTION subscriber_global_eod_canary_execution_member_guard();

-- An append-only, per-symbol evidence ledger. Future execution work may add
-- observations but can never alter the frozen plan or prior evidence.
CREATE TABLE subscriber_global_eod_canary_evidence_events (
  evidence_event_id text PRIMARY KEY,
  execution_plan_id text NOT NULL,
  global_asset_id text NOT NULL,
  evidence_kind text NOT NULL CHECK (evidence_kind IN ('execution_planned', 'provider_request_started', 'provider_response_received', 'normalization_completed', 'parity_compared', 'execution_blocked')),
  event_ordinal integer NOT NULL CHECK (event_ordinal > 0),
  payload jsonb NOT NULL DEFAULT '{}'::jsonb,
  provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  recorded_by text NOT NULL,
  recorded_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (execution_plan_id, global_asset_id)
    REFERENCES subscriber_global_eod_canary_execution_members(execution_plan_id, global_asset_id) ON DELETE RESTRICT,
  UNIQUE (execution_plan_id, global_asset_id, evidence_kind, event_ordinal)
);

CREATE FUNCTION subscriber_global_eod_canary_evidence_disabled_guard()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  plan_is_disabled boolean;
BEGIN
  SELECT execution_state= 'disabled' AND provider_execution_enabled=false AND scheduled_execution_enabled=false AND kill_switch_engaged=true
    INTO plan_is_disabled
  FROM subscriber_global_eod_canary_execution_plans
  WHERE execution_plan_id=NEW.execution_plan_id;
  IF NOT FOUND OR NOT plan_is_disabled OR NEW.evidence_kind <> 'execution_planned' THEN
    RAISE EXCEPTION 'global EOD canary execution is disabled; only execution_planned evidence is permitted';
  END IF;
  RETURN NEW;
END;
$$;

CREATE TRIGGER trg_subscriber_global_eod_canary_evidence_disabled_guard
BEFORE INSERT ON subscriber_global_eod_canary_evidence_events
FOR EACH ROW EXECUTE FUNCTION subscriber_global_eod_canary_evidence_disabled_guard();

REVOKE ALL ON FUNCTION subscriber_global_eod_canary_execution_member_guard() FROM PUBLIC;
REVOKE ALL ON FUNCTION subscriber_global_eod_canary_evidence_disabled_guard() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION subscriber_global_eod_canary_execution_member_guard() TO signalops_subscriber_global_eod;
GRANT EXECUTE ON FUNCTION subscriber_global_eod_canary_evidence_disabled_guard() TO signalops_subscriber_global_eod;

-- A provider request can be represented at most once for each frozen member;
-- with two frozen request ordinals, this is a database-enforced two-request
-- ceiling for this execution plan.
CREATE UNIQUE INDEX idx_subscriber_global_eod_canary_provider_request_once
  ON subscriber_global_eod_canary_evidence_events(execution_plan_id, global_asset_id)
  WHERE evidence_kind = 'provider_request_started';

ALTER TABLE subscriber_global_eod_canary_execution_plans OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_global_eod_canary_execution_members OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_global_eod_canary_evidence_events OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_global_eod_canary_execution_plans, subscriber_global_eod_canary_execution_members, subscriber_global_eod_canary_evidence_events FROM PUBLIC;
GRANT SELECT, INSERT ON subscriber_global_eod_canary_execution_plans, subscriber_global_eod_canary_execution_members, subscriber_global_eod_canary_evidence_events TO signalops_subscriber_global_eod;
