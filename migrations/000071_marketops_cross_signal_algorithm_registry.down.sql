DELETE FROM algorithm_definitions
WHERE tenant_id='tenant-local' AND algorithm_id IN (
  'signalops.algorithms.options_flow_extreme_v1',
  'signalops.algorithms.convergence_opportunity_v2'
);
