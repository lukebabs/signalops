# MarketOps Graph Proposals Read-Only UI Specification

Status: implemented in G079; superseded by G080 review controls
Gate: G079 frontend follow-up  
Author: Codex  
Date: 2026-07-11  
Backend baseline: G079 `c95e6d9` plus live closeout `d33ad84`

## Purpose

Add read-only visibility for first-class MarketOps DSM graph proposals in the existing DSM Workbench.

G079 made graph target candidates durable in `marketops_dsm_graph_proposals` and exposed authenticated `/v1/marketops/dsm/graph-proposals` APIs. The frontend task is to make those persisted proposal records visible to operators without adding graph editing, graph visualization, or graph database mutation.

## Scope

In scope:

- Extend `/marketops/dsm` to fetch graph proposal records for the selected signal or selected first-class artifact.
- Show graph proposal counts and statuses in the existing signal detail flow.
- Show node/relationship proposal details in a compact read-only panel.
- Preserve the existing raw `graph_targets` evidence view as fallback/source evidence.
- Add API client/query/types/tests following existing G078 artifact patterns.

Out of scope:

- Decision buttons or status mutation UI.
- Graph canvas, node-link visualization, graph layout, graph editing, or drag/drop interactions.
- Production graph database writes.
- New top-level navigation or a separate graph-proposals page.
- Changing DSM Workbench `persisted` versus `signal-only` artifact semantics.
- Replacing signal/artifact detail panels.

## Backend Contract

Use existing authenticated gateway conventions. Auth is required when enabled.

List graph proposals:

```http
GET /v1/marketops/dsm/graph-proposals?tenant_id=tenant-local&app_id=marketops&domain=market_data&use_case=daily_market_surveillance&signal_id={signal_id}&limit=50
Authorization: Bearer <access_token>
```

Detail graph proposal:

```http
GET /v1/marketops/dsm/graph-proposals/{proposal_id}
Authorization: Bearer <access_token>
```

Decision endpoint exists but must not be used in this frontend task:

```http
POST /v1/marketops/dsm/graph-proposals/{proposal_id}/decision
```

## Response Shape

The list endpoint returns:

```json
{
  "graph_proposals": [
    {
      "proposal_id": "graphprop_marketops_dsm_v1_...",
      "tenant_id": "tenant-local",
      "app_id": "marketops",
      "domain": "market_data",
      "use_case": "daily_market_surveillance",
      "artifact_id": "artifact_marketops_dsm_v1_...",
      "signal_id": "sig_marketops_dsm_taxonomy_v1_...",
      "signal_type": "marketops.dsm.pinning_risk",
      "detector_id": "marketops.dsm.taxonomy_v1",
      "severity": "high",
      "confidence": 0.84,
      "event_ids": ["normalized_marketops_g079_graph_live"],
      "subject_symbol": "AAPL",
      "candidate_type": "relationship_candidate",
      "node_id": "",
      "from_node": "ticker:AAPL",
      "relationship": "EXHIBITS_SIGNAL",
      "to_node": "signal_type:marketops.dsm.pinning_risk",
      "labels": [],
      "properties": { "severity": "high" },
      "raw_candidate": { "type": "relationship_candidate" },
      "status": "proposed",
      "reviewed_by": "",
      "decision_note": "",
      "created_at": "2026-07-11T17:49:00Z",
      "updated_at": "2026-07-11T17:49:00Z"
    }
  ]
}
```

Detail endpoint returns:

```json
{
  "graph_proposal": { }
}
```

Use type guards over `unknown`. The gateway returns parsed JSON fields for `properties` and `raw_candidate`; do not `JSON.parse` those values.

## Required Implementation

### 1. Types

Add frontend types in `web/src/types.ts` near the G078 DSM artifact types:

- `MarketOpsDSMGraphProposal`
- `MarketOpsDSMGraphProposalsResponse`
- `MarketOpsDSMGraphProposalResponse`
- `MarketOpsDSMGraphProposalFilter`

Recommended status union:

```ts
export type MarketOpsDSMGraphProposalStatus = 'proposed' | 'accepted' | 'rejected' | 'superseded';
```

Recommended candidate type union:

```ts
export type MarketOpsDSMGraphProposalCandidateType = 'node_candidate' | 'relationship_candidate';
```

Keep JSON fields typed as `unknown`:

- `properties`
- `raw_candidate`

### 2. API Client

Add methods in `web/src/api/client.ts`:

```ts
listMarketOpsDSMGraphProposals(filter?: MarketOpsDSMGraphProposalFilter): Promise<MarketOpsDSMGraphProposalsResponse>
getMarketOpsDSMGraphProposal(proposalId: string): Promise<MarketOpsDSMGraphProposalResponse>
```

List query params should include only defined values:

- `tenant_id`
- `app_id`
- `domain`
- `use_case`
- `artifact_id`
- `signal_id`
- `signal_type`
- `subject_symbol`
- `candidate_type`
- `status`
- `limit`

Default limit should be `50`, consistent with artifact API behavior.

### 3. React Query Hooks

Add query keys and hooks in `web/src/api/queries.ts`:

