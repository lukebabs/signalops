# G147 Frontend-Agent Specification: Market State Analyst Experience

Status: proposed - frontend-agent specification ready

Date: 2026-07-20

## Objective

Extend the existing MarketOps application with an analyst experience that makes the G137-G146 market-state ledgers usable without weakening their research and governance boundaries. The implementation must answer, for a selected asset and session:

1. What is the current underlying, volatility, positioning, premium, liquidity, quality, and event state?
2. What changed from the prior eligible session?
3. Which DTE/delta surface cells changed, accelerated, accumulated positioning, or became unusable?
4. Which hypothesis versions were evaluated, why were they eligible or blocked, and what evidence contributed?
5. What historical calibration is actually available, with its sample warnings intact?
6. Which opportunities resulted, what supports or conflicts with them, and what analyst dispositions have been appended?

This is an integrated inspection and triage workflow. It is not a provider console, state builder, hypothesis evaluator, lifecycle promotion surface, proposal reviewer, signal materializer, graph reviewer, or Syncratic Ask experience.

## Product Placement

Add one MarketOps-only route:

```text
/marketops/state
```

Add `Market State` to the MarketOps navigation immediately after `Assets`. Do not expose it in Console navigation. Add an `Open Market State` link from an asset row when a symbol is known; the link changes navigation only and must not start provider acquisition or state construction.

Use one workbench route rather than separate routes for state, surface, transitions, and hypotheses. Extend the existing `/marketops/opportunities` detail rather than creating a second opportunity queue.

Persist analyst context in URL search parameters:

- `symbol`
- `session_date=YYYY-MM-DD`
- `tab=overview|surface|transitions|hypotheses`
- `hypothesis_key` and `hypothesis_version` when a hypothesis row is selected
- preserve the existing `opportunity_id` behavior on `/marketops/opportunities`

Unknown or invalid tab values fall back to `overview`. A valid URL must survive refresh and browser back/forward navigation. Changing symbol or session clears an incompatible hypothesis selection.

## Existing Backend Contract

Use the existing bearer-authenticated API client. G147 requires frontend composition over current endpoints; it does not require or authorize a new aggregate endpoint.

### State And Feature Reads

```text
GET /v1/marketops/states
GET /v1/marketops/states/{market_state_id}
GET /v1/marketops/states/{market_state_id}/lineage
GET /v1/marketops/features/definitions
GET /v1/marketops/features/observations
GET /v1/marketops/transitions
GET /v1/marketops/evidence
GET /v1/marketops/evidence/{evidence_id}
```

Required list filters and envelopes:

| Read | Relevant filters | Envelope |
| --- | --- | --- |
| states | `tenant_id`, `app_id`, `asset_id`, `symbol`, `state_schema_version`, `quality_state`, `session_start`, `session_end`, `limit` | `market_states` |
| definitions | `tenant_id`, `feature_key`, `feature_version`, `domain`, `status`, `limit` | `feature_definitions` |
| observations | `tenant_id`, `app_id`, `asset_id`, `symbol`, `feature_key`, `feature_version`, `domain`, `quality_state`, JSON `dimensions`, session range, `limit` | `feature_observations` |
| transitions | state filters plus `current_state_id`, `feature_key`, `feature_version`, `transition_type`, session range, `limit` | `transitions` |
| evidence | state filters plus `evidence_type`, `evidence_version`, `domain`, `direction`, session range, `limit` | `evidence` |

The selected-state lineage response is authoritative for the feature observations belonging to that state:

```json
{
  "lineage": {
    "market_state": {},
    "feature_observations": [],
    "source_event_ids": [],
    "source_artifact_ids": [],
    "missing_feature_observation_ids": []
  }
}
```

Do not rebuild state membership by issuing a broad observation query. Use the lineage response for the overview and selected-session surface. Use observation list reads only for a bounded, explicit analytical need not already satisfied by the lineage response.

### Hypothesis And Outcome Reads

