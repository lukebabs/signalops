# S6 Options-Demand Workload Preflight

Status: implementation ready; a dedicated secret-managed login is still required.

Run only with a login granted to `signalops_subscriber_options_demand` and no other Subscriber worker group:

```bash
SIGNALOPS_SUBSCRIBER_WORKLOAD_IDENTITY=subscriber-options-demand-planner \
SIGNALOPS_SUBSCRIBER_OPTIONS_DEMAND_DATABASE_URL='<secret-managed dedicated DSN>' \
  ./scripts/subscriber_project_options_demand_workload_preflight.sh
```

The preflight requires a non-superuser, non-CREATEROLE, NOBYPASSRLS identity. It verifies execute-only access to the aggregate demand function; append-only `SELECT, INSERT` access to S6 snapshots; and zero direct access to subscriber entitlements, lists, memberships, global catalog, or MarketOps Options tables.

It has no provider client and cannot create a demand snapshot. A passing preflight authorizes only the subsequent shadow-planner command, never Options capture.
