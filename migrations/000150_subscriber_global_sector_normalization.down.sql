-- This migration records provider-backed catalog classification evidence and
-- updates the live global catalog. Rolling it back would erase audit evidence
-- while leaving references ambiguous, so it is intentionally blocked.
DO $$
BEGIN
  RAISE EXCEPTION '000150 is append-only catalog governance and cannot be rolled back automatically';
END;
$$;
