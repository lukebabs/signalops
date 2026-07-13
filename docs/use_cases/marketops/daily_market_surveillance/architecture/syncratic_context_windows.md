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
