-- pg_dump restores set search_path to empty. The audit trigger must qualify
-- its table reference so valid data-only restores cannot fail at runtime.
CREATE OR REPLACE FUNCTION public.audit_platform_primitive_definition_mutation()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  audit_actor text;
BEGIN
  audit_actor := COALESCE(NULLIF(current_setting('signalops.audit_actor', true), ''), 'system');
  INSERT INTO public.platform_primitive_definition_audit_events (
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
