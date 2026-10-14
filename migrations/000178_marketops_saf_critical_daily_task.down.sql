DELETE FROM marketops_job_dependency_definitions
WHERE version = 'task-manager.v2'
  AND dependency_id IN (
    'dep_saf_benchmark_postclose_v1',
    'dep_saf_benchmark_risk_reward_v1',
    'dep_saf_evaluation_benchmark_v1',
    'dep_saf_evaluation_risk_reward_v1'
  );

DELETE FROM schema_migrations
WHERE version = '000178_marketops_saf_critical_daily_task';
