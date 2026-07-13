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
- G081: back-test substrate MVP implementation.

Closed gate notes:

- G079: `G079_graph_proposal_acceptance.md`.
- G080: `G080_operator_graph_proposal_review.md`.

Recommended next gate:

- G081 implemented: isolated back-test storage, CLI runner, deterministic policy evaluation, and read-only APIs.
- G082 candidate: back-test operator ergonomics, run creation API, or expanded calibration metrics after smoke review.

## G083 Implemented Backend Slice

- G083: named back-test calibration baselines and stored baseline-to-summary comparisons; label/evaluation scoring remains follow-on.
- Specification: `G083_backtest_baselines_and_evaluation.md`.

## G084 Implemented

- G084: evaluation label sync from G080 graph proposal decisions.
- Specification: `G084_evaluation_label_sync.md`.

## G085 Implemented

- G085: label-aware back-test evaluation scoring over G084 labels.
- Specification: `G085_label_aware_backtest_evaluation.md`.

## G086 Proposed

- G086: operator-reviewed calibration promotion candidates over G083/G085 evidence, without runtime deployment.
- Specification: `G086_calibration_promotion_review.md`.

## G087 Proposed

- G087: deployment planning records for approved G086 promotion candidates, without runtime execution.
- Specification: `G087_deployment_planning.md`.

## G088 Proposed

- G088: Syncratic context windows and multi-event insight synthesis from existing ledgers, without a new ingestion layer.
- Specification: `G088_syncratic_context_windows.md`.
