-- The quote-cache fallback resolves canonical symbols through the global asset
-- catalog. Keep that catalog read available only to the projection owner.
GRANT SELECT ON subscriber_global_assets TO signalops_subscriber_migrator;

