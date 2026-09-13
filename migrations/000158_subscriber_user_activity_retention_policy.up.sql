INSERT INTO retention_policies (tenant_id, policy_id, app_id, domain, data_class, retention_days, mode, preservation_rule, description)
SELECT tenant_id,
  'subscriber.user_activity_180d',
  'marketops',
  'subscriber_administration',
  'user_activity_detail',
  180,
  'dry_run',
  'summarized_activity_before_detail_prune',
  'Subscriber user activity detail retention. Detail rows older than 180 days are candidates only until this policy is explicitly enforced after summary/analytics retention is approved.'
FROM (VALUES ('tenant-local'), ('tenant-pilot-b')) AS tenants(tenant_id)
ON CONFLICT (tenant_id, policy_id) DO UPDATE SET
  app_id = EXCLUDED.app_id,
  domain = EXCLUDED.domain,
  data_class = EXCLUDED.data_class,
  retention_days = EXCLUDED.retention_days,
  mode = CASE WHEN retention_policies.mode = 'enforced' THEN retention_policies.mode ELSE EXCLUDED.mode END,
  preservation_rule = EXCLUDED.preservation_rule,
  description = EXCLUDED.description,
  updated_at = now();
