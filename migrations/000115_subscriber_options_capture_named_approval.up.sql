-- A human approval attestation is distinct from execution authorization. It
-- cannot update the disabled gate or make provider execution possible.

CREATE TABLE subscriber_options_capture_named_approvals (
  approval_id text PRIMARY KEY,
  authorization_request_id text NOT NULL UNIQUE REFERENCES subscriber_options_capture_authorization_requests(authorization_request_id) ON DELETE RESTRICT,
  approver_subject text NOT NULL CHECK (approver_subject = 'luke@strategiclabs.io'),
  approval_state text NOT NULL CHECK (approval_state = 'approved_pending_recovery'),
  approved_provider text NOT NULL CHECK (approved_provider = 'massive'),
  approved_provider_budget integer NOT NULL CHECK (approved_provider_budget = 1),
  approved_retry_budget integer NOT NULL CHECK (approved_retry_budget = 0),
  approval_statement text NOT NULL CHECK (char_length(approval_statement) BETWEEN 1 AND 1000),
  recovery_gate_status text NOT NULL CHECK (recovery_gate_status = 'blocked'),
  correlation_id text NOT NULL,
  approval_provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  approved_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE FUNCTION subscriber_options_capture_named_approval_guard()
RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE request_state text; gate_state text; provider_enabled boolean; kill_switch boolean;
BEGIN
  SELECT request.request_state, plan.execution_state, plan.provider_execution_enabled, plan.kill_switch_engaged
    INTO request_state, gate_state, provider_enabled, kill_switch
  FROM subscriber_options_capture_authorization_requests request
  JOIN subscriber_options_capture_canary_plans plan ON plan.capture_plan_id=request.capture_plan_id
  WHERE request.authorization_request_id=NEW.authorization_request_id;
  IF NOT FOUND OR request_state <> 'pending_approval' OR gate_state <> 'disabled' OR provider_enabled OR NOT kill_switch THEN
    RAISE EXCEPTION 'named approval requires unchanged pending request and disabled capture gate';
  END IF;
  RETURN NEW;
END;
$$;
CREATE TRIGGER trg_subscriber_options_capture_named_approval_guard
BEFORE INSERT ON subscriber_options_capture_named_approvals
FOR EACH ROW EXECUTE FUNCTION subscriber_options_capture_named_approval_guard();

ALTER TABLE subscriber_options_capture_named_approvals OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_options_capture_named_approvals FROM PUBLIC;
REVOKE ALL ON FUNCTION subscriber_options_capture_named_approval_guard() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION subscriber_options_capture_named_approval_guard() TO signalops_subscriber_migrator;
