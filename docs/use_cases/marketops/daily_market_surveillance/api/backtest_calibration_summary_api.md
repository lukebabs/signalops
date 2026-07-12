# Back-Test Calibration Summary API

G082 adds persisted calibration summary snapshots over isolated MarketOps back-test runs.

These summaries are derived from `marketops_backtest_runs.metrics` and do not mutate production signal, artifact, graph proposal, alert, or insight ledgers.

## Create Summary

`POST /v1/marketops/backtest-calibration-summaries`

Request body:

```json
{
  "summary_id": "btcal-marketops-example",
  "tenant_id": "tenant-local",
  "app_id": "marketops",
  "domain": "market_data",
  "use_case": "daily_market_surveillance",
  "source_id": "src-massive",
  "dataset": "equity_eod_prices",
  "detector_id": "marketops.dsm.taxonomy_v1",
  "status": "succeeded",
  "limit": 50,
  "requested_by": "operator"
}
```

`summary_id` is optional; the gateway generates `btcal_marketops-*` when omitted.

The filter selects existing back-test runs, then persists a snapshot containing:

- selected `run_ids`
- run/succeeded/failed/zero-input counts
- scanned/signals/artifacts/graph-proposal/policy-result totals
- signal yield
- policy-results-per-signal
- recommendation counts and shares
- dominant recommendation
- original filter and summary parameters

## List Summaries

`GET /v1/marketops/backtest-calibration-summaries?tenant_id=tenant-local&dataset=equity_eod_prices&detector_id=marketops.dsm.taxonomy_v1&limit=50`

Response envelope:

```json
{
  "calibration_summaries": []
}
```

## Get Summary

`GET /v1/marketops/backtest-calibration-summaries/{summary_id}`

Response envelope:

```json
{
  "calibration_summary": {}
}
```

## Auth And Scope

Gateway authentication applies when enabled. Unauthenticated probes should return `401 unauthorized`.

This API is a stored calibration substrate, not a policy promotion workflow. Promotion, baseline naming, and detector-threshold changes remain future gate work.
