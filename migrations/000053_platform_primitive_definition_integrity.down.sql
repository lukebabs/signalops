BEGIN;

DROP TRIGGER IF EXISTS platform_primitive_definitions_policy_types ON platform_primitive_definitions;
DROP FUNCTION IF EXISTS enforce_platform_primitive_definition_policy_types();
DROP TRIGGER IF EXISTS platform_primitive_definitions_immutable_published ON platform_primitive_definitions;
DROP FUNCTION IF EXISTS enforce_platform_primitive_definition_immutability();

ALTER TABLE platform_primitive_definitions
  DROP CONSTRAINT IF EXISTS fk_platform_primitive_quality_policy,
  DROP CONSTRAINT IF EXISTS fk_platform_primitive_retention_policy,
  DROP CONSTRAINT IF EXISTS fk_platform_primitive_lineage_policy,
  DROP COLUMN IF EXISTS quality_policy_ref,
  DROP COLUMN IF EXISTS retention_policy_ref,
  DROP COLUMN IF EXISTS lineage_policy_ref;

DROP INDEX IF EXISTS idx_platform_primitive_definitions_global_id;

COMMIT;
