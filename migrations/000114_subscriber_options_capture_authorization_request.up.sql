-- S6 approval request only. It records a named request for later review but
-- cannot authorize a provider call or alter the disabled capture gate.

CREATE TABLE subscriber_options_capture_authorization_requests (
  authorization_request_id text PRIMARY KEY,
  capture_plan_id text NOT NULL UNIQUE REFERENCES subscriber_options_capture_canary_plans(capture_plan_id) ON DELETE RESTRICT,
  request_version text NOT NULL,
  requested_worker_identity text NOT NULL CHECK (requested_worker_identity = 'subscriber-options-capture'),
  requested_provider text NOT NULL CHECK (requested_provider = 'massive'),
  requested_provider_budget integer NOT NULL CHECK (requested_provider_budget = 1),
  request_state text NOT NULL CHECK (request_state = 'pending_approval'),
  requested_by text NOT NULL CHECK (requested_by = 'subscriber-options-capture'),
  request_reason text NOT NULL CHECK (char_length(request_reason) BETWEEN 1 AND 500),
  correlation_id text NOT NULL,
  request_provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  requested_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE FUNCTION subscriber_options_capture_authorization_request_guard()
RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
  plan_state text; provider_enabled boolean; scheduler_enabled boolean; kill_switch boolean; budget integer;
BEGIN
  SELECT execution_state, provider_execution_enabled, scheduled_execution_enabled, kill_switch_engaged, max_provider_requests
    INTO plan_state, provider_enabled, scheduler_enabled, kill_switch, budget
  FROM subscriber_options_capture_canary_plans WHERE capture_plan_id=NEW.capture_plan_id;
  IF NOT FOUND OR plan_state <> 'disabled' OR provider_enabled OR scheduler_enabled OR NOT kill_switch OR budget <> 1 THEN
    RAISE EXCEPTION 'authorization request requires unchanged disabled one-request capture gate';
  END IF;
  RETURN NEW;
END;
$$;
CREATE TRIGGER trg_subscriber_options_capture_authorization_request_guard
BEFORE INSERT ON subscriber_options_capture_authorization_requests
FOR EACH ROW EXECUTE FUNCTION subscriber_options_capture_authorization_request_guard();

ALTER TABLE subscriber_options_capture_authorization_requests OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_options_capture_authorization_requests FROM PUBLIC;
GRANT SELECT, INSERT ON subscriber_options_capture_authorization_requests TO signalops_subscriber_options_capture;
REVOKE ALL ON FUNCTION subscriber_options_capture_authorization_request_guard() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION subscriber_options_capture_authorization_request_guard() TO signalops_subscriber_options_capture;
