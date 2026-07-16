# Frontend-Agent Spec: G119 Algorithm Signal Materialization Preflight UI

## Goal

Add read-only materialization preflight visibility for algorithm signal proposals inside the existing Algorithms / Signal Proposals UI.

This builds on:

- G113/G114 proposal list/detail/review UI.
- G115/G116 proposal review summary UI.
- G117 materialization design boundary.
- G118 backend materialization preflight API.

The UI must help operators understand whether reviewed algorithm signal proposals are ready for a later materialization gate. It must not materialize production signals or create alerts, insights, graph proposals, policies, deployments, Syncratic work, or new backend behavior.

## Existing Backend API

Use this existing endpoint only:

- `GET /v1/algorithms/signal-proposals/materialization-preflight?tenant_id={tenant_id}&algorithm_id={algorithm_id}&execution_request_id={execution_request_id}&algorithm_result_id={algorithm_result_id}&status={status}&severity={severity}&correlation_id={correlation_id}&limit=200&min_reviewed_ratio=1&policy_version=materialization_preflight.v1`

Response envelope:

```json
{
  "algorithm_signal_materialization_preflight": {
    "tenant_id": "tenant-local",
    "policy_version": "materialization_preflight.v1",
    "total_proposals": 4,
    "eligible_count": 0,
    "duplicate_risk_count": 1,
    "blocked_count": 2,
    "invalid_count": 1,
    "would_write_count": 0,
    "reviewed_ratio": 0.75,
    "min_reviewed_ratio": 1,
    "review_coverage_satisfied": false,
    "high_critical_unreviewed_count": 1,
    "global_blocking_reasons": {
      "review_coverage_below_threshold": 1,
      "high_critical_unreviewed_proposals": 1
    },
    "item_reason_counts": {
      "unreviewed_proposal": 1,
      "duplicate_signal_event_overlap": 1,
      "missing_source_events": 1
    },
    "items": [
      {
        "proposal_id": "algsigprop-reviewed",
        "algorithm_result_id": "algres-reviewed",
        "algorithm_id": "signalops.algorithms.zscore_anomaly_v1",
        "execution_request_id": "algexec-1",
        "proposed_signal_type": "signalops.algorithm.anomaly_candidate",
        "status": "reviewed",
        "severity": "medium",
        "confidence": 0.9,
        "preflight_status": "blocked",
        "reasons": [],
        "duplicate_signal_ids": [],
        "source_event_ids": ["evt-1"],
        "would_write": false,
        "materialization_policy": "materialization_preflight.v1"
      }
    ]
  }
}
```

## UI Placement

Extend the existing `Signal Proposals` area in the Algorithms route.

Recommended placement:

- Keep the existing G116 proposal summary panel at the top.
- Add a compact `Materialization Preflight` panel below the summary and above the proposal list.
- Keep the proposal list and proposal detail/review panel unchanged.

Do not create a new top-level route.

## Filter Coupling

The preflight query must use the same active proposal filters as the proposal list, including `limit`.

Couple these filters:

- tenant id
- algorithm id
- execution request id
- algorithm result id, if present in the UI
- status
- severity
- correlation id
- limit

Additional preflight parameters:

- `min_reviewed_ratio`: default `1`; expose as a compact numeric/select control only if it fits existing UI patterns. Otherwise hard-code the default.
- `policy_version`: default `materialization_preflight.v1`; do not make this prominent unless the current Algorithms UI already exposes policy/config metadata controls.

When proposal filters change, refresh:

- proposal list
- proposal summary
- materialization preflight

## Required Preflight Content

Show primary metrics:

- total proposals
- eligible
- duplicate risk
- blocked
- invalid
- would write
- reviewed ratio
- minimum reviewed ratio
- high/critical unreviewed

Show global blockers:

- `review_coverage_below_threshold`
- `high_critical_unreviewed_proposals`
- any future unknown reason token as plain text

Show reason breakdown:

- `item_reason_counts`, sorted by count descending and then token ascending.

Show per-proposal preflight rows:

- proposal id
- proposed signal type
- preflight status
- proposal review status
- severity
- confidence
- algorithm id
- execution request id
- algorithm result id
- would write
- reasons
- duplicate signal ids

The panel should be dense and scannable. Avoid large decorative cards. A compact metrics strip plus a small reason list/table is preferred.

## Visual Semantics

Use existing badge/status styles where possible.

Recommended status semantics:

- `eligible`: neutral/success tone, but do not imply signal creation.
- `duplicate_risk`: warning tone.
- `blocked`: warning tone.
- `invalid`: error tone.

Warnings:

- `review_coverage_satisfied=false` should be prominent.
- `high_critical_unreviewed_count > 0` should be prominent.
- `would_write_count > 0` must still be labeled as hypothetical/preflight, not as an action.

Required copy:

- The panel must clearly state this is `Read-only preflight`.
- The panel must not say proposals are accepted, deployed, materialized, or production signals.

## Empty, Loading, And Error States

Loading:

- Show a compact loading state in the preflight panel only.
- Do not block proposal list/detail/review interactions while preflight loads.

Empty:

- If `total_proposals=0`, show: `No materialization preflight rows for this filter.`

Error:

- Follow existing API error rendering patterns.
- The proposal list and review workflow should remain usable if only preflight fails.
- Do not expose bearer tokens, auth headers, secrets, raw environment variables, or upstream internals.

## Out Of Scope

- New backend endpoints.
- Materialize button or materialization mutation.
- Bulk materialization controls.
- New proposal review statuses.
- Materialization ledger UI.
- Production `signal.v1` creation.
- Alert or insight creation.
- Graph proposal creation.
- Algorithm execution triggering.
- Algorithm tuning.
- Policy deployment.
- Syncratic Ask/Search.
- New navigation shell.

## Suggested Implementation Areas

Frontend-agent should inspect the G114/G116 implementation first. Likely touch points:

- algorithms API client: add `getAlgorithmSignalMaterializationPreflight` or equivalent.
- React Query hooks: add a preflight query keyed by tenant and active proposal filters.
- type definitions: add response and item types for `algorithm_signal_materialization_preflight`.
- algorithms summarizers: add a small helper to normalize preflight response values and format reason counts.
- Algorithms route: render the read-only preflight panel above the proposal list.
- tests: API path construction, envelope parsing, filter coupling, preflight status rendering, empty/error states, and no materialization controls.

Use existing app styling and component conventions.

## Validation Checklist

Frontend-agent should verify:

- Preflight endpoint is called with active proposal filters.
- `limit` is sent to preflight and remains coupled to the proposal list limit.
- `min_reviewed_ratio` defaults to `1` unless the UI explicitly changes it.
- `policy_version` defaults to `materialization_preflight.v1`.
- Top-level counts render correctly.
- Global blockers render prominently.
- Unknown reason tokens render safely as text.
- `preflight_status` badges render for `eligible`, `duplicate_risk`, `blocked`, and `invalid`.
- Empty preflight renders cleanly for `total_proposals=0`.
- Preflight query errors do not crash or block proposal list/detail/review workflow.
- Existing G114/G116 proposal list/detail/review/summary workflows still work.
- No materialization, alert, insight, graph, deploy, tune, or run controls are added.
- Desktop and mobile layouts remain usable.

## Handoff Note

Backend G118 is implemented and validated. It is intentionally read-only. A later backend gate is still required before any algorithm proposal can become a production signal.
