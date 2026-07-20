# Architecture Notes

Use this folder for MarketOps Daily Market Surveillance architecture and data-flow documentation.

Belongs here:

- DSM domain model and taxonomy explanations.
- Signal versus artifact persistence semantics.
- Graph proposal model and acceptance/storage direction.
- Reconciliation between target MarketOps architecture and implemented gates.

Current key rule:

- A persisted DSM artifact is a first-class record in `marketops_dsm_artifacts` derived from a persisted signal. It is linked by `artifact_id` and `signal_id`; it does not replace the canonical `signal_ledger` record.

Current notes:

- `signal_artifact_persistence.md`: canonical signal/artifact/lifecycle relationship and DSM Workbench Ledger semantics.
- `graph_proposal_acceptance.md`: proposed G079 boundary for first-class graph proposal review/storage.
- `backtest_substrate.md`: implemented G081 back-test substrate boundary, isolated storage model, CLI/API surfaces, and replay distinction.
- `syncratic_context_windows.md`: proposed G088 Syncratic context-window and multi-event insight architecture.
- `syncratic_user_api_boundary.md`: Syncratic user facade OpenAPI and environment/auth boundary.
- `../gates/G090_syncratic_ask_enrichment.md`: proposed Ask-based LLM synthesis boundary over deterministic context windows.
- `functional_components.md`: broad summary of implemented MarketOps functional components and technical component purposes.
- `market_state_intelligence_evaluation.md`: evaluation of the Market State Intelligence target design against implemented MarketOps functional outcomes and design gaps.
- `../gates/G136_market_state_foundation.md`: implemented storage and read-API foundation for feature observations, market states, transitions, evidence, and exact lineage.
- `../gates/G137_aapl_market_state_vertical_slice.md`: implemented bounded AAPL feature/state/transition/evidence materialization over persisted equity and options evidence.
- `../gates/G138_research_hypothesis_evaluator.md`: implemented research-only H001/H004/H006/H007 registry and deterministic evaluation over bounded AAPL states.
- `../gates/G139_opportunity_layer.md`: implemented research-only opportunity grouping, overlap control, conflict scoring, evidence lineage, and read APIs.
- `../gates/G140_forward_outcome_evaluation.md`: implemented point-in-time forward outcome ledger, bounded materializer, and read APIs.
