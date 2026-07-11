# Architecture Notes

Use this folder for MarketOps Daily Market Surveillance architecture and data-flow documentation.

Belongs here:

- DSM domain model and taxonomy explanations.
- Signal versus artifact persistence semantics.
- Graph proposal model and acceptance/storage direction.
- Reconciliation between target MarketOps architecture and implemented gates.

Current key rule:

- A persisted DSM artifact is a first-class record in `marketops_dsm_artifacts` derived from a persisted signal. It is linked by `artifact_id` and `signal_id`; it does not replace the canonical `signal_ledger` record.
