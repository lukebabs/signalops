# MarketOps Intelligence Readiness Frontend-Agent Specification

Status: implemented and locally validated

Date: 2026-07-20

## Objective

Add a read-only Intelligence readiness view to the existing MarketOps Assets experience using the validated G148-C aggregate endpoint. The view explains research rollout coverage per symbol; it must never imply production readiness or expose execution controls.

Implementation commit: `6619269` (`Implement MarketOps intelligence readiness UI`). Focused API/helper tests and the TypeScript/Vite production build pass.

## Backend contract

Use one request:

`GET /v1/marketops/intelligence/readiness?tenant_id={tenant}&universe_group={group}&symbols={csv}&latest_session_date={YYYY-MM-DD}&rollout_status={status}&limit={1..50}`

The authenticated API client supplies the bearer token and tenant. Do not issue per-symbol state, evaluation, opportunity, calibration, outcome, or proposal requests.

The response is:

- `readiness.aggregate.symbol_count`
- `readiness.aggregate.latest_session_date`, when present
- `readiness.aggregate.dimension_counts` for coverage, evaluation, governance, calibration, outcome, and rollout states
- `readiness.aggregate.production_ready_supported=false`
- `readiness.symbols[]`, each containing identity, latest state fields, coverage ratios, stage status/errors, evaluation and outcome counts, proposal status counts, calibration state, five independent readiness dimensions, rollout status, and explicit reasons

Allowed rollout statuses are `not_observed`, `inspection_ready`, `research_evaluation_ready`, `review_ready`, and `blocked`. There is no `production_ready`.

Field notes (authoritative): the five readiness dimensions are the flat per-symbol string fields `coverage_state`, `evaluation_state`, `governance_state`, `calibration_state`, and `outcome_state` (not nested objects). `aggregate.dimension_counts` is grouped by those same dimension keys plus `rollout_status`, each a `string→int` count map. `latest_session_date` filters the persisted `latest_state_date`; the gateway clamps `limit` to a 200 maximum (the UI bounds its selector to 50). A symbol with an empty `latest_market_state_id` is unobserved — render its state columns as “Not observed,” never as zero coverage.

## UI placement

Add an `Intelligence readiness` tab or section under MarketOps Assets.

The top region must render aggregate-first:

- observed symbol count and latest session date;
- compact counts for each rollout status;
- a persistent “Research readiness only — production readiness is unsupported” notice;
- filters for universe group, explicit symbols, latest session, and rollout status.

Render one bounded table or card list from `readiness.symbols` with:

- symbol and universe membership;
- latest state date, schema, quality, completeness, required-feature coverage, and seven-cell surface coverage;
- evaluation, governance, calibration, and outcome badges as separate dimensions;
- rollout status;
- last cohort run ID and per-stage status;
- explicit readiness reasons;
- a link to Market State only when `latest_market_state_id` is non-empty.

Missing state must render as “Not observed,” never zero coverage. Missing/blocked evidence must not render as neutral. Calibration below minimum must retain a visible warning.

## Prohibited behavior

Do not add controls for provider acquisition, cohort execution, Ask execution, graph decisions, proposal review, hypothesis promotion, signal materialization, trading, orders, or portfolio actions. Do not call the Syncratic facade, run automatic Ask, or combine generated prose with deterministic readiness fields.

Do not derive a production-looking boolean, hide readiness reasons, or perform Top 50 N+1 queries.

## States and accessibility

Provide distinct loading, empty, unauthorized, error, and stale-data states. An empty response means no durable cohort readiness exists; it does not mean all assets are ready.

Badges need text labels in addition to color. The table must support keyboard navigation, narrow-screen card layout, and readable reason expansion without horizontal-only interaction.

## Tests

Add tests proving:

- one aggregate request serves the view and no per-symbol queries occur;
- `production_ready_supported=false` is visible;
- AAPL `blocked` and MSFT `not_observed` fixtures render differently;
- missing state is not displayed as zero/neutral;
- calibration warnings and stage errors remain visible;
- state links appear only with a persisted state ID;
- filters are bounded and encoded correctly;
- no mutation, provider, Ask, graph-decision, or proposal-review calls exist;
- desktop and mobile layouts remain usable.

## Live fixture

The validated backend dry-run used AAPL and MSFT over 2026-07-09 through 2026-07-18:

- AAPL: latest state 2026-07-18, schema `marketops.market_state.v1`, quality `partial`, completeness `0.20`, evaluation state `evaluated_no_trigger`, governance `proposal_pending`, calibration unavailable, rollout `blocked`.
- MSFT: no persisted market state, coverage `unavailable`, evaluation `not_run`, governance `research_only`, calibration/outcome unavailable, rollout `not_observed`.

These are truthful sparse-data fixtures, not evidence of empirical market effectiveness.
