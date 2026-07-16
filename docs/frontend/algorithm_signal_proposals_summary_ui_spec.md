# Frontend-Agent Spec: G116 Algorithm Signal Proposal Summary UI

## Goal

Add a compact read-only summary panel for algorithm signal proposal review coverage inside the existing Algorithms / Signal Proposals UI.

This builds on:

- G113/G114 proposal list/detail/review UI.
- G115 backend summary endpoint.

The UI must help operators understand proposal review coverage and unresolved high-priority proposal risk without adding new backend behavior or production materialization.

## Existing Backend API

Use this existing endpoint only:

- `GET /v1/algorithms/signal-proposals/summary?tenant_id={tenant_id}&algorithm_id={algorithm_id}&execution_request_id={execution_request_id}&algorithm_result_id={algorithm_result_id}&status={status}&severity={severity}&correlation_id={correlation_id}`

Response envelope:

```json
{
  "algorithm_signal_proposal_summary": {
    "tenant_id": "tenant-local",
    "total_proposals": 1,
    "proposed_count": 0,
    "reviewed_count": 1,
    "rejected_count": 0,
    "superseded_count": 0,
    "reviewed_ratio": 1,
    "high_critical_unreviewed_count": 0,
    "status_counts": { "reviewed": 1 },
    "severity_counts": { "critical": 1 },
    "proposed_signal_type_counts": {
      "signalops.algorithm.change_point_candidate": 1
    },
    "algorithm_id_counts": {
      "signalops.algorithms.ruptures_change_point_v1": 1
    },
    "reviewer_counts": { "operator-local": 1 }
  }
}
```

## UI Placement

Extend the existing `Signal Proposals` area created for G114.

Recommended placement:

- Summary panel at the top of the Signal Proposals tab/panel.
- Proposal list below it.
- Proposal detail/review panel remains unchanged.

Do not create a new top-level route.

## Filter Coupling

The summary should use the same active filters as the proposal list, except `limit`.

Couple these filters:

- tenant id
- algorithm id
- execution request id
- algorithm result id, if present in the UI
- status
- severity
- correlation id

When the operator changes proposal list filters, refresh both:

- proposal list
- proposal summary

This lets the summary describe the currently visible proposal review slice.

## Required Summary Content

Show primary metrics:

- total proposals
- reviewed ratio
- proposed count
- reviewed count
- rejected count
- superseded count
- high/critical unreviewed count

Show compact breakdowns:

- status counts
- severity counts
- proposed signal type counts
- algorithm id counts
- reviewer counts

The summary should be dense and scannable. Avoid large decorative cards. A compact metrics strip plus small breakdown tables/lists is preferred.

## Visual Semantics

Use existing status/severity badge styles where available.

Recommended emphasis:

- `high_critical_unreviewed_count > 0`: prominent warning tone.
- `reviewed_ratio`: show as percentage with one decimal place or whole percent if existing patterns do that.
- Empty maps should render as `None` or an equivalent compact empty state, not raw `{}`.

Do not imply that reviewed proposals are accepted, materialized, or deployed.

## Empty And Loading States

Loading:

- Show a compact loading state in the summary panel only.
- Do not block the existing proposal list if the summary query is loading independently.

Empty:

- If `total_proposals=0`, show: “No proposal summary for this filter.”

Error:

- Follow existing API error rendering patterns.
- The proposal list should remain usable if only the summary query fails.
- Do not expose bearer tokens, auth headers, secrets, raw environment variables, or upstream internals.

## Out Of Scope

- New backend endpoints.
- Proposal decision mutations beyond what G114 already implemented.
- Bulk decisions.
- Production `signal.v1` creation.
- Alert or insight creation.
- Graph proposal creation.
- Algorithm execution triggering.
- Algorithm tuning.
- Policy deployment.
- Syncratic Ask/Search.
- New navigation shell.

## Suggested Implementation Areas

Frontend-agent should inspect the G114 implementation first. Likely touch points:

- algorithms API client: add `getAlgorithmSignalProposalSummary` or equivalent.
- React Query hooks: add summary query keyed by tenant and active filters.
- Signal Proposals route/panel: render summary above list.
- Proposal filter state: share with summary query.
- Tests for API path construction, envelope parsing, filter coupling, empty states, and warning rendering.

Use existing app styling and component conventions.

## Validation Checklist

Frontend-agent should verify:

- Summary endpoint is called with active proposal filters.
- `limit` is not sent to the summary endpoint.
- Primary metrics render correctly.
- `reviewed_ratio` displays as a human-readable percentage.
- `high_critical_unreviewed_count > 0` has a visible warning state.
- Empty summary renders cleanly for `total_proposals=0`.
- Summary query errors do not crash or block the proposal list/detail workflow.
- Existing G114 proposal list/detail/review workflow still works.
- No materialization, alert, insight, graph, deploy, tune, or run controls are added.
- Desktop and mobile layouts remain usable.

## Handoff Note

Backend G115 is implemented and validated. A local authenticated smoke returned this summary for `tenant-local` after the G112 review smoke:

- `total_proposals=1`
- `reviewed_count=1`
- `reviewed_ratio=1`
- `high_critical_unreviewed_count=0`
- `proposed_signal_type_counts.signalops.algorithm.change_point_candidate=1`

The UI must also handle future non-zero proposed and high/critical unreviewed counts.
