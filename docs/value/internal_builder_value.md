# SignalOps Internal Builder Value

## Evolution Note

This document is for internal builders who need to explain, extend, or evaluate
SignalOps. It should evolve with implementation reality. When this document and
the technical specs disagree, the technical specs win and this document should
be updated.

## Why SignalOps Exists

SignalOps exists because Syncratic needs a disciplined way to reason over
continuous operational change without weakening the existing document-centered
Engine.

The core Syncratic Engine remains the authority for durable knowledge,
retrieval, graph construction, and Ask workflows. SignalOps adds a separate
event intelligence subsystem that can ingest streams, detect meaningful
patterns, preserve evidence, and propose downstream knowledge updates through
explicit extension contracts.

The internal builder mental model is:

SignalOps is the system that turns event flow into evaluated signal flow.

It should not be treated as a one-off MarketOps pipeline, a generic queue
consumer, or a dashboard backend. Those may be expressions of the system, but
they are not the system's purpose.

## The Core Problem To Solve

The core problem is not "receive events." The core problem is to make
continuous data usable as trustworthy operational intelligence.

That requires the platform to answer:

- What event did we receive?
- How did we normalize it?
- Which detector evaluated it?
- What signal was emitted?
- What evidence supports the result?
- What lifecycle state is the output in?
- Can we replay or inspect the processing path?
- Should this output influence the knowledge graph?

Any implementation that cannot answer those questions is likely bypassing the
core value of SignalOps.

## How To Explain The Engine

Use this explanation as the internal default:

SignalOps is a domain-neutral event intelligence engine. It ingests raw
operational events, normalizes them into replayable contracts, routes them
through durable processing, runs detector plugins, persists signal lineage,
derives alerts and insights, and prepares evidence-backed outputs for Syncratic
Engine evaluation.

When speaking to different audiences, adjust the emphasis:

- For product and leadership: emphasize operational intelligence, reusable
  platform leverage, and safe expansion across domains.
- For technical evaluators: emphasize contracts, replayability, idempotency,
  runtime boundaries, and evidence lineage.
- For builders: emphasize separation of concerns, extension points, and the
  discipline of keeping domain logic out of core infrastructure.

## What SignalOps Should Do

SignalOps should provide the common substrate for event intelligence:

- Ingest external or scheduled source events.
- Preserve tenant, source, dataset, time, correlation, and entity context.
- Normalize raw events into stable contracts.
- Publish and consume through durable broker topics.
- Run detector plugins through clear runtime boundaries.
- Persist signal, alert, and insight lineage.
- Route retryable and invalid work through explicit failure paths.
- Support replay, backfill, and evaluation workflows.
- Produce evidence-backed outputs suitable for Engine extension APIs.

## What SignalOps Should Not Do

SignalOps should not collapse every concern into the core engine.

Avoid these patterns:

- Putting domain-specific business logic into the gateway when it belongs in a
  detector, adapter, or use-case layer.
- Treating detector output as accepted knowledge without evaluation.
- Bypassing event contracts for convenience.
- Writing Python algorithm behavior directly into Go infrastructure services.
- Making a specialized use case define the whole platform.
- Mutating existing Syncratic Engine graph or Ask workflows directly.

These boundaries are what make the system reusable.

## Extension Principles

When adding new use cases, start from the core engine and add only what is
domain-specific.

Use `docs/use_cases/` for app, domain, or workflow-specific behavior. Keep
cross-use-case mechanics in top-level docs and keep value framing in
`docs/value/`.

When adding detector behavior, preserve the distinction between:

- Event: what happened.
- Normalized event: what the platform can process consistently.
- Signal: what a detector believes is meaningful.
- Alert: what requires operational attention.
- Insight: what may guide interpretation or action.
- Artifact or graph proposal: what may become durable knowledge after review
  or Engine evaluation.

This vocabulary should stay consistent across documents, code, and UI labels.

## Builder Checklist

Before claiming a new SignalOps capability is part of the core engine, verify:

- It works across or can generalize beyond one domain.
- It preserves source and processing lineage.
- It fits the contract-first event model.
- It does not bypass replay, retry, or idempotency expectations.
- It does not directly interfere with existing Syncratic Engine workflows.
- It can be explained in terms of events, signals, evidence, and lifecycle.

Before creating a domain-specific value narrative, verify:

- The core value docs still describe the shared engine accurately.
- The use-case docs explain only the domain-specific behavior.
- Any stronger claims are supported by implementation or explicitly marked as
  roadmap intent.

## How This Narrative Should Evolve

This document should be updated when:

- A new core processing stage is added.
- A new class of detector or runtime boundary becomes supported.
- Alert, insight, artifact, or graph proposal behavior changes materially.
- A domain use case proves a reusable pattern that belongs in the core engine
  narrative.
- A value claim becomes too broad for current implementation reality.

The goal is not to freeze positioning. The goal is to keep the positioning
honest, useful, and aligned with the system being built.