```text
GET /v1/marketops/hypotheses
GET /v1/marketops/hypotheses/{hypothesis_key}/{hypothesis_version}?tenant_id=...
GET /v1/marketops/hypothesis-evaluations
GET /v1/marketops/outcomes
GET /v1/marketops/outcomes/{outcome_id}?tenant_id=...
```

The hypothesis list supports `tenant_id`, exact key/version, domain, lifecycle status, and `limit`. Evaluation filters include exact key/version, `market_state_id`, asset/symbol, `eligible`, `triggered`, `invalidated`, session range, and `limit`. Outcome filters include exact source type/source ID, hypothesis key/version, symbol, direction, status, one of the supported 1/5/10/20-session horizons, origin-session range, and `limit`.

All nullable score and outcome fields remain unavailable when absent. Pending or missing outcomes are not zero returns, misses, or negative evidence.

### Calibration Reads

Reuse:

```text
GET /v1/marketops/backtest-calibration-summaries
GET /v1/marketops/backtest-calibration-summaries/{summary_id}
```

For a selected hypothesis key, request summaries with:

- `tenant_id={tenant}`
- `app_id=marketops`
- `domain=market_data`
- `use_case=daily_market_surveillance`
- `source_id=marketops.research_ledgers`
- `dataset=hypothesis_evaluations`
- `detector_id=marketops.hypothesis.{lowercase hypothesis key}`
- a bounded `limit`

The existing client currently defaults an omitted detector ID to the DSM taxonomy detector. G147 callers must pass the hypothesis detector explicitly.

The rich G145 report is the summary `parameters` object only when all of these checks pass:

- `summary_version === "marketops.hypothesis_calibration.v1"`;
- `hypothesis_key` exactly matches the selected definition;
- `hypothesis_versions` contains the selected exact version;
- `versions[selectedVersion]` exists.

Parse this payload with a runtime type guard. Never reinterpret generic summary counters as hypothesis performance when the versioned payload is missing or invalid. Show malformed or mismatched reports as unavailable with a compact diagnostic.

Render `mode`, window/as-of, symbols, minimum sample size, exact version metrics, horizons and segments, comparison or walk-forward data when present, and all warnings. `promotion_allowed=false` is authoritative and must remain visible. A favorable report never enables promotion, proposal review, or materialization.

### Opportunity And Disposition Reads/Writes

Retain the existing G139 opportunity APIs and add the G146 disposition contract:

```text
GET  /v1/marketops/opportunities/{opportunity_id}/dispositions?tenant_id=...&limit=50
POST /v1/marketops/opportunities/{opportunity_id}/dispositions
```

POST body:

```json
{
  "tenant_id": "tenant-local",
  "disposition": "needs_more_evidence",
  "note": "Await another session.",
  "metadata": {}
}
```

Current dispositions are `watch`, `advance`, `needs_more_evidence`, `dismiss`, and `resolved`; preserve unknown future strings when rendering existing rows. The response envelope is `opportunity_disposition`; the list envelope is `opportunity_dispositions`.

The frontend must not send an actor merely to impersonate one. Follow the repository's authenticated actor convention and let the gateway derive actor identity. Metadata must be an object and should default to `{}`.

Submitting a disposition appends an audit row. It does not change `lifecycle_status`, recompute the opportunity, approve a hypothesis, review a proposal, create a signal, or resolve an alert. The UI must never imply those effects.

## Required Frontend Contracts

Add or complete types for:

- feature definition and list response/filter;
- feature observation and list response/filter;
- market state and state list/detail/filter;
- state transition and response/filter;
- hypothesis list response/filter (detail types already exist);
- signal outcome list/detail/filter;
- G145 hypothesis calibration report, version metrics, segments, comparisons, folds, and runtime parse result;
- opportunity disposition, list/detail envelopes, create request, and filter.

JSON objects such as `state_payload`, `dimensions`, `quality_details`, `quality_summary`, definition rules, transition/evaluation payloads, and calibration `parameters` arrive parsed and must remain `unknown` until narrowed. Do not call `JSON.parse` on them.

