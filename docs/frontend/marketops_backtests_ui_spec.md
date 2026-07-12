# MarketOps Back-Tests UI Specification

Status: ready for frontend-agent implementation  
Gate: G081 frontend follow-up  
Author: Codex  
Date: 2026-07-12  
Backend baseline: G081 back-test substrate through `a601e05 Document authenticated G081 API smoke`

## Purpose

Add a bounded MarketOps back-test workspace so operators can create and inspect isolated historical DSM runs from the browser.

The UI should make the G081 substrate usable without leaving the MarketOps shell: start a small historical run, see whether it succeeded, inspect aggregate metrics, review generated signals, and inspect generated graph proposal policy recommendations. Back-test outputs are experimental and isolated. They must not be presented as production signals, production graph proposals, replay jobs, or graph database state.

## Scope

In scope:

- Add a MarketOps-only route for back-tests.
- List recent isolated MarketOps back-test runs.
- Create a bounded synchronous run through `POST /v1/marketops/backtests`.
- Show run detail, aggregate metrics, generated back-test signals, and generated back-test graph proposals.
- Show policy recommendation counts and filter graph proposals by recommendation.
- Preserve existing auth and same-origin API behavior.
- Add API client, query hooks, types, unit tests, and route tests following existing frontend patterns.

Out of scope:

- Async job orchestration, cancellation, queue progress, or worker-heartbeat UI.
- Replay controls or reuse of replay job lifecycle semantics.
- Production graph proposal decision controls: accept, reject, supersede, restore.
- Graph visualization, graph editing, or graph database writes.
- ML training, PnL simulation, model comparison, or calibration dashboards beyond simple aggregate counts.
- Changing existing `/marketops/dsm`, `/marketops/signals`, replay, alert, or insight workflows.

## Route And Navigation

Add a route:

```text
/marketops/backtests
```

Add one MarketOps nav item near DSM or Replay:

```ts
{ module: 'backtests', to: '/marketops/backtests', label: 'Back-Tests' }
```

Use a lucide icon already available in the project, such as `FlaskConical`, `History`, or `RefreshCcwDot`. Do not add the route to Console nav.

Keep `/marketops` redirect behavior unchanged.

## Backend Contract

Use authenticated same-origin `/v1/*` APIs. Auth is required when enabled.

Create run:

```http
POST /v1/marketops/backtests
Authorization: Bearer <access_token>
Content-Type: application/json
```

Request body:

```json
{
  "run_id": "bt-ui-optional-id",
  "tenant_id": "tenant-local",
  "source_id": "src-massive",
  "dataset": "equity_eod_prices",
  "detector_id": "marketops.dsm.taxonomy_v1",
  "detector_version": "v1",
  "symbols": ["SPY"],
  "window_start": "2026-07-09T00:00:00Z",
  "window_end": "2026-07-10T00:00:00Z",
  "max_records": 5,
  "batch_size": 5,
  "auto_accept_confidence": 0.75
}
```

`run_id` is optional from the client: when omitted, the gateway generates one via `newID("bt_marketops")` (`internal/api/router.go`). `source_adapter` and `requested_by` are also accepted by the gateway but need not be sent — the backend defaults `source_adapter` and derives `requested_by` from the bearer token when auth is enabled. `window_start`/`window_end` must be RFC3339 (e.g. `...Z`); `window_end` must be after `window_start`; `max_records` must be 1–1000.

Response status: `201 Created` with:

```json
{
  "backtest_run": {},
  "metrics": {}
}
```

Response envelopes (verified against `a601e05`):

- create / detail: `{ "backtest_run": <run>, "metrics"?: <metrics> }` (create returns both keys; detail returns `backtest_run` only)
- list: `{ "backtest_runs": [<run>, ...] }`
- signals: `{ "backtest_signals": [{ "run_id": "...", "signal": <SignalRecord> }, ...] }`
- graph-proposals: `{ "backtest_graph_proposals": [{ "run_id": "...", "graph_proposal": <MarketOpsDSMGraphProposal> }, ...], "policy_results": [<policy_result>, ...] }`

Each back-test signal wraps a production-shaped `SignalRecord` under `signal`; each back-test graph proposal wraps a `MarketOpsDSMGraphProposal` under `graph_proposal`. Recommendation and reason are NOT fields on the graph proposal — they live on the paired `policy_results` entry (joined by `proposal_id`). See §5.

List runs:

```http
GET /v1/marketops/backtests?tenant_id=tenant-local&detector_id=marketops.dsm.taxonomy_v1&status=succeeded&limit=50
```

Run detail:

```http
GET /v1/marketops/backtests/{run_id}?tenant_id=tenant-local
```

