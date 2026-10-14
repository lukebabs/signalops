-- This migration changes governed canonical projection metadata. Automatic
-- rollback would restate projection history; use an approved compensating
-- migration instead.
DO $$
BEGIN
  RAISE EXCEPTION '000170 rollback is not automatic; preserve immutable evidence and apply an approved compensating migration';
END;
$$;
