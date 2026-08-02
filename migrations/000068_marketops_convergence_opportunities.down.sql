-- Rollback is only valid after v2 records have been removed by an explicit operator action.
ALTER TABLE marketops_opportunities
  DROP CONSTRAINT IF EXISTS marketops_opportunities_hypothesis_evaluation_ids_check;

ALTER TABLE marketops_opportunities
  ADD CONSTRAINT marketops_opportunities_hypothesis_evaluation_ids_check
  CHECK (cardinality(hypothesis_evaluation_ids) > 0);
