-- Human-readable subscriber pricing display metadata.
-- Stripe Price IDs remain the billing authority; these fields are customer-facing
-- display copy governed by Subscription Administration.

ALTER TABLE subscriber_subscription_products
  ADD COLUMN monthly_display_price text NOT NULL DEFAULT '',
  ADD COLUMN annual_display_price text NOT NULL DEFAULT '';

UPDATE subscriber_subscription_products
SET monthly_display_price='$24.99/mo',
    annual_display_price='$249/yr'
WHERE product_key='explorer';

UPDATE subscriber_subscription_products
SET monthly_display_price='$99/mo',
    annual_display_price='$999/yr'
WHERE product_key='professional';

UPDATE subscriber_subscription_products
SET monthly_display_price='Contact Sales',
    annual_display_price='Contact Sales'
WHERE product_key='institutional';

INSERT INTO schema_migrations (version)
VALUES ('000165_subscriber_subscription_display_prices')
ON CONFLICT (version) DO NOTHING;
