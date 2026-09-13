CREATE TABLE subscriber_quota_reservation_audit (
  audit_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  reservation_id text NOT NULL REFERENCES subscriber_quota_reservations(reservation_id) ON DELETE RESTRICT,
  actor_subject text NOT NULL,
  mutation text NOT NULL CHECK (mutation IN ('consume', 'release')),
  before_status text NOT NULL CHECK (before_status IN ('reserved', 'consumed', 'released')),
  after_status text NOT NULL CHECK (after_status IN ('reserved', 'consumed', 'released')),
  correlation_id text NOT NULL DEFAULT '',
  occurred_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE subscriber_quota_reservation_audit OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_quota_reservation_audit FROM PUBLIC;
GRANT SELECT, INSERT, UPDATE, DELETE ON subscriber_quota_reservation_audit TO signalops_subscriber_gateway;
ALTER TABLE subscriber_quota_reservation_audit ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscriber_quota_reservation_audit FORCE ROW LEVEL SECURITY;

CREATE POLICY subscriber_quota_reservation_audit_tenant_isolation
  ON subscriber_quota_reservation_audit
  FOR ALL TO signalops_subscriber_gateway
  USING (tenant_id = current_setting('signalops.tenant_id', true))
  WITH CHECK (tenant_id = current_setting('signalops.tenant_id', true));
