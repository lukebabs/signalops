# FMP Annual v4 Entitlement Preflight — 2026-08-16

**Status:** passed; one-symbol dry run only.

## Scope and result

The root-owned, allowlisted `fmp-annual-entitlement-preflight` action rebuilt
the isolated annual-financial worker and ran it with:

- `--dry-run`;
- `--max-assets 1`;
- `--request-interval 250ms`; and
- the dedicated MarketOps primary connection.

It completed successfully with `warm_assets=1`, `succeeded=1`, `failed=0`,
and `fmp_calls=5`. The five calls are the annual income statement, balance
sheet, cash-flow statement, ratios, and key-metrics endpoints. This confirms
the configured FMP Starter credential supports the planned annual capture
contract.

## Safety evidence

The worker was in dry-run mode. It wrote no evidence run or evidence record,
did not run the v4 materializer, restart a Gateway, change the v3 profile,
enable a timer, or alter a tenant/list. Request pacing is enforced in the FMP
client per individual call; the full worker cannot be configured below 250 ms.

## Next approval gate

A full warm-cohort capture is now technically eligible but remains disabled.
It will make at most five annual/reference FMP calls per enabled warm asset
(currently no more than 1,000 assets), append immutable provider evidence, and
then require a separate dry-run/materialization and reader/calibration review.
It does not require or enable a browser trigger.
