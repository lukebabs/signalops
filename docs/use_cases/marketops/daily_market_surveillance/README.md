# MarketOps Daily Market Surveillance

Canonical metadata:

- `app_id=marketops`
- `domain=market_data`
- `use_case=daily_market_surveillance`
- primary source adapter: `market_data.massive`
- current detector: `marketops.dsm.taxonomy_v1`

This use case covers daily surveillance over Massive normalized equity EOD and option contract daily data, deterministic DSM taxonomy signals, derived alert/insight lifecycle records, first-class DSM artifact materialization, and the MarketOps DSM Workbench.

## Current Folder Layout

- `architecture/`: DSM model, signal/artifact persistence semantics, graph proposal direction.
- `api/`: MarketOps-specific endpoints and request scopes.
- `frontend/`: DSM Workbench and MarketOps UI operator semantics.
- `operations/`: smoke tests, replay/publish checks, auth-dependent validation notes.
- `gates/`: concise gate summaries and cross-links for G070 onward.

## Important Semantics

A DSM table row marked `persisted` in the frontend Ledger column means the signal has a first-class artifact record in `marketops_dsm_artifacts`.

It does not mean the signal itself only just became persistent. Signals are persisted separately in `signal_ledger`. The current persistence relationship is:

- `signal_ledger`: canonical durable signal record.
- `marketops_dsm_artifacts`: materialized DSM artifact proposal derived from the signal semantic evidence.
- `alert_ledger` and `insight_ledger`: lifecycle records derived from persisted signals.

## Cross-References

- Functional component map: `architecture/functional_components.md`
- Market State Intelligence evaluation: `architecture/market_state_intelligence_evaluation.md`
- G136 Market State Foundation: `gates/G136_market_state_foundation.md`
- G137 AAPL Market State Vertical Slice: `gates/G137_aapl_market_state_vertical_slice.md`
- Market state materialization operations: `operations/market_state_materialization.md`
- G138 Research Hypothesis Evaluator: `gates/G138_research_hypothesis_evaluator.md`
- Research hypothesis evaluation operations: `operations/hypothesis_evaluation.md`
- G139 Opportunity Layer: `gates/G139_opportunity_layer.md`
- Opportunity-building operations: `operations/opportunity_building.md`
- Opportunities workbench implementation and contract: `../../../frontend/marketops_opportunities_workbench_spec.md`
- Global API contract: `docs/api.md`
- Python worker/detector behavior: `docs/python_worker.md`
- Original MarketOps target specs: `docs/marketops/`
- Frontend DSM workbench implementation spec: `docs/frontend/marketops_dsm_workbench_ui_spec.md`
- Signal/artifact persistence note: `architecture/signal_artifact_persistence.md`
- DSM Workbench operator validation: `frontend/dsm_workbench_operator_validation.md`
- G079 graph proposal acceptance gate brief: `gates/G079_graph_proposal_acceptance.md`
- G080 operator graph proposal review gate brief: `gates/G080_operator_graph_proposal_review.md`
- G081 back-test substrate gate brief: `gates/G081_backtest_substrate.md`
- Back-test substrate architecture: `architecture/backtest_substrate.md`
- Back-test substrate operations: `operations/backtest_substrate.md`
- G094 calibration readiness gate brief: `gates/G094_backtest_calibration_readiness.md`
- Syncratic user API boundary: `architecture/syncratic_user_api_boundary.md`
