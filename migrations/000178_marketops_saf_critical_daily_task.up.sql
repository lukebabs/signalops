-- SAF is a critical daily evidence task. Keep its dependency contract in the
-- MarketOps database so UI/task-manager state and scheduler behavior agree.
INSERT INTO marketops_job_dependency_definitions
  (dependency_id, job_id, upstream_job_id, required_statuses, max_age, version)
VALUES
  ('dep_saf_benchmark_postclose_v1', 'marketops-saf-benchmark', 'marketops-daily-postclose', ARRAY['succeeded'], interval '24 hours', 'task-manager.v2'),
  ('dep_saf_benchmark_risk_reward_v1', 'marketops-saf-benchmark', 'marketops-risk-reward', ARRAY['succeeded'], interval '24 hours', 'task-manager.v2'),
  ('dep_saf_evaluation_benchmark_v1', 'marketops-saf-evaluation', 'marketops-saf-benchmark', ARRAY['succeeded'], interval '24 hours', 'task-manager.v2'),
  ('dep_saf_evaluation_risk_reward_v1', 'marketops-saf-evaluation', 'marketops-risk-reward', ARRAY['succeeded'], interval '24 hours', 'task-manager.v2')
ON CONFLICT (job_id, upstream_job_id, version) DO UPDATE
SET required_statuses = EXCLUDED.required_statuses,
    max_age = EXCLUDED.max_age,
    active = true;

INSERT INTO schema_migrations (version, applied_at)
VALUES ('000178_marketops_saf_critical_daily_task', now())
ON CONFLICT (version) DO NOTHING;
