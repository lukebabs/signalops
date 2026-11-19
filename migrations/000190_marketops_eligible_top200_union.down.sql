DO $$
BEGIN
  RAISE EXCEPTION '000190 rollback is not automatic; preserve operational union evidence and apply an approved compensating migration';
END
$$;
