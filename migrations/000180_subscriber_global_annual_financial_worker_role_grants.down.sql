REVOKE SELECT, INSERT, UPDATE ON subscriber_global_annual_financial_workflows,
  subscriber_global_annual_financial_tasks,
  subscriber_global_marketops_evidence_runs,
  subscriber_global_marketops_evidence_records
  FROM signalops_subscriber_global_eod;
REVOKE SELECT ON subscriber_global_assets, subscriber_global_warm_eod_assets
  FROM signalops_subscriber_global_eod;
REVOKE USAGE ON SCHEMA public FROM signalops_subscriber_global_eod;

DELETE FROM schema_migrations
WHERE version = '000180_subscriber_global_annual_financial_worker_role_grants';
