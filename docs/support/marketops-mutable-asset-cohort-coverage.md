# MarketOps mutable asset-cohort coverage

## Incident

After the tenant-local primary list expanded from the historical 132-member
cohort to 225 assets, Dashboard Risk/Reward history reported older sessions as
`Not materialized` (for example, 93 rows). Those rows were not missing work:
the older sessions were produced before the list expansion.

## Root cause

The signal-overview reader used a fixed SQL `LIMIT 200`, and the coverage
formatter compared every historical date with the *current* active-list size.
Both assumptions violated the mutable-list contract. A list revision changes
the governed cohort; it does not retroactively make prior sessions incomplete.

## Permanent guardrails

- Signal-overview asset and intraday reads are no longer capped at 200; they
  read the governed active cohort (bounded by the repository safety limit).
- The latest session is checked against the current active list for live
  completeness.
- Earlier sessions use the evidence cohort that actually produced that date's
  records, preventing false unprocessed counts after list expansion.
- The regression test asserts that historical two-member evidence remains
  complete after the active list grows to three, while the latest date still
  reports missing members.

## Operational interpretation

`unprocessed` on the newest session means work is missing for the current
asset-list revision. Historical `unprocessed` is not inferred from today's
membership. Any genuine historical backfill gap must be reported by a
date-specific reconciliation job, not by comparing against a later list
revision.
