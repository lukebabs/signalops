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
