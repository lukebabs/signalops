# G123 Algorithm Signal Materialization Action UI Spec

Status: proposed - frontend-agent specification ready
Timestamp: 2026-07-16T00:00:00Z

## Purpose

G123 hands off the frontend work needed to expose the G122 single-proposal materialization mutation in the existing Algorithms / Signal Proposals UI.

## Specification

Frontend-agent specification:

- `../../../../frontend/algorithm_signal_materialization_action_ui_spec.md`

## Scope

The frontend should:

- enable a materialize action only for selected reviewed proposals with eligible preflight status;
- require explicit confirmation before POST;
- call the G122 materialization endpoint;
- show succeeded, duplicate, blocked, and failed materialization results;
- show selected-proposal materialization ledger rows;
- preserve existing proposal review, summary, and preflight workflows.

## Explicitly Out Of Scope

- Bulk materialization.
- New backend endpoints.
- Policy deployment controls.
- Alert, insight, graph proposal, DSM taxonomy, Syncratic, tuning, or execution controls.
- New navigation shell.

## Result

Frontend-agent can implement G123 without expanding materialization beyond the G122 single-proposal backend contract.
