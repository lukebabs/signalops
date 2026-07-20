# MarketOps Functional Components

Status: implemented component map
Date: 2026-07-19

## Purpose

MarketOps Daily Market Surveillance is the first specialized SignalOps app profile. Its current implementation turns normalized market data into deterministic signals, reviewable artifacts, algorithm outputs, quality-gated proposals, and bounded reasoning context for operators.

This document summarizes the implemented functional components and explains the intended purpose of the main technical components. It is a snapshot of what exists today, not a target architecture for every future DSM capability.

## Functional Overview

MarketOps currently supports this end-to-end workflow:

1. Maintain a first-class Top 50 asset universe.
2. Ingest or replay Massive market data into SignalOps raw and normalized ledgers.
3. Detect deterministic DSM signals from equity EOD and options-derived data.
4. Persist signals, alerts, insights, DSM artifacts, and graph proposal candidates.
5. Back-test and evaluate detector behavior over historical normalized events.
6. Run generic algorithms over MarketOps feature rows.
7. Convert algorithm results into quality-gated signal proposals.
8. Review, preflight, and optionally materialize reviewed algorithm proposals.
9. Build bounded Syncratic context windows and Ask explanations from SignalOps evidence.
10. Expose MarketOps operational views in the frontend.

## Component Summary

| Functional component | Implemented purpose | Primary technical components |
| --- | --- | --- |
| App profile and metadata boundary | Identify MarketOps records consistently across ledgers, APIs, workers, and frontend filters. | `app_id=marketops`, `domain=market_data`, `use_case=daily_market_surveillance`, app profile shell, MarketOps route aliases. |
| Asset universe | Provide the canonical working universe for daily surveillance and bounded expansion. | `marketops_asset_universe`, Top 50 mega-cap seed, asset universe API, `/marketops/assets`. |
| Market data ingestion and normalization | Bring provider data into immutable raw/normalized SignalOps ledgers before detection or algorithms run. | Massive adapter, raw event ledger, normalizer, `normalized_event_ledger`, replay jobs, idempotency records. |
| Deterministic DSM detector pack | Convert normalized equity/option evidence into deterministic DSM-style `signal.v1` records. | Python worker detector `marketops.dsm.taxonomy_v1`, earlier `marketops.dsm.eod_price_v1`, worker loader/default detector config, `signal.v1` schema validation. |
| Signal, alert, and insight ledgers | Store durable signal records and derive operator lifecycle records. | `signal_ledger`, `alert_ledger`, `insight_ledger`, signal persister lifecycle derivation. |
| DSM artifact persistence | Preserve structured DSM evidence as first-class artifact rows instead of only nested signal JSON. | `marketops_dsm_artifacts`, artifact extraction, DSM Workbench persisted/signal-only ledger semantics. |
| Graph proposal workflow | Let DSM evidence propose graph facts while keeping graph mutation operator-controlled. | `marketops_dsm_graph_proposals`, graph proposal API, accept/reject/supersede/restore lifecycle, DSM Workbench graph proposal UI. |
| Back-test substrate | Replay historical normalized events through detectors without mutating production runtime behavior. | `/v1/marketops/backtests`, `marketops_backtest_runs`, replay-style filters, baseline/comparison/evaluation APIs. |
| Calibration and label workflow | Track whether detector changes are ready for promotion based on coverage, baselines, labels, and regressions. | Back-test baselines, evaluations, label sync, label-aware metrics, promotion candidates, readiness snapshots. |
| Algorithm plugin substrate | Provide generic SignalOps time-series algorithm execution independent of MarketOps-specific detector code. | `algorithm_definitions`, `algorithm_execution_requests`, `algorithm_results`, CLI `signalops-algorithm-runner`, algorithms route. |
| Algorithm adapter pack | Score normalized feature rows with multiple reusable statistical/ML algorithms. | `signalops.algorithms.zscore_anomaly_v1`, `river_anomaly_v1`, `ruptures_change_point_v1`, `statsmodels_forecast_v1`, `sklearn_classifier_v1`, `sklearn_isolation_forest_v1`. |
| Algorithm signal proposal workflow | Convert algorithm results into reviewable signal proposals without writing production signals immediately. | `algorithm_signal_proposals`, proposal generator, proposal summary API, review lifecycle, proposal review UI. |
| Proposal materialization safety | Preflight and materialize reviewed algorithm proposals through an audited path. | Materialization preflight API, `algorithm_signal_materializations`, single-proposal materialization mutation, duplicate/block handling. |
| Options chain substrate | Store option-chain rows and derived distribution snapshots for asset-level options analysis. | `marketops_options_chain_daily`, `marketops_options_distribution_daily`, options coverage/distribution/chain APIs. |
| Options distribution features | Convert persisted option distributions into normalized algorithm-ready feature events. | `signalops-marketops-options-feature-materializer`, `options_distribution_daily`, stable derived feature IDs/raw offsets. |
| Bounded options coverage expansion | Pull selected or capped Top 50 options snapshots without scheduling broad provider fanout. | `signalops-marketops-options-coverage-runner`, explicit `--symbols`/`--max-symbols`/`--limit`/`--max-pages`, quality-count reporting. |
| Options quality gating | Prevent low-quality call/put OI ratio evidence from becoming reviewable proposals while retaining algorithm audit rows. | `open_interest_quality`, `call_put_oi_ratio_quality`, G131 proposal gate `g131.options_distribution_quality.v1`. |
| Market state intelligence foundation | Persist reusable point-in-time feature, state, transition, and evidence abstractions above event-level signals. | `marketops_feature_definitions`, `marketops_feature_observations`, `marketops_market_states`, `marketops_state_transitions`, `marketops_evidence`, G136 read APIs. |
| Bounded AAPL state materialization | Prove deterministic market-state construction and quality blocking over existing equity/options evidence. | `signalops-marketops-state-materializer`, 25 feature slots, canonical state schema, exact lineage, idempotent upserts. |
| Forward outcome evaluation | Measure realized behavior after triggered research sources without mutating source records. | `marketops_signal_outcomes`, `signalops-marketops-outcome-materializer`, 1/5/10/20-session outcomes, point-in-time status and event lineage. |
| Historical research execution | Acquire bounded equity history, enforce analytics coverage, and coordinate state-to-outcome research without look-ahead. | `signalops-marketops-history-runner`, exact-symbol Massive date ranges, 60 equity and 20 analytics-ready option-session preflight, trailing transition statistics. |
| Syncratic context and Ask integration | Provide bounded explainability context and generated interpretation without ingesting MarketOps data into Syncratic core. | Syncratic context windows, selective materialization, Ask enrichment, evidence purity checks, data-quality blocking. |
| Frontend operator surfaces | Let analysts inspect MarketOps state and review evidence through app-specific workflows. | `/marketops/assets`, `/marketops/dsm`, `/marketops/algorithms`, `/marketops/backtests`, `/marketops/syncratic`. |

