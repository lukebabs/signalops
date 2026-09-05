DROP TABLE IF EXISTS subscriber_entitlement_provisioning_audit;
ALTER TABLE subscriber_entitlement_decision_audit
  ADD CONSTRAINT subscriber_entitlement_decision_audit_tenant_id_fkey
  FOREIGN KEY (tenant_id) REFERENCES subscriber_tenant_entitlements(tenant_id) ON DELETE RESTRICT NOT VALID;
