# SignalOps Technical Buyer Value

## Evolution Note

This document describes the technical value of the general SignalOps core
engine. It is detailed by design, but it is not the source of truth for
contracts, schemas, or deployment behavior. Those details remain in the
canonical technical specifications linked from `README.md`.

## The Problem

Technical teams are often asked to add event intelligence to an existing
platform after the platform was built around documents, requests, dashboards,
or batch workflows. The typical result is a cluster of custom pipelines with
different ingestion rules, retry behavior, event schemas, detector runtimes,
and audit models.

That approach creates predictable risks:

- Ingestion contracts drift by source or team.
- Duplicate events create duplicate side effects.
- Failed processing is hard to replay safely.
- Algorithm outputs are difficult to compare or audit.
- Event pipelines become coupled to a single domain implementation.
- Streaming logic threatens to interfere with existing platform behavior.

SignalOps is valuable because it treats these concerns as platform
requirements, not as afterthoughts inside each use case.

## What SignalOps Solves

SignalOps provides a shared event intelligence substrate with clear boundaries:

- Public event ingestion and source validation at the gateway.
- Durable broker topics for raw, normalized, signal, retry, and dead-letter
  flows.
- Normalized event contracts that preserve tenant, source, timing, entity,
  evidence, and correlation context.
- Python-first detector plugins for algorithmic and ML/NLP processing.
- Go-owned infrastructure, persistence, idempotency, and Engine handoff.
- Signal, alert, and insight ledgers that preserve lineage and lifecycle state.
- Replay and backfill patterns for validation, recovery, and improvement.

The result is a system that can be integrated, operated, and extended without
requiring every new domain to invent its own event platform.

## Architecture Value

SignalOps uses a deliberate split between infrastructure ownership and
algorithm ownership.

Go owns gateway APIs, broker publishing and consuming, durable persistence,
idempotency, replay orchestration, offset discipline, API serving, and Engine
extension adapters. Python owns detector logic, feature extraction, model
scoring, explanation generation, and algorithm-specific evaluation.

The durable boundary is the broker contract. This keeps the system resilient to
process restarts and allows detector workers to evolve without embedding Python
execution inside Go services.

This architecture provides practical value:

- Runtime isolation: infrastructure code and detector code can change at
  different speeds.
- Operational resilience: broker-backed paths support retry, replay, and
  dead-letter handling.
- Auditability: durable ledgers preserve source and processing lineage.
- Extensibility: new detectors and domains can be added through contracts and
  configuration.
- Non-interference: SignalOps does not embed streaming behavior inside the
  existing Syncratic Engine.

## Trust Signals

SignalOps is designed around technical properties that matter in production
event systems.

Replayability: event history, broker coordinates, normalized ledgers, and
signal ledgers allow teams to reprocess or inspect previous behavior.

Idempotency: event IDs, idempotency keys, deterministic signal identifiers,
and upsert-oriented persistence reduce duplicate side effects.

Contract-first processing: cross-runtime event schemas under `contracts/`
define the wire shape shared by Go services and Python workers.

Lifecycle persistence: alerts and insights are derived from persisted signals
with explicit statuses rather than only emitted as transient notifications.

Failure routing: retry and dead-letter topics preserve failed work with
processing metadata so failures can be inspected and remediated.

Tenant and source context: event envelopes carry tenant, source, dataset,
domain, use-case, correlation, and timing metadata.

Engine boundary discipline: SignalOps prepares artifacts, insights, and graph
mutation proposals; the Syncratic Engine remains responsible for validating and
applying durable knowledge changes.

## Integration Posture

SignalOps is intended to integrate with external data sources and downstream
Syncratic capabilities without becoming a monolith.

For sources, the gateway and adapters provide structured entry points for push
events, scheduled pulls, bulk files, replay, and future streaming modes. Events
are normalized into contracts before downstream processing depends on them.

For algorithms, detector plugins produce structured signal outputs. The system
can support statistical detectors, change detectors, drift detectors, semantic
enrichment, and domain-specific model scoring without making any single
detector the platform.

For downstream consumers, persisted signals, alerts, insights, artifacts, and
graph proposal paths provide stable points of integration. API consumers can
inspect the current operational state while deeper Engine integrations can
evaluate which outputs should become durable knowledge.

## Operational Value

SignalOps makes production behavior easier to reason about because processing
is staged and observable.

An operator or evaluator should be able to trace a meaningful output backward:
from alert or insight, to signal, to detector identity and version, to source
event IDs, to broker coordinates, to normalized payload and evidence. That
lineage is central to debugging, audit, model evaluation, and trust.

The system also supports safer evolution. New detectors can be introduced,
schemas can be versioned, and use-case metadata can segment behavior without
rewriting the core ingestion and processing substrate.

## Example Evaluation Questions

A technical buyer can evaluate SignalOps with questions such as:

- Can we prove where an alert or insight came from?
- Can a failed event be retried or sent to a dead-letter path without losing
  evidence?
- Can detectors evolve without changing the gateway or persistence model?
- Can the platform support multiple domains with consistent contracts?
- Can event intelligence connect to knowledge graph workflows without directly
  mutating core Engine state?

SignalOps is valuable when these questions matter more than simply displaying a
stream in a dashboard.
