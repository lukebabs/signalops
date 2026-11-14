-- Persist deterministic dependency watermarks before downstream materialization.
CREATE TABLE IF NOT EXISTS marketops_dependency_watermarks (
  watermark_id text PRIMARY KEY,
  tenant_id text NOT NULL,
  session_date date NOT NULL,
  asset_list_id text NOT NULL,
  asset_list_revision bigint NOT NULL,
  asset_list_fingerprint text NOT NULL DEFAULT '',
  expected_assets integer NOT NULL CHECK (expected_assets >= 0),
  ready_assets integer NOT NULL CHECK (ready_assets >= 0),
  projected_assets integer NOT NULL CHECK (projected_assets >= 0),
  required_minimum_inputs integer NOT NULL CHECK (required_minimum_inputs > 0),
  status text NOT NULL CHECK (status IN ('ready','degraded','blocked')),
  reason text NOT NULL DEFAULT '',
  correlation_id text NOT NULL,
  observed_at timestamptz NOT NULL DEFAULT now(),
  UNIQUE (tenant_id, session_date, asset_list_id, asset_list_revision)
);

CREATE INDEX IF NOT EXISTS idx_marketops_dependency_watermarks_lookup
  ON marketops_dependency_watermarks (tenant_id, session_date DESC, observed_at DESC);

CREATE OR REPLACE FUNCTION marketops_record_dependency_watermark(
  p_watermark_id text,
  p_tenant_id text,
  p_session_date date,
  p_asset_list_id text,
  p_asset_list_revision bigint,
  p_asset_list_fingerprint text,
  p_expected_assets integer,
  p_ready_assets integer,
  p_projected_assets integer,
  p_required_minimum_inputs integer,
  p_status text,
  p_reason text,
  p_correlation_id text
) RETURNS void
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $function$
BEGIN
  IF p_tenant_id <> 'tenant-local' THEN
    RAISE EXCEPTION 'dependency watermark tenant is not permitted';
  END IF;
  IF p_asset_list_id <> 'marketops-primary-tenant-local' THEN
    RAISE EXCEPTION 'dependency watermark asset list is not permitted';
  END IF;
  IF p_status NOT IN ('ready','degraded','blocked') THEN
    RAISE EXCEPTION 'dependency watermark status is not permitted';
  END IF;

  INSERT INTO marketops_dependency_watermarks (
    watermark_id, tenant_id, session_date, asset_list_id, asset_list_revision,
    asset_list_fingerprint, expected_assets, ready_assets, projected_assets,
    required_minimum_inputs, status, reason, correlation_id
  ) VALUES (
    p_watermark_id, p_tenant_id, p_session_date, p_asset_list_id, p_asset_list_revision,
    COALESCE(p_asset_list_fingerprint, ''), p_expected_assets, p_ready_assets, p_projected_assets,
    p_required_minimum_inputs, p_status, COALESCE(p_reason, ''), p_correlation_id
  )
  ON CONFLICT (tenant_id, session_date, asset_list_id, asset_list_revision)
  DO UPDATE SET
    asset_list_fingerprint = EXCLUDED.asset_list_fingerprint,
    expected_assets = EXCLUDED.expected_assets,
    ready_assets = EXCLUDED.ready_assets,
    projected_assets = EXCLUDED.projected_assets,
    required_minimum_inputs = EXCLUDED.required_minimum_inputs,
    status = EXCLUDED.status,
    reason = EXCLUDED.reason,
    correlation_id = EXCLUDED.correlation_id,
    observed_at = now();

  INSERT INTO marketops_job_dependency_evaluations (
    evaluation_id, session_date, job_id, state, reason, dependencies, correlation_id
  ) VALUES (
    p_watermark_id, p_session_date, 'dagster.marketops_dependency_shadow_graph', p_status,
    COALESCE(p_reason, ''), jsonb_build_object(
      'asset_list_id', p_asset_list_id,
      'asset_list_revision', p_asset_list_revision,
      'expected_assets', p_expected_assets,
      'ready_assets', p_ready_assets,
      'projected_assets', p_projected_assets,
      'required_minimum_inputs', p_required_minimum_inputs
    ), p_correlation_id
  )
  ON CONFLICT (evaluation_id) DO UPDATE SET
    state = EXCLUDED.state,
    reason = EXCLUDED.reason,
    dependencies = EXCLUDED.dependencies,
    correlation_id = EXCLUDED.correlation_id,
    evaluated_at = now();
END;
$function$;

REVOKE ALL ON FUNCTION marketops_record_dependency_watermark(text,text,date,text,bigint,text,integer,integer,integer,integer,text,text,text) FROM PUBLIC;

DO $grant$
BEGIN
  IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'signalops_dagster_readonly') THEN
    GRANT SELECT ON marketops_asset_lists, marketops_asset_list_revisions, marketops_asset_list_members TO signalops_dagster_readonly;
    GRANT EXECUTE ON FUNCTION marketops_record_dependency_watermark(text,text,date,text,bigint,text,integer,integer,integer,integer,text,text,text) TO signalops_dagster_readonly;
  END IF;
END;
$grant$;
