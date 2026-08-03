# MarketOps Analyst Capability Guide

## A practical evidence-first workflow

MarketOps is designed for an analyst who needs to decide where to spend attention. It starts with a covered asset universe, surfaces strategic and tactical evidence separately, and elevates only supported combinations into a review workflow.

No score is a recommendation. The useful question is not “what should I trade?” but “what does the available evidence say, how complete is it, and does this name warrant further research?”

## Start with the asset universe and dashboard

The MarketOps dashboard gives a current cross-section of covered assets and post-close evidence. Use its interactive cards and tables to narrow to a meaningful cohort—such as options-flow extremes, current breadth, or the reversal review queue—then pivot to the selected asset.

The asset list is the governed universe reference. It is intentionally shared across MarketOps views so an asset’s Market State, valuation, reversal context, and review membership refer to the same coverage population.

## Investigate a selected asset in Market State

Market State brings together the asset’s latest completed price, session condition, technical evidence, research context, and corroboration. It is the primary place to resolve a result before treating it as a research lead.

Use it to answer:

- What was the last completed price and session date?
- Is the technical condition constructive, neutral, or cautionary?
- Is the price extended versus range, moving averages, or recent returns?
- Does aggregate options flow support, oppose, or fail to inform the current context?
- Which facts are current, incomplete, stale, or unavailable?

The Asset overview remains the detailed source for the price/sentiment/corroboration chart and options-distribution evidence. Market State reuses this analyst context so a workflow pivot does not discard the supporting data.

## Use strategic valuation as context, not timing

The **Valuation & DOSM** view has two related but different results:

- **Valuation Composite (VC):** a 0–10 relative-valuation context based on P/S, GAAP P/E, and EV/EBITDA, with peer adjustment only when coverage is adequate.
- **Distressed Opportunity Scoring Model (DOSM):** a 0–10 strategic research rank combining VC with operating quality, profitability, free cash flow, debt profile, capital efficiency, and a bounded technical adjustment.

Select an asset row to open the calculation details directly below it. Inspect metric provenance, model version, data profile, component scores, confidence, and withheld inputs before drawing a conclusion.

Current production behavior is TTM-only. Revenue CAGR and the high-valuation/low-growth penalty are withheld until 16 rolling quarters are retained. A missing growth component is not treated as a zero; it reduces completeness and confidence.

## Use daily tactical evidence to understand condition

| View or result | What it means | Appropriate analyst use |
|---|---|---|
| Risk/Reward Temporal | A bounded technical posture based on range, RSI, returns, volume, SMAs, ATR, and separately labeled put/call context | Assess current price/technical condition; do not use it as a standalone instruction |
| Tactical Market Posture | Constructive, neutral, or caution daily context from RSI, five-day return, SMA position, and slope | Add concise daily context to strategic VC/DOSM research |
| Exhaustive Reversal | A signed stance for an extended up/down drift whose participation may be fading or climactic | Decide whether a potential reversal deserves investigation, while checking whether the trend remains supported |
| Options-flow extreme | Aggregate put/call volume outside defined thresholds and above activity minimum | Treat as descriptive participation context; never infer intent by itself |

## Read Exhaustive Reversal correctly

Exhaustive Reversal (EROC) makes both directions visible. A positive/bullish reversal-review stance corresponds to a sustained downward price drift; a negative/bearish stance corresponds to sustained upward drift. Neither is an automatic position. Each represents a potentially exhausted extension that merits review.

The calculation checks directional persistence, the magnitude of extension relative to typical movement, volume regime, and reversal-direction flow proximity. The review assessment makes the regime explicit:

- **Fading drift:** participation has declined as the move persists.
- **Climactic extension:** recent participation is unusually elevated.
- **Trend-supported:** price and activity remain aligned; this is monitoring context and is not reversal-ranked.

Complete reversal evidence requires same-session aggregate calls/puts. When that input is unavailable, the system preserves the observed condition as incomplete rather than manufacturing certainty.

## Use Opportunities and Review Queue selectively

The Opportunities view and Review Queue are not a stream of all signals. They are a convergence layer.

- A directional review item requires at least two independent sources to agree on the same asset, direction, and completed session.
- Risk/Reward, EROC, Tactical Market Posture, and options-flow extreme can contribute, but no one method is enough alone.
- Material opposing evidence becomes a non-directional mixed-conviction review instead of a misleading directional conclusion.
- An empty queue means the governance threshold was not met; it is not a failure of the system.

When an item appears, inspect the Review Assessment and pivot through the asset hyperlink to Market State. Confirm the date, evidence sources, condition, confidence, and data quality before allocating research time.

## Interpret options data with discipline

MarketOps uses aggregate put/call contract volume. Put/call below 1 means calls are elevated relative to puts; above 1 means puts are elevated. The extreme layer uses values below 0.30 or above 1.20 and requires at least 1,000 aggregate contracts.

Aggregate options activity cannot show whether positions were bought or sold, opened or closed, directional or hedged. It should only corroborate—or challenge—other completed-session evidence.

## Analyst checklist

1. Confirm the completed session and freshness of the data.
2. Separate strategic VC/DOSM context from daily tactical evidence.
3. Read score explanations and withheld-data reasons; do not substitute missing inputs with assumptions.
4. Treat options flow as context, not intent.
5. Prioritize convergent items, but investigate disagreement when it is material.
6. Record the analyst’s own thesis and risk assessment outside the system’s deterministic artifact; MarketOps does not make that judgment.
