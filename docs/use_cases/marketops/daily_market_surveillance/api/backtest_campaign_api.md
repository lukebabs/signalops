# MarketOps Back-Test Campaign API

G095 adds a bounded operator API for creating historical back-test coverage across multiple symbols, datasets, and windows while reusing the existing G081 back-test runner and isolated back-test ledgers.

## Boundary

Campaigns are orchestration metadata. Each child execution is still an ordinary `marketops_backtest_runs` row with its own signals, artifacts, graph proposals, and policy results in the back-test ledgers. Campaigns do not deploy policies, mutate detector runtime configuration, write production signal/artifact/proposal ledgers, or update graph state.

## Endpoints

- `POST /v1/marketops/backtest-campaigns`
- `GET /v1/marketops/backtest-campaigns`
- `GET /v1/marketops/backtest-campaigns/{campaign_id}`

## Create Request

Required:

- `tenant_id`
- `window_start` and `window_end` as RFC3339 timestamps
- either `symbols` or `universe_group`

Optional defaults:

- `campaign_id`: generated with `btcamp_marketops` prefix when omitted
- `dataset_scope`: defaults to `["equity_eod_prices"]`
- `detector_id`: defaults to `marketops.dsm.taxonomy_v1`
- `source_adapter`: defaults to `market_data.massive`
- `requested_by`: resolved from the authenticated actor/header when present
- `window_step_days`: defaults to `1`
- `max_symbols`: defaults to `5`, capped at `50`
- `max_windows`: defaults to `5`, capped at `60`
- `max_runs`: defaults to `25`, capped at `250`
- `max_records`: defaults to `50`, capped at `1000`

Example:

```json
{
  "tenant_id": "tenant-local",
  "source_id": "src-massive",
  "universe_group": "top50_megacap",
  "dataset_scope": ["equity_eod_prices"],
  "window_start": "2026-07-01T00:00:00Z",
  "window_end": "2026-07-08T00:00:00Z",
  "max_symbols": 10,
  "max_windows": 7,
  "max_runs": 70,
  "max_records": 50
}
```

## Response

The response returns a `campaign` object with lifecycle status, requested scope, child run ids, aggregate metrics, and timestamps. Metrics include planned/completed/failed run counts, aggregate scanned/signals/artifacts/graph proposal/policy result counts, recommendation counts, and child run ids.

## Operational Use

Use campaigns to build the historical run volume needed by calibration summaries, baselines, label-aware evaluations, promotion candidates, and G094 readiness snapshots. After a campaign succeeds, operators can create a calibration summary over the resulting child runs and then re-run readiness checks.
