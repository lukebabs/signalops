-- Display-only subscriber identity labels. This table is not an access-control source.

CREATE TABLE IF NOT EXISTS subscriber_subject_identity_labels (
  tenant_id text NOT NULL,
  subject text NOT NULL,
  display_name text NOT NULL DEFAULT '',
  email text NOT NULL DEFAULT '',
  source text NOT NULL DEFAULT 'activity_jwt' CHECK (source IN ('activity_jwt','tenant_user_access','subscription_admin','migration_backfill')),
  first_seen_at timestamptz NOT NULL DEFAULT now(),
  last_seen_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, subject),
  CONSTRAINT subscriber_subject_identity_labels_nonempty CHECK (display_name <> '' OR email <> '')
);

CREATE INDEX IF NOT EXISTS idx_subscriber_subject_identity_labels_tenant_email
  ON subscriber_subject_identity_labels (tenant_id, lower(email));

ALTER TABLE subscriber_subject_identity_labels OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_subject_identity_labels FROM PUBLIC;
GRANT SELECT, INSERT, UPDATE ON subscriber_subject_identity_labels TO signalops_subscriber_gateway;

ALTER TABLE subscriber_subject_identity_labels ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscriber_subject_identity_labels FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS subscriber_subject_identity_labels_gateway_tenant_scope ON subscriber_subject_identity_labels;
CREATE POLICY subscriber_subject_identity_labels_gateway_tenant_scope
  ON subscriber_subject_identity_labels
  FOR ALL
  TO signalops_subscriber_gateway
  USING (tenant_id = current_setting('signalops.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('signalops.tenant_id', true));

INSERT INTO subscriber_subject_identity_labels
  (tenant_id, subject, display_name, email, source, first_seen_at, last_seen_at, updated_at)
SELECT tenant_id, subject, max(display_name), lower(max(email)), 'tenant_user_access', min(granted_at), max(updated_at), now()
FROM tenant_user_access
WHERE subject <> '' AND (display_name <> '' OR email <> '')
GROUP BY tenant_id, subject
ON CONFLICT (tenant_id, subject) DO UPDATE SET
  display_name = CASE WHEN EXCLUDED.display_name <> '' THEN EXCLUDED.display_name ELSE subscriber_subject_identity_labels.display_name END,
  email = CASE WHEN EXCLUDED.email <> '' THEN EXCLUDED.email ELSE subscriber_subject_identity_labels.email END,
  source = EXCLUDED.source,
  last_seen_at = GREATEST(subscriber_subject_identity_labels.last_seen_at, EXCLUDED.last_seen_at),
  updated_at = now();

INSERT INTO subscriber_subject_identity_labels
  (tenant_id, subject, display_name, email, source, first_seen_at, last_seen_at, updated_at)
SELECT tenant_id, subject,
  max(metadata->>'subject_display_name'),
  lower(max(metadata->>'subject_email')),
  'activity_jwt', min(occurred_at), max(occurred_at), now()
FROM subscriber_user_activity_events
WHERE subject <> ''
  AND (COALESCE(metadata->>'subject_display_name', '') <> '' OR COALESCE(metadata->>'subject_email', '') <> '')
GROUP BY tenant_id, subject
ON CONFLICT (tenant_id, subject) DO UPDATE SET
  display_name = CASE WHEN EXCLUDED.display_name <> '' THEN EXCLUDED.display_name ELSE subscriber_subject_identity_labels.display_name END,
  email = CASE WHEN EXCLUDED.email <> '' THEN EXCLUDED.email ELSE subscriber_subject_identity_labels.email END,
  source = EXCLUDED.source,
  last_seen_at = GREATEST(subscriber_subject_identity_labels.last_seen_at, EXCLUDED.last_seen_at),
  updated_at = now();

CREATE OR REPLACE FUNCTION subscriber_subscription_admin_identity_labels(p_tenant_id text)
RETURNS TABLE(subject text, display_name text, email text)
LANGUAGE sql
SECURITY DEFINER
SET search_path = public
AS $$
  WITH visible_subjects AS (
    SELECT subject FROM subscriber_subject_identity_labels WHERE tenant_id = p_tenant_id AND subject <> ''
    UNION
    SELECT subject FROM tenant_user_access WHERE tenant_id = p_tenant_id AND subject <> ''
    UNION
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
    UNION
    SELECT subject FROM subscriber_refund_requests WHERE tenant_id = p_tenant_id AND subject <> ''
    UNION
    SELECT actor_subject FROM subscriber_refund_requests WHERE tenant_id = p_tenant_id AND actor_subject <> ''
  )
  SELECT visible.subject,
    COALESCE(NULLIF(label.display_name, ''), (
      SELECT access.display_name
      FROM tenant_user_access access
      WHERE access.subject = visible.subject AND access.display_name <> ''
      ORDER BY CASE WHEN access.tenant_id = p_tenant_id THEN 0 ELSE 1 END,
        CASE WHEN access.app_id = 'marketops' THEN 0 ELSE 1 END,
        access.updated_at DESC
      LIMIT 1
    ), '') AS display_name,
    COALESCE(NULLIF(label.email, ''), (
      SELECT access.email
      FROM tenant_user_access access
      WHERE access.subject = visible.subject AND access.email <> ''
      ORDER BY CASE WHEN access.tenant_id = p_tenant_id THEN 0 ELSE 1 END,
        CASE WHEN access.app_id = 'marketops' THEN 0 ELSE 1 END,
        access.updated_at DESC
      LIMIT 1
    ), '') AS email
  FROM visible_subjects visible
  LEFT JOIN subscriber_subject_identity_labels label ON label.tenant_id = p_tenant_id AND label.subject = visible.subject;
$$;

ALTER FUNCTION subscriber_subscription_admin_identity_labels(text) OWNER TO signalops;
REVOKE ALL ON FUNCTION subscriber_subscription_admin_identity_labels(text) FROM PUBLIC;
GRANT EXECUTE ON FUNCTION subscriber_subscription_admin_identity_labels(text) TO signalops_subscriber_gateway;

INSERT INTO schema_migrations (version)
VALUES ('000167_subscriber_subject_identity_labels')
ON CONFLICT (version) DO NOTHING;
