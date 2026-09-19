-- The annual-financial worker connects with the bounded eod worker role and
-- then SET ROLEs to this role before touching the global annual-financial
-- workflow tables. Keep the grant limited to its task/evidence surface.
GRANT USAGE ON SCHEMA public TO signalops_subscriber_global_eod;
GRANT SELECT, INSERT, UPDATE ON subscriber_global_annual_financial_workflows,
  subscriber_global_annual_financial_tasks,
  subscriber_global_marketops_evidence_runs,
  subscriber_global_marketops_evidence_records
  TO signalops_subscriber_global_eod;
GRANT SELECT ON subscriber_global_assets, subscriber_global_warm_eod_assets
  TO signalops_subscriber_global_eod;

INSERT INTO schema_migrations (version, applied_at)
VALUES ('000180_subscriber_global_annual_financial_worker_role_grants', now())
ON CONFLICT (version) DO NOTHING;
