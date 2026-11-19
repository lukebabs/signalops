-- Subscriber refund request ledger and identity-label repair.
-- Subscribers may request refunds; only subscription administrators can action them.

CREATE TABLE IF NOT EXISTS subscriber_refund_requests (
  refund_request_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  subject text NOT NULL,
  subscription_id text NOT NULL DEFAULT '',
  stripe_customer_id text NOT NULL DEFAULT '',
  stripe_subscription_id text NOT NULL DEFAULT '',
  stripe_session_id text NOT NULL DEFAULT '',
  requested_amount_cents integer CHECK (requested_amount_cents IS NULL OR requested_amount_cents >= 0),
  currency text NOT NULL DEFAULT 'usd',
  reason text NOT NULL DEFAULT '',
  status text NOT NULL CHECK (status IN ('requested','reviewing','approved_for_manual_refund','rejected','manual_refund_completed','closed')) DEFAULT 'requested',
  admin_note text NOT NULL DEFAULT '',
  requested_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  resolved_at timestamptz,
  actor_subject text NOT NULL DEFAULT '',
  correlation_id text NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_subscriber_refund_requests_tenant_status
  ON subscriber_refund_requests (tenant_id, status, requested_at DESC);
CREATE INDEX IF NOT EXISTS idx_subscriber_refund_requests_tenant_subject
  ON subscriber_refund_requests (tenant_id, subject, requested_at DESC);

ALTER TABLE subscriber_refund_requests OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_refund_requests FROM PUBLIC;
GRANT SELECT, INSERT, UPDATE ON subscriber_refund_requests TO signalops_subscriber_gateway;

ALTER TABLE subscriber_refund_requests ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscriber_refund_requests FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS subscriber_refund_requests_tenant_scope ON subscriber_refund_requests;
CREATE POLICY subscriber_refund_requests_tenant_scope
  ON subscriber_refund_requests
  USING (tenant_id = current_setting('signalops.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('signalops.tenant_id', true));

CREATE OR REPLACE FUNCTION subscriber_subscription_admin_identity_labels(p_tenant_id text)
RETURNS TABLE(subject text, display_name text, email text)
LANGUAGE sql
SECURITY DEFINER
SET search_path = public
AS $$
  WITH visible_subjects AS (
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

INSERT INTO schema_migrations (version)
VALUES ('000166_subscriber_refund_requests')
ON CONFLICT (version) DO NOTHING;
