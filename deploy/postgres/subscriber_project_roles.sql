-- Subscriber Project database-role bootstrap.
--
-- Run this once with a cluster administrator before any RLS-protected
-- subscriber-private table is migrated. These are NOLOGIN group roles:
-- deployment secret management must create distinct workload login roles and
-- grant each exactly one applicable group role. This script intentionally
-- grants no access to existing MarketOps tables and changes no current service.

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'signalops_subscriber_migrator') THEN
    CREATE ROLE signalops_subscriber_migrator NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'signalops_subscriber_gateway') THEN
    CREATE ROLE signalops_subscriber_gateway NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'signalops_subscriber_catalog_sync') THEN
    CREATE ROLE signalops_subscriber_catalog_sync NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'signalops_subscriber_global_eod') THEN
    CREATE ROLE signalops_subscriber_global_eod NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'signalops_subscriber_options_demand') THEN
    CREATE ROLE signalops_subscriber_options_demand NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'signalops_subscriber_options_capture') THEN
    CREATE ROLE signalops_subscriber_options_capture NOLOGIN NOINHERIT NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS;
  END IF;
END
$$;
