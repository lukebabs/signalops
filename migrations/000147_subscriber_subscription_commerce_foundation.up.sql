-- Subscription commerce foundation. This is intentionally distinct from the
-- catalog/EOD/options provider-demand entitlements introduced in 000088.
-- Commercial plans govern analytical access; they do not authorize provider
-- requests or tenant-local market-data persistence.

CREATE TABLE subscriber_subscription_products (
  product_key text PRIMARY KEY CHECK (product_key IN ('explorer', 'professional', 'institutional')),
  billing_scope text NOT NULL CHECK (billing_scope IN ('subject', 'tenant')),
  display_name text NOT NULL,
  is_free boolean NOT NULL DEFAULT false,
  trial_days integer NOT NULL DEFAULT 0 CHECK (trial_days >= 0 AND trial_days <= 31),
  stripe_product_id text NOT NULL DEFAULT '',
  stripe_monthly_price_id text NOT NULL DEFAULT '',
  stripe_annual_price_id text NOT NULL DEFAULT '',
  feature_policy jsonb NOT NULL DEFAULT '{}'::jsonb,
  limit_policy jsonb NOT NULL DEFAULT '{}'::jsonb,
  revision integer NOT NULL DEFAULT 1 CHECK (revision > 0),
  active boolean NOT NULL DEFAULT true,
  changed_by text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE subscriber_subject_subscriptions (
  subscription_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  subject text NOT NULL,
  product_key text NOT NULL REFERENCES subscriber_subscription_products(product_key) ON DELETE RESTRICT,
  status text NOT NULL CHECK (status IN ('trialing', 'active', 'past_due', 'suspended', 'canceled')),
  stripe_customer_id text NOT NULL DEFAULT '',
  stripe_subscription_id text NOT NULL DEFAULT '',
  trial_ends_at timestamptz,
  current_period_ends_at timestamptz,
  grace_ends_at timestamptz,
  canceled_at timestamptz,
  provisioned_by text NOT NULL,
  correlation_id text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, subject)
);

