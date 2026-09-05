DO $$
BEGIN
  RAISE EXCEPTION '000168 rollback is not automatic; SAF usefulness policy evidence requires an approved compensating version';
END;
$$;
