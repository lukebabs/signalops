ALTER TABLE raw_event_ledger
  ADD COLUMN IF NOT EXISTS app_id text NOT NULL DEFAULT 'console',
  ADD COLUMN IF NOT EXISTS domain text NOT NULL DEFAULT 'custom',
  ADD COLUMN IF NOT EXISTS use_case text NOT NULL DEFAULT 'general';

ALTER TABLE normalized_event_ledger
  ADD COLUMN IF NOT EXISTS app_id text NOT NULL DEFAULT 'console',
  ADD COLUMN IF NOT EXISTS domain text NOT NULL DEFAULT 'custom',
  ADD COLUMN IF NOT EXISTS use_case text NOT NULL DEFAULT 'general';

ALTER TABLE signal_ledger
  ADD COLUMN IF NOT EXISTS app_id text NOT NULL DEFAULT 'console',
  ADD COLUMN IF NOT EXISTS domain text NOT NULL DEFAULT 'custom',
  ADD COLUMN IF NOT EXISTS use_case text NOT NULL DEFAULT 'general';

ALTER TABLE alert_ledger
  ADD COLUMN IF NOT EXISTS app_id text NOT NULL DEFAULT 'console',
  ADD COLUMN IF NOT EXISTS domain text NOT NULL DEFAULT 'custom',
  ADD COLUMN IF NOT EXISTS use_case text NOT NULL DEFAULT 'general';

ALTER TABLE insight_ledger
  ADD COLUMN IF NOT EXISTS app_id text NOT NULL DEFAULT 'console',
  ADD COLUMN IF NOT EXISTS domain text NOT NULL DEFAULT 'custom',
  ADD COLUMN IF NOT EXISTS use_case text NOT NULL DEFAULT 'general';

UPDATE raw_event_ledger SET domain = payload->>'source_domain' WHERE domain = 'custom' AND payload->>'source_domain' <> '';
UPDATE normalized_event_ledger SET domain = event->>'source_domain' WHERE domain = 'custom' AND event->>'source_domain' <> '';
UPDATE signal_ledger SET domain = source_domain WHERE domain = 'custom' AND source_domain <> '';
UPDATE alert_ledger SET domain = source_domain WHERE domain = 'custom' AND source_domain <> '';
UPDATE insight_ledger SET domain = source_domain WHERE domain = 'custom' AND source_domain <> '';

CREATE INDEX IF NOT EXISTS idx_raw_event_ledger_app_time
  ON raw_event_ledger (tenant_id, app_id, domain, use_case, observation_time DESC);

CREATE INDEX IF NOT EXISTS idx_normalized_event_app_time
  ON normalized_event_ledger (tenant_id, app_id, domain, use_case, observation_time DESC);

CREATE INDEX IF NOT EXISTS idx_signal_ledger_app_time
  ON signal_ledger (tenant_id, app_id, domain, use_case, signal_time DESC);

CREATE INDEX IF NOT EXISTS idx_alert_ledger_app_status_time
  ON alert_ledger (tenant_id, app_id, domain, use_case, status, last_observed_at DESC);

CREATE INDEX IF NOT EXISTS idx_insight_ledger_app_status_time
  ON insight_ledger (tenant_id, app_id, domain, use_case, status, observed_at DESC);
