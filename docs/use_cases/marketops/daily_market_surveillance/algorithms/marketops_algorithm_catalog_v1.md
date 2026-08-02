# MarketOps Algorithm Catalog and Expected Outcomes

**Status:** implemented operational catalog as of 2026-08-02  
**Scope:** deterministic, research-only MarketOps analytics. None of these algorithms submits trades, changes a holding, or makes an investment recommendation.

## Operating model

MarketOps separates **strategic** financial context, refreshed weekly, from **tactical** post-close context, refreshed after each completed session. A third layer requires independent same-session evidence before creating a selective analyst-review item.

| Layer | Algorithms | Refresh | Analyst question |
| --- | --- | --- | --- |
| Strategic | Valuation Composite (VC), DOSM | Weekly FMP refresh; persisted snapshots reused daily | Is the available financial evidence relatively attractive, weak, or incomplete? |
| Tactical | Risk/Reward Temporal, Tactical Market Posture, Exhaustive Reversal | Daily post-close | What is the current price/technical condition and does it merit monitoring or review? |
| Cross-signal | Options-flow extremes, convergence opportunity builder | Daily post-close | Do at least two independent sources agree strongly enough for research review? |

All inputs are persisted before calculation. Results are deterministic for identical snapshots and configuration, retain provenance, and are evidence—not advice.

## Strategic valuation

### Valuation Composite (VC)

- **Registry ID:** `signalops.algorithms.valuation_composite_v3`
- **Purpose:** relative valuation from canonical price/market capitalization and normalized GAAP financial inputs.
- **Logic:** P/S contributes 40%, GAAP P/E 30%, and EV/EBITDA 30%; a peer adjustment applies once. Score is clamped to 0–10. The explainable fair-value anchor is `price × exp(0.1 × (VC − 5))`, not a target price.
- **Profile:** FMP Income Statement, Balance Sheet, and Cash Flow are retained as four-quarter TTM data. Three-year revenue CAGR and its high-valuation/low-growth penalty are withheld until the 16-quarter point-in-time gate is met.
- **Expected outcome:** a valuation-quality research context with raw metrics, component scores, confidence, and withheld-data reasons. It is not a timing signal.

### Distressed Opportunity Scoring Model (DOSM)

- **Registry ID:** `signalops.algorithms.distressed_opportunity_scoring_v3`
- **Purpose:** rank research candidates by strategic valuation and operating resilience.
- **Logic:** `0.50 × final VC + 0.50 × fundamental score + bounded technical adjustment`, clamped to 0–10. Fundamental quality uses revenue growth when available plus operating margin, profitability, free-cash-flow margin, debt profile, and capital efficiency. With TTM-only data, the five available dimensions are equally reweighted; missing growth is never treated as zero.
- **Expected outcome:** a slow-moving strategic ranking and fair-value anchor. Confidence falls with stale financials, thin peer coverage, missing technical inputs, or absent growth history; it must not be treated as a daily trade trigger.

Normative formulas and TTM restrictions: [deterministic specification](../../../../marketops/SignalOps_MarketOps_VC_DOSM_Deterministic_Algorithm_Specification_v3.0.md) and [TTM profile](../../../../marketops/SignalOps_MarketOps_VC_DOSM_TTM_Operational_Profile_v1.md).

## Daily tactical algorithms

### Risk/Reward Temporal

- **Registry ID:** `signalops.algorithms.risk_reward_temporal_v1`
- **Inputs:** 252-session range position; RSI; 5-session return; 10-session volume ratio; 50/200-session SMA distances and slope; ATR; put/call volume ratio and deviation.
- **Logic:** bounded technical inputs produce technical direction and score. Put/call is `puts ÷ calls`: below 1 is calls elevated and above 1 is puts elevated. It is speculative corroboration only and cannot determine direction.
- **Expected outcome:** persisted EOD technical posture in Assets and Market State; one independent evidence source only, never a standalone instruction.

### Tactical Market Posture

