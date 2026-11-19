DELETE FROM schema_migrations WHERE version='000166_subscriber_refund_requests';

DROP TABLE IF EXISTS subscriber_refund_requests;

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
