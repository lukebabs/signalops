# G095 Back-Test Historical Campaigns

Status: implemented backend/API slice.

## Purpose

G095 closes the immediate G094 gap around broader historical validation by adding a bounded campaign layer above existing MarketOps back-test runs. The gate lets an operator request repeated child runs across symbols, dataset scope, and time windows without introducing policy deployment or a background scheduler.

## Implemented Scope

- Persisted `marketops_backtest_campaigns` control-plane rows.
- Synchronous `POST /v1/marketops/backtest-campaigns` execution that plans and runs bounded child back-tests sequentially.
- `GET` list/detail APIs for campaign audit and UI/API follow-up.
- Universe-based symbol resolution through `marketops_asset_universe` when `symbols` are omitted and `universe_group` is supplied.
- Hard request bounds for symbols, windows, child runs, and records per child run.
- Aggregate campaign metrics plus first-class `child_run_ids` linking back to ordinary back-test run rows.

## Out Of Scope

- Runtime policy deployment.
- Detector threshold mutation.
- Model training.
- Production signal/artifact/graph writes.
- Unbounded or asynchronous campaign workers.
- Automatic promotion or readiness mutation.

## Validation

- Targeted API/storage tests cover bounded child run creation, universe symbol resolution, unbounded request rejection, and list filtering.
- Full validation should include migration application, gateway rebuild, authenticated create/list/detail smoke, and a follow-on calibration summary/readiness check over the created child runs.

## Follow-On

The next narrow task after G095 is to run a real historical campaign against Top 50 equity/options windows, then summarize/evaluate those child runs and re-check G094 readiness.
