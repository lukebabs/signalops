CREATE TABLE IF NOT EXISTS tenant_user_access (
  tenant_id text NOT NULL,
  subject text NOT NULL,
  display_name text NOT NULL DEFAULT '',
  email text NOT NULL DEFAULT '',
  app_id text NOT NULL CHECK (app_id IN ('marketops','cyberops')),
  permission text NOT NULL CHECK (permission IN ('read','write')),
  granted_by text NOT NULL,
  granted_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, subject, app_id)
);
CREATE INDEX IF NOT EXISTS tenant_user_access_tenant_subject_idx ON tenant_user_access (tenant_id, subject);
CREATE TABLE IF NOT EXISTS tenant_user_access_audit (
  audit_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  tenant_id text NOT NULL, subject text NOT NULL, app_id text NOT NULL,
  mutation text NOT NULL CHECK (mutation IN ('grant','update','revoke')),
  actor_subject text NOT NULL, actor_display_name text NOT NULL DEFAULT '',
  before_value jsonb NOT NULL DEFAULT '{}'::jsonb, after_value jsonb NOT NULL DEFAULT '{}'::jsonb,
  occurred_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS tenant_user_access_audit_tenant_subject_idx ON tenant_user_access_audit (tenant_id, subject, occurred_at DESC);
