# G139 Frontend-Agent Specification: MarketOps Opportunities Workbench

Status: ready for frontend-agent implementation after the G139 backend is deployed.

## Objective

Add an analyst-facing MarketOps Opportunities workbench that turns compatible hypothesis evaluations into a fast triage queue. The experience must answer, with minimal navigation:

1. What opportunity needs attention?
2. Why is it ranked here?
3. Which independent hypotheses and evidence support it?
4. What conflicts, quality limits, or invalidating evidence reduce confidence?
5. Is this research-only or operationally materialized?

The page is an inspection workflow. G139 has no opportunity review, trade, signal-materialization, Syncratic Ask, or build mutation API.

## Backend Contract

Use the existing bearer-authenticated API client.

### List

```text
GET /v1/marketops/opportunities
```

Supported query parameters:

- `tenant_id`
- `app_id`
- `opportunity_id`
- `asset_id`
- `symbol`
- `direction`
- `horizon`
- `lifecycle_status`
- `research_only=true|false`
- `session_start=YYYY-MM-DD`
- `session_end=YYYY-MM-DD`
- `limit` (default 50, max 200)

Response envelope:

```json
{"opportunities": []}
```

### Detail

```text
GET /v1/marketops/opportunities/{opportunity_id}?tenant_id={tenant_id}
```

Response envelope:

```json
{"opportunity": {}}
```

### Supporting Reads

Use bounded supporting requests only when required:

- `GET /v1/marketops/hypothesis-evaluations` for linked evaluation detail and empty-queue rejection diagnostics.
- `GET /v1/marketops/hypotheses/{key}/{version}?tenant_id=...` only when the analyst opens a hypothesis contribution.
- `GET /v1/marketops/evidence/{evidence_id}` only when the analyst opens an evidence row.
- `GET /v1/marketops/states/{market_state_id}/lineage` only from an explicit state-lineage action.

Do not fan out one request per row during list rendering. Fetch opportunity detail on selection; fetch linked records only when their section is opened, and cache them with existing TanStack Query patterns.

## Required Types

Add typed contracts for the opportunity record, filters, list envelope, and detail envelope. Preserve unknown future lifecycle/direction strings in rendering while strongly typing current values:

- lifecycle: `emerging`, `active`, `strengthening`, `weakening`, `invalidated`, `resolved`, `expired`;
- direction: `upside`, `downside`, `non_directional`.

Important fields:

- identity and tenant/application fields;
- opened and last-evaluated dates;
- direction, horizon, lifecycle, and `research_only`;
- opportunity, confidence, domain-diversity, and conflict scores;
- hypothesis, conflict, signal, supporting-evidence, and invalidating-evidence IDs;
- deterministic summary and structured `opportunity_payload`;
- version, build run, deterministic key, and timestamps.

Treat array and payload fields defensively. Missing optional data must render as unavailable, not as zero evidence.

Payload-sourced and nullable fields (authoritative):

- `opportunity_payload` embeds the structured detail: `contributions` (per-contribution `evaluation_id`, `hypothesis_key`, `hypothesis_version`, `domain`, and `trigger_score` / `confidence_score` / `quality_score`), `overlap_suppressed_evaluation_ids`, `conflicting_evaluation_ids`, `hypothesis_families`, `scoring_version`, and `research_only`. Render Contributions and suppressed-overlap IDs from this payload; the top-level DTO also carries `conflicting_evaluation_ids`, `conflict_score`, and `invalidating_evidence_ids`.
- Per-contribution `reason_codes` are NOT in the payload — they come from the linked hypothesis-evaluations supporting read (match by `evaluation_id`). Fetch that once (scoped, bounded) when the Contributions section is open.
- Hypothesis-evaluation scores (`trigger_score`, `confidence_score`, `magnitude_score`, `rarity_score`, `persistence_score`, `corroboration_score`, `quality_score`) and evidence scores (`magnitude`, `rarity_score`, `persistence_score`, `quality_score`) are server-nullable (omitempty pointers) — render absent values as unavailable, not 0.
- There is no separate `data_quality_warnings` field; do not render one.

## Navigation

- Add `/marketops/opportunities` to MarketOps routing only.
- Add `Opportunities` to MarketOps navigation near `Assets` and `DSM`; it must not appear in Console navigation.
- Use a relevant Lucide icon such as `Telescope` or `Target` from the installed icon library.
- Preserve the selected opportunity in `?opportunity_id=` so refresh/back/forward navigation retains analyst context.

## Workbench Layout

This is a dense operational surface, not a landing page.

### Header And Filter Bar

Use a compact page header with title `Opportunities` and a one-line current-scope status. Do not add descriptive marketing copy.

Place one restrained filter toolbar directly below it:

- symbol input;
- lifecycle menu;
- direction segmented control or menu;
- horizon menu;
- date range controls;
- research-only toggle;
- refresh icon button with tooltip;
- clear-filters icon button with tooltip.

Do not wrap each filter in a card. Keep controls stable in size and allow them to wrap cleanly on narrow screens.

