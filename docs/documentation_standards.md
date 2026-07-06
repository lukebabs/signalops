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

