-- The production K8s FMP worker authenticates as the non-login runtime role.
-- Keep its write surface limited to annual-financial task/evidence tables.
GRANT USAGE ON SCHEMA public TO signalops_subscriber_global_eod_runtime;
GRANT SELECT, INSERT, UPDATE ON subscriber_global_annual_financial_workflows,
  subscriber_global_annual_financial_tasks,
  subscriber_global_marketops_evidence_runs,
  subscriber_global_marketops_evidence_records
  TO signalops_subscriber_global_eod_runtime;
GRANT SELECT ON subscriber_global_assets, subscriber_global_warm_eod_assets
  TO signalops_subscriber_global_eod_runtime;

INSERT INTO schema_migrations (version, applied_at)
VALUES ('000179_subscriber_global_annual_financial_runtime_grants', now())
ON CONFLICT (version) DO NOTHING;
