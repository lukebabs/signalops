-- The reconciliation changes only governed catalog-reference metadata, but
-- SAF v3 rows may already be immutable audit evidence. Rollback is therefore
-- intentionally manual and requires an approved replacement calculation.
DO $$
BEGIN
  RAISE EXCEPTION '000151 rollback is not automatic; preserve immutable SAF benchmark evidence and apply an approved compensating migration';
END;
$$;
