-- This append-only evidence migration is intentionally irreversible once
-- Option or Risk/Reward records have been materialized. Restore a pre-migration
-- backup rather than dropping global evidence or widening a tenant fallback.
SELECT 1;
