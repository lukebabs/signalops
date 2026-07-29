ALTER TABLE cyberops_connect_integrity_failures
  ADD COLUMN IF NOT EXISTS resolved_at timestamptz,
  ADD COLUMN IF NOT EXISTS resolution_actor text,
  ADD COLUMN IF NOT EXISTS resolution_reason text;

UPDATE cyberops_connect_integrity_failures
SET resolved_at = COALESCE(resolved_at, last_seen_at),
    resolution_actor = COALESCE(resolution_actor, 'migration-backfill'),
    resolution_reason = COALESCE(resolution_reason, 'pre-existing local acceptance disposition')
WHERE resolution_status <> 'open';

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'cyberops_connect_integrity_resolution_status_check') THEN
    ALTER TABLE cyberops_connect_integrity_failures ADD CONSTRAINT cyberops_connect_integrity_resolution_status_check CHECK (resolution_status IN ('open', 'resolved_false_positive', 'resolved_test_fixture', 'resolved_confirmed_conflict'));
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'cyberops_connect_integrity_resolution_evidence_check') THEN
    ALTER TABLE cyberops_connect_integrity_failures ADD CONSTRAINT cyberops_connect_integrity_resolution_evidence_check CHECK ((resolution_status = 'open' AND resolved_at IS NULL AND resolution_actor IS NULL AND resolution_reason IS NULL) OR (resolution_status <> 'open' AND resolved_at IS NOT NULL AND resolution_actor IS NOT NULL AND resolution_reason IS NOT NULL));
  END IF;
END
$$;

CREATE TABLE IF NOT EXISTS cyberops_connect_integrity_resolution_audit (
  audit_id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  tenant_id text NOT NULL,
  failure_id text NOT NULL REFERENCES cyberops_connect_integrity_failures(failure_id),
  previous_status text NOT NULL,
  resolution_status text NOT NULL,
  actor text NOT NULL,
  reason text NOT NULL,
  occurred_at timestamptz NOT NULL,
  CHECK (previous_status = 'open'),
  CHECK (resolution_status IN ('resolved_false_positive', 'resolved_test_fixture', 'resolved_confirmed_conflict')),
  CHECK (char_length(btrim(actor)) BETWEEN 1 AND 256),
  CHECK (char_length(btrim(reason)) BETWEEN 3 AND 1024)
);

CREATE INDEX IF NOT EXISTS idx_cyberops_connect_integrity_resolution_audit_failure ON cyberops_connect_integrity_resolution_audit (tenant_id, failure_id, occurred_at DESC);
