CREATE TABLE IF NOT EXISTS cyberops_iot_network_configs (
  tenant_id text PRIMARY KEY,
  internal_cidrs jsonb NOT NULL DEFAULT '[]'::jsonb,
  updated_at timestamptz NOT NULL DEFAULT now()
);
