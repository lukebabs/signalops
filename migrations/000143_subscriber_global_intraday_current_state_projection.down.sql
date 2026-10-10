-- The projection is a restricted reader over immutable evidence. Do not remove
-- it through ordinary rollback after any consumer has been enabled; restore a
-- verified pre-migration backup instead.
SELECT 1;
