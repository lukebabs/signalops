-- The S&P 200 union is an additive operational promotion. It must not be
-- rolled back automatically because activation requests, task evidence, and
-- historical observations may already reference the promoted assets.
DO $$
BEGIN
  RAISE EXCEPTION '000189 rollback is not automatic; preserve S&P 200 union evidence and apply an approved compensating migration';
END
$$;
