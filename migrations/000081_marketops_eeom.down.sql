DROP TABLE IF EXISTS marketops_eeom_results;
DELETE FROM algorithm_definitions WHERE tenant_id='tenant-local' AND algorithm_id='signalops.algorithms.earnings_event_opportunity_v1';
