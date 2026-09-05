# SAF-2 — Multi-Horizon Signal Usefulness Sprint

Status: first implementation slice source-ready; production migration/deploy remains gated.

Last updated: 2026-09-05.

## Purpose

This sprint upgrades Signal Assurance from a shallow directional outcome view into a practical signal-usefulness framework.

The core rule is that SAF must not treat the next completed close as the default success or failure condition. A bullish signal that is down after one day is not automatically a failed signal. It may still be developing, may later materialize, may carry useful early-warning value, or may be invalidated by an explicit contract condition.

SAF should answer:

- Did the signal eventually move in the expected direction?
- How long did it take to materialize?
- How much favorable movement was available before expiry?
- How much adverse movement had to be tolerated first?
- Did the asset outperform the broad market or its sector?
- Under which algorithms, score bands, sectors, and market regimes was the signal useful?

This sprint remains research and governance work. It does not create trading advice, execute trades, or automatically modify production algorithm weights.

## Primary cohort

The first execution target remains the tenant-local **MarketOps Legacy Default** cohort of 132 assets.

This is the correct initial scope because it preserves the most mature MarketOps operating history and avoids blending the legacy hot cohort with the wider 1,000-symbol warm-EOD universe. Future cohorts may be added, but they must be labeled and evaluated separately.

Operational SAF reporting continues to use:

- start cutoff: 2026-08-20;
- default UI window: 10 trading days;
- maximum standard UI window: 20 trading days.

## Outcome taxonomy

SAF outcome reporting must use lifecycle states instead of presenting binary hit/miss as the primary interpretation.

| State | Meaning |
|---|---|
| `confirmed` | A signal assertion exists and its baseline is frozen. |
| `developing` | Favorable evidence is emerging, but materialization is not complete. |
| `materialized` | The validation contract threshold was reached within the horizon. |
| `outperformed` | The assertion materialized relative to broad-market or sector benchmark. |
| `adverse_warning` | The assertion moved against the thesis beyond a warning threshold, but not enough to invalidate it. |
| `invalidated` | The assertion breached an explicit invalidation rule. |
| `expired` | The horizon ended without materialization or invalidation. |
| `censored` | The evaluation window is still open; final outcome is not mature. |

`hit` and `miss` may remain as simplified derived labels for compact reporting, but they must not hide the lifecycle state, horizon, or adverse/favorable path.

## Evaluation horizons

Every confirmed assertion should be evaluated across fixed trading-session horizons:

| Horizon | Use |
|---|---|
| 1 trading day | Early reaction diagnostic only. It must not be the default final judgment. |
| 5 trading days | Short confirmation window. |
| 10 trading days | Default operational usefulness window. |
| 20 trading days | Maximum standard operational window for slower-developing signals. |

Each horizon is evaluated independently. A signal may be adverse at 1D, developing at 5D, materialized at 10D, and then expired or invalidated under a longer contract only if the versioned validation contract says so.

## Required measurements

The sprint should define and later implement derived measurements for each assertion/horizon pair:

- directional return at horizon;
- broad-market relative return;
- sector-relative return;
- maximum favorable excursion;
- maximum adverse excursion;
- time to first favorable movement;
- time to materialization;
- drawdown before materialization;
- persistence or continuation score;
- invalidation rate;
- expiration rate;
- benchmark coverage state;
- exclusion reason where evaluation is not possible.

Missing benchmark, sector, price, or contract data must be represented as an explicit coverage state. It must not be converted to zero and must not be treated as a pass or fail.

## Usefulness score

Introduce a versioned research metric named `saf_usefulness.v1`.

Default score composition:

| Component | Weight |
|---|---:|
| Directional resolution | 25% |
| Materialization strength / favorable excursion | 25% |
| Adverse excursion control | 20% |
| Broad-market and sector-relative performance | 20% |
| Timeliness and persistence | 10% |

The score is a research-quality indicator, not a recommendation. Formula changes require a new version and must not restate prior results in place.

## Drift controls

The sprint must preserve SAF’s existing no-drift rules:

- freeze the assertion baseline at confirmation time;
- retain algorithm, version, direction, score, confidence, validation contract, and provenance;
- evaluate only against declared horizons and contract rules;
- store derived metrics as versioned outputs;
- do not overwrite immutable assertion baselines or historical outcome evidence;
- do not silently blend cohorts;
- do not use future information in historical replay;
- do not infer missing sectors, benchmarks, or prices without governed evidence.

## UI and analyst experience

The future UI should make the SAF lifecycle understandable without forcing analysts to infer it from raw scores.

Required experience:

- Signal Assurance remains under MarketOps Tools.
- The default view shows the 10-trading-day operational window.
- Analysts can switch to 20 trading days.
- Each row shows lifecycle state, usefulness score, horizon, materialization state, MFE, MAE, benchmark-relative result, and evidence coverage.
- Clicking a row reveals the assertion baseline, validation contract, benchmark evidence, materialization path, adverse path, and exclusion details.
- Summary cards should distinguish confirmed assertions, developing assertions, materialized assertions, invalidated assertions, expired assertions, and censored assertions.

## Execution sequence

1. Document the SAF-2 contract and acceptance criteria.
2. Add schema support for versioned multi-horizon usefulness observations without modifying immutable assertion baselines.
3. Add a deterministic materializer that derives `saf_usefulness.v1` from existing MarketOps evidence.
4. Add API projections for lifecycle state, horizon metrics, score components, and coverage reasons.
5. Update the MarketOps Tools Signal Assurance view to expose the lifecycle drill-down.
6. Add automated tests for the key interpretation rule: one-day adverse movement does not automatically equal a miss.
7. Validate with Playwright against the tenant-local legacy cohort.

## Acceptance criteria

- A one-day adverse move after a bullish confirmation remains `developing`, `adverse_warning`, or `censored` unless an explicit invalidation rule is breached.
- 1D, 5D, 10D, and 20D outcomes are independently visible.
- Usefulness scoring is versioned and reproducible.
- MFE, MAE, time-to-materialization, and benchmark-relative evidence are visible or explicitly marked unavailable.
- The tenant-local legacy 132 cohort remains separate from the wider warm-EOD universe.
- No provider polling is required for the initial materializer if existing MarketOps evidence is sufficient.
- No historical evidence, assertion baseline, or algorithm output is deleted or restated.
- Playwright validates the SAF Tools path, default 10-day window, max 20-day window, row drill-down, and lifecycle language.

## Implementation slice 1 — 2026-09-05

The first source implementation adds deterministic SAF-2 usefulness semantics without rewriting historical evidence:

- `saf_usefulness.v1` scoring and lifecycle classification are computed from existing immutable SAF/outcome metrics.
- Existing effectiveness and observation APIs now expose usefulness lifecycle state, score, score components, policy version, and time-to-materialization where inferable.
- The Signal Assurance Tools UI shows usefulness in summary cards, effectiveness rows, observation rows, mobile cards, and audit drill-downs.
- Migration `000168_subscriber_global_saf_usefulness_policy` records the versioned policy contract only; it does not poll providers or mutate assertion/outcome evidence.
- Regression coverage includes the required rule that a one-day adverse bullish observation is developing or adverse-warning, not an automatic miss.

## Deferred work

- Wider 1,000-symbol warm-EOD cohort comparison.
- Additional validation horizons beyond 20 trading days.
- Algorithm threshold calibration from usefulness evidence.
- Admin-facing degradation alerts for rolling SAF usefulness decay.
- Historical replay beyond the current operational cutoff.