Note: `tenant_id` on the detail path is accepted by the gateway but currently ignored (lookup is by `run_id` only). Send it anyway for contract consistency.

Generated back-test signals:

```http
GET /v1/marketops/backtests/{run_id}/signals?tenant_id=tenant-local&signal_type=marketops.dsm.pinning_risk&limit=50
```

Generated back-test graph proposals and policy results:

```http
GET /v1/marketops/backtests/{run_id}/graph-proposals?tenant_id=tenant-local&recommendation=manual_review_required&limit=50
```

Filter semantics on the graph-proposals endpoint (verified against `internal/storage/postgres/marketops_backtests.go`):

- `tenant_id`, `signal_type`, `subject_symbol`, `candidate_type`, `limit` narrow the `backtest_graph_proposals` list.
- `tenant_id`, `subject_symbol`, `candidate_type`, `recommendation`, `limit` narrow the `policy_results` list.
- `recommendation` does **not** narrow `backtest_graph_proposals` — only `policy_results`. To keep the displayed proposal table consistent when a recommendation filter is active, the UI joins `policy_results` → `graph_proposal` by `proposal_id` and hides proposals whose joined recommendation does not match the filter (client-side).

Recommendation values:

- `auto_accept_candidate`
- `auto_reject_candidate`
- `manual_review_required`
- `supersede_candidate`

## Data Semantics

Back-test rows are isolated experiment records:

- Runs come from `marketops_backtest_runs`.
- Signals come from `marketops_backtest_signals`, not production `signal_ledger`.
- Graph proposals come from `marketops_backtest_graph_proposals`, not production `marketops_dsm_graph_proposals`.
- Policy results come from `marketops_backtest_policy_results`.
- The UI must label the workspace as back-test or experimental wherever ambiguity is likely.

Back-test run metrics currently include:

- `scanned`
- `signals`
- `artifacts`
- `graph_proposals`
- `policy_results`
- `recommendation_counts`
- `batches`
- `max_records`
- `batch_size`
- `started_at`
- `completed_at`

JSON fields are already parsed by the gateway. Use type guards over `unknown`; do not `JSON.parse` nested values unless a field is actually a string.

## Required UI

### 1. Back-Test Run List

Show a compact run table or dense list with:

- run id
- status
- dataset
- detector id/version
- requested by
- window start/end
- scanned count
- signal count
- graph proposal count
- dominant recommendation count summary
- created/updated time

Filters:

- status
- detector id
- limit

Defaults:

- `tenant_id=tenant-local`
- `detector_id=marketops.dsm.taxonomy_v1`
- `limit=50`

Selecting a run should open a detail panel in the same route.

### 2. Create Run Panel

Provide a compact form for bounded synchronous execution:

- run id: optional text; when blank the backend generates one (`bt_marketops…`). Exposing it lets an operator reproduce a named run such as `bt-g081-auth-api-smoke-20260712`.
- source id: default `src-massive`
- dataset: default `equity_eod_prices`; allow `options_contracts_daily` only if the UI already has a safe option source context, otherwise keep it visible but disabled with a short label
- symbols: comma-separated input converted to uppercase array
- window start: RFC3339-compatible datetime input
- window end: RFC3339-compatible datetime input
- max records: numeric input, default `5`, min `1`, max `1000`
- batch size: numeric input, default equal to max records or `50`, max `1000`
- detector id: default `marketops.dsm.taxonomy_v1`
- detector version: optional text, default `v1`
- auto accept confidence: numeric input, default `0.75`, min `0`, max `1` (a value of `0` is normalized to `0.75` by the backend)

Behavior:

- Validate required fields before POST.
- Disable submit while the synchronous request is in flight.
- On success, select the returned run and refresh the run list.
- On failure, show the API error envelope using existing error presentation patterns.
- Do not silently generate broad historical runs. Keep `max_records` bounded and visible.

### 3. Run Detail Panel

Show:

- status and timing
- filters JSON summary
- parameters JSON summary
- metrics cards: scanned, signals, artifacts, graph proposals, policy results
- recommendation count chips
- error message if status is failed

Include raw JSON disclosure sections for metrics, filters, and parameters, using existing JSON viewer behavior.

### 4. Generated Signals Section

For the selected run, fetch generated signals and show:

- signal id
- signal type short label
- severity
- confidence
- subject ticker if available
- event ids
- supporting metrics summary

This section must be labeled `Back-Test Signals` or equivalent. Do not link these records as if they were production signal ledger rows unless the backend exposes a dedicated detail route later.

### 5. Generated Graph Proposals Section

