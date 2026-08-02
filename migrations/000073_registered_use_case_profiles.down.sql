ALTER TABLE tenant_user_access
  DROP CONSTRAINT IF EXISTS tenant_user_access_app_id_registered_fkey;

ALTER TABLE tenant_user_access
  ADD CONSTRAINT tenant_user_access_app_id_check
  CHECK (app_id IN ('marketops', 'cyberops'));

DROP TABLE IF EXISTS registered_use_case_profiles;
