-- Remove only the registration row; dependent access grants must be revoked
-- before a down migration is applied.
DELETE FROM registered_use_case_profiles
WHERE app_id = 'narrativeops'
  AND NOT EXISTS (SELECT 1 FROM tenant_user_access WHERE app_id = 'narrativeops');
