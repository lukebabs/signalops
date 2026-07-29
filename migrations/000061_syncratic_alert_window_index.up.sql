CREATE INDEX IF NOT EXISTS idx_alert_ledger_app_time
  ON alert_ledger (tenant_id, app_id, domain, use_case, last_observed_at DESC);
