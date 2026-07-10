DROP INDEX IF EXISTS idx_signal_ledger_app_time;
DROP INDEX IF EXISTS idx_normalized_event_app_time;
DROP INDEX IF EXISTS idx_raw_event_ledger_app_time;

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
