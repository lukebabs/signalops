#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${SIGNALOPS_K8S_MARKETOPS_DATA_NAMESPACE:-signalops-data}"
POD="${SIGNALOPS_K8S_MARKETOPS_POSTGRES_POD:-marketops-postgres-staging-0}"
SECRET="${SIGNALOPS_K8S_MARKETOPS_POSTGRES_SECRET:-marketops-postgres-staging-auth}"
DB="${SIGNALOPS_K8S_MARKETOPS_POSTGRES_DB:-marketops}"
USER="${SIGNALOPS_K8S_MARKETOPS_POSTGRES_USER:-signalops}"

fail() {
  echo "signalops_k8s_staging_enrollment_schema_bootstrap_failed: $*" >&2
  exit 1
}

command -v kubectl >/dev/null 2>&1 || fail "kubectl is required"

password="$(kubectl get secret "$SECRET" -n "$NAMESPACE" -o jsonpath='{.data.POSTGRES_PASSWORD}' 2>/dev/null | base64 -d || true)"
[[ -n "$password" ]] || fail "could not read staging Postgres password from ${NAMESPACE}/${SECRET}"

kubectl get pod "$POD" -n "$NAMESPACE" >/dev/null 2>&1 || fail "missing staging Postgres pod ${NAMESPACE}/${POD}"

