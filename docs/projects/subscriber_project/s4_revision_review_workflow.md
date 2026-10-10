# S4 EOD Revision Review Workflow

Status: deployed read-only analyst transparency workflow.

## Analyst experience

Open **MarketOps → Assets → select an asset → Overview**. When immutable provider comparison evidence exists, the **Provider revision review** panel shows each EOD field side by side:

- initial tenant-local capture;
- subsequent global provider re-observation;
- session date and the two observed-at times;
- `informational` versus `review_required` materiality;
- expandable source identifiers and payload fingerprints.

Review-required fields are visually prioritized. The initial and revised records remain immutable, and the panel does not provide an approval, overwrite, recalculation, or version-selection action.

## API contract

The existing asset-detail response contains `eod_revision_review`:

```text
GET /v1/tenants/{tenant_id}/marketops/assets/{symbol}/algorithm-observations
```

The response declares:

- `usage_context=revision_review`
- `initial_observation_role=initial_tenant_local_capture`
- `revised_observation_role=global_reobservation`

This is comparison evidence, not an EOD selection context. Historical assurance remains fixed to the initial capture; current MarketOps context remains fixed to the usable global re-observation.

## Access boundary

Migration `000107_subscriber_gateway_eod_revision_review` adds a security-barrier view with only immutable comparison columns. The gateway receives no direct grant on global assets, raw revision observations, or delta tables. The storage query joins the narrow view to an active asset in the request tenant's MarketOps universe, so another tenant cannot request review data for a non-authorized symbol.

## Current canary evidence

For both AAPL and NVDA, the review workflow exposes:

- `volume`: `review_required` provider revision;
- `vwap`: `review_required` provider revision;
- `open`, `high`, `low`, `close`: informational unchanged comparisons.

No provider request, scheduler change, coverage activation, SAF/outcome restatement, or algorithm recalculation was made by this workflow.
