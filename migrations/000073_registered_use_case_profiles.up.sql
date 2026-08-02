-- Registered profiles are the database authorization boundary for use-case
-- grants. Application metadata supplies presentation and routing information;
-- a profile must also be registered here before it can be granted to a user.
CREATE TABLE IF NOT EXISTS registered_use_case_profiles (
  app_id text PRIMARY KEY,
  registered_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO registered_use_case_profiles (app_id)
VALUES ('marketops'), ('cyberops')
ON CONFLICT (app_id) DO NOTHING;

ALTER TABLE tenant_user_access
  DROP CONSTRAINT IF EXISTS tenant_user_access_app_id_check;

ALTER TABLE tenant_user_access
  ADD CONSTRAINT tenant_user_access_app_id_registered_fkey
  FOREIGN KEY (app_id)
  REFERENCES registered_use_case_profiles (app_id)
  ON UPDATE RESTRICT
  ON DELETE RESTRICT;