Preserve unknown backend enum strings for forward compatibility. Treat all pointer/optional numeric and boolean fields as genuinely nullable. The distinction between absent, zero, and false is material.

Add API client methods and TanStack Query hooks with complete filter-bearing keys. Detail and lineage hooks must use `enabled` gating. Add a disposition mutation hook that updates the returned detail row and invalidates only disposition queries for the selected opportunity; it must not optimistically mutate computed opportunity data.

## Workbench Layout

### Scope Bar

Use a compact page header titled `Market State` followed by a sticky, restrained scope bar:

- symbol selector/input;
- session-date selector populated from available states;
- state schema and quality badges;
- completeness as `required present / required expected` plus percentage;
- as-of time;
- refresh icon button with tooltip.

Do not add a provider refresh, state build, evaluation run, or materialize button. If no symbol is supplied, offer an asset-selection link. If a symbol has no persisted states, say so directly and link back to Assets; do not imply that the browser can create coverage.

For a selected symbol, fetch a bounded state window and sort session dates explicitly. When multiple revisions exist for one session/schema, choose the newest `as_of_time`, then a deterministic ID tie-break, and expose the selected revision in audit detail. The prior comparison uses the nearest earlier persisted session, not calendar-day arithmetic. Never compare a feature across different feature keys, versions, or canonical dimensions.

### Tabs

Place four tabs below the scope bar:

1. Overview
2. Surface
3. Transitions
4. Hypotheses

Desktop favors dense tables and aligned small multiples. Narrow screens stack sections and allow horizontal scrolling only inside the surface grid or data tables. Do not fill the page with independently floating dashboard cards.

## Overview Tab

Group exact selected-state observations into six ordered domains:

1. underlying;
2. volatility and volatility surface;
3. option positioning;
4. premium;
5. liquidity and quality;
6. event context.

Use feature definition titles/units when available, with feature key/version as secondary text. Each row shows current typed value, prior exact-match value, absolute change when both values are usable, quality state/score, and a lineage affordance. Text and boolean features remain text/boolean; do not coerce them into numeric deltas.

The summary band shows state quality, completeness, eligible hypothesis keys, required-versus-total feature counts, missing feature count, and prior-session identity. `quality_summary` may be shown through a narrowed summary plus collapsed JSON, but it must not replace observation-level truth.

Missing and unusable observations remain visible in their domain with the backend quality state and reason from `quality_details` when available. Render an em dash and reason, never zero. IDs, build run, deterministic key, payload, timestamps, source event IDs, and artifact IDs belong in a collapsed Audit and lineage section rather than the initial reading order.

## Surface Tab

Render the canonical persisted seven-cell surface:

- 30-, 60-, and 90-DTE ATM cells;
- 30- and 60-DTE 25-delta put cells;
- 30- and 60-DTE 25-delta call cells.

Derive cell identity from exact feature key plus canonical `dimensions` (`option_type`, `target_dte`, `target_delta`, or the explicit ATM feature keys). Do not infer absent cells from array order.

For each cell show, where persisted:

- IV;
- 1-session and 5-session IV change;
- applicable transition z-score or percentile;
- midpoint/extrinsic premium and 1-/5-session premium change;
- 1-/5-session OI change;
- spread, quote/selection quality, or explicit unavailable reason.

Use a metric selector or aligned sub-rows so the grid remains readable. Heatmap color may supplement, but never replace, a formatted value and quality marker. Use a neutral unavailable pattern for unusable cells.

Above the grid, derive concise answers only from returned observations/transitions:

- cells with the largest absolute IV move;
- cells with acceleration transitions;
- maturity with the broadest expansion/contraction;
- largest unusual OI change;
- unusable cells and backend reasons.

If the needed observation or transition is absent, say `Unavailable`; do not calculate a proxy from unrelated fields. Show the canonical selected surface even when every cell is unavailable because that missingness is operationally meaningful.

## Transitions Tab

