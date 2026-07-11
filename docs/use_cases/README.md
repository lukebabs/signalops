# SignalOps Use-Case Documentation

SignalOps is domain-neutral. Use-case documentation lives here when behavior is specific to an app/domain/use-case tuple rather than the core platform.

Current active use-case folders:

- `console/general/`: default Console behavior where `app_id=console` and `use_case=general`.
- `marketops/daily_market_surveillance/`: MarketOps market-data surveillance where `app_id=marketops`, `domain=market_data`, and `use_case=daily_market_surveillance`.

Core platform contracts that apply across use cases remain in top-level docs such as `docs/api.md`, `docs/deployment.md`, and `docs/python_worker.md`. Gate-by-gate evidence remains in `docs/build_journal.md` and `docs/gate_audit.md`.

## Folder Pattern

Each concrete use case should use this structure when enough documentation exists:

- `architecture/`: domain model, persistence semantics, data flow, and target architecture reconciliation.
- `api/`: use-case-specific API behavior and examples that supplement `docs/api.md`.
- `frontend/`: use-case UI semantics, labels, operator validation checklists, and frontend-agent specs.
- `operations/`: deployment, environment, replay, smoke validation, and runbook notes.
- `gates/`: gate-specific implementation notes that are too detailed for the global audit journal.

Do not duplicate the global API or deployment docs wholesale. Add concise use-case notes here and link to the canonical core documents.
