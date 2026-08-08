# SignalOps MarketOps

## Deterministic market surveillance for explainable opportunity research

SignalOps MarketOps helps research teams turn fragmented equity, options, technical, and financial evidence into a consistent daily review process. It operates as a specialized SignalOps domain: data is persisted, normalized, evaluated by versioned deterministic algorithms, and presented with the inputs and reasons an analyst needs to assess it.

MarketOps is research decision support—not a trading system. It does not place orders, alter portfolios, or issue investment recommendations.

## The challenge

Analysts often have to reconcile end-of-day price movement, options activity, trend indicators, reported financials, and research hypotheses in separate tools. The result is slow triage, inconsistent interpretation, and alerts that are difficult to explain or audit. A single factor can look compelling while being incomplete, stale, or contradicted by other evidence.

MarketOps addresses this by separating slow-moving strategic context from daily tactical conditions, then only elevating selective cross-signal convergence for review.

## What MarketOps delivers

| Capability | What it provides | Analyst value |
|---|---|---|
| Unified asset universe | A governed list of covered assets and canonical completed-session market data | A consistent surveillance perimeter and a common reference point across views |
| Market State | Price, technical, options, research, and corroborating evidence for a selected asset | A fast way to understand the current condition without losing provenance |
| Strategic valuation research | Valuation Composite (VC) and Distressed Opportunity Scoring Model (DOSM) | A transparent strategic ranking based on retained financial and market snapshots |
| Daily tactical surveillance | Risk/Reward Temporal, Tactical Market Posture, and Exhaustive Reversal | A current-session view of extension, trend, participation, and reversal conditions |
| Sector Intelligence Foundation | Price-led cross-sectional ranks for core sectors and industries versus SPY, QQQ, and RSP | A transparent way to choose where deeper research may be most relevant, without claiming rotation or a trade |
| Options-flow context | Aggregate put/call-volume extremes with activity thresholds | A descriptive corroborating input, clearly separated from directional inference |
| Convergent review | An Opportunities/Review Queue requiring independent same-session evidence | Less noise and a more selective research queue |
| Administrative governance | Algorithm registry, schedules, statuses, data provenance, and retention governance | Operational visibility and repeatable research controls |

## Deterministic evidence model

Every MarketOps result is calculated from persisted inputs and a versioned rule set. Given the same input snapshot and configuration, the calculation produces the same result. Results retain model identity, source/provenance, freshness, completeness, score trace, and withheld-data reasons where applicable.

This makes the system explainable by design. It does not ask an analyst to trust an opaque signal; it shows the factors, boundaries, and confidence that produced the research artifact.

## Three evidence layers

| Layer | Cadence | Core question | Implemented methods |
|---|---|---|---|
| Strategic | Weekly financial refresh; cached snapshots reused between refreshes | Is the available financial evidence relatively attractive, weak, or incomplete? | VC and DOSM |
| Tactical | After each completed market session | What is the current price and technical condition? | Risk/Reward Temporal, Tactical Market Posture, Exhaustive Reversal |
| Convergent | Daily post-close | Do independent inputs align strongly enough to merit review? | Options-flow extremes and convergence opportunity builder |

Strategic and tactical outputs remain distinct. A daily technical condition does not silently overwrite valuation context, and a valuation score is not presented as a trade-timing instruction.

## Implemented proprietary algorithms

- **Valuation Composite (VC):** an explainable 0–10 relative-valuation score using price-to-sales, GAAP price-to-earnings, and EV/EBITDA, with a bounded peer adjustment when peer coverage is adequate.
- **Distressed Opportunity Scoring Model (DOSM):** a 0–10 strategic research rank that combines VC, operating quality, cash generation, debt profile, capital efficiency, and a bounded technical adjustment.
- **Risk/Reward Temporal:** an end-of-day technical posture derived from range position, RSI, returns, volume, moving averages, ATR, and separately labeled put/call corroboration.
- **Tactical Market Posture:** a concise daily constructive, neutral, or caution context based on RSI, recent return, SMA position, and trend slope.
- **Exhaustive Reversal:** a signed reversal-review stance that detects extended directional movement and distinguishes fading/climactic participation from a trend that remains supported.
- **Convergence Opportunity Builder:** a selective review layer that requires at least two independent sources to agree on asset, direction, and completed session; material disagreement becomes a non-directional mixed-conviction review.
- **Sector Rotation Intelligence Foundation (SRI):** a versioned price-led cross-sectional context rank built from 5-, 20-, and 60-session ETF returns, benchmark-relative strength, momentum, and acceleration. It is explicitly not a rotation, flow, breadth, or recommendation engine.

## Data and operating model

- **Massive:** canonical completed-session adjusted close, market capitalization, shares, technical indicators, and options market data.
- **Financial Modeling Prep (FMP):** quarterly income statement, balance sheet, and cash-flow rows for strategic financial analysis. FMP consumption is controlled: routine recalculation reuses retained snapshots, while scheduled weekly and continuation jobs perform authorized financial refreshes.
- **Post-close operation:** weekday processing begins after the completed session and materializes market state, tactical evidence, reversal analysis, convergence, and outcome maturation.
- **Outcome lineage:** matured 1-, 5-, 10-, and 20-trading-session observations support later calibration. They are evidence for evaluation, not a performance claim.

## Governance and retention

MarketOps uses a metadata-first model rather than a raw-feed archive. Provider payloads are retained for 30 days; normalized canonical EOD/features/states/scores/outcomes for 12 months; detailed option-chain rows for 92 days with derived results retained for 12 months; and financial facts/snapshots for four years. Governance runs daily in dry-run mode until an approved enforcement change is made.

## Current boundaries

- VC/DOSM is currently **TTM-only**. Three-year revenue CAGR and its valuation penalty are intentionally withheld until a 16-quarter point-in-time history is available.
- Scores and fair-value anchors are mathematical research artifacts, not price targets or recommendations.
- Aggregate options data cannot distinguish buyer/seller, opening/closing, premium direction, or hedge intent.
- An empty selective review queue can be the correct outcome when independent evidence does not converge.
- SRI requires 61 canonical price sessions for each primary ETF and each benchmark. Partial availability is displayed as not-ready context, never as a low rank.

> **Common reusable disclaimer:** SignalOps MarketOps provides deterministic, research-only analytics. It does not provide investment advice, recommendations, trade instructions, or guarantees of performance. Analysts remain responsible for independent research, risk assessment, and decision-making.