Request a bounded transition window for the selected symbol and show material transitions grouped by session and domain. Provide filters for domain, transition type, lookback, direction, and quality. Default to a material view that prioritizes:

- acceleration and regime transitions;
- non-zero multi-session changes;
- transitions with z-score/percentile rarity;
- quality degradation or recovery.

Do not render every feature revision as an alert. The material filter is a presentation filter over persisted rows, not a mutation or a claim that hidden rows do not exist. Show `displayed / returned` counts and provide `Show all returned transitions`.

Each row includes feature/dimensions, current and baseline session/state, transition type/lookback, current/baseline/transition values, z-score or percentile, persistence, direction, and quality. Nullable values remain unavailable. A detail expansion shows transition payload, lineage IDs, calculation run, deterministic key, and timestamps.

## Hypotheses Tab

Use a master/detail layout over exact definition versions and the selected state's evaluations. Fetch definitions once with a bounded list request and evaluations once using `market_state_id`; do not issue definition or evidence requests per row during initial render.

Each hypothesis row shows:

- title, key, and exact version;
- domain and direction;
- lifecycle status;
- eligible, triggered, and invalidated states;
- trigger, confidence, quality, rarity, persistence, magnitude, and corroboration scores when present;
- compact reason-code summary.

The selected detail is ordered as follows:

1. **Definition and status**: title, description, rationale, owner, lifecycle, approval audit if present.
2. **Required evidence**: required features/transitions and quality policy, with present/missing status derived from exact selected-state observations/transitions.
3. **Current evaluation**: eligibility, trigger, invalidation, nullable scores, reason codes, and narrowed contribution/check data from `evaluation_payload`.
4. **Evidence**: linked evidence IDs with lazy detail expansion; one request per analyst-opened evidence row is acceptable and cached.
5. **Historical calibration**: latest exact-version G145 report plus optional older reports, sample counts, horizons, segments, warnings, and permanent advisory status.
6. **Governance status**: lifecycle/approval facts and source-aware proposal status when already available through the existing proposal read contract.
7. **Audit**: exact IDs, run IDs, deterministic keys, raw versioned payloads, and timestamps.

Definitions with no evaluation for the selected state remain visible as `Not evaluated for this state`. An ineligible evaluation is not a failed hypothesis, and a non-trigger is not a negative realized outcome.

The route is read-only for definition lifecycle and proposal state. It may link to the existing Algorithms proposal-review surface when a source-aware proposal exists, but it must not embed proposal accept/reject controls. There is no current hypothesis promotion mutation endpoint; do not invent one. G146 hypothesis materialization is deliberately unsupported and returns fail-closed, so do not offer a hypothesis materialize action.

## Opportunity Detail Extension

Keep the existing G139 queue, URL selection, contribution ordering, conflicts, evidence, empty diagnostics, and lazy supporting reads. Extend only the selected detail with these sections:

### Historical Calibration

For each exact hypothesis contribution, show whether a valid G145 report exists. Aggregate no cross-version numbers. Prefer a compact comparison of independent samples, matured outcomes, directional hit rate, mean/median forward return, favorable/adverse excursion, drawdown incidence, realized-volatility change, and calibration error. Warnings and insufficient-sample state must be more prominent than favorable point estimates.

Do not label an opportunity `calibrated` merely because a summary exists. Use `Calibration available`, `Below minimum sample`, or `Unavailable` based on the parsed report.

### Data Quality Warnings

There is no dedicated `data_quality_warnings` DTO field. Derive warnings only from persisted opportunity/evaluation evidence:

- state/observation `quality_state` and observation `quality_details`;
- evaluation/evidence quality scores and evaluation reason codes;
- evaluation invalidation status;
- missing linked evidence;
- calibration warnings and sample threshold flags.

Label the section `Quality and evidence limits`; do not manufacture warning strings or treat absent scores as warnings without context.

### Analyst Disposition

Show an append-only disposition history newest first. Provide a compact form with one allowed disposition, an optional note, and explicit confirmation for `dismiss` or `resolved`. On success, append/refetch the ledger and show the actor/time returned by the backend.

