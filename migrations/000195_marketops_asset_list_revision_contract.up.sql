-- Make the operational asset list a durable, versioned contract.
-- The current list may change; each workflow resolves one immutable revision.

CREATE TABLE IF NOT EXISTS marketops_asset_lists (
  list_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  app_id text NOT NULL DEFAULT 'marketops',
  list_key text NOT NULL,
  display_name text NOT NULL DEFAULT '',
  status text NOT NULL DEFAULT 'active' CHECK (status IN ('active','archived')),
  current_revision bigint NOT NULL DEFAULT 1 CHECK (current_revision > 0),
  created_at timestamptz NOT NULL DEFAULT now(),
  updated_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, list_key)
);

CREATE TABLE IF NOT EXISTS marketops_asset_list_revisions (
  list_id text NOT NULL REFERENCES marketops_asset_lists(list_id) ON DELETE CASCADE,
  revision bigint NOT NULL CHECK (revision > 0),
  membership_fingerprint text NOT NULL,
  effective_at timestamptz NOT NULL DEFAULT now(),
  created_by text NOT NULL DEFAULT 'system',
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (list_id, revision)
);

CREATE TABLE IF NOT EXISTS marketops_asset_list_members (
  list_id text NOT NULL,
  revision bigint NOT NULL,
  global_asset_id text NOT NULL,
  ticker text NOT NULL,
  rank integer,
  metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
  PRIMARY KEY (list_id, revision, global_asset_id),
  UNIQUE (list_id, revision, ticker),
  FOREIGN KEY (list_id, revision)
    REFERENCES marketops_asset_list_revisions(list_id, revision) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_marketops_asset_list_members_lookup
  ON marketops_asset_list_members (list_id, revision, rank, ticker);

-- Seed the current tenant-local operational list from the live governed view.
INSERT INTO marketops_asset_lists (list_id, tenant_id, list_key, display_name)
VALUES ('marketops-primary-tenant-local', 'tenant-local', 'primary_eod', 'MarketOps Primary Assets')
ON CONFLICT (list_id) DO UPDATE SET updated_at = now();

INSERT INTO marketops_asset_list_revisions (list_id, revision, membership_fingerprint, created_by)
SELECT 'marketops-primary-tenant-local', 1,
       md5(COALESCE(string_agg(ticker, ',' ORDER BY universe_priority, rank NULLS LAST, ticker), '')),
       'migration-000195'
FROM marketops_primary_assets
WHERE tenant_id = 'tenant-local';

INSERT INTO marketops_asset_list_members (list_id, revision, global_asset_id, ticker, rank, metadata)
SELECT 'marketops-primary-tenant-local', 1,
       'ticker:' || ticker, ticker, rank,
       COALESCE(metadata, '{}'::jsonb) || jsonb_build_object('source', 'marketops_primary_assets')
FROM marketops_primary_assets
WHERE tenant_id = 'tenant-local'
ON CONFLICT (list_id, revision, global_asset_id) DO NOTHING;

ALTER TABLE marketops_task_workflows
  ADD COLUMN IF NOT EXISTS asset_list_id text,
  ADD COLUMN IF NOT EXISTS asset_list_revision bigint,
  ADD COLUMN IF NOT EXISTS asset_list_fingerprint text;

ALTER TABLE marketops_task_items
  ADD COLUMN IF NOT EXISTS global_asset_id text,
  ADD COLUMN IF NOT EXISTS asset_list_id text,
  ADD COLUMN IF NOT EXISTS asset_list_revision bigint;

UPDATE marketops_task_workflows
SET asset_list_id = 'marketops-primary-tenant-local', asset_list_revision = 1
WHERE tenant_id = 'tenant-local' AND asset_list_id IS NULL;

UPDATE marketops_task_items
SET global_asset_id = CASE WHEN symbol <> '' THEN 'ticker:' || upper(symbol) ELSE '' END,
    asset_list_id = 'marketops-primary-tenant-local', asset_list_revision = 1
WHERE tenant_id = 'tenant-local' AND asset_list_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_marketops_task_workflows_asset_list
  ON marketops_task_workflows (tenant_id, asset_list_id, asset_list_revision, session_date);
CREATE INDEX IF NOT EXISTS idx_marketops_task_items_asset_list
  ON marketops_task_items (tenant_id, asset_list_id, asset_list_revision, session_date, task_type);

ALTER TABLE marketops_intelligence_cohort_runs
  ADD COLUMN IF NOT EXISTS asset_list_id text,
  ADD COLUMN IF NOT EXISTS asset_list_revision bigint,
  ADD COLUMN IF NOT EXISTS asset_list_fingerprint text;

UPDATE marketops_intelligence_cohort_runs
SET asset_list_id = 'marketops-primary-tenant-local', asset_list_revision = 1
WHERE tenant_id = 'tenant-local' AND asset_list_id IS NULL;

COMMENT ON TABLE marketops_asset_lists IS
  'Mutable named asset-list registry; membership changes create a new revision.';
COMMENT ON TABLE marketops_asset_list_revisions IS
  'Immutable membership snapshots used by workflows and retries.';
COMMENT ON TABLE marketops_asset_list_members IS
  'Stable asset identities in each immutable list revision; ticker is display metadata.';

CREATE OR REPLACE FUNCTION marketops_publish_primary_asset_list_revision(
  p_tenant_id text DEFAULT 'tenant-local',
  p_actor text DEFAULT 'system'
) RETURNS bigint
LANGUAGE plpgsql
AS $function$
DECLARE
  v_list_id text := 'marketops-primary-' || p_tenant_id;
  v_revision bigint;
  v_fingerprint text;
BEGIN
  SELECT current_revision + 1 INTO v_revision
  FROM marketops_asset_lists
  WHERE list_id = v_list_id AND status = 'active'
  FOR UPDATE;
  IF v_revision IS NULL THEN
    RAISE EXCEPTION 'active MarketOps asset list does not exist: %', v_list_id;
  END IF;

  SELECT md5(COALESCE(string_agg(ticker, ',' ORDER BY universe_priority, rank NULLS LAST, ticker), ''))
    INTO v_fingerprint
  FROM marketops_primary_assets
  WHERE tenant_id = p_tenant_id;

  INSERT INTO marketops_asset_list_revisions(list_id, revision, membership_fingerprint, created_by)
  VALUES (v_list_id, v_revision, v_fingerprint, COALESCE(NULLIF(p_actor, ''), 'system'));

  INSERT INTO marketops_asset_list_members(list_id, revision, global_asset_id, ticker, rank, metadata)
  SELECT v_list_id, v_revision, 'ticker:' || ticker, ticker, rank,
         COALESCE(metadata, '{}'::jsonb) || jsonb_build_object('source', 'marketops_primary_assets')
  FROM marketops_primary_assets
  WHERE tenant_id = p_tenant_id;

  UPDATE marketops_asset_lists
  SET current_revision = v_revision, updated_at = now()
  WHERE list_id = v_list_id;
  RETURN v_revision;
END;
$function$;

COMMENT ON FUNCTION marketops_publish_primary_asset_list_revision(text,text) IS
  'Publishes an immutable snapshot of the live primary list; workflows bind to the returned revision.';

DO $verify$
DECLARE
  member_count integer;
BEGIN
  SELECT count(*) INTO member_count
  FROM marketops_asset_list_members
  WHERE list_id = 'marketops-primary-tenant-local' AND revision = 1;
  IF member_count = 0 THEN
    RAISE EXCEPTION '000195 seeded an empty MarketOps primary asset list';
  END IF;
END;
$verify$;