## Technical Component Purposes

### App Metadata

MarketOps records carry `app_id`, `domain`, and `use_case` so the shared SignalOps infrastructure can safely host multiple domains. These fields let API queries, workers, ledgers, frontend routes, and algorithms filter MarketOps evidence without creating a separate platform.

### Asset Universe

The Top 50 asset universe anchors the scope of daily surveillance. It prevents ad hoc symbol selection from becoming the default operating model and gives bounded jobs a canonical source for `top50_megacap` expansion.

### Raw And Normalized Ledgers

Raw event storage preserves provider lineage and immutability. Normalized event storage provides the stable boundary consumed by detectors, back-tests, algorithms, and replay. MarketOps logic should operate on normalized records except for explicit ingestion or provider-debug workflows.

### Deterministic DSM Detectors

DSM detectors encode deterministic market surveillance rules. They produce auditable `signal.v1` records with subject entities, metrics, evidence, taxonomy, severity, and confidence. They are intentionally separate from exploratory algorithms so operational rules remain explainable and reproducible.

### Signal, Alert, And Insight Ledgers

Signals are the durable source of detected events. Alerts represent event-level operator attention. Insights are intended to represent higher-level observations or synthesized context, not one-to-one duplicates of every alert. Current lifecycle records are derived from persisted signals, with further distinction still part of ongoing product design.

### DSM Artifacts And Graph Proposals

DSM artifacts extract structured evidence from signals into first-class rows. Graph proposals convert evidence into candidate graph facts, but graph mutation remains review-controlled. This keeps the data model explainable and avoids automatic graph pollution from raw detector output.

### Back-Test And Calibration Substrate

Back-tests run detectors over historical normalized events and persist metrics for comparison. Baselines, evaluations, labels, and readiness snapshots are intended to make promotion decisions evidence-based rather than anecdotal. This substrate does not deploy runtime policy changes by itself.

### Algorithm Plugin Substrate

The algorithm layer is generic SignalOps infrastructure. MarketOps is currently its primary consumer, but the substrate is intended for time-series use cases broadly. Algorithms read normalized feature rows and write immutable `algorithm_results`; they do not mutate production signal state directly.

