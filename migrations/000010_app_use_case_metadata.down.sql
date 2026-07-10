DROP INDEX IF EXISTS idx_insight_ledger_app_status_time;
DROP INDEX IF EXISTS idx_alert_ledger_app_status_time;
DROP INDEX IF EXISTS idx_signal_ledger_app_time;
DROP INDEX IF EXISTS idx_normalized_event_app_time;
DROP INDEX IF EXISTS idx_raw_event_ledger_app_time;

ALTER TABLE insight_ledger
  DROP COLUMN IF EXISTS use_case,
  DROP COLUMN IF EXISTS domain,
  DROP COLUMN IF EXISTS app_id;

ALTER TABLE alert_ledger
  DROP COLUMN IF EXISTS use_case,
  DROP COLUMN IF EXISTS domain,
  DROP COLUMN IF EXISTS app_id;

ALTER TABLE signal_ledger
  DROP COLUMN IF EXISTS use_case,
  DROP COLUMN IF EXISTS domain,
  DROP COLUMN IF EXISTS app_id;

ALTER TABLE normalized_event_ledger
  DROP COLUMN IF EXISTS use_case,
  DROP COLUMN IF EXISTS domain,
  DROP COLUMN IF EXISTS app_id;

ALTER TABLE raw_event_ledger
  DROP COLUMN IF EXISTS use_case,
  DROP COLUMN IF EXISTS domain,
  DROP COLUMN IF EXISTS app_id;
