# G119 Algorithm Signal Materialization Preflight UI Spec

Status: proposed - frontend-agent specification ready
Timestamp: 2026-07-16T00:00:00Z

## Purpose

G119 hands off the frontend work needed to expose G118 read-only materialization preflight results in the existing Algorithms / Signal Proposals UI.

## Specification

Frontend-agent specification:

- `../../../../frontend/algorithm_signal_materialization_preflight_ui_spec.md`

## Scope

The frontend should:

- call the G118 preflight endpoint with active proposal filters;
- show readiness counts and global blockers;
- show per-proposal preflight status and reason tokens;
- preserve the existing proposal list/detail/review workflow;
- clearly label the panel as read-only preflight.

## Explicitly Out Of Scope

- Materialize button or materialization mutation.
- New backend endpoints.
- New proposal review statuses.
- Production signal writes.
- Alert, insight, graph proposal, policy deployment, Syncratic, tuning, or execution controls.

## Result

Frontend-agent can implement G119 without backend scope creep or materialization semantics.
