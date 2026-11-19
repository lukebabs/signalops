# S4 Current Market Context Integration

Status: deployed for the MarketOps asset-detail API. This is a live-context read path; it does not alter historical assurance or cause an algorithm recalculation.

## Contract

`GET /v1/tenants/{tenant_id}/marketops/assets/{symbol}/algorithm-observations` now includes `current_eod_context` when the requested symbol is active in that tenant's MarketOps universe and a usable global re-observation exists.

The returned object is fixed to:

- `usage_context`: `current_market_context`
- `selected_observation_role`: `global_reobservation`
- `policy_version`: `s4-as-of-selection-v1`

It includes the selected EOD OHLCV/VWAP fields and audit provenance: session date, provider, payload fingerprint, source event/run, algorithm version, quality state, and `as_of_time`.

When no usable current re-observation exists, `current_eod_context` is `null`. The API never falls back to a tenant-local initial capture and never accepts a browser-selected revision.

## Authorization boundary

Migration `000106_subscriber_gateway_current_eod_context` creates `subscriber_gateway_current_eod_context`, a `security_barrier` view. It deliberately grants only that narrow view to the subscriber gateway role; it does not grant the gateway direct access to global asset or provider-revision tables.

The storage query joins this view to `marketops_universal_assets` using the request tenant and active symbol. Therefore a tenant cannot obtain a global current-EOD row for a symbol outside its authorized active MarketOps universe.

## Separation from historical evaluation

The live context exists alongside, not inside, SAF outcomes and backtests:

| Consumer | EOD context | Selected role |
|---|---|---|
| SAF, outcomes, backtests | `historical_assurance` | `initial_tenant_local_capture` |
| Asset-detail current context | `current_market_context` | `global_reobservation` |

No historical evidence, assertion, outcome, or existing backtest was restated. This release makes no provider request, scheduler change, global coverage activation, or automated recalculation.

## Deployment evidence

- Application commit: `6e6e6ce`.
- Migration `000106` applied from a clean archive.
- The projection resolves two usable current-context records: AAPL and NVDA.
- Focused API/storage tests and the clean image's full Go test suite passed.
- Local and public gateway health checks returned `200` after the gateway-only release.
