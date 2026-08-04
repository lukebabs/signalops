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
- Event timing
- Sector context
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

The score estimates the probability that the earnings event offers a favorable risk/reward profile.

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

## Future

- Bayesian probability fusion
- Cross-asset propagation
- Peer earnings effects
- Analyst feedback calibration

