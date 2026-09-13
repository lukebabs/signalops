-- Restore the v4-preferred projection from migration 000164.
-- This is intentionally blocked here because applying it safely requires
-- checking currently materialized v5 evidence and dependent readers.
DO $$
BEGIN
  RAISE EXCEPTION '000171 rollback is not automatic; apply an approved compensating projection migration';
END;
$$;