Copy must state that the action records analyst judgment and does not alter computed lifecycle. Do not replace prior rows, present the newest disposition as authoritative computed state, or add a generic `Resolve opportunity` button.

## Control Separation

The following controls must stay visibly distinct:

| Control | G147 behavior |
| --- | --- |
| provider acquisition | existing Providers/Assets workflows only; no action in Market State |
| feature materialization/state construction | no browser mutation; inspect persisted state only |
| research evaluation | no browser mutation; inspect persisted evaluation only |
| lifecycle promotion | unavailable; no backend mutation exists |
| proposal review | existing Algorithms surface; optional navigation link only |
| signal materialization | existing governed proposal flow only; no state/opportunity action |
| graph proposal review | existing DSM/graph review surface; outside G147 detail |
| opportunity disposition | append-only action in selected opportunity detail |

The UI must not visually collapse research lifecycle, evaluation result, proposal review status, signal materialization, and analyst disposition into a single status or action rail.

## Query Composition And Performance

The initial selected-state load should normally consist of:

1. one bounded state-list request for the symbol/session window;
2. one lineage request for the selected state;
3. one definitions request, shared and cached.

Tab-specific reads are enabled only when that tab opens:

- Surface: selected lineage plus bounded current-state/session transitions; no cell fan-out.
- Transitions: one bounded symbol/window request, maximum 200.
- Hypotheses: one evaluation request for selected `market_state_id`, maximum 200; calibration only for the opened hypothesis; evidence detail only when opened.
- Prior comparison: one lineage read for the chosen prior state, cached.
- Opportunity calibration/dispositions: only after an opportunity is selected and the section is opened.

Never issue one state, transition, outcome, definition, or calibration request per table row or surface cell. Cache exact detail reads with stable IDs and include all filters in query keys. Preserve visible data during background refresh.

If a list reaches its requested maximum, show that the view may be truncated. Do not silently claim completeness.

## Empty, Loading, And Error States

- No symbol: show `Choose an asset to inspect market state` and an Assets link.
- No state: show `No persisted market state for this scope`; do not offer a fake build action.
- State exists but lineage is empty/incomplete: render state quality and explicit missingness.
- No transitions: distinguish no returned rows from rows hidden by the material filter.
- No evaluations: keep definitions visible and show `Not evaluated for this state`.
- No calibration: show `No exact-version calibration report`.
- Malformed calibration payload: show `Calibration report is not compatible with marketops.hypothesis_calibration.v1`.
- No opportunities: preserve the G139 bounded rejection diagnostic.
- No dispositions: show `No analyst dispositions recorded.`

Use fixed-dimension skeletons for first load. Isolate errors by section so a calibration or evidence failure does not erase the state or queue. Retry actions must repeat only the failed query. Sanitize all errors; never expose bearer tokens, headers, secrets, environment values, stack traces, provider payloads, or database internals.

## Accessibility And Responsive Behavior

- Every tab, row, expansion, selector, and action is keyboard reachable.
- State and hypothesis rows support visible focus; Enter selects where appropriate.
- Heatmaps, quality, direction, lifecycle, and disposition never rely on color alone.
- Surface cells have accessible names containing DTE, delta/type, metric, value, and quality.
- Tables retain real headers and semantic relationships; do not implement the surface as unlabeled colored divs.
- Tooltips supplement visible labels and are not the sole source of required information.
- On mobile, tabs remain accessible, overview groups stack, and hypothesis master/detail becomes list-first with a familiar back action.

## Suggested Implementation Scope

Expected frontend touch points:

- `web/src/types.ts`
- `web/src/api/client.ts`
- `web/src/api/queries.ts`
- `web/src/apps/appRouting.ts` and navigation tests
- `web/src/router.tsx`
- new `web/src/routes/MarketOpsStateRoute.tsx`
- focused state/surface/calibration helpers and unit tests
- existing `web/src/routes/MarketOpsAssetsRoute.tsx` for the navigation link
- existing `web/src/routes/MarketOpsOpportunitiesRoute.tsx` for calibration, quality limits, and disposition history/action

