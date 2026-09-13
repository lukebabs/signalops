-- Admin-managed Stripe billing reconciliation.
-- This does not enable customer self-service checkout or billing portal flows.

ALTER TABLE subscriber_billing_webhook_events
  DROP CONSTRAINT IF EXISTS subscriber_billing_webhook_events_processing_status_check;
ALTER TABLE subscriber_billing_webhook_events
  ADD CONSTRAINT subscriber_billing_webhook_events_processing_status_check
  CHECK (processing_status IN ('received', 'processed', 'failed', 'unmatched'));

CREATE INDEX IF NOT EXISTS idx_subscriber_billing_webhook_events_received
  ON subscriber_billing_webhook_events(received_at DESC, event_type, processing_status);

GRANT SELECT, INSERT, UPDATE ON subscriber_billing_webhook_events TO signalops_subscriber_gateway;

CREATE POLICY subscriber_subject_subscriptions_stripe_reconcile_select ON subscriber_subject_subscriptions
  FOR SELECT TO signalops_subscriber_gateway
  USING (current_setting('signalops.stripe_webhook_reconcile', true) = 'true' AND stripe_subscription_id <> '');
CREATE POLICY subscriber_subject_subscriptions_stripe_reconcile_update ON subscriber_subject_subscriptions
  FOR UPDATE TO signalops_subscriber_gateway
  USING (current_setting('signalops.stripe_webhook_reconcile', true) = 'true' AND stripe_subscription_id <> '')
  WITH CHECK (current_setting('signalops.stripe_webhook_reconcile', true) = 'true' AND stripe_subscription_id <> '');

CREATE POLICY subscriber_tenant_subscriptions_stripe_reconcile_select ON subscriber_tenant_subscriptions
  FOR SELECT TO signalops_subscriber_gateway
  USING (current_setting('signalops.stripe_webhook_reconcile', true) = 'true' AND stripe_subscription_id <> '');
CREATE POLICY subscriber_tenant_subscriptions_stripe_reconcile_update ON subscriber_tenant_subscriptions
  FOR UPDATE TO signalops_subscriber_gateway
  USING (current_setting('signalops.stripe_webhook_reconcile', true) = 'true' AND stripe_subscription_id <> '')
  WITH CHECK (current_setting('signalops.stripe_webhook_reconcile', true) = 'true' AND stripe_subscription_id <> '');
