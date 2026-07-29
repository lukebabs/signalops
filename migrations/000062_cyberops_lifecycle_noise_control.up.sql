CREATE TABLE IF NOT EXISTS cyberops_lifecycle_policies (
  tenant_id text NOT NULL,
  policy_id text NOT NULL,
  policy_version text NOT NULL,
  mode text NOT NULL CHECK (mode IN ('disabled','shadow','enforced')),
  selector jsonb NOT NULL DEFAULT '{}'::jsonb,
  policy_hash text NOT NULL,
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, policy_id)
);
CREATE TABLE IF NOT EXISTS cyberops_approved_services (
  tenant_id text NOT NULL, destination_ip inet NOT NULL, protocol text NOT NULL CHECK (protocol IN ('tcp','udp')), destination_port integer NOT NULL CHECK (destination_port BETWEEN 1 AND 65535), approved_by text NOT NULL, reason text NOT NULL DEFAULT '', created_at timestamptz NOT NULL DEFAULT now(), PRIMARY KEY (tenant_id, destination_ip, protocol, destination_port)
);
CREATE TABLE IF NOT EXISTS cyberops_lifecycle_episodes (
  episode_id text PRIMARY KEY, tenant_id text NOT NULL, policy_id text NOT NULL, fingerprint text NOT NULL, disposition text NOT NULL, first_observed_at timestamptz NOT NULL, last_observed_at timestamptz NOT NULL, observation_count integer NOT NULL DEFAULT 0, signal_ids text[] NOT NULL DEFAULT '{}', insight_id text NOT NULL DEFAULT '', alert_id text NOT NULL DEFAULT '', UNIQUE (tenant_id, policy_id, fingerprint)
);
CREATE TABLE IF NOT EXISTS cyberops_lifecycle_decisions (
  decision_id text PRIMARY KEY, tenant_id text NOT NULL, signal_id text NOT NULL, policy_id text NOT NULL, policy_version text NOT NULL, mode text NOT NULL, disposition text NOT NULL, reason text NOT NULL, fingerprint text NOT NULL, episode_id text NOT NULL, insight_id text NOT NULL DEFAULT '', alert_id text NOT NULL DEFAULT '', created_at timestamptz NOT NULL DEFAULT now(), UNIQUE (tenant_id, signal_id, policy_id)
);
CREATE TABLE IF NOT EXISTS cyberops_lifecycle_policy_audit (
  audit_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY, tenant_id text NOT NULL, mutation text NOT NULL, destination_ip inet, protocol text, destination_port integer, actor text NOT NULL, before_value jsonb NOT NULL DEFAULT '{}'::jsonb, after_value jsonb NOT NULL DEFAULT '{}'::jsonb, occurred_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_cyberops_lifecycle_decisions_tenant_created ON cyberops_lifecycle_decisions (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_cyberops_lifecycle_episodes_tenant_observed ON cyberops_lifecycle_episodes (tenant_id, last_observed_at DESC);
INSERT INTO cyberops_lifecycle_policies (tenant_id, policy_id, policy_version, mode, selector, policy_hash) VALUES
('tenant-local','external-deny','cyberops-lifecycle-v1','shadow','{"signal_type":"cyberops.firewall.external_deny","disposition":"record_only"}','a6ae21ac5c4c6c9f'),
('tenant-local','public-service-exposure','cyberops-lifecycle-v1','shadow','{"signal_type":"cyberops.firewall.new_public_service_exposure","approved":"record_only","unapproved":"create_or_update_insight"}','f3d94cebe3e7197e'),
('tenant-local','port-scan','cyberops-lifecycle-v1','shadow','{"signal_type":"cyberops.firewall.external_deny","threshold":10,"window_seconds":300,"disposition":"create_or_update_alert"}','09b8e9223c51d61f')
ON CONFLICT (tenant_id, policy_id) DO NOTHING;
