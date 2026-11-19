-- Shared per-user MarketOps watchlist context. The preference is deliberately
-- separate from list membership: it contains only a selected authorized list
-- (or the all-lists union mode) and is revalidated on every gateway read.

CREATE TABLE subscriber_watchlist_context_preferences (
  tenant_id text NOT NULL,
  subject text NOT NULL,
  selection_mode text NOT NULL CHECK (selection_mode IN ('list', 'all')),
  list_id text NOT NULL DEFAULT '',
  provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, subject),
  CHECK ((selection_mode = 'all' AND list_id = '') OR (selection_mode = 'list' AND list_id <> ''))
);

ALTER TABLE subscriber_watchlist_context_preferences OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_watchlist_context_preferences FROM PUBLIC;
GRANT SELECT, INSERT, UPDATE, DELETE ON subscriber_watchlist_context_preferences TO signalops_subscriber_gateway;
ALTER TABLE subscriber_watchlist_context_preferences ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscriber_watchlist_context_preferences FORCE ROW LEVEL SECURITY;
CREATE POLICY subscriber_watchlist_context_preferences_tenant_isolation
  ON subscriber_watchlist_context_preferences
  FOR ALL TO signalops_subscriber_gateway
  USING (tenant_id = current_setting('signalops.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('signalops.tenant_id', true));
