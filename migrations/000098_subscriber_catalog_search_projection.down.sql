REVOKE ALL ON FUNCTION subscriber_search_global_catalog(text, integer) FROM signalops_subscriber_gateway;
DROP FUNCTION IF EXISTS subscriber_search_global_catalog(text, integer);
