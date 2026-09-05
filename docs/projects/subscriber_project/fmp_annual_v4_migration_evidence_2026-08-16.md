# FMP Annual v4 Evidence-Schema Deployment — 2026-08-16

**Status:** passed; schema only. No FMP provider request, annual capture,
valuation materialization, scheduler enablement, Gateway restart, or UI change
occurred in this action.

## Applied change

The existing allowlisted MarketOps migration action applied
`000136_subscriber_global_annual_financial_evidence` to the dedicated
MarketOps primary database at `2026-08-16 03:33:39 UTC`.

The migration adds `fundamental_annual` as an allowed immutable global evidence
kind and permits the existing global evidence ledger's `provider_capture`
execution mode. It does not create a table, a browser grant, a tenant copy, or
a reader projection. The append-only constraints and the restricted
`signalops_subscriber_global_eod` writer identity remain in force.

## Verification retained

- The ordered migration runner recorded version `000136` in `schema_migrations`.
- The dedicated primary database stayed `marketops`; no temporal migration ran.
- The action restarted no Gateway, worker, or scheduler.
- The inherited Market State projection checks completed unchanged, confirming
  the migration action stayed within the dedicated primary boundary.

## Next controlled action

Refresh the root-owned deployment-agent allowlist, then run only
`fmp-annual-entitlement-preflight`. That action builds the isolated worker and
makes one warm-symbol, five-endpoint annual/reference request at 250 ms minimum
spacing. It is dry-run only and writes no evidence. A provider rejection,
incomplete annual response, or rate-limit result leaves the capture and all
scheduling disabled.
