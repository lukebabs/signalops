# Gate Notes

Use this folder for MarketOps Daily Market Surveillance gate-specific summaries when a gate needs more detail than `docs/gate_audit.md`.

Current gate sequence:

- G070: deterministic MarketOps DSM v0 detector.
- G071: MarketOps asset universe storage/API and UI.
- G072: Massive option contract daily normalization.
- G073: option-interest and price-derived feature enrichment.
- G074: DSM artifact and graph proposal payloads.
- G075: broader DSM taxonomy pack.
- G076: DSM Workbench UI.
- G077: first-class DSM artifact ledger backend.
- G078: DSM artifact API frontend integration.
- G079: graph proposal acceptance/storage backend, read-only frontend visibility, authenticated API smoke, and historical persister-lag cleanup.
- G080: operator graph proposal review workflow.
- G081: back-test substrate specification and architecture.

Closed gate notes:

- G079: `G079_graph_proposal_acceptance.md`.
- G080: `G080_operator_graph_proposal_review.md`.

Recommended next gate:

- G081 review: operator/architecture review of `G081_backtest_substrate.md`, `architecture/backtest_substrate.md`, and `operations/backtest_substrate.md`.
- G082 implementation: thin MVP back-test runner and isolated storage boundary after G081 go/no-go.
