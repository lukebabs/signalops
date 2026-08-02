# SignalOps MarketOps Exhaustive Reversal (EROC)

**Version:** 6.1  
**Status:** Implemented deterministic research model  
**Subsystem:** SignalOps → MarketOps  
**Data sources:** Persisted MarketOps EOD closes, EOD volumes, and same-session options distributions

## Purpose

Exhaustive Reversal is a daily analyst-review model, not a reversal prediction. It separates a supported ten-session trend from a fading or climactic extension that merits reversal review. It is research-only and does not alter VC, DOSM, or risk/reward scores.

## Inputs and readiness

Each active asset requires 21 completed EOD observations: the latest ten establish drift, persistence, and extension; the preceding twenty daily moves and volumes establish asset-relative normal activity. The runner also reads same-session aggregate call and put option volume. Insufficient history produces `UNAVAILABLE`; absent options flow produces an explicitly incomplete candidate, never a complete one.

## Regime classification

The ten-session net drift sets a potential reversal stance: downward drift maps to bullish reversal review, upward drift maps to bearish reversal review. A candidate must also meet both price tests:

- persistence: four consecutive directional closes or at least 80% directional closes in a recent 5, 6, or 7-session window;
- extension: distance from the relevant ten-session close extreme is at least 3.0 times the asset’s 20-session mean absolute daily move.

Price-qualified assets are then classified deterministically:

| Regime | Rule | Analyst treatment |
| --- | --- | --- |
| `trend_supported` | Current five-day volume is at least 0.95× the prior five days and options flow aligns with the drift at 1.20× or more | Monitor only; never reversal-ranked |
| `fading_drift` | Current five-day average volume is 0.85× or less of the prior five days | Reversal review candidate |
| `climactic_extension` | Latest volume is at least 1.75× the prior 20-session average | Reversal review candidate |
| `unresolved` | No qualifying reversal or supported-trend regime | Monitor only |

“Accumulation” and “distribution” are not asserted: aggregate EOD and option-volume data can describe a trend as supported, but cannot establish trade-side accumulation or distribution.

## Review priority and evidence completeness

Only fading and climactic regimes receive a signed −100 to +100 stance score. Its absolute evidence score remains 0–100; the UI may present that signed value on a base-10 display scale. The score is a transparent weighted summary:

| Component | Weight |
| --- | ---: |
| Directional persistence | 25% |
| Normalized price extension | 30% |
| Regime participation evidence | 25% |
| Reversal-direction options flow | 20% |

A candidate is **complete** only when its reversal-direction call/put or put/call flow ratio is at least 1.20. Complete candidates rank before incomplete candidates regardless of raw score. Incomplete candidates remain visible with the missing or failed evidence stated in their trace.

## Persistence and interface

The post-close runner writes one immutable result per active asset/session:

```text
algorithm_id: signalops.algorithms.eroc_v6
model_version: eroc-v6.1
provider: marketops_eroc
GET /v1/tenants/{tenant_id}/marketops/eroc
```

The view presents a reversal review queue followed by a collapsed trend-supported and unresolved monitor. Every active asset remains observable; only qualified exhaustion regimes are ranked as reversal work.

## Limits and follow-on measurement

- Options volume does not identify opening versus closing activity, premium direction, or hedging intent.
- The model has no forward-outcome calibration or predictive-performance claim. A future workstream should measure 5-, 10-, and 20-session outcomes by regime and evidence-completeness state before thresholds are retuned.
- No network calls occur during scoring; all inputs are persisted before the deterministic run.
