-- pg_dump restores set search_path to empty. The deferred policy-reference
-- trigger must qualify its own table for data-only restore safety.
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
