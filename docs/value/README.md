# SignalOps Value Documentation

## Document Change Management

Status: evolving value narrative.

Last updated: 2026-07-12.

Primary scope: the general SignalOps core engine, independent of any single
domain application.

Document owner: SignalOps maintainers.

Review expectation: update these documents whenever the system's positioning,
implemented capability, or public-facing value claims materially change.

Change policy:

- Treat these documents as product and strategy framing, not as executable
  contracts.
- Prefer conservative claims that can be traced to implemented behavior or
  accepted architecture.
- Keep technical guarantees aligned with the canonical specifications linked
  below.
- Add domain-specific value narratives under `docs/use_cases/` when the value
  depends on a particular app, domain, or workflow.
- Record meaningful documentation changes alongside related build or design
  changes according to `docs/documentation_standards.md`.

Canonical references:

- `../Syncratic_SignalOps_Infrastructure_Specification.md`
- `../Syncratic_SignalOps_Processing_Specification.md`
- `../api.md`
- `../use_cases/README.md`

## Purpose

This folder explains the value of SignalOps from multiple perspectives. It is
designed to help people understand why the system exists, what problems the
core engine solves, and how its architecture creates durable operational
advantage.

The initial value narrative is intentionally general. SignalOps can support
market data, CRM events, security telemetry, IoT streams, procurement signals,
and operational metrics, but the core engine is not defined by any one of
those domains.

## Core Engine Overview

SignalOps turns continuous events into replayable, explainable, evidence-backed
operational intelligence.

At the core, the system accepts raw events, normalizes them into stable
contracts, routes them through durable broker topics, runs detector plugins,
persists signal lineage, derives alerts and insights, and prepares graph-aware
evidence for downstream evaluation. The engine is designed around idempotency,
replay, auditability, tenant-aware boundaries, and non-interference with the
existing Syncratic Engine.

The value is not only that SignalOps can detect something. The value is that it
can explain what changed, preserve the evidence, make processing repeatable,
and let downstream systems decide what should become durable knowledge.

## Audience Guide

- `product_executive_value.md`: for product, executive, commercial, and
  strategy readers who need to understand the business problem and operational
  outcomes.
- `technical_buyer_value.md`: for architects, platform owners, technical
  buyers, and evaluators who need trust signals about integration, reliability,
  replayability, and extensibility.
- `internal_builder_value.md`: for engineers, product builders, and internal
  stakeholders who need a shared explanation of what SignalOps should become
  and how to extend the value narrative responsibly.

## What These Docs Are Not

These documents are not API specifications, deployment runbooks, schema
contracts, or gate evidence. When a value statement depends on exact behavior,
the linked technical specs remain authoritative.
