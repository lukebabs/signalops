DELETE FROM algorithm_definitions WHERE tenant_id='tenant-local' AND algorithm_id IN ('signalops.algorithms.valuation_composite_v3','signalops.algorithms.distressed_opportunity_scoring_v3');
DROP TABLE IF EXISTS marketops_valuation_results;
DROP TABLE IF EXISTS marketops_valuation_snapshots;