CREATE TABLE subscriber_tenant_subscriptions (
  subscription_id text PRIMARY KEY,
  tenant_id text NOT NULL UNIQUE,
  product_key text NOT NULL REFERENCES subscriber_subscription_products(product_key) ON DELETE RESTRICT,
  status text NOT NULL CHECK (status IN ('trialing', 'active', 'past_due', 'suspended', 'canceled')),
  stripe_customer_id text NOT NULL DEFAULT '',
  stripe_subscription_id text NOT NULL DEFAULT '',
  current_period_ends_at timestamptz,
  grace_ends_at timestamptz,
  canceled_at timestamptz,
  provisioned_by text NOT NULL,
  correlation_id text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE subscriber_subscription_seats (
  tenant_id text NOT NULL,
  subject text NOT NULL,
  tenant_subscription_id text NOT NULL REFERENCES subscriber_tenant_subscriptions(subscription_id) ON DELETE RESTRICT,
  seat_role text NOT NULL CHECK (seat_role IN ('member', 'tenant_admin')),
  status text NOT NULL CHECK (status IN ('active', 'revoked')),
  assigned_by text NOT NULL,
  correlation_id text NOT NULL DEFAULT '',
  assigned_at timestamptz NOT NULL DEFAULT now(),
  revoked_at timestamptz,
  PRIMARY KEY (tenant_id, subject),
  FOREIGN KEY (tenant_id) REFERENCES subscriber_tenant_subscriptions(tenant_id) ON DELETE RESTRICT
);

CREATE TABLE subscriber_subscription_feature_decisions (
  decision_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  subject text NOT NULL,
  feature_key text NOT NULL,
  decision text NOT NULL CHECK (decision IN ('allowed', 'blocked_subscription', 'blocked_role', 'blocked_limit', 'invalid_request')),
  product_key text NOT NULL DEFAULT '',
  subscription_id text NOT NULL DEFAULT '',
  policy_revision integer NOT NULL DEFAULT 0,
  correlation_id text NOT NULL DEFAULT '',
  provenance jsonb NOT NULL DEFAULT '{}'::jsonb,
  decided_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE subscriber_billing_webhook_events (
  provider_event_id text PRIMARY KEY,
  event_type text NOT NULL,
  payload jsonb NOT NULL,
  received_at timestamptz NOT NULL DEFAULT now(),
  processed_at timestamptz,
  processing_status text NOT NULL CHECK (processing_status IN ('received', 'processed', 'failed')),
  error_message text NOT NULL DEFAULT ''
);

CREATE TABLE subscriber_subscription_audit_events (
  audit_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  subject text NOT NULL DEFAULT '',
  subscription_id text NOT NULL DEFAULT '',
  actor_subject text NOT NULL,
  event_type text NOT NULL,
  before_state jsonb,
  after_state jsonb,
  correlation_id text NOT NULL DEFAULT '',
  occurred_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO subscriber_subscription_products
  (product_key, billing_scope, display_name, is_free, trial_days, feature_policy, limit_policy, changed_by)
VALUES
  ('explorer', 'subject', 'Explorer', true, 0,
   '{"market_dashboards":true,"public_signals":true,"sector_rotation_discovery":true}'::jsonb,
   '{"private_watchlists":3,"assets_per_watchlist":25}'::jsonb, 'subscription-foundation'),
  ('professional', 'subject', 'Professional', false, 7,
   '{"market_dashboards":true,"public_signals":true,"sector_rotation_discovery":true,"value_intelligence":true,"distressed_opportunity_intelligence":true,"earnings_opportunity_intelligence":true,"sector_rotation_detail":true,"options_signals":true,"earnings_calendar":true,"research_reports":true}'::jsonb,
   '{"private_watchlists":20,"assets_per_watchlist":100}'::jsonb, 'subscription-foundation'),
  ('institutional', 'tenant', 'Institutional', false, 0,
   '{"market_dashboards":true,"public_signals":true,"sector_rotation_discovery":true,"value_intelligence":true,"distressed_opportunity_intelligence":true,"earnings_opportunity_intelligence":true,"sector_rotation_detail":true,"options_signals":true,"earnings_calendar":true,"research_reports":true,"signal_assurance_analytics":true,"portfolio_analysis":true,"batch_screening":true,"historical_replay":true,"strategy_validation":true,"custom_universes":true,"api":true,"white_label":true}'::jsonb,
   '{"private_watchlists":-1,"assets_per_watchlist":-1}'::jsonb, 'subscription-foundation');

CREATE INDEX idx_subscriber_subject_subscriptions_tenant_subject ON subscriber_subject_subscriptions (tenant_id, subject);
CREATE UNIQUE INDEX idx_subscriber_subject_subscriptions_stripe_id ON subscriber_subject_subscriptions (stripe_subscription_id) WHERE stripe_subscription_id <> '';
CREATE UNIQUE INDEX idx_subscriber_tenant_subscriptions_stripe_id ON subscriber_tenant_subscriptions (stripe_subscription_id) WHERE stripe_subscription_id <> '';
CREATE INDEX idx_subscriber_subscription_seats_subscription ON subscriber_subscription_seats (tenant_subscription_id, status);
CREATE INDEX idx_subscriber_subscription_feature_decisions_tenant_time ON subscriber_subscription_feature_decisions (tenant_id, decided_at DESC);
CREATE INDEX idx_subscriber_subscription_audit_tenant_time ON subscriber_subscription_audit_events (tenant_id, occurred_at DESC);

ALTER TABLE subscriber_subscription_products OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_subject_subscriptions OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_tenant_subscriptions OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_subscription_seats OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_subscription_feature_decisions OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_billing_webhook_events OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_subscription_audit_events OWNER TO signalops_subscriber_migrator;

REVOKE ALL ON subscriber_subscription_products, subscriber_subject_subscriptions,
  subscriber_tenant_subscriptions, subscriber_subscription_seats,
  subscriber_subscription_feature_decisions, subscriber_billing_webhook_events,
  subscriber_subscription_audit_events FROM PUBLIC;

GRANT SELECT, INSERT, UPDATE, DELETE ON subscriber_subscription_products,
  subscriber_subject_subscriptions, subscriber_tenant_subscriptions,
  subscriber_subscription_seats, subscriber_subscription_feature_decisions,
  subscriber_subscription_audit_events TO signalops_subscriber_gateway;

ALTER TABLE subscriber_subject_subscriptions ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscriber_subject_subscriptions FORCE ROW LEVEL SECURITY;
ALTER TABLE subscriber_tenant_subscriptions ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscriber_tenant_subscriptions FORCE ROW LEVEL SECURITY;
ALTER TABLE subscriber_subscription_seats ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscriber_subscription_seats FORCE ROW LEVEL SECURITY;
ALTER TABLE subscriber_subscription_feature_decisions ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscriber_subscription_feature_decisions FORCE ROW LEVEL SECURITY;
ALTER TABLE subscriber_subscription_audit_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscriber_subscription_audit_events FORCE ROW LEVEL SECURITY;

CREATE POLICY subscriber_subject_subscriptions_tenant_isolation ON subscriber_subject_subscriptions
  FOR ALL TO signalops_subscriber_gateway
  USING (tenant_id = current_setting('signalops.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('signalops.tenant_id', true));
CREATE POLICY subscriber_tenant_subscriptions_tenant_isolation ON subscriber_tenant_subscriptions
  FOR ALL TO signalops_subscriber_gateway
  USING (tenant_id = current_setting('signalops.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('signalops.tenant_id', true));
CREATE POLICY subscriber_subscription_seats_tenant_isolation ON subscriber_subscription_seats
  FOR ALL TO signalops_subscriber_gateway
  USING (tenant_id = current_setting('signalops.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('signalops.tenant_id', true));
CREATE POLICY subscriber_subscription_feature_decisions_tenant_isolation ON subscriber_subscription_feature_decisions
  FOR ALL TO signalops_subscriber_gateway
  USING (tenant_id = current_setting('signalops.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('signalops.tenant_id', true));
CREATE POLICY subscriber_subscription_audit_events_tenant_isolation ON subscriber_subscription_audit_events
  FOR ALL TO signalops_subscriber_gateway
  USING (tenant_id = current_setting('signalops.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('signalops.tenant_id', true));
