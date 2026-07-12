# API Notes

Use this folder for MarketOps Daily Market Surveillance API notes that supplement `docs/api.md`.

Current MarketOps-specific endpoints:

- `GET /v1/tenants/{tenant_id}/marketops/assets`
- `GET /v1/marketops/dsm/artifacts`
- `GET /v1/marketops/dsm/artifacts/{artifact_id}`
- `GET /v1/marketops/dsm/graph-proposals`
- `GET /v1/marketops/dsm/graph-proposals/{proposal_id}`
- `POST /v1/marketops/dsm/graph-proposals/{proposal_id}/decision`
- `POST /v1/marketops/backtest-calibration-summaries`
- `GET /v1/marketops/backtest-calibration-summaries`
- `GET /v1/marketops/backtest-calibration-summaries/{summary_id}`
- `POST /v1/marketops/backtest-calibration-baselines`
- `GET /v1/marketops/backtest-calibration-baselines`
- `GET /v1/marketops/backtest-calibration-baselines/{baseline_id}`
- `POST /v1/marketops/backtest-calibration-comparisons`
- `GET /v1/marketops/backtest-calibration-comparisons`
- `GET /v1/marketops/backtest-calibration-comparisons/{comparison_id}`
- `POST /v1/marketops/backtest-evaluation-labels/sync`
- `GET /v1/marketops/backtest-evaluation-labels`
- `GET /v1/marketops/backtest-evaluation-labels/{label_id}`

MarketOps signal, alert, insight, raw-event, normalized-event, and back-test run views use the shared `/v1/*` APIs with metadata filters where applicable:

- `app_id=marketops`
- `domain=market_data`
- `use_case=daily_market_surveillance`

Authentication is enforced by the gateway when enabled. Positive live API validation requires a real bearer token; unauthenticated probes should return `401 unauthorized`.

Current notes:

- `graph_proposal_api.md`: G079 graph proposal list/detail/decision API boundary.
- `backtest_calibration_summary_api.md`: G082 persisted back-test calibration summary API boundary.
- `backtest_baseline_comparison_api.md`: G083 baseline and stored comparison API boundary.
- `backtest_evaluation_label_api.md`: G084 graph-proposal decision label sync API boundary.
