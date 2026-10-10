CREATE TABLE subscriber_upgrade_interactions (
  interaction_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  subject text NOT NULL,
  app_id text NOT NULL DEFAULT 'marketops',
  interaction_type text NOT NULL CHECK (interaction_type IN ('prompt_shown', 'prompt_clicked', 'checkout_started', 'contact_sales_clicked')),
  source_feature text NOT NULL DEFAULT '',
  source_route text NOT NULL DEFAULT '',
  source_url text NOT NULL DEFAULT '',
  asset_symbol text NOT NULL DEFAULT '',
  current_tier text NOT NULL DEFAULT '',
  required_tier text NOT NULL CHECK (required_tier IN ('professional', 'institutional')),
  prompt_variant text NOT NULL DEFAULT 'contextual',
  cta_label text NOT NULL DEFAULT '',
  correlation_id text NOT NULL DEFAULT '',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  occurred_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT subscriber_upgrade_interactions_app_check CHECK (app_id IN ('marketops')),
  CONSTRAINT subscriber_upgrade_interactions_metadata_object_check CHECK (jsonb_typeof(metadata) = 'object')
);

CREATE INDEX idx_subscriber_upgrade_interactions_tenant_time
  ON subscriber_upgrade_interactions (tenant_id, occurred_at DESC);
CREATE INDEX idx_subscriber_upgrade_interactions_tenant_subject_time
  ON subscriber_upgrade_interactions (tenant_id, subject, occurred_at DESC);
CREATE INDEX idx_subscriber_upgrade_interactions_source_time
  ON subscriber_upgrade_interactions (tenant_id, source_feature, interaction_type, occurred_at DESC);

ALTER TABLE subscriber_upgrade_interactions OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_upgrade_interactions FROM PUBLIC;
GRANT SELECT, INSERT ON subscriber_upgrade_interactions TO signalops_subscriber_gateway;

ALTER TABLE subscriber_upgrade_interactions ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscriber_upgrade_interactions FORCE ROW LEVEL SECURITY;

CREATE POLICY subscriber_upgrade_interactions_gateway_tenant_scope
  ON subscriber_upgrade_interactions
  FOR ALL
  TO signalops_subscriber_gateway
  USING (tenant_id = current_setting('signalops.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('signalops.tenant_id', true));

CREATE OR REPLACE FUNCTION subscriber_subscription_admin_identity_labels(p_tenant_id text)
RETURNS TABLE(subject text, display_name text, email text)
LANGUAGE sql
SECURITY DEFINER
SET search_path = public
AS $$
  WITH visible_subjects AS (
    SELECT subject FROM subscriber_subject_subscriptions WHERE tenant_id = p_tenant_id AND subject <> ''
    UNION
    SELECT subject FROM subscriber_subscription_seats WHERE tenant_id = p_tenant_id AND subject <> ''
    UNION
    SELECT assigned_by FROM subscriber_subscription_seats WHERE tenant_id = p_tenant_id AND assigned_by <> ''
    UNION
    SELECT subject FROM subscriber_subscription_audit_events WHERE tenant_id = p_tenant_id AND subject <> ''
    UNION
    SELECT actor_subject FROM subscriber_subscription_audit_events WHERE tenant_id = p_tenant_id AND actor_subject <> ''
    UNION
    SELECT subject FROM subscriber_user_activity_events WHERE tenant_id = p_tenant_id AND subject <> ''
    UNION
    SELECT subject FROM subscriber_upgrade_interactions WHERE tenant_id = p_tenant_id AND subject <> ''
  )
  SELECT visible.subject,
    COALESCE((
      SELECT access.display_name
      FROM tenant_user_access access
      WHERE access.subject = visible.subject AND access.display_name <> ''
      ORDER BY CASE WHEN access.tenant_id = p_tenant_id THEN 0 ELSE 1 END,
        CASE WHEN access.app_id = 'marketops' THEN 0 ELSE 1 END,
        access.updated_at DESC
      LIMIT 1
    ), '') AS display_name,
    COALESCE((
      SELECT access.email
      FROM tenant_user_access access
      WHERE access.subject = visible.subject AND access.email <> ''
      ORDER BY CASE WHEN access.tenant_id = p_tenant_id THEN 0 ELSE 1 END,
        CASE WHEN access.app_id = 'marketops' THEN 0 ELSE 1 END,
        access.updated_at DESC
      LIMIT 1
    ), '') AS email
  FROM visible_subjects visible;
$$;

ALTER FUNCTION subscriber_subscription_admin_identity_labels(text) OWNER TO signalops;
REVOKE ALL ON FUNCTION subscriber_subscription_admin_identity_labels(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION subscriber_subscription_admin_identity_labels(text) TO signalops_subscriber_gateway;
