-- A distinct canary may preserve source-plan priorities other than 1 and 2.
-- Execution request ordinals remain the bounded slots 1 and 2; source priority
-- remains in immutable canary membership and execution provenance.
CREATE OR REPLACE FUNCTION subscriber_global_eod_canary_execution_member_guard()
RETURNS trigger
LANGUAGE plpgsql
AS $$
DECLARE
  frozen_symbol text;
BEGIN
  SELECT asset.canonical_symbol
    INTO frozen_symbol
  FROM subscriber_global_eod_canary_execution_plans plan
  JOIN subscriber_global_eod_canary_members member ON member.canary_run_id=plan.canary_run_id
  JOIN subscriber_global_assets asset ON asset.global_asset_id=member.global_asset_id
  WHERE plan.execution_plan_id=NEW.execution_plan_id
    AND member.global_asset_id=NEW.global_asset_id;
  IF NOT FOUND OR NEW.request_ordinal < 1 OR NEW.request_ordinal > 2
     OR frozen_symbol <> NEW.expected_symbol THEN
    RAISE EXCEPTION 'canary execution member must match the frozen symbol and bounded request slot';
  END IF;
  RETURN NEW;
END;
$$;

ALTER FUNCTION subscriber_global_eod_canary_execution_member_guard() OWNER TO signalops_subscriber_migrator;
REVOKE ALL ON FUNCTION subscriber_global_eod_canary_execution_member_guard() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION subscriber_global_eod_canary_execution_member_guard() TO signalops_subscriber_global_eod;
