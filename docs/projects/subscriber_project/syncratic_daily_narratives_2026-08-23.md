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

## Production execution — 2026-08-23

- Materialized tenant-local daily narratives for completed session `2026-08-21` using the four v1 strategies.
- Persisted rows are stored in the dedicated MarketOps database, not the shared SignalOps database.
- Artifact read-back:
  - Daily Overview: 400 refs
  - Sector Rotation: 160 refs
  - Risk/Reward: 120 refs
  - Review Queue: 120 refs
- Corrected a routing gap found during execution: Syncratic MarketOps routes now resolve through the dedicated MarketOps repository when configured.
- Corrected deterministic digest behavior by sorting Risk/Reward leader candidates before truncating the top examples.
- Added a Syncratic worker stale-digest guard and updated the post-close Syncratic runner to build before run so obsolete digest jobs are drained without duplicate Ask calls.
- Final production idempotency proof: rerunning materialization returned four `unchanged_evidence_digest` skips and created no additional context windows, insights, or jobs.

## Chunked prompt strategy

Daily narrative Ask now follows a chunked/map-reduce-ready pattern. Focused narratives remain bounded by their own context strategies, while Daily Overview is a compact synthesis over section summaries rather than a full dump of all lineage refs. Full provenance remains in the database. The Ask prompt receives bounded examples, artifact totals, and capped citation samples. SignalOps uses a conservative `10000`-byte local proxy for the current Syncratic AI Gateway `4000` input-token limit.


Current SignalOps policy target:

- Keep Syncratic AI Gateway policy moderate; 4,000 input tokens / 1,000 output tokens remains a reasonable target for focused prompts.
- Daily Overview should rely on compaction and source-section drilldowns before increasing gateway limits.
- If future Institutional/custom-universe contexts require larger prompts, enable that as a separate tier/policy decision.
