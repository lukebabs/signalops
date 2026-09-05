-- Temporary legacy-hot cohort and shadow comparison. It does not change the
-- existing watchlist-derived selector or any scheduler consumer.

CREATE TABLE subscriber_global_intraday_grandfathered_cohorts (
  cohort_id text PRIMARY KEY,
  source_tenant_id text NOT NULL CHECK (source_tenant_id = $$tenant-local$$),
  source_list_id text NOT NULL REFERENCES subscriber_watchlists(list_id) ON DELETE RESTRICT,
  cohort_version text NOT NULL,
  status text NOT NULL CHECK (status IN ($$active$$, $$retired$$)),
  seeded_by text NOT NULL,
  rationale text NOT NULL,
  seeded_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX subscriber_global_intraday_grandfathered_one_active
  ON subscriber_global_intraday_grandfathered_cohorts (source_tenant_id)
  WHERE status = $$active$$;

CREATE TABLE subscriber_global_intraday_grandfathered_members (
  cohort_id text NOT NULL REFERENCES subscriber_global_intraday_grandfathered_cohorts(cohort_id) ON DELETE RESTRICT,
  global_asset_id text NOT NULL REFERENCES subscriber_global_assets(global_asset_id) ON DELETE RESTRICT,
  source_ticker text NOT NULL,
  activation_state text NOT NULL CHECK (activation_state IN ($$active$$, $$deferred_catalog_ineligible$$)),
  eligibility_status_at_seed text NOT NULL CHECK (eligibility_status_at_seed IN ($$eligible$$, $$ineligible$$, $$discovered$$, $$suspended$$)),
  provenance jsonb NOT NULL DEFAULT $$ {} $$::jsonb,
  seeded_at timestamptz NOT NULL,
  PRIMARY KEY (cohort_id, global_asset_id)
);

CREATE TABLE subscriber_global_intraday_hot_shadow_runs (
  shadow_run_id text PRIMARY KEY,
  cohort_id text NOT NULL REFERENCES subscriber_global_intraday_grandfathered_cohorts(cohort_id) ON DELETE RESTRICT,
  selector_version text NOT NULL,
  execution_mode text NOT NULL CHECK (execution_mode = $$shadow_read_only$$),
  cohort_member_count integer NOT NULL CHECK (cohort_member_count >= 0),
  selector_member_count integer NOT NULL CHECK (selector_member_count >= 0),
  shared_member_count integer NOT NULL CHECK (shared_member_count >= 0),
  cohort_only_count integer NOT NULL CHECK (cohort_only_count >= 0),
  selector_only_count integer NOT NULL CHECK (selector_only_count >= 0),
  comparison_status text NOT NULL CHECK (comparison_status IN ($$match$$, $$mismatch$$)),
  membership_fingerprint text NOT NULL,
  correlation_id text NOT NULL DEFAULT $$$$,
  recorded_by text NOT NULL CHECK (recorded_by = $$subscriber-global-eod-reconciler$$),
  recorded_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  CHECK (cohort_member_count = shared_member_count + cohort_only_count),
  CHECK (selector_member_count = shared_member_count + selector_only_count),
  CHECK ((comparison_status = $$match$$ AND cohort_only_count = 0 AND selector_only_count = 0)
      OR (comparison_status = $$mismatch$$ AND (cohort_only_count > 0 OR selector_only_count > 0)))
);
CREATE TABLE subscriber_global_intraday_hot_shadow_entries (
  shadow_run_id text NOT NULL REFERENCES subscriber_global_intraday_hot_shadow_runs(shadow_run_id) ON DELETE RESTRICT,
  global_asset_id text NOT NULL REFERENCES subscriber_global_assets(global_asset_id) ON DELETE RESTRICT,
  canonical_symbol text NOT NULL,
  selection_state text NOT NULL CHECK (selection_state IN ($$shared$$, $$cohort_only$$, $$selector_only$$)),
  selector_watcher_count bigint NOT NULL DEFAULT 0 CHECK (selector_watcher_count >= 0),
  recorded_at timestamptz NOT NULL,
  PRIMARY KEY (shadow_run_id, global_asset_id)
);

DO $preflight$
DECLARE total_members integer; active_members integer;
BEGIN
  SELECT count(*), count(*) FILTER (WHERE asset.eligibility_status = $$eligible$$)
    INTO total_members, active_members
  FROM subscriber_watchlist_memberships membership
  JOIN subscriber_global_assets asset ON asset.global_asset_id = membership.global_asset_id
  WHERE membership.list_id = $$sublist-tenant-local-legacy-default$$;
  IF total_members <> 132 OR active_members <> 125 THEN
    RAISE EXCEPTION $$legacy hot cohort expects 132 preserved members / 125 eligible members; found % / %$$, total_members, active_members;
  END IF;
END;
$preflight$;

INSERT INTO subscriber_global_intraday_grandfathered_cohorts
  (cohort_id, source_tenant_id, source_list_id, cohort_version, status, seeded_by, rationale, seeded_at)
VALUES
  ($$subhotcohort-tenant-local-legacy-132-v1$$, $$tenant-local$$, $$sublist-tenant-local-legacy-default$$,
   $$subscriber-legacy-hot-grandfather-v1$$, $$active$$, $$subscriber-legacy-hot-import$$,
   $$preserve the tenant-local legacy intraday universe while the watchlist-derived selector is dual-run$$, now())
ON CONFLICT (cohort_id) DO NOTHING;

INSERT INTO subscriber_global_intraday_grandfathered_members
  (cohort_id, global_asset_id, source_ticker, activation_state, eligibility_status_at_seed, provenance, seeded_at)
SELECT $$subhotcohort-tenant-local-legacy-132-v1$$, membership.global_asset_id, asset.canonical_symbol,
  CASE WHEN asset.eligibility_status = $$eligible$$ THEN $$active$$ ELSE $$deferred_catalog_ineligible$$ END,
  asset.eligibility_status,
  jsonb_build_object($$source_list_id$$, membership.list_id, $$source_tenant_id$$, membership.tenant_id,
    $$policy$$, $$legacy-hot-grandfathered-v1$$), now()
FROM subscriber_watchlist_memberships membership
JOIN subscriber_global_assets asset ON asset.global_asset_id = membership.global_asset_id
WHERE membership.list_id = $$sublist-tenant-local-legacy-default$$
ON CONFLICT (cohort_id, global_asset_id) DO NOTHING;

CREATE VIEW subscriber_global_grandfathered_hot_intraday_assets WITH (security_barrier = true) AS
SELECT member.global_asset_id, asset.canonical_symbol, cohort.cohort_id, member.source_ticker,
  member.activation_state, member.eligibility_status_at_seed, member.seeded_at
FROM subscriber_global_intraday_grandfathered_cohorts cohort
JOIN subscriber_global_intraday_grandfathered_members member ON member.cohort_id = cohort.cohort_id
JOIN subscriber_global_assets asset ON asset.global_asset_id = member.global_asset_id
WHERE cohort.status = $$active$$
  AND member.activation_state = $$active$$
  AND asset.eligibility_status = $$eligible$$;

CREATE FUNCTION subscriber_global_record_hot_intraday_shadow_run(p_correlation_id text, p_require_match boolean DEFAULT false)
RETURNS TABLE (shadow_run_id text, comparison_status text, cohort_members integer, selector_members integer, shared_members integer, cohort_only integer, selector_only integer)
LANGUAGE plpgsql
AS $fn$
DECLARE cohort text; run_id text; now_at timestamptz := now(); fingerprint text;
BEGIN
  SELECT cohort_id INTO cohort FROM subscriber_global_intraday_grandfathered_cohorts
  WHERE status = $$active$$ ORDER BY seeded_at DESC, cohort_id DESC LIMIT 1;
  IF cohort IS NULL THEN RAISE EXCEPTION $$no active legacy hot cohort$$; END IF;
  CREATE TEMP TABLE hot_shadow_source ON COMMIT DROP AS
  WITH cohort_members AS (SELECT global_asset_id, canonical_symbol FROM subscriber_global_grandfathered_hot_intraday_assets),
  selector_members AS (SELECT global_asset_id, canonical_symbol, watcher_count FROM subscriber_global_hot_intraday_assets)
  SELECT COALESCE(c.global_asset_id,s.global_asset_id) AS global_asset_id,
    COALESCE(c.canonical_symbol,s.canonical_symbol) AS canonical_symbol,
    CASE WHEN c.global_asset_id IS NOT NULL AND s.global_asset_id IS NOT NULL THEN $$shared$$
         WHEN c.global_asset_id IS NOT NULL THEN $$cohort_only$$ ELSE $$selector_only$$ END AS selection_state,
    COALESCE(s.watcher_count,0)::bigint AS selector_watcher_count
  FROM cohort_members c FULL JOIN selector_members s USING (global_asset_id);
  SELECT md5(COALESCE(string_agg(global_asset_id || $$:$$ || selection_state || $$:$$ || selector_watcher_count::text, $$|$$ ORDER BY global_asset_id), $$$$)) INTO fingerprint FROM hot_shadow_source;
  SELECT $$subhotshadow-$$ || md5(cohort || $$:$$ || now_at::text || $$:$$ || fingerprint) INTO run_id;
  SELECT count(*) FILTER (WHERE selection_state IN ($$shared$$,$$cohort_only$$)), count(*) FILTER (WHERE selection_state IN ($$shared$$,$$selector_only$$)), count(*) FILTER (WHERE selection_state=$$shared$$), count(*) FILTER (WHERE selection_state=$$cohort_only$$), count(*) FILTER (WHERE selection_state=$$selector_only$$)
    INTO cohort_members, selector_members, shared_members, cohort_only, selector_only FROM hot_shadow_source;
  comparison_status := CASE WHEN cohort_only=0 AND selector_only=0 THEN $$match$$ ELSE $$mismatch$$ END;
  IF p_require_match AND comparison_status <> $$match$$ THEN RAISE EXCEPTION $$hot intraday shadow mismatch: cohort_only %, selector_only %$$, cohort_only, selector_only; END IF;
  INSERT INTO subscriber_global_intraday_hot_shadow_runs VALUES (run_id, cohort, $$subscriber-watchlist-context-v1$$, $$shadow_read_only$$, cohort_members, selector_members, shared_members, cohort_only, selector_only, comparison_status, fingerprint, COALESCE(p_correlation_id,$$$$), $$subscriber-global-eod-reconciler$$, now_at, now_at);
  INSERT INTO subscriber_global_intraday_hot_shadow_entries (shadow_run_id,global_asset_id,canonical_symbol,selection_state,selector_watcher_count,recorded_at)
    SELECT run_id,global_asset_id,canonical_symbol,selection_state,selector_watcher_count,now_at FROM hot_shadow_source;
  shadow_run_id := run_id;
  RETURN NEXT;
END;
$fn$;

ALTER TABLE subscriber_global_intraday_grandfathered_cohorts OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_global_intraday_grandfathered_members OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_global_intraday_hot_shadow_runs OWNER TO signalops_subscriber_migrator;
ALTER TABLE subscriber_global_intraday_hot_shadow_entries OWNER TO signalops_subscriber_migrator;
ALTER VIEW subscriber_global_grandfathered_hot_intraday_assets OWNER TO signalops_subscriber_migrator;
ALTER FUNCTION subscriber_global_record_hot_intraday_shadow_run(text,boolean) OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON subscriber_global_intraday_grandfathered_cohorts, subscriber_global_intraday_grandfathered_members, subscriber_global_intraday_hot_shadow_runs, subscriber_global_intraday_hot_shadow_entries FROM PUBLIC;
REVOKE ALL ON subscriber_global_grandfathered_hot_intraday_assets FROM PUBLIC;
REVOKE ALL ON FUNCTION subscriber_global_record_hot_intraday_shadow_run(text,boolean) FROM PUBLIC;
GRANT SELECT ON subscriber_global_intraday_grandfathered_cohorts, subscriber_global_intraday_grandfathered_members, subscriber_global_intraday_hot_shadow_runs, subscriber_global_intraday_hot_shadow_entries, subscriber_global_grandfathered_hot_intraday_assets TO signalops_subscriber_global_eod;
GRANT INSERT ON subscriber_global_intraday_hot_shadow_runs, subscriber_global_intraday_hot_shadow_entries TO signalops_subscriber_global_eod;
GRANT EXECUTE ON FUNCTION subscriber_global_record_hot_intraday_shadow_run(text,boolean) TO signalops_subscriber_global_eod;