Reuse existing shared `StatusBadge`, quality badges, loading/error states, refresh/copy controls, and collapsed JSON viewer. Do not refactor unrelated DSM, Back-Tests, Algorithms, Alerts, Insights, Syncratic, auth, or shell code.

## Acceptance Tests

### Contracts And Queries

- API client encodes every used filter and correct response envelope.
- Query keys include full scope/filter and detail hooks are gated by IDs/open sections.
- JSON objects remain parsed `unknown`; calibration uses runtime validation.
- Hypothesis calibration always supplies the explicit hypothesis detector ID.
- Disposition POST sends tenant, allowed disposition, note, and object metadata without spoofing actor.
- Successful disposition invalidates/refetches disposition rows only, not computed opportunity lifecycle.

### State And Surface

- MarketOps navigation exposes `Market State`; Console does not.
- URL symbol/date/tab/hypothesis selection round-trips through refresh/back/forward.
- Duplicate state revisions select newest as-of deterministically.
- Prior comparison chooses nearest earlier persisted session and exact feature key/version/dimensions.
- Numeric zero, boolean false, and missing values render distinctly.
- Seven canonical cells map by dimensions/key rather than array position.
- Missing/unusable surface cells display quality reasons and never fake zero.
- Material transition filtering reports shown/returned counts and can reveal all returned rows.

### Hypotheses And Calibration

- Definitions remain visible without a current evaluation.
- Eligibility, trigger, invalidation, reason codes, and nullable scores remain distinct.
- Required evidence status uses the exact selected state and feature versions.
- Evidence detail is lazy and cached.
- Calibration rejects wrong schema/key/version payloads.
- Minimum-sample warnings and `promotion_allowed=false` remain prominent.
- No definition promotion, evaluation, proposal review, or materialization mutation appears in the Market State route.

### Opportunities And Governance

- Existing G139 queue/detail and empty diagnostics remain green.
- Calibration is exact per contribution version and never pooled.
- Quality limits contain only persisted/derivable reasons.
- Disposition history is append-only and rendered independently from lifecycle.
- `dismiss` and `resolved` require confirmation.
- No disposition result implies signal, alert, graph, proposal, or lifecycle mutation.

### Quality

- Focused helper/component/API/query/navigation tests pass.
- Existing frontend suite passes.
- `npm run build` passes.
- Desktop and mobile viewport checks show usable matrices/tables, no clipped controls, and no nested-card sprawl.
- Network inspection confirms no N+1 calls per surface cell, definition row, evaluation row, contribution, or transition.

## Explicitly Out Of Scope

- New backend endpoints or composite state API.
- Provider acquisition, scheduling, broad universe fanout, or history synthesis.
- Browser-triggered feature/state/hypothesis/opportunity/outcome construction.
- Hypothesis lifecycle mutation or automatic promotion.
- Proposal accept/reject controls outside the existing Algorithms workflow.
- Hypothesis signal materialization; G146 deliberately fails closed.
- Graph proposal review or graph mutation.
- Syncratic Ask, generated narrative, graph exploration, and cohort rollout; those belong to G148.
- Trade, order, portfolio, or execution controls.
- Full-chain options persistence or visualization beyond the seven selected cells.
- Redesigning the MarketOps navigation shell or unrelated routes.

## Handoff Note

The deployed backend is sufficient for a read-oriented G147 experience plus the bounded append-only disposition action. It is intentionally sparse: current definitions remain research-only, existing evaluations are non-triggered/ineligible, hypothesis proposals may be absent, calibration may have insufficient samples, and the opportunity queue may be empty. The frontend must make those truths legible rather than hiding them or manufacturing readiness.

Known backend absences are product boundaries, not frontend TODOs: there is no hypothesis promotion mutation, no browser state/evaluation builder endpoint, no dedicated hypothesis-calibration endpoint, no data-quality-warnings field, and no supported hypothesis signal-materialization adapter.
