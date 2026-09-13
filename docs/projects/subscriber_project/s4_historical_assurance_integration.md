# S4 Historical Assurance Integration

Status: implemented as an additive application-level contract; no historical data, SAF assertion, outcome, or prior backtest run was restated.

## Purpose

Historical evaluation must use information that was available at the evaluation point. It must never silently replace that evidence with a later provider correction. This integration binds SAF effectiveness/outcome read models and newly created MarketOps backtests to the S4 `historical_assurance` selection context.

## Contract

| Consumer | Context | Selected observation role | Policy | Restatement |
|---|---|---|---|---|
| SAF effectiveness, outcome observations, recommendations | `historical_assurance` | `initial_tenant_local_capture` | `s4-as-of-selection-v1` | Disabled |
| New MarketOps backtests | `historical_assurance` | `initial_tenant_local_capture` | `s4-as-of-selection-v1` | Disabled |

The API returns this contract in `data_selection` for:

- `GET /v1/marketops/signal-assurance/effectiveness`
- `GET /v1/marketops/signal-assurance/effectiveness/observations`
- `GET /v1/marketops/signal-assurance/recommendations`

Every new MarketOps backtest persists the same selection metadata in both its immutable `filters` and `parameters` records. The backtest runner rejects `current_market_context`; callers cannot select an EOD revision ad hoc.

## Data behavior

The existing tenant-local normalized EOD ledger is the initial-capture historical record used by SAF and backtests. The S4 global revision projection remains additive. It retains subsequent `global_reobservation` values and their field-level differences, but those values are not substituted into historical outcomes or backtests.

This establishes reproducibility for future runs without claiming that older SAF or backtest rows were recomputed. Legacy records retain their original provenance and must be interpreted using their recorded evidence source.

## Verification

Focused tests cover the policy default, rejection of current-market selection for backtests, and the immutable SAF response metadata:

```bash
go test ./internal/marketops/backtest ./internal/api
```

The independent S4 selection migration and canary evidence remain in [the as-of selection policy](s4_as_of_selection_policy.md) and [its deployment evidence](s4_as_of_selection_evidence_2026-08-13.md).
