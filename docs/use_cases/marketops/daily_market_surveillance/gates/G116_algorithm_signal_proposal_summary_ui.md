# G116 Algorithm Signal Proposal Summary UI

Status: implemented - automated tests, typecheck, and build green; live browser validation pending (auth gate)
Timestamp: 2026-07-16T06:25:00Z

## Purpose

G116 implements the G116 frontend spec: a compact read-only review-coverage
summary panel over the G115 summary endpoint, placed above the G114 proposal
list inside the existing Signal Proposals area. It helps operators understand
proposal review coverage and unresolved high-priority proposal risk for the
currently filtered slice, without adding backend behavior or any production
materialization.

## Implemented Scope

- Added a compact `ProposalSummaryPanel` at the top of the G114 Signal Proposals
  panel (below the filter row, above the list). The proposal list and
  detail/review workflow are unchanged.
- Coupled the summary to the active proposal list filters (tenant id, algorithm
  id, execution request id, status, severity, correlation id). `limit` is never
  sent to the summary endpoint (it aggregates the whole matched slice).
  `algorithm_result_id` is omitted because it is not exposed as a list filter in
  the G114 UI.
- Primary metrics strip: total proposals, reviewed ratio (as a percentage),
  proposed / reviewed / rejected / superseded counts, and high/critical
  unreviewed count. The high/critical unreviewed metric renders in a prominent
  warning tone when greater than zero.
- Compact breakdown chip lists for status, severity, proposed signal type,
  algorithm id, and reviewer counts. Status and severity reuse the existing
  proposal-status / severity styles; empty maps render as `None`, never `{}`.
- Independent loading / error / empty states scoped to the panel: a compact
  loading line, a safe inline error (no tokens/headers/secrets), and
  `No proposal summary for this filter.` when `total_proposals=0`. None of these
  block the proposal list or detail.
- New API client method `getAlgorithmSignalProposalSummary` and React Query hook
  `useAlgorithmSignalProposalSummary`, plus an extracted
  `summarizeAlgorithmSignalProposalSummary` view summarizer and
  `algorithmCountEntries` ordering helper. The G114 decision invalidation helper
  now also invalidates the summary prefix so coverage refreshes after a review.
- `reviewed` is presented as coverage only; it never implies accepted,
  materialized, or deployed. No new materialization, alert, insight, graph,
  deploy, tune, or run controls were added.

## Validation Performed

- `npx vitest run` (full web suite): 21 files, 281 tests passed. New G116
  coverage: summary path construction with coupled filters and no `limit`;
  tenant defaulting and unset-filter omission; summary envelope parsing;
  `summarizeAlgorithmSignalProposalSummary` metrics + ordered breakdowns +
  non-object collapse; `algorithmCountEntries` ordering + non-numeric drop;
  `formatPercent` whole/fractional/non-finite formatting; and the G114 decision
  invalidation now also covers the summary prefix.
- `npx tsc --noEmit`: clean.
- `npm run build` (`tsc && vite build`): succeeded.

## Out Of Scope

- New backend endpoints, proposal decision mutations beyond G114, bulk
  decisions, production `signal.v1` creation, alert/insight creation, graph
  proposal creation, algorithm execution triggering, algorithm tuning, policy
  deployment, Syncratic Ask/Search, and a new navigation shell.
- Live browser validation against the local Docker stack: the gateway requires a
  bearer token (auth gate active) and the IdP login flow requires a browser.
  This is the same follow-on validation step tracked for the broader
  auth-gated surface; it does not change the no-materialization boundary.

## Result

- G116 closes the G116 specification as a read-only review-coverage surface over
  the G115 summary endpoint. Automated tests, typecheck, and build are green;
  browser validation remains the documented follow-on.

## References

- Frontend spec: `../../../../frontend/algorithm_signal_proposals_summary_ui_spec.md`
- Summary endpoint: `G115_algorithm_signal_proposal_summary.md`
- Proposal review UI (extended): `G114_algorithm_signal_proposal_review_ui.md`
