# SignalOps Documentation Standards

## Purpose

SignalOps documentation is part of the build artifact. Design decisions,
implementation gates, verification results, and audit-relevant changes must be
recorded as the subsystem is built.

## Required Documentation During Build

Every meaningful build step must update documentation in the same change set.

Required records:

- `docs/build_journal.md`: ongoing progress journal.
- `docs/gate_audit.md`: gate-by-gate audit log.
- Relevant architecture or implementation specs when behavior, contracts, or
  scope changes.



## Use-Case Documentation Layout

Use-case-specific documentation belongs under `docs/use_cases/`.

Folder pattern:

```text
docs/use_cases/{app_id}/{use_case}/
  README.md
  architecture/
  api/
  frontend/
  operations/
  gates/
```

Use this structure when behavior is scoped by `app_id`, `domain`, or `use_case` metadata. Examples include MarketOps daily surveillance behavior, DSM taxonomy semantics, use-case-specific APIs, operator UI labels, and live validation runbooks.

Top-level docs remain canonical for cross-use-case contracts:

- `docs/api.md` for shared API contracts.
- `docs/deployment.md` for shared deployment behavior.
- `docs/python_worker.md` for worker behavior that is not only one use case.
- `docs/build_journal.md` and `docs/gate_audit.md` for chronological implementation and audit records.

When a new specialized use case is introduced, create its folder and `README.md` in the same change set as the first meaningful documentation for that use case.

## Timestamp Standard

All audit timestamps must use UTC ISO 8601 format:

```text
YYYY-MM-DDTHH:MM:SSZ
```

Example:

```text
2026-07-06T20:02:13Z
```

## Journal Entry Format

Each journal entry must include:

- timestamp
- summary
- files changed
- rationale
- verification performed
- next step

## Gate Audit Format

Each gate record must include:

- gate id
- timestamp
- gate name
- status
- criteria
- evidence
- reviewer or actor
- follow-up items

Allowed gate statuses:

- `planned`
- `in_progress`
- `passed`
- `failed`
- `deferred`

## Minimum Verification Expectations

Documentation-only changes require:

- file readback or diff review
- git status check

Code changes require, at minimum:

- relevant unit tests or explicit reason tests were not run
- build or static validation when available
- documentation update for new behavior or changed contracts

Infrastructure/deployment changes require:

- rendered manifest or dry-run validation when available
- configuration documentation
- rollback or recovery notes when relevant

