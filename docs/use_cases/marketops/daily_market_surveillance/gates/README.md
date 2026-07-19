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

## G089 Implemented

- G089: Syncratic Insights UI for G088 context windows and synthesized insights.
- Specification: `../../../../frontend/syncratic_context_windows_ui_spec.md`.

## G090 Implemented

- G090: server-side Syncratic Ask enrichment for one bounded context window at a time, without Search enrichment or batch generation.
- Specification: `G090_syncratic_ask_enrichment.md`.

## G091 Implemented

- G091: budgeted Syncratic context materialization with dry-run preview and per-asset decision audit.
- Specification: `G091_budgeted_syncratic_materialization.md`.

## G092 Implemented

- G092: Syncratic materialization preview and confirmed write frontend workflow.
- Specification: `../../../../frontend/syncratic_materialization_preview_ui_spec.md`.

## G093 Implemented

- G093: Syncratic insight de-duplication/currentness policy and Ask-state clarity for overlapping context windows.
- Specification: `G093_syncratic_insight_deduplication.md`.

## G094 Implemented

- G094: back-test calibration readiness criteria and persisted snapshots for broader historical coverage, Top 50 equity/options windows, and label volume/quality before runtime policy deployment.
- Specification: `G094_backtest_calibration_readiness.md`.

## G095 Implemented

- G095: bounded historical back-test campaigns over existing isolated back-test runs.
- Specification: `G095_backtest_historical_campaigns.md`.

## G096 Implemented

- G096: read-only normalized-event coverage preflight for data-bearing back-test campaign planning.
- Specification: `G096_backtest_coverage_preflight.md`.

## G097 Implemented

- G097: bounded Massive ingestion smoke path for creating normalized MarketOps input through the existing raw/normalizer pipeline.
- Specification: `G097_backtest_input_ingestion_smoke.md`.

## G098 Implemented

- G098: Massive credential preflight before bounded ingestion smoke.
- Specification: `G098_massive_credential_preflight.md`.

## G099 Implemented

- G099: MarketOps input smoke closeout after env mapping correction, idempotency upsert fix, clean ingestion smoke, coverage check, and one-run campaign validation.
- Specification: `G099_marketops_input_smoke_closeout.md`.

## G100 Implemented

- G100: bounded three-symbol equity ingestion/campaign expansion and calibration summary refresh.
- Specification: `G100_bounded_equity_campaign_expansion.md`.

## G101 Implemented

- G101: bounded options daily ingestion/campaign expansion and calibration summary refresh.
- Specification: `G101_bounded_options_campaign_expansion.md`.

## G102 Implemented

- G102: bounded multi-day equity and options ingestion/campaign expansion with refreshed calibration summaries.
- Specification: `G102_bounded_multiday_campaign_expansion.md`.

## G103 Implemented

- G103: calibration readiness re-check after bounded multi-day evidence.
- Specification: `G103_calibration_readiness_recheck.md`.

## G104 Proposed

- G104: reviewed-label workflow specification for increasing real graph-proposal labels without synthetic labels or threshold relaxation.
- Specification: `G104_reviewed_label_workflow.md`.

## G105 Implemented

- G105: reviewed-label batch inventory and idempotent sync readiness for the first 25-label milestone.
- Specification: `G105_reviewed_label_batch_inventory.md`.

## G106 Implemented

- G106: generic SignalOps algorithm registry, execution request ledger, result ledger, seed definitions, and read APIs.
- Specification: `G106_algorithm_registry_result_ledger.md`.

## G107 Implemented

- G107: first executable generic algorithm runner for `signalops.algorithms.zscore_anomaly_v1`, writing deterministic `algorithm_results`.
- Specification: `G107_zscore_algorithm_runner.md`.

## G108 Implemented

- G108: read-only algorithm execution summary API with result counts, severity counts, max score/confidence, and top result rows.
- Specification: `G108_algorithm_execution_visibility.md`.

## G109 Implemented

- G109: read-only algorithm execution visibility UI for definitions, execution requests, summaries, result rows, and result lineage.
- Specification: `G109_algorithm_execution_visibility_ui.md`.

## G110 Implemented

- G110: deterministic v0 runner adapters for every seeded G106 algorithm id, all writing `algorithm_results`.
- Specification: `G110_algorithm_adapter_pack_v0.md`.

## G111 Implemented

- G111: first-class `algorithm_signal_proposals` ledger, bounded generator CLI, and read-only APIs for converting `algorithm_results` into reviewed signal proposals without direct production signal writes.
- Specification: `G111_algorithm_result_signal_proposal_design.md`.

## G112 Implemented

- G112: operator review lifecycle for `algorithm_signal_proposals` with decision metadata and no production signal materialization.
- Specification: `G112_algorithm_signal_proposal_review.md`.

## G113 Proposed

- G113: frontend visibility and review workflow for `algorithm_signal_proposals`, using G111/G112 APIs without production signal materialization.
- Specification: `../../../../frontend/algorithm_signal_proposals_review_ui_spec.md`.

## G114 Implemented

- G114: implements the G113 frontend spec — proposal list/detail/review UI inside the existing Algorithms route over G111/G112 APIs, with bounded review controls and no production signal materialization. Automated tests, typecheck, and build green; browser validation pending the auth gate.
- Specification: `G114_algorithm_signal_proposal_review_ui.md`.

## G115 Implemented

