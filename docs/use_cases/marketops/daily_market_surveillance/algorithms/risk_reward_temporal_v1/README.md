# Risk/Reward Temporal v1

Algorithm ID: `signalops.algorithms.risk_reward_temporal_v1`

Research-only daily technical posture for MarketOps assets. It reports a directional technical score, confidence, risk level, weighted factor evidence, and a separate speculative put/call-volume corroboration. It is not a recommendation and never materializes a platform signal.

The Python implementation uses the reusable `signalops.platform_algorithm_execution.v2` multi-feature temporal-vector contract. Existing scalar algorithm contracts remain unchanged.

## Canonical metadata

- Runtime contract: `signalops.platform_algorithm_execution.v2`
- Runtime: `python_plugin`; algorithm type: `trend_detection`; version `v1`, status `active`
- `research_only=true`; MarketOps role: `technical_risk_reward`
- Excluded from algorithm adjudication and signal materialization

## Inputs and scoring

Ten persisted Market State observations drive it — eight technical (`range_position_252d`, `rsi_14`, `return_5d`, `volume_ratio_10d`, `distance_sma_50_pct`, `distance_sma_200_pct`, `sma_50_slope_20d_pct`, `atr_14_pct`) and two speculative put/call (`put_call_volume_ratio`, `put_call_volume_ratio_10d_deviation_pct`). Missing, invalid, stale, or insufficient-history values stay unavailable and are never coerced to zero. Each run writes immutable `algorithm_results` with result type `risk_reward_temporal`. See `input_features.md` and `scoring.md`.

## Operations

Run after Market State materialization through `scripts/marketops_algorithm_corroboration.sh`, once per active asset over a 400-calendar-day feature window. Failed assets are counted without blocking the rest; no output is produced when no usable persisted observations exist. See `operations.md`.

## Cross-references

- Inputs: `input_features.md`
- Scoring and direction rules: `scoring.md`
- Scheduling and backfills: `operations.md`
- Frontend Risk/Reward panel handoff: `frontend_handoff.md`
- Selected-asset API: `risk_reward` object on `GET /v1/tenants/{tenant_id}/marketops/assets/{symbol}/algorithm-observations` (see `frontend_handoff.md`, `docs/deployment.md`).
- Seed definition: `migrations/000044_marketops_risk_reward_algorithm.up.sql`.
- Functional context: `../../architecture/functional_components.md`.
