# Syncratic Context Windows Architecture

Status: proposed
Use case: MarketOps Daily Market Surveillance

## Purpose

Syncratic should provide the explainability layer over existing SignalOps and MarketOps ledgers. It should build durable context windows and synthesized insights from persisted evidence, not ingest external data in the MVP.

## Design Boundary

SignalOps records observations and operational lifecycle records.

MarketOps applies domain-specific DSM logic, graph proposals, back-tests, evaluations, and promotion review.

Syncratic explains why a pattern matters by assembling bounded evidence windows across these records.

## Alert And Insight Boundary

Alerts are event-level operational work items. They may map directly to incidents.

Syncratic insights are multi-event analytical findings. They should reference supporting alerts, signals, events, artifacts, graph proposals, labels, and evaluations.

## No New Ingestion Layer For MVP

G088 should consume existing internal ledgers first:

- normalized events;
- signals;
- alerts;
- DSM artifacts;
- graph proposals;
- operator decisions and labels;
- back-test/evaluation/promotion records.

External data ingestion for news, filings, macro calendars, analyst notes, or third-party taxonomies should be deferred until internal explainability works and provenance/licensing rules are clear.

## Reproducibility

Each context window should capture:

- strategy;
- builder version;
- subject and time range;
- evidence references;
- summary metrics;
- evidence digest.

A reproduced context build with the same inputs should produce the same id and digest.

## Selective Materialization

Syncratic should scan broadly but materialize narrowly.

For the MarketOps Top 50 universe, the daily process should perform a cheap aggregate candidate scan across all covered assets, then build full context windows only for candidates that cross deterministic thresholds.

Default materialization triggers:

- at least two related alerts or signals in the strategy window;
- at least one critical alert in the strategy window;
- related graph proposal, evaluation, promotion, or operator-label evidence for the same subject/window.

The context builder should skip quiet assets, skip unchanged evidence digests, and enforce configurable caps for scanned assets, candidate windows, context windows, and synthesized insights. This keeps Syncratic explainability available across the universe without creating excessive unused batch work.

The deterministic materialization key should be based on tenant, use case, context strategy, subject symbol, window start, window end, and builder version.

## Evidence Purity

Subject-scoped context windows must not include evidence for another ticker unless a later cross-asset context strategy explicitly allows that behavior. For the current `symbol_signal_cluster_5d` strategy, signal and alert inclusion requires an exact entity-symbol match for the context subject, and supporting evidence must not mention a different known ticker. Evidence that fails this check is excluded before the context window and Syncratic Ask prompt are built.
