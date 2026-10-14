-- Stripe-backed subscriber activation checkout ledger.
-- Stripe receives only checkout_ref as opaque metadata; tenant and subject mapping
-- remain inside the dedicated MarketOps database.

CREATE TABLE subscriber_checkout_sessions (
  checkout_ref text PRIMARY KEY,
  tenant_id text NOT NULL,
  subject text NOT NULL,
  product_key text NOT NULL REFERENCES subscriber_subscription_products(product_key) ON DELETE RESTRICT,
  billing_period text NOT NULL CHECK (billing_period IN ('monthly', 'annual')),
  stripe_price_id text NOT NULL,
  stripe_session_id text NOT NULL DEFAULT '',
  stripe_subscription_id text NOT NULL DEFAULT '',
  status text NOT NULL CHECK (status IN ('created', 'checkout_started', 'webhook_processed', 'expired', 'failed')),
  checkout_url_returned boolean NOT NULL DEFAULT false,
  actor_subject text NOT NULL DEFAULT '',
  correlation_id text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_subscriber_checkout_sessions_tenant_subject ON subscriber_checkout_sessions (tenant_id, subject, created_at DESC);
CREATE INDEX idx_subscriber_checkout_sessions_ref_status ON subscriber_checkout_sessions (checkout_ref, status);
CREATE UNIQUE INDEX idx_subscriber_checkout_sessions_stripe_session ON subscriber_checkout_sessions (stripe_session_id) WHERE stripe_session_id <> '';
CREATE UNIQUE INDEX idx_subscriber_checkout_sessions_stripe_subscription ON subscriber_checkout_sessions (stripe_subscription_id) WHERE stripe_subscription_id <> '';

ALTER TABLE subscriber_checkout_sessions OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_checkout_sessions FROM PUBLIC;
GRANT SELECT, INSERT, UPDATE ON subscriber_checkout_sessions TO signalops_subscriber_gateway;

ALTER TABLE subscriber_checkout_sessions ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscriber_checkout_sessions FORCE ROW LEVEL SECURITY;

CREATE POLICY subscriber_checkout_sessions_tenant_isolation ON subscriber_checkout_sessions
  FOR ALL TO signalops_subscriber_gateway
  USING (tenant_id = current_setting('signalops.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('signalops.tenant_id', true));

CREATE POLICY subscriber_checkout_sessions_stripe_reconcile_select ON subscriber_checkout_sessions
  FOR SELECT TO signalops_subscriber_gateway
  USING (current_setting('signalops.stripe_webhook_reconcile', true) = 'true' AND checkout_ref <> '');

CREATE POLICY subscriber_checkout_sessions_stripe_reconcile_update ON subscriber_checkout_sessions
  FOR UPDATE TO signalops_subscriber_gateway
  USING (current_setting('signalops.stripe_webhook_reconcile', true) = 'true' AND checkout_ref <> '')
  WITH CHECK (current_setting('signalops.stripe_webhook_reconcile', true) = 'true' AND checkout_ref <> '');
