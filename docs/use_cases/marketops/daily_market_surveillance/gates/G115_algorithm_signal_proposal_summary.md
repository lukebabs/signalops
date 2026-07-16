# G115 Algorithm Signal Proposal Summary

Status: implemented
Timestamp: 2026-07-16T00:00:00Z

## Purpose

G115 adds a read-only summary/readiness surface over `algorithm_signal_proposals`. The goal is to help operators understand review volume and unresolved high-priority proposal risk before any future materialization gate.

## Implemented Scope

- Added `SummarizeAlgorithmSignalProposals` to the algorithm repository contract.
- Added Postgres aggregate query over existing `algorithm_signal_proposals` rows.
- Added API endpoint:
  - `GET /v1/algorithms/signal-proposals/summary`
- Added API response with:
  - total proposals
  - proposed/reviewed/rejected/superseded counts
  - reviewed ratio
  - high/critical unreviewed count
  - counts by status
  - counts by severity
  - counts by proposed signal type
  - counts by algorithm id
  - counts by reviewer

## Filters

The summary endpoint accepts the same core filters as the proposal list endpoint:

- `tenant_id`
- `algorithm_id`
- `execution_request_id`
- `algorithm_result_id`
- `status`
- `severity`
- `correlation_id`

## Explicitly Out Of Scope

- New mutations.
- Production signal materialization.
- Alert or insight creation.
- Graph proposal creation.
- Frontend changes.
- Policy deployment.

## Validation

- Focused API tests cover summary aggregation and filter propagation.
- Focused algorithm/generator tests pass against the expanded repository interface.
- Full Go test suite passed.
- JSON schema validation passed.
- Gateway build passed.
- Authenticated live summary smoke returned the expected reviewed proposal counts for `tenant-local`.

## Next Gate

G116 should be chosen after G114 browser validation. Likely options are frontend summary visibility or a formal materialization design gate, but production materialization should remain blocked until enough reviewed proposal evidence exists.
