-- Subscriber Project S3: tenant-default and subject-private list foundation.
-- These private preference tables are feature-gated at the gateway. They never
-- own market data and reference only the platform-owned global asset identity.

CREATE TABLE subscriber_watchlists (
  list_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  list_kind text NOT NULL CHECK (list_kind IN ('tenant_default', 'private')),
  owner_subject text NOT NULL DEFAULT '',
  list_name text NOT NULL CHECK (char_length(list_name) BETWEEN 1 AND 120),
  created_by_subject text NOT NULL,
  updated_by_subject text NOT NULL,
  provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  CHECK (
    (list_kind = 'tenant_default' AND owner_subject = '') OR
    (list_kind = 'private' AND owner_subject <> '')
  ),
  UNIQUE (tenant_id, list_id)
);

-- A tenant has one durable shared default list. A private owner cannot create
-- duplicate names that differ only by case within the same tenant.
CREATE UNIQUE INDEX subscriber_watchlists_one_tenant_default
  ON subscriber_watchlists (tenant_id)
  WHERE list_kind = 'tenant_default';
CREATE UNIQUE INDEX subscriber_watchlists_private_owner_name
  ON subscriber_watchlists (tenant_id, owner_subject, lower(list_name))
  WHERE list_kind = 'private';
CREATE INDEX subscriber_watchlists_tenant_owner
  ON subscriber_watchlists (tenant_id, list_kind, owner_subject, updated_at DESC);

CREATE TABLE subscriber_watchlist_memberships (
  tenant_id text NOT NULL,
  list_id text NOT NULL,
  global_asset_id text NOT NULL,
  added_by_subject text NOT NULL,
  provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  added_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (list_id, global_asset_id),
  FOREIGN KEY (tenant_id, list_id)
    REFERENCES subscriber_watchlists (tenant_id, list_id) ON DELETE RESTRICT,
  FOREIGN KEY (global_asset_id)
    REFERENCES subscriber_global_assets (global_asset_id) ON DELETE RESTRICT
);
CREATE INDEX subscriber_watchlist_memberships_tenant_list
  ON subscriber_watchlist_memberships (tenant_id, list_id, added_at DESC);

CREATE TABLE subscriber_watchlist_audit (
  audit_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  list_id text NOT NULL,
  actor_subject text NOT NULL,
  mutation text NOT NULL CHECK (mutation IN ('create_list', 'rename_list', 'add_asset', 'remove_asset')),
  global_asset_id text NOT NULL DEFAULT '',
  before_value jsonb NOT NULL DEFAULT '{}'::jsonb,
  after_value jsonb NOT NULL DEFAULT '{}'::jsonb,
  correlation_id text NOT NULL DEFAULT '',
  occurred_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX subscriber_watchlist_audit_tenant_list_time
  ON subscriber_watchlist_audit (tenant_id, list_id, occurred_at DESC);

ALTER TABLE subscriber_watchlists OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_watchlist_memberships OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_watchlist_audit OWNER TO signalops_subscriber_migrator;

REVOKE ALL ON subscriber_watchlists, subscriber_watchlist_memberships, subscriber_watchlist_audit FROM PUBLIC;
GRANT SELECT, INSERT, UPDATE, DELETE ON subscriber_watchlists, subscriber_watchlist_memberships, subscriber_watchlist_audit TO signalops_subscriber_gateway;
-- A list can reference a verified global ID without gaining global-catalog read access.
GRANT REFERENCES (global_asset_id) ON subscriber_global_assets TO signalops_subscriber_gateway;

ALTER TABLE subscriber_watchlists ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscriber_watchlists FORCE ROW LEVEL SECURITY;
ALTER TABLE subscriber_watchlist_memberships ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscriber_watchlist_memberships FORCE ROW LEVEL SECURITY;
ALTER TABLE subscriber_watchlist_audit ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscriber_watchlist_audit FORCE ROW LEVEL SECURITY;

CREATE POLICY subscriber_watchlists_tenant_isolation
  ON subscriber_watchlists
  FOR ALL TO signalops_subscriber_gateway
  USING (tenant_id = current_setting('signalops.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('signalops.tenant_id', true));

CREATE POLICY subscriber_watchlist_memberships_tenant_isolation
  ON subscriber_watchlist_memberships
  FOR ALL TO signalops_subscriber_gateway
  USING (tenant_id = current_setting('signalops.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('signalops.tenant_id', true));

CREATE POLICY subscriber_watchlist_audit_tenant_isolation
  ON subscriber_watchlist_audit
  FOR ALL TO signalops_subscriber_gateway
  USING (tenant_id = current_setting('signalops.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('signalops.tenant_id', true));
