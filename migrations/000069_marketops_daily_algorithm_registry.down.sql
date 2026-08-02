DELETE FROM algorithm_definitions
WHERE tenant_id='tenant-local'
  AND algorithm_id IN (
    'signalops.algorithms.eroc_v6',
    'signalops.algorithms.tactical_market_posture_v1'
  );
