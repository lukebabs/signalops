DELETE FROM retention_runs
WHERE policy_id = 'subscriber.user_activity_180d'
  AND tenant_id IN ('tenant-local', 'tenant-pilot-b');

DELETE FROM retention_policies
WHERE policy_id = 'subscriber.user_activity_180d'
  AND tenant_id IN ('tenant-local', 'tenant-pilot-b');
