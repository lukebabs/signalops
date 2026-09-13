DELETE FROM schema_migrations WHERE version='000165_subscriber_subscription_display_prices';

ALTER TABLE subscriber_subscription_products
  DROP COLUMN IF EXISTS monthly_display_price,
  DROP COLUMN IF EXISTS annual_display_price;
