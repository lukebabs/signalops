# Syncratic daily narratives — 2026-08-23

## Purpose

Syncratic Ask was expanded from narrow per-asset context windows toward platform-level MarketOps explainability. The new daily narrative layer materializes deterministic contexts over persisted MarketOps artifacts, then optionally queues Syncratic Ask to produce analyst-facing prose. The generated prose is explainability only; it is not a signal, recommendation, provider poll, graph mutation, or lifecycle decision.

## Implemented slice

- Added daily narrative context strategies:
  - `marketops_daily_overview_v1`
  - `marketops_sri_daily_v1`
  - `marketops_risk_reward_daily_v1`
  - `marketops_review_queue_daily_v1`
- Added same-origin API: `POST /v1/syncratic/daily-narratives/materialize`.
- Reused existing `syncratic_context_windows`, `syncratic_insights`, and `syncratic_intelligence_jobs`; no new schema was required.
- Added prompt builder `marketops.syncratic.daily_narrative_prompt.v1` with artifact citation, data-quality, and no-trading-instruction requirements.
- Updated the Syncratic worker to process daily narrative contexts with the daily narrative prompt and insight type.
- Reworked `/marketops/syncratic` so daily narratives are the primary workbench and legacy per-asset context windows remain under Asset Drilldowns.

## Evidence boundary

The v1 context builders read persisted MarketOps evidence only:

- Risk/Reward snapshots
- SRI snapshots and ranks
- Review Queue opportunities
- Combined daily overview from those deterministic sections

The materializer does not call providers, alter algorithms, mutate lifecycle state, or write production signals.

## Subscription value alignment

- Explorer: daily overview and public sector context can become the explainability entry point.
- Professional: full SRI, Risk/Reward, Review Queue, EEOM, VC/DOSM narratives can be surfaced as analyst workflow accelerators.
- Institutional: future extension should add Signal Assurance analytics, custom universes, portfolio contexts, historical replay summaries, and API access.

## Validation

- `go test ./internal/api`
- `npm --prefix web test -- --run src/api/syncratic.test.ts src/lib/syncratic.test.ts`
- `npm --prefix web run build`
