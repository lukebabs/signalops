BEGIN;

-- Repair the two original policy IDs so every seeded policy reference resolves to
-- the canonical opaque identifier used by the registry.
UPDATE platform_primitive_definitions
SET definition_id = CASE definition_key
  WHEN 'signalops.quality_states' THEN 'policy_signalops_quality_states_v1'
  WHEN 'signalops.complete_lineage' THEN 'policy_signalops_lineage_v1'
  ELSE definition_id
END
WHERE tenant_id = 'tenant-local'
  AND primitive_type = 'policy'
  AND definition_key IN ('signalops.quality_states', 'signalops.complete_lineage');

-- Backfill every referenced policy as a typed policy definition before adding the
-- deferred reference constraints below.
WITH policy_references AS (
  SELECT tenant_id, quality_policy_id AS policy_id FROM platform_primitive_definitions WHERE quality_policy_id <> ''
  UNION
  SELECT tenant_id, retention_policy_id FROM platform_primitive_definitions WHERE retention_policy_id <> ''
  UNION
  SELECT tenant_id, lineage_policy_id FROM platform_primitive_definitions WHERE lineage_policy_id <> ''
)
INSERT INTO platform_primitive_definitions (
  tenant_id, primitive_type, definition_id, definition_key, version, app_id, domain, use_case,
  status, implementation_ref, point_in_time_required, contract, metadata
)
SELECT
  tenant_id,
  'policy',
  policy_id,
  CASE policy_id
    WHEN 'policy_signalops_quality_states_v1' THEN 'signalops.quality_states'
    WHEN 'policy_signalops_lineage_v1' THEN 'signalops.complete_lineage'
    ELSE substr(policy_id, length('policy_') + 1)
  END,
  '1.0.0',
  CASE WHEN policy_id LIKE 'policy_marketops_%' THEN 'marketops' ELSE '' END,
  CASE WHEN policy_id LIKE 'policy_marketops_%' THEN 'markets' ELSE '' END,
  CASE WHEN policy_id LIKE 'policy_marketops_%' THEN 'daily_market_surveillance' ELSE '' END,
  'active',
  'registry:policy_reference:v1',
  true,
  jsonb_build_object(
    'policy_id', policy_id,
    'policy_class', CASE
      WHEN policy_id LIKE '%quality%' THEN 'quality_gate'
      WHEN policy_id LIKE '%retention%' THEN 'retention'
      WHEN policy_id LIKE '%lineage%' THEN 'lineage'
      ELSE 'governance'
    END
  ),
  jsonb_build_object('registry_seed', 'platform_primitive_definitions_v1')
FROM policy_references
ON CONFLICT (tenant_id, primitive_type, definition_key, version) DO NOTHING;

CREATE UNIQUE INDEX idx_platform_primitive_definitions_global_id
  ON platform_primitive_definitions (definition_id);

ALTER TABLE platform_primitive_definitions
  ADD COLUMN quality_policy_ref text GENERATED ALWAYS AS (NULLIF(quality_policy_id, '')) STORED,
  ADD COLUMN retention_policy_ref text GENERATED ALWAYS AS (NULLIF(retention_policy_id, '')) STORED,
  ADD COLUMN lineage_policy_ref text GENERATED ALWAYS AS (NULLIF(lineage_policy_id, '')) STORED,
  ADD CONSTRAINT fk_platform_primitive_quality_policy
    FOREIGN KEY (tenant_id, quality_policy_ref)
    REFERENCES platform_primitive_definitions (tenant_id, definition_id)
    DEFERRABLE INITIALLY DEFERRED,
  ADD CONSTRAINT fk_platform_primitive_retention_policy
    FOREIGN KEY (tenant_id, retention_policy_ref)
    REFERENCES platform_primitive_definitions (tenant_id, definition_id)
    DEFERRABLE INITIALLY DEFERRED,
  ADD CONSTRAINT fk_platform_primitive_lineage_policy
    FOREIGN KEY (tenant_id, lineage_policy_ref)
    REFERENCES platform_primitive_definitions (tenant_id, definition_id)
    DEFERRABLE INITIALLY DEFERRED;

CREATE FUNCTION enforce_platform_primitive_definition_immutability()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  IF OLD.status IN ('active', 'deprecated', 'retired') THEN
    IF ROW(
      NEW.tenant_id, NEW.primitive_type, NEW.definition_id, NEW.definition_key, NEW.version,
      NEW.app_id, NEW.domain, NEW.use_case, NEW.schema_ref, NEW.implementation_ref,
      NEW.quality_policy_id, NEW.retention_policy_id, NEW.lineage_policy_id,
      NEW.point_in_time_required, NEW.contract, NEW.metadata
    ) IS DISTINCT FROM ROW(
      OLD.tenant_id, OLD.primitive_type, OLD.definition_id, OLD.definition_key, OLD.version,
      OLD.app_id, OLD.domain, OLD.use_case, OLD.schema_ref, OLD.implementation_ref,
      OLD.quality_policy_id, OLD.retention_policy_id, OLD.lineage_policy_id,
      OLD.point_in_time_required, OLD.contract, OLD.metadata
    ) THEN
      RAISE EXCEPTION 'published platform primitive definition % is immutable', OLD.definition_id
        USING ERRCODE = '55000';
    END IF;

    IF (OLD.status = 'active' AND NEW.status NOT IN ('active', 'deprecated', 'retired'))
      OR (OLD.status = 'deprecated' AND NEW.status NOT IN ('deprecated', 'retired'))
      OR (OLD.status = 'retired' AND NEW.status <> 'retired') THEN
      RAISE EXCEPTION 'invalid lifecycle transition from % to %', OLD.status, NEW.status
        USING ERRCODE = '55000';
    END IF;
  END IF;
  RETURN NEW;
END;
$$;

CREATE TRIGGER platform_primitive_definitions_immutable_published
BEFORE UPDATE ON platform_primitive_definitions
FOR EACH ROW
EXECUTE FUNCTION enforce_platform_primitive_definition_immutability();

CREATE FUNCTION enforce_platform_primitive_definition_policy_types()
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
      FROM platform_primitive_definitions policy
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
      FROM platform_primitive_definitions dependent
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

CREATE CONSTRAINT TRIGGER platform_primitive_definitions_policy_types
AFTER INSERT OR UPDATE ON platform_primitive_definitions
DEFERRABLE INITIALLY DEFERRED
FOR EACH ROW
EXECUTE FUNCTION enforce_platform_primitive_definition_policy_types();

COMMIT;
