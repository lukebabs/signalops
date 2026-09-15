-- This migration changes governed catalog identity/sector projections and may
-- already be referenced by append-only SAF benchmark observations. Automatic
-- rollback would restate projection history; use an approved compensating
-- migration instead.
DO $$
BEGIN
  RAISE EXCEPTION '000169 rollback is not automatic; preserve immutable SAF benchmark evidence and apply an approved compensating migration';
END;
$$;
