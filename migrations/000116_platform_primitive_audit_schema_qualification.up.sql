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

-- The deferred policy-reference trigger must also resolve its own table when
-- pg_dump restores with an empty search_path.
CREATE OR REPLACE FUNCTION public.enforce_platform_primitive_definition_policy_types()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  policy_id text;
BEGIN
  FOREACH policy_id IN ARRAY ARRAY[NEW.quality_policy_id, NEW.retention_policy_id, NEW.lineage_policy_id]
  LOOP
    IF policy_id <> '' AND NOT EXISTS (
      SELECT 1
      FROM public.platform_primitive_definitions policy
      WHERE policy.tenant_id = NEW.tenant_id
        AND policy.primitive_type = 'policy'
        AND policy.definition_id = policy_id
    ) THEN
      RAISE EXCEPTION 'platform primitive policy reference % must identify a policy definition', policy_id
        USING ERRCODE = '23514';
    END IF;
  END LOOP;

  IF TG_OP = 'UPDATE'
    AND OLD.primitive_type = 'policy'
    AND NEW.primitive_type <> 'policy'
    AND EXISTS (
      SELECT 1
      FROM public.platform_primitive_definitions dependent
      WHERE dependent.tenant_id = OLD.tenant_id
        AND OLD.definition_id IN (
          dependent.quality_policy_id,
          dependent.retention_policy_id,
          dependent.lineage_policy_id
        )
    ) THEN
    RAISE EXCEPTION 'referenced policy definition % cannot change primitive type', OLD.definition_id
      USING ERRCODE = '23514';
  END IF;
  RETURN NEW;
END;
$$;
