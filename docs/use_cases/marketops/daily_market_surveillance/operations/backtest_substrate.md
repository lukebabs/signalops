# Back-Test Substrate Operations

Status: MVP implemented
Use case: MarketOps Daily Market Surveillance

## Operator Workflow

The first back-test workflow is explicit and bounded:

1. Choose tenant and MarketOps use case.
2. Choose historical observation window.
3. Choose dataset: `equity_eod_prices` or `options_contracts_daily`.
4. Choose symbol scope (resolves payload-level against `normalized_event_ledger.normalized_payload`):
   - explicit symbols, or
   - a known universe group such as `top50_megacap` (expanded to symbols via `marketops_asset_universe`).
5. Choose detector id/version to run (an execution parameter, not an input filter — normalized events are pre-detector).
6. Choose policy pack id/version.
7. Set max records.
8. Start the back-test run with `cmd/marketops-backtest`.
9. Review run metrics and proposal policy recommendations through `/v1/marketops/backtests`.
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

## Documentation Links

- Gate note: `../gates/G081_backtest_substrate.md`
- Architecture: `../architecture/backtest_substrate.md`


## CLI Usage

Example smoke command:

```bash
docker run --rm -v "$PWD:/work" -w /work golang:1.24 go run ./cmd/marketops-backtest \
  --tenant-id tenant-local \
  --source-id src-massive \
  --dataset equity_eod_prices \
  --symbols AAPL,SPY \
  --window-start 2026-07-01T00:00:00Z \
  --window-end 2026-07-12T00:00:00Z \
  --max-records 25
```

Required environment:

- `SIGNALOPS_DATABASE_URL`
- `SIGNALOPS_TEMPORAL_DATABASE_URL`
- Python available as `python3`, or pass `--python-bin`

The runner writes only to `marketops_backtest_*` tables. It does not publish to Redpanda and does not write operational signal/artifact/proposal ledgers.

## Read APIs

- `POST /v1/marketops/backtests` to create and synchronously execute a bounded run
- `GET /v1/marketops/backtests?tenant_id=tenant-local&limit=50`
- `GET /v1/marketops/backtests/{run_id}`
- `GET /v1/marketops/backtests/{run_id}/signals`
- `GET /v1/marketops/backtests/{run_id}/graph-proposals?recommendation=manual_review_required`

## Calibration Readiness Campaign Direction

G094 expands the operational expectation from smoke runs to calibration campaigns. Operators should use the isolated back-test runner/API to build broader coverage before trusting promotion candidates.

Recommended campaign shape:

- run equity EOD back-tests across the Top 50 universe over multiple historical windows;
- run options daily back-tests separately for symbols with normalized options coverage;
- keep `max_records` bounded per run and aggregate with G082 summaries rather than creating unbounded single runs;
- sync reviewed graph proposal decisions into G084 labels after operator review;
- run G085 label-aware evaluations only after enough labels exist;
- treat weak label volume, label conflicts, or sparse historical windows as `needs_more_data`, not as a deployment signal.

Policy deployment remains out of scope for calibration operations until a later execution gate is explicitly approved.
