-- Allow durable default-deny decisions before a tenant receives an entitlement
-- and retain immutable provisioning mutations under the same forced-RLS model.

ALTER TABLE subscriber_entitlement_decision_audit
  DROP CONSTRAINT IF EXISTS subscriber_entitlement_decision_audit_tenant_id_fkey;

CREATE TABLE subscriber_entitlement_provisioning_audit (
  audit_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  actor_subject text NOT NULL,
  mutation text NOT NULL CHECK (mutation IN ('provision', 'update', 'suspend')),
  before_value jsonb NOT NULL DEFAULT '{}'::jsonb,
  after_value jsonb NOT NULL DEFAULT '{}'::jsonb,
  correlation_id text NOT NULL DEFAULT '',
  occurred_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE subscriber_entitlement_provisioning_audit OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_entitlement_provisioning_audit FROM PUBLIC;
GRANT SELECT, INSERT, UPDATE, DELETE ON subscriber_entitlement_provisioning_audit TO signalops_subscriber_gateway;
ALTER TABLE subscriber_entitlement_provisioning_audit ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscriber_entitlement_provisioning_audit FORCE ROW LEVEL SECURITY;

CREATE POLICY subscriber_entitlement_provisioning_audit_tenant_isolation
  ON subscriber_entitlement_provisioning_audit
  FOR ALL TO signalops_subscriber_gateway
  USING (tenant_id = current_setting('signalops.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('signalops.tenant_id', true));