```ts
marketOpsDSMGraphProposals(filter)
marketOpsDSMGraphProposal(proposalId)
```

Hooks:

```ts
useMarketOpsDSMGraphProposals(filter)
useMarketOpsDSMGraphProposal(proposalId)
```

Only enable detail query when `proposalId` is truthy.

### 4. DSM Workbench Integration

Update `web/src/routes/MarketOpsDsmRoute.tsx`.

When a signal is selected, query graph proposals with:

```ts
{
  tenant_id: TENANT_ID,
  app_id: 'marketops',
  domain: 'market_data',
  use_case: 'daily_market_surveillance',
  signal_id: selectedSignal.signal_id,
  limit: 50,
}
```

If a selected first-class artifact is available, it is acceptable to filter by `artifact_id` instead of or in addition to `signal_id`, but avoid issuing duplicate queries for the same selection.

Add a read-only graph proposal section to the existing detail panel near artifact/graph evidence:

- Header: `Graph Proposal Ledger`
- Count summary: total, node candidates, relationship candidates
- Status summary: proposed, accepted, rejected, superseded
- Compact list/table with:
  - status
  - candidate type
  - node id for node candidates
  - from / relationship / to for relationship candidates
  - confidence
  - updated timestamp

Selecting an item can expand inline or show a small detail block in the same panel. Do not open a modal unless the existing route already uses that pattern.

Detail block should show:

- proposal id
- artifact id
- signal id
- subject symbol
- labels
- properties JSON
- raw candidate JSON
- reviewed by / decision note / decided at when present

### 5. Empty And Error States

Use concise operational states:

- Loading: skeleton or existing loading affordance.
- Empty: `No graph proposals persisted for this signal.`
- Error: `Graph proposals unavailable.`

Do not imply the signal is invalid when graph proposals are absent. A signal can still be valid and artifact-backed while graph proposal materialization is unavailable or not yet deployed.

### 6. UI Constraints

Keep the DSM Workbench dense and operational.

- Do not add a hero, landing page, graph illustration, or separate graph route.
- Do not use a graph canvas.
- Do not add edit controls, accept/reject buttons, menus, or mutation affordances.
- Do not create nested cards. Use the existing detail-panel visual language.
- Keep text compact so long proposal ids do not overflow; truncate visually but preserve full value in copyable/selectable text or title attributes if existing patterns support that.

### 7. Tests

Add or extend frontend tests to cover:

- API client builds `/v1/marketops/dsm/graph-proposals` with filters and bearer auth.
- API client encodes proposal ids for detail fetches.
- Query keys are stable.
- DSM Workbench displays graph proposal counts when records are returned.
- DSM Workbench renders empty graph proposal state when none are returned.
- Relationship candidates without labels render without crashing.

Run:

```sh
cd web && npm test
cd web && npm run build
cd web && npm audit --audit-level=low
```

### 7.1 Test Coverage Note (applied during implementation)

The first three items above (API client path/filters/bearer auth, proposal id
encoding, stable query keys) are covered in
`web/src/api/marketopsGraphProposals.test.ts`, matching the established G078
artifact test pattern (`marketopsAssets.test.ts`).

The remaining three items ("displays counts", "renders empty state",
"relationship candidates without labels render without crashing") are verified
as **pure functions** in `web/src/lib/marketopsDsm.test.ts`
(`summarizeGraphProposals`, `formatGraphProposalLabels`,
`graphProposalSubjectLine`) rather than as React component renders. This project
ships no component test harness — `vitest` only, with no `jsdom` /
`@testing-library/react` dependency and no `test` block in `vite.config.ts` —
and the cited G078 pattern likewise has no component render tests. Adding a
render harness would expand scope beyond a read-only follow-up and risk the
`npm audit --audit-level=low` gate. The ledger UI itself is covered by
`tsc` + `vite build` (type-check + compile), and its display logic is fully
exercised through the unit-tested helpers it calls.

## Acceptance Criteria

- `/marketops/dsm` still renders existing signal/artifact workflows.
- Selecting the G079 smoke signal `sig_marketops_dsm_taxonomy_v1_g079_graph_live`, when available to the authenticated browser session, shows five graph proposals.
- The UI distinguishes three node candidates and two relationship candidates.
- Proposal status is visible and read-only.
- No decision mutation requests are issued by the UI.
- Existing `persisted` and `signal-only` artifact labels keep their G078 meaning.

## Handoff Notes

Live backend smoke record from G079 closeout:

- Signal: `sig_marketops_dsm_taxonomy_v1_g079_graph_live`
- Artifact: `artifact_marketops_dsm_v1_g079_graph_live`
- Subject: `AAPL`
- Expected graph proposals: `5`
- Expected status: all `proposed`

Positive browser validation requires an operator bearer token. Unauthenticated graph-proposal API requests should return `401 unauthorized` when gateway auth is enabled.


## G080 Note

G080 intentionally supersedes the read-only UI constraint for one narrow action surface: inline proposal review decisions in the DSM Workbench. The new controls update `marketops_dsm_graph_proposals` review status and metadata only. Graph visualization, graph editing, and production graph database writes remain out of scope.
