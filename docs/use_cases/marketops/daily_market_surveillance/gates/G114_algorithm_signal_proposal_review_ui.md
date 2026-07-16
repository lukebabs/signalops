# G114 Algorithm Signal Proposal Review UI

Status: implemented - automated tests, typecheck, and build green; live browser validation pending (auth gate)
Timestamp: 2026-07-16T05:24:00Z

## Purpose

G114 implements the G113 frontend-agent specification: operator visibility and
review controls for the G111/G112 `algorithm_signal_proposals` workflow inside
the existing MarketOps Algorithms route. It lets operators inspect
algorithm-derived candidate signal proposals, review their evidence, and record
review decisions. It does not materialize production signals and creates no
alerts, insights, graph proposals, policies, deployments, or Syncratic work.

## Implemented Scope

- Extended the existing `/marketops/algorithms` route (G109) with a `Signal
  Proposals` review panel. No new top-level route; the Definitions / Execution
  Requests / Results drill-down flow is left intact.
- Proposal list table with all spec-required columns (proposal id, proposed
  signal type, status, severity, score, confidence, algorithm id, execution
  request id, algorithm result id, correlation id, reviewed by, decided at,
  updated at) inside an `overflow-x-auto` dense table.
- Filters: status (default `proposed`), severity, algorithm id, execution
  request id, correlation id, and limit (default `50`). Selecting an execution
  request in the existing drill-down carries its id into the proposals filter;
  the field stays editable so operators can view all recent tenant proposals.
- Selected-proposal detail panel rendering every spec-required field, lineage id
  lists (source event ids, evidence refs), and collapsible formatted
  `proposal_payload` / `rationale` JSON via the existing `JsonViewer`.
- Bounded review controls in the detail panel only: a status selector
  (`reviewed` / `rejected` / `superseded` / restore to `proposed`), a note
  textarea, and a single neutral `Save review` button. A non-empty note is
  required for `rejected` and `superseded` (frontend validation). Submit is
  disabled while the request is in flight.
- New API client methods (`listAlgorithmSignalProposals`,
  `getAlgorithmSignalProposal`, `decideAlgorithmSignalProposal`) and React Query
  hooks (`useAlgorithmSignalProposals`, `useAlgorithmSignalProposal`,
  `useDecideAlgorithmSignalProposal`) plus an extracted
  `applyAlgorithmSignalProposalDecisionResult` invalidation helper. The decision
  POST sends `{tenant_id, status, note, metadata}` with no actor header — the
  gateway derives the reviewer via `replayActor`, matching the promotion-
  candidate decision pattern.
- Review semantics stay review-only: status badges use neutral / positive-but-
  not-production / negative / muted tones, and no `Accept`, `Materialize`,
  `Create Signal`, `Promote`, `Deploy`, `Create Alert`, or `Create Insight`
  controls exist. `reviewed` never reads as accepted or deployed.
- Empty and error states follow existing patterns (`No algorithm signal proposals
  found.`, `Select a proposal to inspect its evidence.`, `ErrorState` for list
  failures, a safe inline message for mutation failures). No bearer tokens,
  auth headers, secrets, or upstream internals are surfaced.

## Validation Performed

- `npx vitest run` (full web suite): 21 files, 273 tests passed. New G114
  coverage: proposal list path construction + filters + tenant + bearer +
  default limit; detail path URL-encoding; decision POST body / path / bearer /
  no-actor-header; list + detail envelope parsing; query invalidation seeds the
  detail cache and invalidates only the proposal list/detail prefixes;
  `summarizeAlgorithmSignalProposal` scalar / lineage / JSON / review-metadata
  reads plus non-object collapse; `algorithmProposalStatusStyle` mapping +
  fallback (including `accepted` rendering neutral, not positive).
- `npx tsc --noEmit`: clean.
- `npm run build` (`tsc && vite build`): succeeded; `AlgorithmsRoute` chunk
  builds and the rest of the bundle is unaffected.

## Out Of Scope

- Production `signal.v1` creation, alert/insight creation, graph proposal
  creation, algorithm execution triggering, algorithm tuning, policy deployment,
  Syncratic Ask/Search, bulk decisions, multi-review history, and new backend
  endpoints.
- Live browser validation against the local Docker stack: the gateway requires a
  bearer token (auth gate active), and the IdP login flow requires a browser.
  This is the same follow-on validation step tracked separately for the broader
  auth-gated surface (G052/G053); it does not change the no-materialization
  boundary.

## Result

- G114 closes the G113 specification as a review-only operator surface over the
  G111/G112 proposal ledger. Automated tests, typecheck, and build are green;
  browser validation remains the documented follow-on.

## References

- Frontend spec: `../../../../frontend/algorithm_signal_proposals_review_ui_spec.md` (G113)
- Proposal ledger APIs: `G111_algorithm_result_signal_proposal_design.md`
- Review lifecycle: `G112_algorithm_signal_proposal_review.md`
- Algorithms route (extended): `G109_algorithm_execution_visibility_ui.md`
