# Earnings Event Opportunity Model (EEOM) v1.0

## Engineering Specification for SignalOps / MarketOps

### Purpose
The Earnings Event Opportunity Model (EEOM) is a SignalOps orchestration algorithm that evaluates the pre-earnings opportunity of an asset.

It does **not** predict earnings outcomes. It estimates the probability that the current pre-earnings setup presents an attractive risk/reward opportunity.

The absolute goal is to bring raise awareness of  material events that can produce unique opportunities

### Core Principles
- Tactical signals dominate as earnings approach.
- Strategic signals (VC/DOSM) provide context.
- EEOM orchestrates existing algorithms rather than duplicating analytics.
- Deterministic, explainable, idempotent and auditable.
- Leverage existing signals. Rework is not necessary.


## SignalOps Philosophy

Turn Data into Signals.
Turn Signals into Evidence.
Turn Evidence into Opportunity.

Conviction belongs to the analyst.

## Inputs

### Strategic
- VC Score
- DOSM Score
- Valuation
- Balance sheet
- Growth
- Peer comparison

### Tactical (Primary)
Technical Opportunity Engine:
- Existing bullish/bearish technical signals
- Trend
- Relative strength
- Momentum
- Accumulation/distribution
- Support/resistance
- Volume

Options Opportunity Engine:
- Expected move
- ATM straddle
- IV Rank
- IV percentile
- OI changes
- Put/Call positioning
- Skew

### Event
- Rolling 30-day earnings calendar
- Historical earnings volatility
- Portfolio relevance

## Dynamic Weighting

30–21 days: Strategic > Tactical

20–10 days: Balanced

10–5 days: Tactical > Strategic

5–0 days:
Technicals and Options dominate.
VC/DOSM remain contextual.

## Primary Output

Risk / Reward Opportunity Probability (RROP)

The setup-quality score is expressed on a 0.0–10.0 analyst scale; it measures the quality of available pre-earnings evidence.

It does not predict:
- EPS beat
- Revenue beat
- Price direction

## Opportunity Drivers

EEOM aggregates:
- Technical Opportunity Score
- Options Opportunity Score
- VC Score
- DOSM Score
- Event Materiality
- Risk/Reward Model
- Evidence Quality

## Explainability

Every score must expose:
- Supporting evidence
- Contradicting evidence
- Bullish signals
- Bearish signals
- Opportunity classification
- Primary risks

## Classifications

Priority A – High Opportunity

Priority B – Await Validation

Priority C – Better Post-Earnings Opportunity

Priority D – Distressed Inflection

Priority E – Avoid

Informational Only

## Engineering Requirements

- Deterministic
- Explainable
- Idempotent
- Time-series persistence
- Versioned scores
- Modular orchestration
- JSON API outputs

## Earnings-aware material-event context

Financial Modeling Prep (FMP) is the v1 calendar authority. A single bounded calendar request during post-close persists a canonical `market_event_calendar` record before Market State cohorts materialize. The normalized event contract is `symbol`, `event_type`, `event_date`, `event_time: null`, `status: date_reported`, `confidence: null`, `source`, `last_verified`, `known_at`, and derived `days_to_event`. FMP does not provide a reliable BMO/AMC timing or confirmed-status field, so those values are not inferred.

Market State uses only events whose `known_at` predates the session being materialized. The awareness window is ten calendar days before earnings through two calendar days after. Technical, options, volume, and Risk/Reward outputs receive an explainable event-proximity annotation but no numerical score adjustment; score calibration requires separate prospective evidence. The dashboard, Market State, and EEOM surface the same persisted event object.

## Future

- Bayesian probability fusion
- Cross-asset propagation
- Peer earnings effects
- Analyst feedback calibration

