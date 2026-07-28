CREATE TABLE platform_primitive_definition_audit_events (
  event_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  tenant_id text NOT NULL,
  definition_id text NOT NULL,
  primitive_type text NOT NULL,
  definition_key text NOT NULL,
  version text NOT NULL,
  event_type text NOT NULL CHECK (event_type IN ('created', 'updated', 'status_changed')),
  actor text NOT NULL,
  previous_status text,
  result_status text NOT NULL,
  before_state jsonb,
  after_state jsonb NOT NULL,
  occurred_at timestamptz NOT NULL DEFAULT now(),
  FOREIGN KEY (tenant_id, definition_id)
    REFERENCES platform_primitive_definitions (tenant_id, definition_id)
);

CREATE INDEX idx_platform_primitive_definition_audit_events_definition
  ON platform_primitive_definition_audit_events (tenant_id, definition_id, occurred_at DESC);

CREATE FUNCTION audit_platform_primitive_definition_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  audit_actor text;
BEGIN
  audit_actor := COALESCE(NULLIF(current_setting('signalops.audit_actor', true), ''), 'system');
  INSERT INTO platform_primitive_definition_audit_events (
    tenant_id, definition_id, primitive_type, definition_key, version,
    event_type, actor, previous_status, result_status, before_state, after_state
  ) VALUES (
    NEW.tenant_id,
    NEW.definition_id,
    NEW.primitive_type,
    NEW.definition_key,
    NEW.version,
    CASE
      WHEN TG_OP = 'INSERT' THEN 'created'
      WHEN OLD.status IS DISTINCT FROM NEW.status THEN 'status_changed'
      ELSE 'updated'
    END,
    audit_actor,
    CASE WHEN TG_OP = 'UPDATE' THEN OLD.status ELSE NULL END,
    NEW.status,
    CASE
      WHEN TG_OP = 'UPDATE' THEN to_jsonb(OLD) - ARRAY['quality_policy_ref', 'retention_policy_ref', 'lineage_policy_ref']
      ELSE NULL
    END,
    to_jsonb(NEW) - ARRAY['quality_policy_ref', 'retention_policy_ref', 'lineage_policy_ref']
  );
  RETURN NEW;
END;
$$;

CREATE TRIGGER platform_primitive_definitions_audit_mutation
AFTER INSERT OR UPDATE ON platform_primitive_definitions
FOR EACH ROW
EXECUTE FUNCTION audit_platform_primitive_definition_mutation();
