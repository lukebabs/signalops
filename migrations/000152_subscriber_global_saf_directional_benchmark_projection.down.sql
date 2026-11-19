-- This projection changes the preferred benchmark calculation, while prior
-- SAF rows remain immutable audit evidence. Rollback is therefore
-- intentionally manual and requires an approved replacement calculation.
DO $$
BEGIN
  RAISE EXCEPTION '000152 rollback is not automatic; preserve immutable SAF benchmark evidence and apply an approved compensating calculation migration';
END;
$$;
