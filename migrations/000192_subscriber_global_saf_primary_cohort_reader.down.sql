DELETE FROM schema_migrations WHERE version = '000192_subscriber_global_saf_primary_cohort_reader';
REVOKE ALL ON FUNCTION subscriber_global_saf_benchmark_primary_members() FROM signalops_subscriber_global_eod;
DROP FUNCTION IF EXISTS subscriber_global_saf_benchmark_primary_members();
