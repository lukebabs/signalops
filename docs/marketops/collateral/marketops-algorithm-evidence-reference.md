# MarketOps Algorithm and Evidence Reference

## Operating principles

MarketOps algorithms are deterministic, research-only artifacts. They run on persisted input snapshots, preserve provenance and model version, and are designed to explain a condition—not to make an investment recommendation. Scores are bounded representations of their own method; they are not interchangeable confidence, probability, return, or price-target measures.

| Evidence layer | Refresh | Purpose | Must not be interpreted as |
|---|---|---|
| Strategic | Weekly financial refresh; snapshots reused | Relative valuation and operating resilience | A daily trade trigger |
| Tactical | Daily after completed session | Current price/technical state | A forecast or position instruction |
| Convergent | Daily after tactical results | Selective review prioritization | Guaranteed opportunity quality |

## Strategic algorithms

### Valuation Composite (VC)

**Registry ID:** `signalops.algorithms.valuation_composite_v3`  
**Question answered:** Given retained trailing financials and the canonical price/market-cap snapshot, how relatively expensive or attractive is this equity on the defined valuation measures?

VC uses P/S (40%), GAAP P/E (30%), and EV/EBITDA (30%) mapped to deterministic component curves. Where at least three same-sector/same-industry peers exist, one bounded peer-relative adjustment is applied. The final score is clamped to 0–10.

The displayed fair-value anchor is `price × exp(0.1 × (VC − 5))`. It is an explainable mathematical translation of the score, not a target price, forecast, or recommendation.

**Completeness boundaries:** VC requires usable TTM statements, a valid completed-session price, and market capitalization. In the current profile, FMP supplies four quarterly rows per statement. Three-year revenue CAGR and its valuation penalty are withheld pending the 16-quarter history gate.

### Distressed Opportunity Scoring Model (DOSM)

**Registry ID:** `signalops.algorithms.distressed_opportunity_scoring_v3`  
**Question answered:** Which assets have a relatively attractive or distressed strategic evidence profile that warrants research?

DOSM combines 50% final VC, 50% fundamental quality, and a bounded technical adjustment, then clamps the result to 0–10. In the TTM-only profile, fundamental quality equally reweights operating margin, GAAP profitability, free-cash-flow margin, debt profile, and capital efficiency/ROIC. Revenue growth is not assumed absent; it is withheld and confidence is reduced.

Massive RSI-14, SMA-50, and SMA-200 supplement DOSM through provider-backed technical provenance. They do not force an FMP refresh and do not make DOSM a daily timing model.

**Appropriate outcome:** a strategic research ranking with score trace, confidence, data completeness, and a valuation anchor. It is not a recommendation to buy, sell, hold, or allocate capital.

## Tactical algorithms

### Risk/Reward Temporal

**Registry ID:** `signalops.algorithms.risk_reward_temporal_v1`  
**Question answered:** What does the completed-session technical evidence say about present price/risk posture?

Inputs include 252-session range position, RSI, five-session return, ten-session volume ratio, SMA distances and slope, ATR, and put/call volume ratio/deviation. Missing, invalid, stale, or insufficient-history values remain unavailable; they are never coerced to zero. Aggregate put/call is distinct speculative corroboration and cannot decide direction.

**Appropriate outcome:** one independent technical evidence source shown in Asset and Market State workflows. It is intentionally not a platform action, alert, or standalone instruction.

### Tactical Market Posture

**Registry ID:** `signalops.algorithms.tactical_market_posture_v1`  
**Question answered:** Is the daily technical condition constructive, neutral, or cautionary?

RSI reversal, aligned SMA trend, and five-day extension each contribute `−0.5`, `0`, or `+0.5`; the sum is bounded from `−1.5` to `+1.5`. Scores at or above `+0.5` are constructive; scores at or below `−0.5` are caution; the remainder is neutral.

**Appropriate outcome:** a compact tactical lens beside strategic valuation evidence. It does not change VC/DOSM values or fair-value anchors.

### Exhaustive Reversal

**Registry ID:** `signalops.algorithms.eroc_v6`; **model:** `eroc-v6.1`  
**Question answered:** Has a sustained price extension reached a participation pattern that merits a reversal review, or does the trend still appear supported?

The method needs 21 completed EOD closes/volumes. It identifies four consecutive directional closes or at least 80% directional closes in a five-, six-, or seven-session window, then requires the extension to reach at least three times the 20-session mean absolute daily move. A downward drift produces a positive/bullish reversal-review stance; an upward drift produces a negative/bearish stance. The signed stance is `−100…+100`; its evidence score is `0…100`.

Its score is 25% directional persistence, 30% extension, 25% volume regime, and 20% reversal-flow proximity. Fading drift, climactic extension, and trend-supported regimes prevent a generic interpretation of directional movement. Trend-supported is monitoring context and is never reversal-ranked. A confirmed result additionally requires reversal-direction aggregate flow at or above 1.20; absent flow leaves the observed condition incomplete.

**Appropriate outcome:** a prioritization mechanism for analyst reversal review in either direction, not a claim that a reversal will occur.

## Cross-signal evidence

### Options-flow extremes

Aggregate options volume must be at least 1,000 contracts. A put/call ratio below 0.30 is a call-volume extreme; above 1.20 is a put-volume extreme. Ratio means puts divided by calls.

This evidence is descriptive. It cannot establish whether contracts were bought or sold, opened or closed, used as a hedge, or intended to express a directional view. It should never stand alone.

### Convergence Opportunity Builder

The convergence layer requires two independent completed-session sources—Risk/Reward, Exhaustive Reversal, Tactical Market Posture, or an options-flow extreme—to agree on asset and direction. Material opposing evidence (each strength at least 0.20) becomes a non-directional mixed-conviction review rather than a false directional assertion.

Active rows expire on a symbol rebuild; their history and outcome lineage remain. The system may correctly produce no active review items when evidence does not meet this threshold.

**Appropriate outcome:** a selective research queue that gives an analyst a reason to investigate, together with the evidence to challenge or confirm that reason.

## Evidence, provenance, and evaluation

Every output is associated with input/session context, a model identity, and source/freshness information. MarketOps later materializes deterministic one-, five-, ten-, and 20-trading-session outcome observations after a final convergence refresh. These support calibration only after enough outcomes mature across relevant conditions, direction, and completeness.

No algorithm should be described as validated, predictive, or performance-generating merely because it has stored historical outcomes. Threshold changes require replay and calibration evidence, not a small live sample.

## Current data and retention constraints

- Financial evidence is refreshed strategically, not on every tactical run; routine jobs reuse retained snapshots to control FMP consumption.
- Financial analysis is currently four-quarter TTM only. The 16-quarter CAGR enhancement is not yet active.
- MarketOps raw provider payloads are retained for 30 days; normalized EOD/features/states/scores/outcomes for 12 months; detailed option chain rows for 92 days; financial facts/snapshots for four years.
- Retention governance currently performs daily dry-run evaluation. Deletion remains separately controlled and is not enabled by the schedule.