- G115: read-only summary/readiness API for `algorithm_signal_proposals` review coverage and unresolved high/critical proposal counts.
- Specification: `G115_algorithm_signal_proposal_summary.md`.

## G116 Implemented

- G116: implements the G116 frontend spec — compact read-only review-coverage summary panel above the G114 proposal list, over the G115 summary API, with coupled filters (no `limit`) and independent loading/error/empty states. Automated tests, typecheck, and build green; browser validation pending the auth gate.
- Specification: `G116_algorithm_signal_proposal_summary_ui.md`.

## G117 Proposed

- G117: design-only architecture for future materialization of reviewed `algorithm_signal_proposals` into production `signal.v1` rows.
- Specification: `G117_algorithm_signal_materialization_design.md`.

## G118 Implemented

- G118: read-only backend materialization preflight for `algorithm_signal_proposals`, reporting eligible, duplicate-risk, blocked, invalid, and would-write counts without production signal writes.
- Specification: `G118_algorithm_signal_materialization_preflight.md`.

## G119 Proposed

- G119: frontend-agent specification for read-only algorithm signal materialization preflight visibility over the G118 API.
- Specification: `G119_algorithm_signal_materialization_preflight_ui_spec.md`.

## G120 Proposed

- G120: design-only architecture for explicit algorithm signal materialization requests, ledger semantics, idempotency, stable signal ids, auth/audit, payload mapping, and failure behavior.
- Specification: `G120_algorithm_signal_materialization_request_design.md`.

## G121 Implemented

- G121: storage migration and read-only APIs for `algorithm_signal_materializations`, with no materialization mutation or production signal writes.
- Specification: `G121_algorithm_signal_materialization_ledger_reads.md`.

## G122 Implemented

- G122: single-proposal algorithm signal materialization mutation with server-side preflight, idempotent ledger rows, duplicate/blocked handling, and one production signal write for eligible reviewed proposals.
- Specification: `G122_algorithm_signal_materialization_write_path.md`.

## G123 Proposed

- G123: frontend-agent specification for single-proposal algorithm signal materialization action UI over the G122 API.
- Specification: `G123_algorithm_signal_materialization_action_ui_spec.md`.

## G124 Proposed

- G124: lifecycle policy decision for production signals created by algorithm proposal materialization; keep G122 signal-ledger-only and defer alert/insight/graph/Syncratic fanout to a separate audited policy gate.
- Specification: `G124_algorithm_materialized_signal_lifecycle_policy.md`.

## G125 Implemented

- G125: MarketOps options-chain substrate with persisted full-chain rows, 10-trade-day distribution snapshots, asset-scoped read APIs, and a reserved non-persisting live-preview endpoint.
- Specification: `G125_marketops_options_chain_substrate.md`.

## G126 Implemented

- G126: converts persisted options distribution snapshots into canonical `options_distribution_daily` normalized feature events and adds a CLI materializer so existing algorithms can score call/put divergence features.
- Specification: `G126_options_distribution_algorithm_features.md`.

## G127 Implemented

- G127: adds a bounded Massive option-chain snapshot ingestor that persists current chain rows and derives the rolling MarketOps options distribution snapshot for one symbol at a time.
- Specification: `G127_options_chain_snapshot_ingestion.md`.

## G128 Proposed

- G128: frontend-agent specification for persisted asset-level options coverage, call/put distribution, and chain-row inspection on `/marketops/assets`.
- Specification: `../../../../frontend/marketops_asset_options_distribution_ui_spec.md`.

## G129 Implemented

- G129: adds a no-provider-call options distribution backfill CLI that derives one `10_trade_days` snapshot per persisted chain trade date.
- Specification: `G129_options_distribution_backfill.md`.

## G130 Implemented

- G130: adds explicit open-interest and call/put ratio quality metadata for options distribution snapshots and normalized feature rows.
- Specification: `G130_options_distribution_quality_metrics.md`.

## G131 Implemented

- G131: gates algorithm signal proposal generation for options call/put open-interest ratio results so only `call_put_oi_ratio_quality=usable` evidence enters the proposal queue.
- Specification: `G131_quality_aware_algorithm_proposals.md`.

## G132 Implemented

- G132: implemented options ratio/open-interest quality visibility in asset options, algorithm results, and signal proposal views, including explicit zero-OI chain-row rendering.
- Specification: `../../../../frontend/marketops_options_quality_visibility_ui_spec.md`.

## G133 Implemented

- G133: adds a bounded operator CLI for selected-symbol or capped Top 50 options coverage expansion, including chain ingest, distribution derivation, normalized feature materialization, and quality reporting.
- Specification: `G133_bounded_top50_options_coverage_expansion.md`.

## G134 Implemented

- G134: validates that expanded AAPL/MSFT options feature rows produce algorithm results but zero proposals when G131 quality gating sees only `all_zero` and `denominator_zero` call/put OI ratio evidence.
- Specification: `G134_expanded_options_quality_gate_validation.md`.

## G135 Implemented

- G135: validates a real live Massive pull for non-NVDA options data, persists AMZN coverage, runs algorithm scoring, and confirms G131 generates a proposal only for usable call/put OI ratio evidence.
- Specification: `G135_live_options_positive_quality_path.md`.

## G136 Implemented

- G136: adds first-class feature-definition, feature-observation, market-state, state-transition, and evidence ledgers with read-only APIs and exact state lineage resolution.
- Specification: `G136_market_state_foundation.md`.