### Algorithm Proposals And Materialization

Algorithm results become operationally relevant only through signal proposals. Proposals are reviewable and auditable. Materialization is a separate step with preflight checks, duplicate detection, and materialization ledger records. This separation lets data scientists tune algorithms without bypassing operator governance.

### Options Chain And Distribution Substrate

Options chain rows preserve contract-level evidence. Distribution snapshots aggregate call/put open interest and volume across trade-date windows, moneyness buckets, and expiration buckets. This gives analysts a tractable way to inspect options pressure over time rather than reading raw chain rows only.

### Options Quality Metadata

Provider open-interest data often contains zero or missing values. MarketOps records explicit quality states such as `usable`, `partial_zero`, `all_zero`, and `denominator_zero`. Algorithm results remain durable for audit, but proposal generation only admits usable call/put OI ratio evidence for the current options ratio policy.

### Bounded Live Coverage Runner

The options coverage runner exists to expand coverage deliberately. It can process explicit symbols or a capped Top 50 slice, but it does not schedule itself or fan out automatically. Its purpose is operator-controlled data acquisition with visible quality counts and provider budget limits.

### Market State Intelligence

G136 provides first-class feature, state, transition, and evidence ledgers. G137 materializes one bounded AAPL path from persisted equity and options evidence. G138 adds the first versioned research hypothesis registry and deterministic evaluation ledger for H001, H004, H006, and H007. G139 groups compatible triggered evaluations into research-only opportunities with overlap suppression, conflict scoring, contribution/evidence lineage, and deterministic summaries. Missing history, unusable open interest, absent bid/ask, and uncovered IV cells remain explicit rejection reasons; current AAPL inputs therefore produce no opportunities.

### Syncratic Reasoning Boundary

SignalOps does not ingest MarketOps data into Syncratic core. Instead, SignalOps builds bounded context windows from its own persisted evidence and calls Syncratic Ask for explanation when useful. Evidence purity and data-quality gates prevent the reasoning layer from interpreting mismatched or unsupported evidence as market analysis.

### Frontend Workflows

The frontend surfaces are operational tools, not marketing pages. They let analysts inspect assets, options quality, DSM artifacts, graph proposals, back-tests, algorithm runs, proposal reviews, materialization preflight, and Syncratic context. The UI should keep ingestion, algorithm execution, review, and materialization controls distinct so analysts understand what state is being changed.

## Current Evidence Of Implementation

Recent validated gates include:

- G130: options distribution quality metadata.
- G131: quality-aware algorithm proposal generation.
- G132: options quality visibility UI and zero-OI clarification.
- G133: bounded multi-symbol options coverage runner.
- G134: validation that non-usable AAPL/MSFT options evidence produces no proposals.
- G135: live AMZN positive path where usable options evidence produced exactly one proposal.
- G136: first-class feature/state/transition/evidence foundation and read APIs.
- G137: bounded AAPL materialization with exact lineage, deterministic reruns, and live OI evidence blocking.
- G138: research-only H001/H004/H006/H007 registry and explainable AAPL trigger/non-trigger evaluation.
- G139: research-only opportunity grouping with overlap control, conflict scoring, read APIs, and a deployed UX-first analyst workbench.
- G140: immutable forward outcome ledger, deterministic AAPL materializer, maturity/missing-price semantics, and read APIs.
- G141: 135-session live AAPL equity history, trailing-only transition statistics, and strict historical research orchestration with options-quality blocking.

The latest live AMZN closeout expanded usable options samples to 3 persisted usable rows across 8 AMZN trade days while leaving proposal materialization blocked until operator review.

## Important Boundaries Still In Place

- No automatic Top 50 options scheduler exists.
- No runtime detector/policy deployment path exists yet.
- Algorithm results do not automatically become production signals.
- Materialization requires reviewed proposals and preflight success.
- Syncratic receives bounded context for reasoning; MarketOps data is not ingested into Syncratic core.
- Broader algorithm usefulness and runtime policy deployment remain the next separate workstream.

## Prospective Options Capture

G142 extends the bounded options coverage runner with a deterministic capture-quality ledger. The session workflow requires canonical same-session equity close before provider acquisition, filters Massive by analytical DTE and moneyness bounds, aggregates a capped transient candidate set, and retains at most five deterministic source contracts for the currently implemented surface cells. It separates candidate acquisition metrics from compact persisted evidence; G141 consumes only selected rows from the exact analytics-ready capture run.