- **Registry ID:** `signalops.algorithms.tactical_market_posture_v1`
- **Inputs:** RSI-14, 5-day return, distance from 50- and 200-day SMAs, and 50-day SMA slope.
- **Logic:** RSI reversal, aligned SMA trend, and five-day extension each add `−0.5`, `0`, or `+0.5`; sum is clamped to `−1.5…+1.5`. `≥+0.5` is constructive, `≤−0.5` caution, otherwise neutral.
- **Expected outcome:** daily current-condition context beside VC/DOSM. It does **not** modify strategic valuation scores or fair-value anchors.

### Exhaustive Reversal (EROC)

- **Registry ID:** `signalops.algorithms.eroc_v6`; **model:** `eroc-v6.1`
- **Purpose:** identify a price extension whose participation pattern merits analyst reversal review, while distinguishing a still-supported trend.
- **Readiness:** 21 completed EOD closes/volumes; same-session aggregate calls/puts are optional but required for *complete* reversal evidence.
- **Direction:** ten-session downward drift maps to positive/bullish reversal review; upward drift maps to negative/bearish review. Stance is `−100…+100`; the underlying evidence score is `0…100`.
- **Price gate:** four consecutive directional closes or at least 80% directional closes in a 5-, 6-, or 7-session window, plus extension ≥3× the 20-session mean absolute daily move.
- **Regimes:** fading drift = current five-day volume ≤85% of prior five days; climactic extension = latest volume ≥1.75× prior 20-session average; trend-supported = current/prior five-day volume ≥0.95 with drift-aligned flow ≥1.20. Trend-supported is monitor context, never reversal-ranked.
- **Score:** `25% persistence + 30% extension + 25% regime volume + 20% reversal-flow proximity`. A candidate is `CONFIRMED` only when reversal-direction flow is ≥1.20; otherwise it remains an observed, incomplete review.
- **Expected outcome:** a prioritized queue for fading/climactic extensions, not a prediction. Both signed directions are review opportunities, not required positions.

## Cross-signal review and outcome measurement

### Options-flow extremes

An extreme requires aggregate options volume ≥1,000. Put/call `<0.30` is call-volume extreme and `>1.20` is put-volume extreme. Aggregate volume cannot identify buys/sells, opening/closing, premium direction, or hedging; it is descriptive corroboration, never standalone advice.

### Convergence opportunity queue

V2 requires two independent sources—Risk/Reward, EROC, Tactical Market Posture, or options-flow extreme—to agree on asset, direction, and completed session. A material opposing pair (each strength ≥0.20) becomes non-directional mixed-conviction review rather than a false directional conclusion. Active v2 rows expire on a symbol rebuild; history and outcome lineage remain.

**Expected outcome:** a selective, explainable research queue. An empty queue is healthy when no sources agree. Membership creates no alert, recommendation, or trade.

### Outcome maturity

After final convergence refresh, outcome materialization reruns over the preceding 45 calendar days, changing deterministic 1-, 5-, 10-, and 20-trading-session records from pending to matured when canonical closes exist. These are calibration evidence only; no performance claim is valid until sufficient mature results exist by regime, direction, and completeness.

## Schedule, UI, and limits

The weekday post-close run starts **18:01:55 ET**: collection/normalization, Market State, hypotheses and Risk/Reward; retention and weekly valuation; Tactical Posture and EROC; final convergence and outcome sweep; universal completion gate and Syncratic intelligence.

- `/marketops/valuation`: VC/DOSM and tactical calculation trace.
- `/marketops/eroc`: reversal assessment and evidence.
- `/marketops/opportunities` and `/marketops/review`: convergent review items and pivots.
- `/admin/algorithms`: registry definitions for VC, DOSM, Risk/Reward, Tactical Posture, and EROC.

FMP is rate-limited and strategic/weekly. EROC/options flow use aggregate volume and cannot infer intent. Technical results can change after each completed session. Threshold changes require replay/calibration evidence, not a small live sample.
