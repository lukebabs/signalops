# Signal And Artifact Persistence

Status: current as of G078  
Use case: MarketOps Daily Market Surveillance

## Purpose

This note defines how persisted DSM signals, first-class DSM artifacts, graph proposal payloads, and lifecycle records relate to each other. It exists because the DSM Workbench uses the word `persisted` in the Ledger column, and that label refers specifically to artifact-ledger materialization, not signal persistence.

## Canonical Records

### `signal_ledger`

`signal_ledger` is the canonical durable signal store.

A row here means a validated `signal.v1` event was consumed by `signal-persister` and persisted. MarketOps DSM signals include:

- stable `signal_id`
- `app_id=marketops`
- `domain=market_data`
- `use_case=daily_market_surveillance`
- `detector_id=marketops.dsm.taxonomy_v1`
- `signal_type`, severity, confidence, entities, metrics, semantic evidence, graph targets, and source event ids

### `marketops_dsm_artifacts`

`marketops_dsm_artifacts` is the first-class DSM artifact ledger added in G077.

A row here means the signal persister found a DSM artifact proposal inside the signal semantic evidence and materialized it as its own durable artifact record. It is linked back to the signal by:

- `artifact_id`
- `signal_id`
- `signal_type`
- `detector_id`
- source/app/use-case metadata

The artifact ledger stores the extracted artifact JSON, semantic evidence, graph targets, supporting metrics, subject symbol, quality issues, and source event ids.

### `alert_ledger` and `insight_ledger`

These lifecycle ledgers are derived from persisted signals:

- every valid signal derives one active insight
- medium/high/critical signals derive one open alert

They are lifecycle/operator records, not replacements for the signal or artifact ledgers.

## Frontend Label Semantics

In `/marketops/dsm`, the table column named `Ledger` has two values:

- `persisted`: the signal has a matching first-class artifact row in `marketops_dsm_artifacts`.
- `signal-only`: the signal exists in `signal_ledger`, and may still contain artifact proposal JSON in semantic evidence, but no matching first-class artifact row was returned by the artifact API query.

Therefore, `persisted` does not mean "this is the persistent signal." The signal is already persistent when it appears in the DSM Workbench. The label means "the artifact for this signal is also persisted as a first-class artifact record."

## Current Data Flow

1. Python worker emits a validated `signal.v1` event for `marketops.dsm.taxonomy_v1`.
2. `signal-persister` validates and persists the event into `signal_ledger`.
3. `signal-persister` derives alert and insight lifecycle rows.
4. `signal-persister` extracts DSM artifact proposals from `semantic_evidence` and upserts them into `marketops_dsm_artifacts`.
5. Gateway serves signals through shared `/v1/signals` APIs and first-class artifacts through `/v1/marketops/dsm/artifacts` APIs.
6. DSM Workbench joins signal data and artifact API data client-side for operator inspection.

## Non-Goals In Current Gates

The current artifact ledger is not graph acceptance. Graph target payloads are proposal evidence only until a later gate creates graph proposal acceptance/storage.

The artifact ledger is not an independent artifact-building service. Current artifacts are materialized from signal semantic evidence inside the existing signal persistence transaction.

## Next Direction

G079 should add graph proposal acceptance/storage or a similarly explicit backend boundary for graph candidates, using `marketops_dsm_artifacts` and persisted signal graph targets as inputs.
