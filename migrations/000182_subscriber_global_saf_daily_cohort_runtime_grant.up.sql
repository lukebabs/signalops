-- Production gateway uses the runtime role, distinct from the migration-era
-- gateway role. Keep the daily cohort projection read-only for that role.
GRANT SELECT ON subscriber_gateway_global_saf_daily_cohort_validation TO signalops_subscriber_gateway_runtime;
