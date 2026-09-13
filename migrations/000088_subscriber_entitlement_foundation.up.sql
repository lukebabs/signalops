-- Subscriber Project entitlement provisioning foundation.
-- This schema is intentionally unreachable from browser/API routes until the
-- S0-A entitlement, workload-credential, audit, and feature-gate conditions
-- are complete.

CREATE TABLE subscriber_tenant_entitlements (
  tenant_id text PRIMARY KEY,
  provisioning_version text NOT NULL,
  product_tier text NOT NULL DEFAULT '',
  status text NOT NULL CHECK (status IN ('active', 'suspended')),
  provisioned_by text NOT NULL,
  correlation_id text NOT NULL DEFAULT '',
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE subscriber_entitlement_capabilities (
  tenant_id text NOT NULL REFERENCES subscriber_tenant_entitlements(tenant_id) ON DELETE RESTRICT,
  capability text NOT NULL CHECK (capability IN ('catalog_search', 'eod_activation', 'options_demand')),
  enabled boolean NOT NULL DEFAULT false,
  quota_limit integer NOT NULL DEFAULT 0 CHECK (quota_limit >= 0),
  PRIMARY KEY (tenant_id, capability)
);

CREATE TABLE subscriber_quota_reservations (
  reservation_id text PRIMARY KEY,
  tenant_id text NOT NULL REFERENCES subscriber_tenant_entitlements(tenant_id) ON DELETE RESTRICT,
  capability text NOT NULL CHECK (capability IN ('catalog_search', 'eod_activation', 'options_demand')),
  provisioning_version text NOT NULL,
  idempotency_key text NOT NULL,
  subject text NOT NULL,
  requested_units integer NOT NULL CHECK (requested_units > 0),
  status text NOT NULL CHECK (status IN ('reserved', 'consumed', 'released')),
  policy_version text NOT NULL,
  correlation_id text NOT NULL DEFAULT '',
  reserved_at timestamptz NOT NULL DEFAULT now(),
  released_at timestamptz,
  UNIQUE (tenant_id, capability, provisioning_version, idempotency_key)
);

CREATE INDEX idx_subscriber_quota_reservations_active
  ON subscriber_quota_reservations (tenant_id, capability, provisioning_version, status);

CREATE TABLE subscriber_entitlement_decision_audit (
  decision_id text PRIMARY KEY,
  tenant_id text NOT NULL REFERENCES subscriber_tenant_entitlements(tenant_id) ON DELETE RESTRICT,
  subject text NOT NULL,
  capability text NOT NULL CHECK (capability IN ('catalog_search', 'eod_activation', 'options_demand')),
  decision_reason text NOT NULL CHECK (decision_reason IN ('allowed', 'blocked_entitlement', 'deferred_quota', 'invalid_request')),
  requested_units integer NOT NULL CHECK (requested_units > 0),
  consumed_units integer NOT NULL CHECK (consumed_units >= 0),
  quota_limit integer NOT NULL CHECK (quota_limit >= 0),
  entitlement_version text NOT NULL,
  policy_version text NOT NULL,
  correlation_id text NOT NULL DEFAULT '',
  decision_at timestamptz NOT NULL,
  provenance jsonb NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX idx_subscriber_entitlement_decision_audit_tenant_time
  ON subscriber_entitlement_decision_audit (tenant_id, decision_at DESC);

ALTER TABLE subscriber_tenant_entitlements OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_entitlement_capabilities OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_quota_reservations OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_entitlement_decision_audit OWNER TO signalops_subscriber_migrator;

REVOKE ALL ON subscriber_tenant_entitlements, subscriber_entitlement_capabilities,
  subscriber_quota_reservations, subscriber_entitlement_decision_audit FROM PUBLIC;

GRANT SELECT, INSERT, UPDATE, DELETE ON subscriber_tenant_entitlements,
  subscriber_entitlement_capabilities, subscriber_quota_reservations,
  subscriber_entitlement_decision_audit TO signalops_subscriber_gateway;

ALTER TABLE subscriber_tenant_entitlements ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscriber_tenant_entitlements FORCE ROW LEVEL SECURITY;
ALTER TABLE subscriber_entitlement_capabilities ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscriber_entitlement_capabilities FORCE ROW LEVEL SECURITY;
ALTER TABLE subscriber_quota_reservations ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscriber_quota_reservations FORCE ROW LEVEL SECURITY;
ALTER TABLE subscriber_entitlement_decision_audit ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscriber_entitlement_decision_audit FORCE ROW LEVEL SECURITY;

CREATE POLICY subscriber_tenant_entitlements_tenant_isolation
  ON subscriber_tenant_entitlements
  FOR ALL TO signalops_subscriber_gateway
  USING (tenant_id = current_setting('signalops.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('signalops.tenant_id', true));

CREATE POLICY subscriber_entitlement_capabilities_tenant_isolation
  ON subscriber_entitlement_capabilities
  FOR ALL TO signalops_subscriber_gateway
  USING (tenant_id = current_setting('signalops.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('signalops.tenant_id', true));

CREATE POLICY subscriber_quota_reservations_tenant_isolation
  ON subscriber_quota_reservations
  FOR ALL TO signalops_subscriber_gateway
  USING (tenant_id = current_setting('signalops.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('signalops.tenant_id', true));

CREATE POLICY subscriber_entitlement_decision_audit_tenant_isolation
  ON subscriber_entitlement_decision_audit
  FOR ALL TO signalops_subscriber_gateway
  USING (tenant_id = current_setting('signalops.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('signalops.tenant_id', true));
