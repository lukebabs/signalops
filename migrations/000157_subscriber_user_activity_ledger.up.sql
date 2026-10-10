CREATE TABLE IF NOT EXISTS subscriber_user_activity_events (
  activity_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  subject text NOT NULL,
  app_id text NOT NULL DEFAULT 'marketops',
  event_type text NOT NULL,
  feature_key text NOT NULL DEFAULT '',
  http_method text NOT NULL DEFAULT '',
  route_path text NOT NULL DEFAULT '',
  status_code integer NOT NULL DEFAULT 0,
  correlation_id text NOT NULL DEFAULT '',
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  occurred_at timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT subscriber_user_activity_app_check CHECK (app_id IN ('marketops')),
  CONSTRAINT subscriber_user_activity_event_check CHECK (event_type IN ('login', 'logout', 'feature_view', 'api_mutation')),
  CONSTRAINT subscriber_user_activity_status_check CHECK (status_code >= 0 AND status_code <= 599),
  CONSTRAINT subscriber_user_activity_metadata_object_check CHECK (jsonb_typeof(metadata) = 'object')
);

CREATE INDEX IF NOT EXISTS idx_subscriber_user_activity_tenant_time
  ON subscriber_user_activity_events (tenant_id, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_subscriber_user_activity_tenant_subject_time
  ON subscriber_user_activity_events (tenant_id, subject, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_subscriber_user_activity_tenant_event_time
  ON subscriber_user_activity_events (tenant_id, event_type, occurred_at DESC);
CREATE INDEX IF NOT EXISTS idx_subscriber_user_activity_feature_time
  ON subscriber_user_activity_events (tenant_id, feature_key, occurred_at DESC);

ALTER TABLE subscriber_user_activity_events OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_user_activity_events FROM PUBLIC;
GRANT SELECT, INSERT ON subscriber_user_activity_events TO signalops_subscriber_gateway;

ALTER TABLE subscriber_user_activity_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscriber_user_activity_events FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS subscriber_user_activity_gateway_tenant_scope ON subscriber_user_activity_events;
CREATE POLICY subscriber_user_activity_gateway_tenant_scope
  ON subscriber_user_activity_events
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
