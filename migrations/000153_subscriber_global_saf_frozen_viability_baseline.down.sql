DO $$
BEGIN
  RAISE EXCEPTION '000153 rollback is not automatic; SAF frozen-baseline evidence requires an approved compensating version';
END;
$$;
