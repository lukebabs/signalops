UPDATE subscriber_subscription_products
SET feature_policy = feature_policy - 'syncratic_explainability',
    revision = revision + 1,
    changed_by = 'rollback:000159_subscriber_syncratic_explainability_feature',
    updated_at = now()
WHERE feature_policy ? 'syncratic_explainability';

DELETE FROM schema_migrations
WHERE version = '000159_subscriber_syncratic_explainability_feature';
