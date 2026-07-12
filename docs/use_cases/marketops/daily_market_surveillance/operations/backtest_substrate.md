# Back-Test Substrate Operations

Status: specification proposed
Use case: MarketOps Daily Market Surveillance

## Operator Workflow

The first back-test workflow should be explicit and bounded:

1. Choose tenant and MarketOps use case.
2. Choose historical observation window.
3. Choose dataset: `equity_eod_prices` or `options_contracts_daily`.
4. Choose symbol scope:
   - explicit symbols, or
   - a known universe group such as `top50_megacap`.
5. Choose detector id/version.
6. Choose policy pack id/version.
7. Set max records.
8. Start the back-test run.
9. Review run metrics and proposal policy recommendations.
10. Decide whether the policy is safe enough for a later automation gate.

## First Smoke Scenario

The first live smoke should be intentionally small:

- tenant: `tenant-local`
- app: `marketops`
- domain: `market_data`
- use case: `daily_market_surveillance`
- source: `src-massive`
- dataset: `equity_eod_prices`
- symbols: `AAPL`, `SPY`
- window: one or two historical observation dates already present in `normalized_event_ledger`
- max records: 25
- detector: current MarketOps DSM detector
- policy: deterministic v0 calibration policy

Expected smoke output:

- one completed back-test run
- nonzero scanned count if source events exist
- generated output records isolated under the run id
- aggregate policy metrics
- no writes to operational signal/artifact/proposal ledgers
- no graph database writes

## Safety Controls

The implementation gate should include:

- required observation window
- required max record limit
- required tenant/app/domain/use-case metadata
- explicit dataset allowlist
- explicit detector id
- explicit policy id
- cancellation or timeout boundary for long runs
- skip/failure accounting
- dry-run style summary before any future automation policy promotion

## Operational Distinction From Replay

Operators should use replay when they want to republish existing ledgers through the operational pipeline.

Operators should use back-test when they want to evaluate detector and policy behavior over historical data without changing operational state.

Back-test runs are experiments. Replay jobs are operational pipeline actions.

## Review Expectations

Before implementation, reviewers should confirm:

- normalized events are the correct first source
- policy calibration is the correct first objective
- isolated ledgers are mandatory
- the initial metrics are sufficient
- graph materialization remains deferred

## Go / No-Go Checklist For G082

Proceed to G082 only when:

- this G081 specification is reviewed and accepted
- the first smoke window and symbols are chosen
- the initial deterministic policy pack semantics are accepted
- storage isolation is accepted

Do not proceed if the next goal changes to raw-provider replay, ML training, PnL simulation, or production graph materialization.