For the selected run, fetch graph proposals and policy results (single endpoint returns both arrays) and join `policy_results` → `graph_proposal` by `proposal_id`. For each proposal show:

- proposal id
- recommendation (from the joined policy result; `—` when no policy result matched)
- status if present (always `proposed` for back-test rows — there is no back-test decision endpoint)
- candidate type
- subject symbol
- node id for node candidates
- from / relationship / to for relationship candidates
- confidence
- policy reason or note if present (from the joined policy result)

Filters (sent to the backend on the graph-proposals endpoint):

- recommendation
- candidate type
- subject symbol
- limit

Because `recommendation` only narrows `policy_results` server-side (not `graph_proposals`), the UI must additionally hide proposals whose joined recommendation does not match the selected `recommendation` filter, so the table stays consistent. `candidate_type` and `subject_symbol` narrow both lists server-side and need no client-side filtering.

Use recommendation colors consistently but keep the UI restrained:

- `auto_accept_candidate`: positive/green
- `auto_reject_candidate`: neutral or red depending existing design convention
- `manual_review_required`: warning/amber
- `supersede_candidate`: purple or gray, whichever better matches existing token colors

Do not add decision buttons. These are back-test policy outputs, not production operator decisions.

## Required Types And API Client Work

Add or extend types in `web/src/types.ts`:

- `MarketOpsBacktestRun`
- `MarketOpsBacktestRunsResponse`
- `MarketOpsBacktestRunResponse`
- `MarketOpsBacktestCreateRequest`
- `MarketOpsBacktestCreateResponse`
- `MarketOpsBacktestSignal` (wraps a production-shaped `SignalRecord` under `signal`)
- `MarketOpsBacktestSignalsResponse`
- `MarketOpsBacktestGraphProposal` (wraps a `MarketOpsDSMGraphProposal` under `graph_proposal`)
- `MarketOpsBacktestGraphProposalsResponse` (carries both `backtest_graph_proposals` and `policy_results`)
- `MarketOpsBacktestPolicyResult` (carries `recommendation`, `reason`, `proposal_id` for the §5 join)
- `MarketOpsBacktestMetrics` (permissive; `[key: string]: unknown` for forward compatibility)
- filter types for list/signals/graph-proposals queries

Add client methods in `web/src/api/client.ts`:

```ts
listMarketOpsBacktests(filter): Promise<MarketOpsBacktestRunsResponse>
getMarketOpsBacktest(runId, tenantId): Promise<MarketOpsBacktestRunResponse>
createMarketOpsBacktest(request): Promise<MarketOpsBacktestCreateResponse>
listMarketOpsBacktestSignals(runId, filter): Promise<MarketOpsBacktestSignalsResponse>
listMarketOpsBacktestGraphProposals(runId, filter): Promise<MarketOpsBacktestGraphProposalsResponse>
```

Add query keys/hooks in `web/src/api/queries.ts`:

- `marketOpsBacktests(filter)`
- `marketOpsBacktest(runId, tenantId)`
- `marketOpsBacktestSignals(runId, filter)`
- `marketOpsBacktestGraphProposals(runId, filter)`
- `useCreateMarketOpsBacktest()` mutation invalidating run list/detail keys on success

## Testing Requirements

Add focused frontend tests for:

- API client builds the correct `/v1/marketops/backtests` URLs and POST body.
- Create mutation invalidates relevant back-test queries.
- Route/nav exposes `Back-Tests` only in MarketOps nav.
- Symbol parsing uppercases and trims values.
- Metrics/recommendation helpers tolerate missing or malformed JSON fields.
- Error envelope from create run is displayed.

Run before handoff:

```bash
cd web && npm test
cd web && npm run build
```

If the frontend-agent changes routed UI, also rebuild the container for local verification:

```bash
docker compose up -d --build web
```

## Acceptance Criteria

The frontend task is complete when:

- `/marketops/backtests` is reachable from the MarketOps nav and never appears in Console nav.
- An authenticated operator can create a bounded run equivalent to `bt-g081-auth-api-smoke-20260712` from the browser.
- The new run appears in the run list without a manual page reload.
- Selecting the run shows metrics, generated back-test signals, generated graph proposals, and policy recommendation counts.
- The UI clearly distinguishes back-test outputs from production ledgers.
- Existing `/marketops/dsm`, `/marketops/assets`, replay, alerts, and insights views still build and test without regressions.

## Non-Goals For This Frontend Gate

Do not implement:

- policy promotion
- graph writeback
- proposal accept/reject/supersede/restore controls
- back-test cancellation
- async job queue dashboards
- model training surfaces
- provider data pull controls
- production replay controls

Those require separate backend gates or explicit product decisions.
