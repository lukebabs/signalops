-- A down migration is intentionally unsupported after a larger immutable
-- ranking snapshot has been retained. Preserve its provenance and use a new
-- current snapshot to roll forward instead.
SELECT 1;
