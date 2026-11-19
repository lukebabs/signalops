REVOKE ALL ON FUNCTION subscriber_add_tenant_default_catalog_membership(text,text,text,text) FROM signalops_subscriber_gateway;
DROP FUNCTION IF EXISTS subscriber_add_tenant_default_catalog_membership(text,text,text,text);
REVOKE ALL ON FUNCTION subscriber_search_global_warm_catalog(text,text,text,text,integer,integer) FROM signalops_subscriber_gateway;
DROP FUNCTION IF EXISTS subscriber_search_global_warm_catalog(text,text,text,text,integer,integer);
