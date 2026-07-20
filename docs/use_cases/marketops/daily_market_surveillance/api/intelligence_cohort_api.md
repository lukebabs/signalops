# MarketOps Intelligence Cohort API

G148-C exposes bounded, read-only cohort run and readiness records. All routes use existing bearer authentication and tenant enforcement.

## Routes

- `GET /v1/marketops/intelligence/cohort-runs?tenant_id=...&universe_group=...&status=...&limit=...`
- `GET /v1/marketops/intelligence/cohort-runs/{run_id}?tenant_id=...`
- `GET /v1/marketops/intelligence/readiness?tenant_id=...&universe_group=...&symbols=AAPL,MSFT&latest_session_date=YYYY-MM-DD&rollout_status=...&limit=...`

The run detail returns `cohort_run` plus its bounded `symbol_results`. Readiness returns an aggregate-first object with `aggregate` and `symbols`; it resolves latest durable result per symbol in one repository query.

Readiness has independent coverage, evaluation, governance, calibration, and outcome dimensions. Rollout status is one of `not_observed`, `inspection_ready`, `research_evaluation_ready`, `review_ready`, or `blocked`. The aggregate always returns `production_ready_supported: false`.

## Explicit runner

`signalops-marketops-intelligence-cohort-runner` requires tenant, either symbols or universe group, a maximum of 10 symbols, inclusive session bounds, explicit stages, and a run ID. Writes require `--acknowledge-writes`; `--dry-run=true` writes neither cohort rows nor analytical rows.

Allowed dependency-ordered stages are preflight, state materialization, hypothesis evaluation, opportunity build, outcome materialization, and hypothesis proposal generation. Provider acquisition, Graph mapping, Ask, proposal decisions, lifecycle promotion, and direct signal materialization are not runner stages.

Actor identity comes from `SIGNALOPS_ACTOR`. Per-symbol failures are recorded and may continue only with `--continue-on-error`. Conflicting run-ID scope is rejected.