kubectl exec -i -n "$NAMESPACE" "$POD" -- env PGPASSWORD="$password" psql -v ON_ERROR_STOP=1 -h 127.0.0.1 -U "$USER" -d "$DB" <<'SQL' >/dev/null
CREATE TABLE IF NOT EXISTS schema_migrations (
  version text PRIMARY KEY,
  applied_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS tenant_user_access (
  tenant_id text NOT NULL,
  subject text NOT NULL,
  display_name text NOT NULL DEFAULT '',
  email text NOT NULL DEFAULT '',
  app_id text NOT NULL CHECK (app_id IN ('marketops','cyberops')),
  permission text NOT NULL CHECK (permission IN ('read','write')),
  granted_by text NOT NULL,
  granted_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (tenant_id, subject, app_id)
);
CREATE INDEX IF NOT EXISTS tenant_user_access_tenant_subject_idx ON tenant_user_access (tenant_id, subject);

CREATE TABLE IF NOT EXISTS tenant_user_access_audit (
  audit_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  tenant_id text NOT NULL,
  subject text NOT NULL,
  app_id text NOT NULL,
  mutation text NOT NULL CHECK (mutation IN ('grant','update','revoke')),
  actor_subject text NOT NULL,
  actor_display_name text NOT NULL DEFAULT '',
  before_value jsonb NOT NULL DEFAULT '{}'::jsonb,
  after_value jsonb NOT NULL DEFAULT '{}'::jsonb,
  occurred_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS tenant_user_access_audit_tenant_subject_idx ON tenant_user_access_audit (tenant_id, subject, occurred_at DESC);

CREATE TABLE IF NOT EXISTS subscriber_subscription_products (
  product_key text PRIMARY KEY CHECK (product_key IN ('explorer', 'professional', 'institutional')),
  billing_scope text NOT NULL CHECK (billing_scope IN ('subject', 'tenant')),
  display_name text NOT NULL,
  is_free boolean NOT NULL DEFAULT false,
  trial_days integer NOT NULL DEFAULT 0 CHECK (trial_days >= 0 AND trial_days <= 31),
  stripe_product_id text NOT NULL DEFAULT '',
  stripe_monthly_price_id text NOT NULL DEFAULT '',
  stripe_annual_price_id text NOT NULL DEFAULT '',
  monthly_display_price text NOT NULL DEFAULT '',
  annual_display_price text NOT NULL DEFAULT '',
  feature_policy jsonb NOT NULL DEFAULT '{}'::jsonb,
  limit_policy jsonb NOT NULL DEFAULT '{}'::jsonb,
  revision integer NOT NULL DEFAULT 1 CHECK (revision > 0),
  active boolean NOT NULL DEFAULT true,
  changed_by text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS subscriber_subject_subscriptions (
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

CREATE TABLE IF NOT EXISTS subscriber_tenant_subscriptions (
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

CREATE TABLE IF NOT EXISTS subscriber_subscription_seats (
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

CREATE TABLE IF NOT EXISTS subscriber_subscription_feature_decisions (
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

CREATE TABLE IF NOT EXISTS subscriber_billing_webhook_events (
  provider_event_id text PRIMARY KEY,
  event_type text NOT NULL,
  payload jsonb NOT NULL,
  received_at timestamptz NOT NULL DEFAULT now(),
  processed_at timestamptz,
  processing_status text NOT NULL CHECK (processing_status IN ('received', 'processed', 'failed')),
  error_message text NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS subscriber_subscription_audit_events (
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
  (product_key, billing_scope, display_name, is_free, trial_days, monthly_display_price, annual_display_price, feature_policy, limit_policy, changed_by)
VALUES
  ('explorer', 'subject', 'Explorer', true, 0, '$24.99/mo', '$249/yr',
   '{"market_dashboards":true,"public_signals":true,"sector_rotation_discovery":true}'::jsonb,
   '{"private_watchlists":3,"assets_per_watchlist":25}'::jsonb, 'k8s-staging-enrollment-bootstrap'),
  ('professional', 'subject', 'Professional', false, 7, '$99/mo', '$999/yr',
   '{"market_dashboards":true,"public_signals":true,"sector_rotation_discovery":true,"value_intelligence":true,"distressed_opportunity_intelligence":true,"earnings_opportunity_intelligence":true,"sector_rotation_detail":true,"options_signals":true,"earnings_calendar":true,"research_reports":true}'::jsonb,
   '{"private_watchlists":20,"assets_per_watchlist":100}'::jsonb, 'k8s-staging-enrollment-bootstrap'),
  ('institutional', 'tenant', 'Institutional', false, 0, 'Contact Sales', 'Contact Sales',
   '{"market_dashboards":true,"public_signals":true,"sector_rotation_discovery":true,"value_intelligence":true,"distressed_opportunity_intelligence":true,"earnings_opportunity_intelligence":true,"sector_rotation_detail":true,"options_signals":true,"earnings_calendar":true,"research_reports":true,"signal_assurance_analytics":true,"portfolio_analysis":true,"batch_screening":true,"historical_replay":true,"strategy_validation":true,"custom_universes":true,"api":true,"white_label":true}'::jsonb,
   '{"private_watchlists":-1,"assets_per_watchlist":-1}'::jsonb, 'k8s-staging-enrollment-bootstrap')
ON CONFLICT (product_key) DO UPDATE
SET display_name=EXCLUDED.display_name,
    monthly_display_price=EXCLUDED.monthly_display_price,
    annual_display_price=EXCLUDED.annual_display_price,
    feature_policy=EXCLUDED.feature_policy,
    limit_policy=EXCLUDED.limit_policy,
    updated_at=now();

CREATE INDEX IF NOT EXISTS idx_subscriber_subject_subscriptions_tenant_subject ON subscriber_subject_subscriptions (tenant_id, subject);
CREATE UNIQUE INDEX IF NOT EXISTS idx_subscriber_subject_subscriptions_stripe_id ON subscriber_subject_subscriptions (stripe_subscription_id) WHERE stripe_subscription_id <> '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_subscriber_tenant_subscriptions_stripe_id ON subscriber_tenant_subscriptions (stripe_subscription_id) WHERE stripe_subscription_id <> '';
CREATE INDEX IF NOT EXISTS idx_subscriber_subscription_seats_subscription ON subscriber_subscription_seats (tenant_subscription_id, status);
CREATE INDEX IF NOT EXISTS idx_subscriber_subscription_feature_decisions_tenant_time ON subscriber_subscription_feature_decisions (tenant_id, decided_at DESC);
CREATE INDEX IF NOT EXISTS idx_subscriber_subscription_audit_tenant_time ON subscriber_subscription_audit_events (tenant_id, occurred_at DESC);

INSERT INTO schema_migrations (version)
VALUES ('k8s_staging_enrollment_schema_bootstrap')
ON CONFLICT (version) DO NOTHING;
SQL

table_count="$(kubectl exec -n "$NAMESPACE" "$POD" -- env PGPASSWORD="$password" psql -h 127.0.0.1 -U "$USER" -d "$DB" -Atc "SELECT count(*) FROM pg_tables WHERE schemaname='public' AND tablename IN ('tenant_user_access','tenant_user_access_audit','subscriber_subscription_products','subscriber_subject_subscriptions','subscriber_tenant_subscriptions','subscriber_subscription_seats','subscriber_subscription_feature_decisions','subscriber_billing_webhook_events','subscriber_subscription_audit_events');")"
[[ "$table_count" == "9" ]] || fail "expected 9 enrollment tables, found ${table_count}"

cat <<EOF
signalops_k8s_staging_enrollment_schema_bootstrap_verified
namespace=${NAMESPACE}
pod=${POD}
database=${DB}
tables=${table_count}
provider_polling=false
production_cutover_allowed=false
EOF
