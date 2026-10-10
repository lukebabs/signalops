# S4 As-Of Selection Policy

Status: deployed as a shared, deterministic selection contract for versioned EOD observations.

## Consistent rule

| Consumer context | Selected version | Reason |
|---|---|---|
| `historical_assurance` | Immutable `initial_tenant_local_capture` | Signal Assurance effectiveness, historical outcomes, and backtests must reproduce information actually available at the time. |
| `current_market_context` | Latest usable `global_reobservation` | Current MarketOps context and new calculations should use the most recently verified provider data. |

Migration `000105_subscriber_global_eod_revision_selection` stores the two policy rows and provides `subscriber_global_eod_resolved_observations`. Each resolved record includes the policy/version, selected observation role, provider, payload fingerprint, provenance, algorithm version, quality state, and `as_of_time`.

The policy does not permit a browser, tenant, analyst, or job to select a version ad hoc. A consumer must declare one of the two bounded contexts. This produces the same version for every consumer using the same context and preserves both values for audit.

## Scope

The S4 canary is the first active data set using this projection. Existing tenant-local MarketOps services remain unchanged until a later, separately reviewed integration migrates each consumer to declare its selection context. Historical SAF/backtest records are not restated; current context is not silently substituted into historical evaluation.
