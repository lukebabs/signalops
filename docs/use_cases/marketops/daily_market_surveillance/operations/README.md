# Operations Notes

Use this folder for MarketOps Daily Market Surveillance runbooks, smoke checks, replay checks, and environment notes.

Current recurring validations:

- Rebuild/redeploy relevant services after backend or frontend changes.
- Apply Postgres migrations with `make compose-storage-migrate` when new relational tables are added.
- Publish bounded events or signals for live smoke validation.
- Verify Redpanda consumer lag for the relevant consumer group.
- Verify Postgres rows in `signal_ledger`, `marketops_dsm_artifacts`, `alert_ledger`, and `insight_ledger`.
- For auth-enabled APIs, unauthenticated probes should return `401`; positive browser/API validation requires a bearer token.

## Back-Test Operations

`backtest_substrate.md` covers the implemented G081 back-test workflow, first smoke scenario, safety controls, CLI command, and read-only inspection APIs. Back-tests are experiments isolated from operational state: a back-test run must never write to `signal_ledger`, `alert_ledger`, `insight_ledger`, `marketops_dsm_artifacts`, `marketops_dsm_graph_proposals`, or any production graph database. Replay remains the correct tool for republishing existing ledgers through the operational pipeline.
