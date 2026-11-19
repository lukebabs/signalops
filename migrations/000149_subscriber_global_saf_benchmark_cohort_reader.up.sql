-- A constrained cohort reader is safer than granting the benchmark worker
-- direct access to all tenant watchlist memberships. It reveals only the
-- preserved tenant-local legacy-default global asset identities.
CREATE FUNCTION subscriber_global_saf_benchmark_legacy_default_members()
RETURNS TABLE(global_asset_id text)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = public
AS $$
  SELECT member.global_asset_id
  FROM subscriber_watchlist_memberships member
  WHERE member.list_id = 'sublist-tenant-local-legacy-default'
$$;

ALTER FUNCTION subscriber_global_saf_benchmark_legacy_default_members() OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON FUNCTION subscriber_global_saf_benchmark_legacy_default_members() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION subscriber_global_saf_benchmark_legacy_default_members() TO signalops_subscriber_global_eod;
