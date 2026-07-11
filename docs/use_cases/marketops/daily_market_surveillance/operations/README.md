# Operations Notes

Use this folder for MarketOps Daily Market Surveillance runbooks, smoke checks, replay checks, and environment notes.

Current recurring validations:

- Rebuild/redeploy relevant services after backend or frontend changes.
- Apply Postgres migrations with `make compose-storage-migrate` when new relational tables are added.
- Publish bounded events or signals for live smoke validation.
- Verify Redpanda consumer lag for the relevant consumer group.
- Verify Postgres rows in `signal_ledger`, `marketops_dsm_artifacts`, `alert_ledger`, and `insight_ledger`.
- For auth-enabled APIs, unauthenticated probes should return `401`; positive browser/API validation requires a bearer token.
