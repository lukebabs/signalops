-- Add Syncratic explainability as the commercial control for interactive Ask/materialization.
-- Explorer keeps read-only deterministic/public Syncratic narratives; Professional+ can run Ask/Regenerate.
UPDATE subscriber_subscription_products
SET feature_policy = feature_policy || '{"syncratic_explainability": true}'::jsonb,
    revision = revision + 1,
    changed_by = 'migration:000159_subscriber_syncratic_explainability_feature',
    updated_at = now()
WHERE product_key IN ('professional', 'institutional')
  AND COALESCE(feature_policy->>'syncratic_explainability', 'false') <> 'true';

UPDATE subscriber_subscription_products
SET feature_policy = feature_policy - 'syncratic_explainability',
    revision = revision + 1,
    changed_by = 'migration:000159_subscriber_syncratic_explainability_feature',
    updated_at = now()
WHERE product_key = 'explorer'
  AND feature_policy ? 'syncratic_explainability';

INSERT INTO schema_migrations(version)
VALUES ('000159_subscriber_syncratic_explainability_feature')
ON CONFLICT (version) DO NOTHING;
