-- Allow v2 convergence opportunities to carry source/evidence lineage instead of legacy hypothesis evaluation IDs.
ALTER TABLE marketops_opportunities
  DROP CONSTRAINT IF EXISTS marketops_opportunities_hypothesis_evaluation_ids_check;

ALTER TABLE marketops_opportunities
  ADD CONSTRAINT marketops_opportunities_hypothesis_evaluation_ids_check
  CHECK (
    (version = 1 AND cardinality(hypothesis_evaluation_ids) > 0)
    OR
    (version >= 2 AND (cardinality(hypothesis_evaluation_ids) > 0 OR cardinality(supporting_evidence_ids) > 0))
  );
