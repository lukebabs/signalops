DROP INDEX IF EXISTS idx_cyberops_connect_integrity_resolution_audit_failure;
DROP TABLE IF EXISTS cyberops_connect_integrity_resolution_audit;
ALTER TABLE cyberops_connect_integrity_failures
  DROP CONSTRAINT IF EXISTS cyberops_connect_integrity_resolution_evidence_check,
  DROP CONSTRAINT IF EXISTS cyberops_connect_integrity_resolution_status_check,
  DROP COLUMN IF EXISTS resolution_reason,
  DROP COLUMN IF EXISTS resolution_actor,
  DROP COLUMN IF EXISTS resolved_at;
