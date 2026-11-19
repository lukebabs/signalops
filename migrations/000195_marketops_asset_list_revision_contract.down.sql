DROP INDEX IF EXISTS idx_marketops_task_items_asset_list;
DROP INDEX IF EXISTS idx_marketops_task_workflows_asset_list;
ALTER TABLE marketops_intelligence_cohort_runs
  DROP COLUMN IF EXISTS asset_list_fingerprint,
  DROP COLUMN IF EXISTS asset_list_revision,
  DROP COLUMN IF EXISTS asset_list_id;
ALTER TABLE marketops_task_items
  DROP COLUMN IF EXISTS asset_list_revision,
  DROP COLUMN IF EXISTS asset_list_id,
  DROP COLUMN IF EXISTS global_asset_id;
ALTER TABLE marketops_task_workflows
  DROP COLUMN IF EXISTS asset_list_fingerprint,
  DROP COLUMN IF EXISTS asset_list_revision,
  DROP COLUMN IF EXISTS asset_list_id;
DROP TABLE IF EXISTS marketops_asset_list_members;
DROP TABLE IF EXISTS marketops_asset_list_revisions;
DROP TABLE IF EXISTS marketops_asset_lists;