### Queue And Detail

Desktop uses a two-pane master/detail layout:

- left pane: 38-44% width, minimum 360px;
- right pane: remaining width;
- both remain within one page band, without nested cards;
- selecting a queue row updates the detail pane without route loss.

Mobile uses a list-first flow. Selecting an item opens a full-width detail view with a familiar back icon button. Do not squeeze both panes side by side.

### Opportunity Queue

Sort by `opportunity_score` descending, then latest evaluated date. Each row must expose enough information to triage without opening it:

- symbol and direction;
- lifecycle and research-only badge;
- concise deterministic summary, clamped to two lines;
- opportunity and confidence scores;
- hypothesis count and independent-domain count;
- conflict indicator when `conflict_score > 0`;
- last evaluated date and horizon.

Use restrained status color plus text/icon; never communicate direction, conflict, or lifecycle by color alone. Keep row height stable and keyboard-focusable. `Enter` selects the focused row; standard tab navigation must reach every command.

### Detail Information Order

The detail pane must prioritize action-oriented interpretation:

1. **Decision snapshot**: symbol, direction, horizon, lifecycle, research-only status, opportunity score, and confidence.
2. **Why now**: the deterministic `summary`, contribution count, domain diversity, and last evaluated date.
3. **Contributions**: from `opportunity_payload.contributions`, ordered by trigger score, showing hypothesis key/version, domain, and trigger/confidence/quality scores. Per-contribution `reason_codes` are merged in from the linked hypothesis-evaluations supporting read (one bounded, scoped request when the section is open).
4. **Conflicts and limits**: opposing evaluation IDs, conflict score, invalidating evidence, and suppressed overlap IDs (from `opportunity_payload.overlap_suppressed_evaluation_ids`). There is no separate data-quality-warnings field.
5. **Evidence**: supporting evidence IDs with lazy detail expansion and source lineage actions.
6. **Audit details**: opportunity version, build run ID, deterministic key, timestamps, and raw payload in a collapsed JSON viewer.

Do not lead with raw JSON or IDs. IDs remain copyable in the audit section and linked-record rows.

## Empty Queue UX

The first live G139 run is expected to contain zero opportunities because all current AAPL evaluations are rejected. A blank generic empty state is insufficient.

When the opportunity list is empty:

1. Run one bounded `GET /v1/marketops/hypothesis-evaluations` request using the same symbol/date scope and `limit=200`.
2. Show `No eligible opportunities in this scope`.
3. Summarize evaluated count, eligible count, triggered count, and the five most frequent rejection reason codes.
4. Translate known reason tokens into concise operator language while preserving the raw reason token in a tooltip or secondary text.
5. Distinguish no source evaluations from evaluations blocked by quality/coverage.
6. Provide a direct command to clear filters; do not offer a build or provider-ingestion action that G139 does not support.

This diagnostic is the primary UX for sparse live data and must receive the same test coverage as populated results.

## Error And Loading States

- Use skeleton rows with fixed dimensions for first load.
- Preserve the current queue and selection during background refresh.
- A list failure shows a compact retry state in the queue pane.
- A detail failure does not destroy the queue or selection.
- A supporting evidence/hypothesis failure is isolated to that section.
- Never expose bearer tokens, raw auth headers, environment values, stack traces, or database/provider internals.

## Suggested File Scope

Follow existing project conventions. Expected areas include:

- `web/src/types.ts`
- `web/src/api/client.ts`
- `web/src/api/queries.ts`
- focused API/query tests
- `web/src/apps/appRouting.ts` and routing tests
- `web/src/router.tsx`
- a new `MarketOpsOpportunitiesRoute.tsx`
- a small pure helper module for score formatting, reason aggregation, and contribution parsing with unit tests

Do not refactor DSM, Assets, Back-Tests, Algorithms, alerts, insights, auth, or generic layout code beyond the minimum route/navigation integration.

## Acceptance Tests

- API client encodes every supported filter and attaches existing bearer auth.
- Query keys are stable and include the complete filter object.
- MarketOps navigation exposes Opportunities; Console does not.
- Populated queue sorts and renders score, confidence, lifecycle, direction, horizon, research state, contributions, and conflicts.
- URL selection persists and detail fetch is lazy.
- Detail sections render defensively with missing optional fields.
- Overlap-suppressed and conflicting evaluations are visually distinct from supporting contributions.
- Empty queue diagnostics aggregate rejection reasons from one bounded supporting request.
- Loading, list error, detail error, and supporting-read error states are covered.
- Desktop and mobile screenshots confirm no overlap, clipped controls, nested cards, or unreadable tables.
- `npm test` and `npm run build` pass.

## Explicitly Out Of Scope

- Opportunity review or lifecycle mutation.
- Trade submission or portfolio actions.
- Signal/proposal/materialization controls.
- Syncratic Ask or generated narrative for opportunities.
- Triggering the opportunity builder from the browser.
- Top 50 expansion, provider acquisition, scheduling, or policy promotion.
- Redesigning existing MarketOps pages.
