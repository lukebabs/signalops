# Operations Notes

Use this folder for MarketOps Daily Market Surveillance runbooks, smoke checks, replay checks, and environment notes.

Current recurring validations:

- Rebuild/redeploy relevant services after backend or frontend changes.
- Apply Postgres migrations with `make compose-storage-migrate` when new relational tables are added.
- Publish bounded events or signals for live smoke validation.
- Verify Redpanda consumer lag for the relevant consumer group.
- Verify Postgres rows in `signal_ledger`, `marketops_dsm_artifacts`, `alert_ledger`, and `insight_ledger`.
- For auth-enabled APIs, unauthenticated probes should return `401`; positive browser/API validation requires a bearer token.

## Back-Test Operations

`backtest_substrate.md` covers the implemented G081 back-test workflow, first smoke scenario, safety controls, CLI command, read-only inspection APIs, and G094 calibration-readiness campaign direction. Back-tests are experiments isolated from operational state: a back-test run must never write to `signal_ledger`, `alert_ledger`, `insight_ledger`, `marketops_dsm_artifacts`, `marketops_dsm_graph_proposals`, or any production graph database. Replay remains the correct tool for republishing existing ledgers through the operational pipeline.

## Syncratic Context Operations

`syncratic_context_windows.md` covers the proposed G088 validation shape for deterministic context windows and synthesized insights over existing ledgers.

## Back-Test Input Ingestion

- `backtest_input_ingestion.md`: bounded Massive puller smoke for creating normalized MarketOps input rows before calibration campaigns.

## Reviewed Label Workflow

- `reviewed_label_workflow.md`: operator workflow for increasing real reviewed graph-proposal labels through existing G080, G084, G085, and G094 APIs.

## Market State Materialization

- `market_state_materialization.md`: bounded G137 AAPL dry-run, write, idempotency, lineage, and quality-blocking checks over existing persisted evidence.

## Research Hypothesis Evaluation

- `hypothesis_evaluation.md`: bounded G138 AAPL dry-run, write, idempotency, trigger/rejection, and reason-code checks over existing G137 states.

## Opportunity Building

- `opportunity_building.md`: bounded G139 AAPL grouping, overlap, conflict, research-only, empty-ledger, and idempotency checks over G138 evaluations.


## Forward Outcome Evaluation

- `outcome_evaluation.md`: bounded G140 AAPL dry-run/write, maturity, missing-price, point-in-time, lineage, and idempotency checks over triggered evaluations and opportunities.


## Historical Research Pipeline

- `historical_research_pipeline.md`: G141 bounded AAPL equity acquisition, strict coverage preflight, dry-run diagnostics, and historical options-data boundary.

## Prospective Options Capture

- `prospective_options_capture.md`: G142 bounded per-session Massive snapshot capture, readiness inspection, retry/resume, and scheduling policy.
- `daily_postclose_pipeline.md`: exchange-aware bounded equity, normalization, prospective options, and ten-symbol cohort orchestration with a user-systemd timer.
