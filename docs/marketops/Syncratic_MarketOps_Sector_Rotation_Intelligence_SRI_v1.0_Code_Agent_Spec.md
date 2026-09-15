# Syncratic MarketOps
# Sector Rotation Intelligence (SRI)
## Engineering Specification for Code Agent
### Version 1.0

---

## 1. Purpose

Sector Rotation Intelligence (SRI) is a MarketOps orchestration algorithm that continuously evaluates the evolution of market sectors, industries, and themes using representative exchange-traded funds (ETFs) and their underlying constituent behavior.

SRI is designed to answer:

> Which market segments are strengthening, weakening, accumulating, distributing, rotating in, or rotating out — and is there sufficient evidence to anticipate an emerging change in market leadership?

SRI SHALL provide market-structure context to asset-level algorithms such as VC, DOSM, the Technical Opportunity Engine, the Options Opportunity Engine, the Earnings Event Opportunity Model (EEOM), and the Signal Assurance Framework (SAF).

SRI is not an ETF recommendation engine. SRI uses ETFs as observable proxies for market segments. The objective is to infer sector and thematic state.

## 2. Core Design Philosophy

SRI follows the SignalOps hierarchy:

```text
DATA
  ↓
SIGNALS
  ↓
EVIDENCE
  ↓
SECTOR / THEME STATE
  ↓
ROTATION OPPORTUNITY
  ↓
ASSET CONTEXT
```

The ETF is the observable representation. The underlying target is the economic or market segment.

Examples:

```text
IGV → Software / SaaS
SMH → Semiconductors
XLF → Financials
XLE → Energy
XBI → Biotechnology
KRE → Regional Banks
ITA → Aerospace & Defense
PAVE → Infrastructure
TAN → Solar
ICLN → Clean Energy
```

SRI SHALL separate ETF identity from Segment identity. This allows the same segment to be represented by multiple ETFs where useful.

## 3. Primary Objectives

SRI SHALL:

1. Maintain a curated registry of representative ETFs.
2. Map ETFs to sectors, industries, themes, factors, countries, and asset classes.
3. Select one primary benchmark ETF for each monitored segment.
4. Optionally track one or more secondary ETFs for validation.
5. Track ETF price evolution.
6. Track relative strength.
7. Track technical state.
8. Track volume and participation.
9. Track options positioning when sufficiently liquid.
10. Track constituent breadth.
11. Track constituent diffusion.
12. Track internal leadership.
13. Track capital-flow proxies.
14. Track momentum.
15. Track momentum acceleration.
16. Track relative ranking changes.
17. Detect improving and deteriorating segments.
18. Detect probable rotation-in behavior.
19. Detect probable rotation-out behavior.
20. Produce deterministic segment scores.
21. Produce explicit state classifications.
22. Provide asset-level context to MarketOps.
23. Publish signals into SignalOps.
24. Integrate with SAF for longitudinal validation.

## 4. Non-Goals

SRI v1 SHALL NOT:

- Predict exact institutional fund flows.
- Claim ETF volume is equivalent to net capital inflow.
- Treat price momentum alone as sector rotation.
- Recommend ETF trades.
- Duplicate the Technical Engine.
- Duplicate the Options Engine.
- Reimplement asset-level technical indicators.
- Infer constituent holdings without a versioned source.
- Use LLM-generated sector classifications as production truth.
- Automatically modify VC or DOSM scores.
- Treat all ETFs as equally representative.
- Treat niche or illiquid ETFs as primary benchmarks without explicit approval.

## 5. Conceptual Architecture

```text
                       Market Data
                           │
                           ▼
                    ETF Market Prices
                           │
                           ▼
                    ETF Registry Service
                           │
                           ├───────────────┐
                           │               │
                           ▼               ▼
                    Technical Engine   Options Engine
                           │               │
                           └───────┬───────┘
                                   │
                                   ▼
                           ETF Signal Layer
                                   │
              ┌────────────────────┼─────────────────────┐
              │                    │                     │
              ▼                    ▼                     ▼
        Relative Strength      Breadth Engine       Flow Proxies
              │                    │                     │
              └────────────────────┼─────────────────────┘
                                   │
                                   ▼
                         Sector Diffusion Engine
                                   │
                                   ▼
                         Rotation Scoring Engine
                                   │
                                   ▼
                           Segment State Model
                                   │
                                   ▼
                         Sector Rotation Signals
                                   │
              ┌────────────────────┼──────────────────────┐
              ▼                    ▼                      ▼
             VC                  DOSM                    EEOM
              │                    │                      │
              └────────────────────┼──────────────────────┘
                                   ▼
                             MarketOps UI
                                   │
                                   ▼
                                 SAF
```

## 6. Domain Model

The canonical monitored entity is `market_segment`.

Example:

```json
{
  "segment_id": "segment_software",
  "segment_type": "industry",
  "name": "Software",
  "primary_etf": "IGV",
  "secondary_etfs": ["SKYY"],
  "parent_segment": "Technology"
}
```

Supported segment types SHOULD include:

```text
sector
industry
subindustry
theme
factor
country
region
asset_class
```

## 7. ETF Registry

SRI SHALL maintain a curated ETF registry.

Canonical structure:

```json
{
  "symbol": "IGV",
  "name": "iShares Expanded Tech-Software Sector ETF",
  "segment_id": "segment_software",
  "segment_type": "industry",
  "role": "primary",
  "benchmark_priority": 1,
  "liquidity_tier": "high",
  "options_liquidity": "high",
  "holdings_supported": true,
  "active": true
}
```

The registry SHALL be versioned.

## 8. Recommended Initial ETF Universe

### 8.1 Broad Market Context

```text
SPY  → S&P 500
QQQ  → Nasdaq-100 / Growth Technology Proxy
IWM  → Russell 2000 / Small Cap
DIA  → Dow Industrials
RSP  → Equal-Weight S&P 500
```

These are contextual benchmarks rather than sector signals.

### 8.2 Technology

```text
XLK  → Broad Technology
IGV  → Software
SMH  → Semiconductors
SOXX → Semiconductors secondary benchmark
SKYY → Cloud Computing
HACK → Cybersecurity
CIBR → Cybersecurity secondary benchmark
```

### 8.3 Communication Services

```text
XLC → Communication Services
```

### 8.4 Consumer

```text
XLY → Consumer Discretionary
XLP → Consumer Staples
XRT → Retail
```

### 8.5 Financials

```text
XLF → Broad Financials
KRE → Regional Banks
KBE → Banks
```

### 8.6 Healthcare

```text
XLV → Broad Healthcare
IBB → Biotechnology large-cap / established
XBI → Biotechnology broad/equal-weight
IHI → Medical Devices
```

### 8.7 Industrials

```text
XLI  → Broad Industrials
IYT  → Transportation
ITA  → Aerospace & Defense
PPA  → Aerospace & Defense secondary
PAVE → Infrastructure
```

### 8.8 Materials

```text
XLB → Materials
XME → Metals & Mining
```

### 8.9 Energy

```text
XLE → Broad Energy
OIH → Oil Services
XOP → Oil & Gas Exploration & Production
```

### 8.10 Utilities

```text
XLU → Utilities
```

### 8.11 Real Estate

```text
XLRE → Real Estate
VNQ  → Real Estate secondary
```

### 8.12 Clean Energy / Transition

```text
ICLN → Clean Energy
TAN  → Solar
FAN  → Wind
```

### 8.13 AI / Automation Themes

```text
BOTZ → Robotics / Automation / AI
IRBO → Robotics / AI broader thematic
```

These thematic ETFs SHOULD NOT be treated as equivalent to GICS sectors.

## 9. ETF Selection Criteria

A primary ETF SHOULD maximize:

```text
segment representativeness
+
liquidity
+
trading history
+
constituent transparency
+
options liquidity
+
institutional recognition
```

Recommended deterministic selection score:

```text
ETFRepresentationScore =
0.30 * Representativeness
+ 0.20 * DollarLiquidity
+ 0.15 * TradingHistory
+ 0.15 * HoldingsTransparency
+ 0.10 * OptionsLiquidity
+ 0.10 * InstitutionalRecognition
```

All inputs SHALL be normalized to 0–100. The score MAY be manually overridden.

## 10. Multiple ETF Validation

When more than one ETF represents a segment, SRI SHOULD support cross-validation.

Example:

```text
Semiconductors:
SMH primary
SOXX secondary
```

If both strengthen, segment evidence confidence increases. If they diverge, confidence decreases. ETF disagreement MUST be preserved as evidence.

## 11. Data Requirements

For each ETF:

- regular-session last trade / close
- OHLCV
- adjusted OHLCV for historical evaluation
- volume
- average volume
- shares outstanding where available
- AUM where available
- options summary where liquid
- holdings
- constituent weights
- holdings effective date
- sector metadata
- segment metadata

For constituents:

- asset_id
- symbol
- weight
- price
- 20DMA
- 50DMA
- 200DMA
- technical score
- relative strength
- return windows
- market capitalization where available

## 12. Update Cadence

Recommended:

```text
ETF market prices           daily after close
Technical scores            daily after close
Options signals             daily after close
ETF holdings                weekly
Breadth                     daily
Diffusion                   daily
Relative strength           daily
Rotation score              daily
Segment state               daily
Aggregate reports           weekly
```

Intraday support MAY be added later.

## 13. Technical Integration

SRI SHALL consume the existing Technical Engine. It SHALL NOT recompute technical indicators.

Recommended input contract:

```json
{
  "symbol": "IGV",
  "technical_score": 82.4,
  "bullish_score": 78.0,
  "bearish_score": 22.0,
  "trend_state": "bullish",
  "momentum_state": "strengthening",
  "support_state": "holding",
  "volume_state": "confirming",
  "as_of": "2026-08-07T20:00:00Z",
  "algorithm_version": "tech-3.1"
}
```

## 14. Options Integration

SRI SHALL consume existing Options Engine outputs where ETF options liquidity is sufficient.

Potential inputs:

- bullish options score
- bearish options score
- put/call positioning
- skew
- IV rank
- IV percentile
- open interest concentration
- unusual options activity
- term structure
- expected move

Example:

```json
{
  "symbol": "IGV",
  "options_opportunity_score": 71.2,
  "bullish_options_score": 74.0,
  "bearish_options_score": 26.0,
  "options_liquidity": "high",
  "as_of": "2026-08-07T20:00:00Z"
}
```

If options liquidity is insufficient, options contribution weight = 0 and remaining weights MUST be renormalized.

## 15. Relative Strength Engine

Relative strength is critical.

For each ETF calculate relative performance against:

```text
SPY
QQQ
RSP
peer sector ETFs
```

Example:

```text
IGV vs SPY
IGV vs QQQ
IGV vs XLK
```

Recommended return windows:

```text
5d
10d
20d
60d
120d
252d
```

Relative return:

```text
RS_t = ETF_return_t - Benchmark_return_t
```

## 16. Relative Strength Score

Suggested components:

```text
5d relative return
20d relative return
60d relative return
relative moving-average slope
relative breakout state
rank percentile among monitored segments
```

Example score:

```text
RelativeStrengthScore =
0.10 * RS_5D
+ 0.25 * RS_20D
+ 0.25 * RS_60D
+ 0.20 * RelativeTrend
+ 0.20 * CrossSectionalRank
```

Normalize to 0–100.

## 17. Momentum

SRI SHALL separate momentum from relative strength.

Momentum measures the segment itself. Relative strength measures the segment versus other market references.

Momentum inputs MAY include:

- 5d return
- 20d return
- 60d return
- trend slope
- rate of change
- moving average separation
- technical momentum state

## 18. Momentum Acceleration

Rotation often begins before absolute strength becomes dominant. Therefore calculate acceleration.

Simple form:

```text
MomentumAcceleration = RecentMomentum - PriorMomentum
```

Example:

```text
20d return now = +8%
previous 20d return = +2%
acceleration = +6 percentage points
```

Alternative:

```text
MA = zscore(ROC_10d - ROC_30d)
```

Store both raw and normalized metrics.

## 19. Breadth Engine

ETF price strength alone is insufficient. Breadth SHALL measure constituent participation.

Core breadth metrics:

```text
% constituents above 20DMA
% constituents above 50DMA
% constituents above 200DMA
% constituents positive over 5d
% constituents positive over 20d
% constituents outperforming SPY over 20d
% constituents outperforming sector ETF over 20d
```

## 20. Weighted Breadth

Because ETF holdings are weighted, calculate:

```text
WeightedBreadth_50DMA =
Σ(weight_i * indicator(price_i > MA50_i))
```

Also calculate equal-weight breadth. This allows SRI to detect concentration.

## 21. Equal-Weight vs Cap-Weight Divergence

A key diagnostic:

```text
ETF cap-weight return
vs
median constituent return
vs
equal-weight constituent return
```

Example:

```text
IGV ETF return:      +8%
Equal-weight return: +2%
Median constituent:  +1%
```

Interpretation: narrow leadership.

Versus:

```text
IGV ETF return:      +8%
Equal-weight return: +7%
Median constituent:  +6%
```

Interpretation: broad participation.

## 22. Sector Diffusion Index

Sector Diffusion Index (SDI) is a first-class SRI metric.

Purpose:

> Measure how broadly strength or weakness is distributed through the segment.

Diffusion is distinct from ETF price trend.

## 23. Diffusion Inputs

Recommended:

```text
% above 20DMA
% above 50DMA
% above 200DMA
% positive 20d
% outperforming SPY
% improving technical score
median constituent relative strength
equal-weight vs cap-weight spread
leadership concentration
```

## 24. Sector Diffusion Index Formula

Initial deterministic formula:

```text
SDI =
0.15 * Above20DMA
+ 0.15 * Above50DMA
+ 0.10 * Above200DMA
+ 0.15 * Positive20D
+ 0.15 * OutperformingBenchmark
+ 0.10 * ImprovingTechnicalShare
+ 0.10 * MedianRelativeStrength
+ 0.10 * BreadthConcentrationAdjustment
```

Normalize 0–100.

## 25. Diffusion Interpretation

Suggested bands:

```text
80–100  Broad Expansion
65–79   Healthy Participation
50–64   Mixed Participation
35–49   Narrow / Weakening
20–34   Broad Deterioration
0–19    Capitulation / Severe Weakness
```

## 26. Leadership Concentration

Calculate concentration using top holdings.

Metrics:

```text
top_1_weight
top_5_weight
top_10_weight
Herfindahl-Hirschman concentration
```

Also measure contribution to ETF return.

Example:

```text
Top 5 holdings contribute 82% of ETF 20d gain
```

This SHOULD reduce diffusion quality.

## 27. Internal Leadership Model

Identify:

```text
leading constituents
emerging leaders
laggards
recovering constituents
deteriorating constituents
```

Recommended ranking inputs:

```text
technical score
relative strength
20d return
60d return
volume confirmation
weight in ETF
```

## 28. Leadership Breadth

Leadership should be considered stronger when multiple constituents participate.

Metric:

```text
LeadershipBreadth =
count(constituents in top technical strength band)
/
constituent_count
```

Track trend. Rising leadership breadth is a rotation-in signal. Falling leadership breadth may precede rotation-out.

## 29. Capital-Flow Proxies

SRI SHALL NOT claim exact flows unless actual creation/redemption data is available.

Instead define:

```text
CapitalFlowProxyScore
```

Potential inputs:

- ETF volume vs 20d average
- dollar volume trend
- price-volume confirmation
- accumulation/distribution signals
- AUM changes where available
- shares outstanding changes where available
- options positioning
- relative turnover
- constituent volume participation

## 30. Volume Expansion

Calculate:

```text
RelativeVolume = CurrentVolume / Avg20Volume
```

Also:

```text
DollarVolume = Price * Volume
```

Track rolling change.

Rising price + rising dollar volume supports accumulation.

Falling price + rising volume supports distribution.

## 31. ETF Flow Data

If external ETF flow / creation-redemption data becomes available, integrate as a distinct evidence source.

Do not replace proxy scores.

Store:

```text
reported_net_flow
flow_source
flow_date
flow_confidence
```

## 32. Flow Score

Initial proxy formula:

```text
FlowScore =
0.25 * RelativeVolume
+ 0.20 * DollarVolumeTrend
+ 0.20 * PriceVolumeConfirmation
+ 0.15 * AccumulationDistribution
+ 0.10 * OptionsPositioning
+ 0.10 * ConstituentParticipation
```

Normalize 0–100.

## 33. Rotation Concept

Rotation is not merely Sector A falls and Sector B rises.

SRI SHALL look for coordinated evidence:

```text
relative strength changes
+
momentum acceleration
+
breadth improvement
+
diffusion expansion
+
flow improvement
+
leadership broadening
```

## 34. Rotation-In Score

Suggested initial formula:

```text
RotationInScore =
0.20 * RelativeStrengthScore
+ 0.15 * MomentumScore
+ 0.15 * MomentumAccelerationScore
+ 0.20 * SectorDiffusionIndex
+ 0.15 * FlowScore
+ 0.10 * TechnicalScore
+ 0.05 * OptionsScore
```

Weights SHALL be configuration-driven.

If OptionsScore unavailable, renormalize remaining components.

## 35. Rotation-Out Score

Suggested:

```text
RotationOutScore =
0.20 * RelativeWeakness
+ 0.15 * NegativeMomentum
+ 0.15 * NegativeAcceleration
+ 0.20 * BreadthDeterioration
+ 0.15 * DistributionScore
+ 0.10 * BearishTechnicalScore
+ 0.05 * BearishOptionsScore
```

## 36. Rotation Differential

For each segment:

```text
RotationDifferential = RotationInScore - RotationOutScore
```

Range after normalization:

```text
-100 to +100
```

Interpretation:

```text
+60 to +100  Strong Rotation In
+25 to +59   Rotation In
-24 to +24   Neutral / Transition
-25 to -59   Rotation Out
-60 to -100  Strong Rotation Out
```

## 37. Segment State Model

Every segment SHALL have a state.

Recommended states:

```text
ACCUMULATION
LEADERSHIP
ROTATION_IN
RECOVERY
NEUTRAL
WEAKENING
DISTRIBUTION
ROTATION_OUT
CAPITULATION
```

## 38. State Rules

### ACCUMULATION

Potential rule:

```text
FlowScore >= 65
AND SDI >= 55
AND MomentumAcceleration > 0
AND RelativeStrengthScore < leadership threshold
```

Meaning: capital appears to be entering before full leadership.

### LEADERSHIP

```text
RelativeStrengthScore >= 80
AND SDI >= 70
AND TechnicalScore >= 70
```

### ROTATION_IN

```text
RotationInScore >= 70
AND RotationDifferential >= +25
AND MomentumAcceleration > 0
```

### RECOVERY

```text
prior_state in {WEAKENING, DISTRIBUTION, ROTATION_OUT, CAPITULATION}
AND MomentumAcceleration > 0
AND SDI improving
```

### WEAKENING

```text
RelativeStrength declining
AND SDI declining
AND MomentumAcceleration < 0
```

### DISTRIBUTION

```text
DistributionScore >= threshold
AND price trend may still be positive
AND breadth deterioration present
```

This state is important because distribution may precede visible weakness.

### ROTATION_OUT

```text
RotationOutScore >= 70
AND RotationDifferential <= -25
```

### CAPITULATION

```text
SDI <= 20
AND Technical bearish >= high threshold
AND relative strength extremely weak
```

## 39. State Transition Constraints

Avoid excessive state flipping.

Use:

```text
minimum persistence
hysteresis
confirmation thresholds
```

Example:

```text
new state must persist 2 daily evaluations
OR exceed strong threshold immediately
```

## 40. Cross-Sector Ranking

Every daily run SHALL rank monitored segments by:

```text
RotationInScore
RelativeStrengthScore
SectorDiffusionIndex
MomentumAcceleration
FlowScore
CompositeSectorScore
```

Also calculate rank change:

```text
today rank
5d ago rank
20d ago rank
```

## 41. Composite Sector Score

Recommended:

```text
CompositeSectorScore =
0.20 * RelativeStrengthScore
+ 0.15 * TechnicalScore
+ 0.15 * MomentumScore
+ 0.15 * SectorDiffusionIndex
+ 0.15 * FlowScore
+ 0.10 * MomentumAcceleration
+ 0.10 * OptionsScore
```

This is a current-state score and is distinct from rotation score.

## 42. Rotation Velocity

Track score change:

```text
RotationVelocity_5D = RotationInScore_today - RotationInScore_5d_ago
```

Also 20d.

This is useful to identify segments rising rapidly through rankings.

## 43. Rotation Acceleration

```text
RotationAcceleration = Velocity_recent - Velocity_prior
```

This can help identify emerging rotation before absolute ranking becomes high.

## 44. Rotation Matrix

Create pairwise rotation views.

Example:

```text
Software       ↑ strengthening
Semiconductors ↑ leadership
Utilities      ↓ weakening
Energy         ↓ rotation out
Healthcare     → neutral
```

A richer model MAY calculate relative score spread between every segment pair.

## 45. Pairwise Relative Rotation

For segment A vs B:

```text
PairwiseRotation(A,B) = CompositeSectorScore_A - CompositeSectorScore_B
```

Track delta.

Example:

```text
Software vs Utilities:
spread today = +38
spread 20d ago = -5
change = +43
```

This is evidence of relative rotation.

## 46. Rotation Network

Future visualization MAY render:

```text
nodes = segments
node size = strength
node state = rotation state
edges = relative rotation change
```

This is a UI concern. Core engine only publishes deterministic data.

## 47. Broad Market Regime Context

SRI SHOULD consume Market Regime Engine state.

Possible context:

```text
risk_on
risk_off
high_volatility
low_volatility
growth_leadership
value_leadership
small_cap_leadership
defensive_rotation
```

Regime context SHALL NOT override sector evidence. It is contextual metadata.

## 48. Macro Sensitivity

Future versions MAY attach macro sensitivities to segments.

Examples:

```text
Utilities → rates sensitive
Banks → yield curve sensitive
Energy → oil sensitive
Semiconductors → growth / capex sensitive
REITs → rates sensitive
```

These mappings MUST be curated and versioned.

## 49. Asset-Level Context Output

SRI SHALL expose segment context for individual assets.

Example:

```json
{
  "symbol": "NET",
  "segment": "Software",
  "segment_etf": "IGV",
  "segment_state": "ROTATION_IN",
  "segment_score": 84.2,
  "relative_strength_rank": 3,
  "sector_diffusion": 78.4,
  "flow_score": 73.1,
  "context": "supportive"
}
```

## 50. Integration with VC

VC remains strategic. SRI may provide a contextual modifier.

Example:

```text
VC = 88
SRI = 91
Interpretation: strong strategic value + supportive segment rotation
```

SRI SHALL NOT directly mutate VC in v1. Instead expose `segment_context = supportive`.

## 51. Integration with DOSM

Example:

```text
DOSM = 82
segment_state = RECOVERY
sector_diffusion rising
```

This may strengthen the interpretation that distress is segment-supported rather than isolated.

Again: context only in v1.

## 52. Integration with EEOM

EEOM MAY consume:

```text
segment_state
segment_score
rotation_in_score
rotation_velocity
sector_diffusion
```

Example:

```text
Asset approaching earnings
+
technical bullish
+
options bullish
+
segment rotating in
```

This becomes stronger event context.

## 53. Integration with SAF

Every SRI signal SHALL be eligible for Signal Assurance Framework validation.

Example assertions:

```text
SEGMENT_ROTATION_IN
SEGMENT_LEADERSHIP
SEGMENT_ROTATION_OUT
SEGMENT_RECOVERY
SEGMENT_DISTRIBUTION
```

## 54. SRI Assertion Examples

### Rotation In

Expected materialization:

```text
segment ETF outperforms SPY by >= 3% within 20 trading days
```

### Leadership

```text
segment remains top-quartile relative strength for >= 10 of next 20 trading sessions
```

### Rotation Out

```text
segment underperforms SPY by >= 3% within 20 trading days
```

### Recovery

```text
relative strength improves by defined percentile within 30 trading days
```

Validation contracts MUST be versioned.

## 55. Canonical Segment Snapshot

```json
{
  "segment_id": "segment_software",
  "name": "Software",
  "segment_type": "industry",
  "primary_etf": "IGV",
  "as_of": "2026-08-07T20:00:00Z",
  "state": "ROTATION_IN",
  "composite_sector_score": 84.2,
  "rotation_in_score": 88.5,
  "rotation_out_score": 24.1,
  "rotation_differential": 64.4,
  "relative_strength_score": 86.1,
  "technical_score": 81.4,
  "momentum_score": 79.3,
  "momentum_acceleration": 72.0,
  "sector_diffusion_index": 78.4,
  "flow_score": 73.1,
  "options_score": 69.0,
  "rank": 2,
  "rank_change_5d": 4,
  "rotation_velocity_5d": 12.8,
  "evidence_quality": 0.94
}
```

## 56. Evidence Quality

SRI SHOULD output an evidence-quality score.

Potential factors:

```text
ETF liquidity
holdings freshness
constituent coverage
technical data completeness
options liquidity
price freshness
secondary ETF agreement
```

Range: 0.00–1.00.

## 57. Data Quality Flags

Support:

```text
MISSING_HOLDINGS
STALE_HOLDINGS
LOW_OPTIONS_LIQUIDITY
LOW_ETF_LIQUIDITY
INCOMPLETE_CONSTITUENT_COVERAGE
BENCHMARK_MISSING
SECONDARY_ETF_DIVERGENCE
PRICE_STALE
```

Scores SHOULD degrade gracefully rather than fail entirely.

## 58. ETF Holdings Snapshot

Canonical:

```json
{
  "etf": "IGV",
  "effective_date": "2026-08-07",
  "holdings": [
    {
      "asset_id": "asset_...",
      "symbol": "MSFT",
      "weight": 0.081
    }
  ],
  "source": "provider",
  "snapshot_version": "2026-08-07"
}
```

Holdings are point-in-time data.

Historical evaluation MUST use historical holdings snapshots when available.

## 59. Point-in-Time Correctness

SRI historical analysis SHALL NOT use future ETF holdings.

Store:

```text
holdings effective date
ingestion time
source
version
```

When replaying August 2025, use holdings known in August 2025. This is essential for SAF validation.

## 60. Persistence Tables

Recommended:

```text
sri_segments
sri_etf_registry
sri_etf_holdings
sri_segment_snapshots
sri_segment_constituent_metrics
sri_segment_state_events
sri_pairwise_rotation
```

## 61. sri_segments

```sql
CREATE TABLE sri_segments (
    segment_id UUID PRIMARY KEY,
    segment_key VARCHAR(128) UNIQUE NOT NULL,
    name VARCHAR(128) NOT NULL,
    segment_type VARCHAR(32) NOT NULL,
    parent_segment_id UUID,
    primary_etf_symbol VARCHAR(32),
    active BOOLEAN NOT NULL DEFAULT TRUE,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## 62. sri_etf_registry

```sql
CREATE TABLE sri_etf_registry (
    etf_symbol VARCHAR(32) PRIMARY KEY,
    segment_id UUID NOT NULL,
    role VARCHAR(32) NOT NULL,
    benchmark_priority INTEGER NOT NULL,
    liquidity_tier VARCHAR(32),
    options_liquidity VARCHAR(32),
    holdings_supported BOOLEAN NOT NULL DEFAULT FALSE,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    config JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## 63. sri_etf_holdings

```sql
CREATE TABLE sri_etf_holdings (
    snapshot_id UUID NOT NULL,
    etf_symbol VARCHAR(32) NOT NULL,
    asset_id UUID NOT NULL,
    symbol VARCHAR(32) NOT NULL,
    weight DOUBLE PRECISION NOT NULL,
    effective_date DATE NOT NULL,
    source VARCHAR(64),
    ingested_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY(snapshot_id, asset_id)
);
```

## 64. sri_segment_snapshots

```sql
CREATE TABLE sri_segment_snapshots (
    segment_id UUID NOT NULL,
    as_of TIMESTAMPTZ NOT NULL,
    state VARCHAR(32) NOT NULL,
    composite_score DOUBLE PRECISION,
    rotation_in_score DOUBLE PRECISION,
    rotation_out_score DOUBLE PRECISION,
    rotation_differential DOUBLE PRECISION,
    relative_strength_score DOUBLE PRECISION,
    technical_score DOUBLE PRECISION,
    momentum_score DOUBLE PRECISION,
    momentum_acceleration DOUBLE PRECISION,
    diffusion_index DOUBLE PRECISION,
    flow_score DOUBLE PRECISION,
    options_score DOUBLE PRECISION,
    rank INTEGER,
    rank_change_5d INTEGER,
    rotation_velocity_5d DOUBLE PRECISION,
    evidence_quality DOUBLE PRECISION,
    algorithm_version VARCHAR(32) NOT NULL,
    PRIMARY KEY(segment_id, as_of, algorithm_version)
);
```

Convert to Timescale hypertable if appropriate.

## 65. Segment State Events

```sql
CREATE TABLE sri_segment_state_events (
    event_id UUID PRIMARY KEY,
    segment_id UUID NOT NULL,
    previous_state VARCHAR(32),
    new_state VARCHAR(32) NOT NULL,
    reason_codes JSONB,
    score_snapshot JSONB,
    occurred_at TIMESTAMPTZ NOT NULL,
    algorithm_version VARCHAR(32) NOT NULL
);
```

## 66. Pairwise Rotation Table

```sql
CREATE TABLE sri_pairwise_rotation (
    as_of TIMESTAMPTZ NOT NULL,
    segment_a UUID NOT NULL,
    segment_b UUID NOT NULL,
    score_spread DOUBLE PRECISION NOT NULL,
    spread_change_5d DOUBLE PRECISION,
    spread_change_20d DOUBLE PRECISION,
    algorithm_version VARCHAR(32) NOT NULL,
    PRIMARY KEY(as_of, segment_a, segment_b, algorithm_version)
);
```

## 67. Idempotency

Daily SRI snapshot key:

```text
segment_id + market_session_date + algorithm_version
```

Holding snapshot key:

```text
etf_symbol + effective_date + source + holdings_hash
```

Repeated ingestion MUST NOT duplicate state.

## 68. Daily Evaluation Pipeline

```text
Market Close
    ↓
Finalize ETF prices
    ↓
Finalize constituent prices
    ↓
Load Technical Engine outputs
    ↓
Load Options Engine outputs
    ↓
Resolve latest valid holdings
    ↓
Calculate relative strength
    ↓
Calculate momentum
    ↓
Calculate acceleration
    ↓
Calculate breadth
    ↓
Calculate diffusion
    ↓
Calculate flow proxy
    ↓
Calculate rotation scores
    ↓
Calculate composite score
    ↓
Determine segment state
    ↓
Rank segments
    ↓
Calculate pairwise rotation
    ↓
Persist snapshots
    ↓
Publish state-change signals
    ↓
Register SAF assertions if applicable
```

## 69. Weekly Evaluation Pipeline

Weekly report SHALL summarize:

```text
top rotation-in segments
top leadership segments
new accumulation states
new recovery states
largest rank improvements
largest rank deterioration
new distribution states
rotation-out segments
breadth expansion
breadth contraction
```

## 70. Internal Events

Publish:

```text
sri.segment.rotation_in
sri.segment.rotation_out
sri.segment.leadership
sri.segment.accumulation
sri.segment.recovery
sri.segment.distribution
sri.segment.capitulation
sri.segment.rank_changed
sri.segment.diffusion_changed
```

## 71. Example Signal Event

```json
{
  "event_type": "sri.segment.rotation_in",
  "segment_id": "segment_software",
  "segment": "Software",
  "primary_etf": "IGV",
  "state": "ROTATION_IN",
  "rotation_in_score": 88.5,
  "rotation_differential": 64.4,
  "sector_diffusion_index": 78.4,
  "relative_strength_score": 86.1,
  "occurred_at": "2026-08-07T20:10:00Z",
  "algorithm_version": "sri-1.0"
}
```

## 72. REST APIs

### List segments

```http
GET /v1/marketops/sectors
```

### Current snapshot

```http
GET /v1/marketops/sectors/{segment_id}
```

### Rankings

```http
GET /v1/marketops/sectors/rankings
```

Filters:

```text
segment_type
state
top
as_of
```

### History

```http
GET /v1/marketops/sectors/{segment_id}/history
```

### Rotation matrix

```http
GET /v1/marketops/sectors/rotation-matrix
```

### Asset context

```http
GET /v1/marketops/assets/{symbol}/sector-context
```

## 73. Rankings Response

```json
{
  "as_of": "2026-08-07T20:00:00Z",
  "segments": [
    {
      "rank": 1,
      "segment": "Semiconductors",
      "etf": "SMH",
      "state": "LEADERSHIP",
      "score": 91.3,
      "rotation_in_score": 89.2,
      "diffusion": 82.1
    },
    {
      "rank": 2,
      "segment": "Software",
      "etf": "IGV",
      "state": "ROTATION_IN",
      "score": 84.2,
      "rotation_in_score": 88.5,
      "diffusion": 78.4
    }
  ]
}
```

## 74. Explainability

Every segment score SHALL expose contributing factors.

Example:

```json
{
  "segment": "Software",
  "state": "ROTATION_IN",
  "drivers": [
    {"factor": "relative_strength", "direction": "positive", "contribution": 17.2},
    {"factor": "diffusion", "direction": "positive", "contribution": 15.7},
    {"factor": "momentum_acceleration", "direction": "positive", "contribution": 11.4}
  ],
  "risks": [
    {"factor": "leadership_concentration", "direction": "negative", "contribution": -4.1}
  ]
}
```

## 75. Confidence / Evidence

SRI SHALL NOT use the term analyst conviction.

Recommended field:

```text
evidence_quality
```

Optional:

```text
signal_confidence
```

This represents data and signal agreement, not human conviction.

## 76. Cross-ETF Agreement

For segments with secondary ETFs:

```text
PrimaryETFScore
SecondaryETFScore
```

Agreement metric:

```text
Agreement = 1 - abs(primary_normalized - secondary_normalized)
```

Segment evidence quality increases with agreement.

## 77. Sector Diffusion Change

Track:

```text
SDI_today
SDI_5d_ago
SDI_20d_ago
```

Calculate:

```text
DiffusionVelocity_5D
DiffusionVelocity_20D
```

This can identify breadth improvement before ETF breakout.

## 78. Early Rotation Candidate

Introduce state flag:

```text
EARLY_ROTATION_CANDIDATE
```

Criteria example:

```text
MomentumAcceleration >= 70
AND DiffusionVelocity_5D strongly positive
AND RelativeStrengthScore between 45 and 70
AND FlowScore improving
```

This is not yet ROTATION_IN. It is an early-warning state.

## 79. Late Leadership Risk

Introduce flag:

```text
LATE_LEADERSHIP_RISK
```

Possible pattern:

```text
RelativeStrength >= 85
BUT Diffusion falling
AND Leadership concentration rising
AND Momentum acceleration negative
```

This may precede distribution.

## 80. Rotation Transition Model

Useful progression:

```text
CAPITULATION
    ↓
RECOVERY
    ↓
EARLY_ROTATION_CANDIDATE
    ↓
ACCUMULATION
    ↓
ROTATION_IN
    ↓
LEADERSHIP
    ↓
LATE_LEADERSHIP_RISK
    ↓
WEAKENING
    ↓
DISTRIBUTION
    ↓
ROTATION_OUT
```

Not all segments follow every state. State transitions MUST remain rule-based.

## 81. Rotation Opportunity vs State

Do not confuse CurrentState with RotationOpportunity.

A segment in LEADERSHIP may have lower incremental opportunity than a segment in EARLY_ROTATION_CANDIDATE.

Future versions MAY publish `rotation_opportunity_score` distinct from composite strength.

## 82. Suggested Rotation Opportunity Score

Optional v1:

```text
RotationOpportunity =
0.25 * MomentumAcceleration
+ 0.20 * DiffusionVelocity
+ 0.20 * RankImprovement
+ 0.15 * FlowImprovement
+ 0.10 * TechnicalImprovement
+ 0.10 * RelativeStrengthImprovement
```

This identifies emerging rather than established leadership.

## 83. MarketOps UI Requirements

### Sector Heatmap

Display:

```text
segment
state
composite score
rotation score
diffusion
rank change
```

### Rotation Table

Columns:

```text
Rank
Segment
ETF
State
Composite
Rotation In
Rotation Out
RS
Diffusion
Flow
Momentum Acceleration
Rank Change
```

### Segment Detail

Show:

- ETF price trend
- relative strength
- diffusion history
- breadth
- flow proxy
- constituent leaders
- constituent laggards
- technical state
- options state
- state-transition timeline

## 84. Rotation Matrix UI

Rows: segments.

Columns: relative comparison or pairwise spread.

Highlight strongest positive and negative changes.

## 85. Constituent Drill-Down

For a segment show:

```text
symbol
weight
technical score
relative strength
20d return
above 20DMA
above 50DMA
above 200DMA
leadership classification
```

This explains diffusion.

## 86. Performance Targets

Initial:

```text
< 100 monitored ETFs
< 5,000 constituent relationships
daily evaluation
< 5 minute compute target
```

Architecture SHALL scale beyond these limits.

## 87. Horizontal Scaling

Partition calculations by segment or ETF.

Avoid cross-worker mutable state.

Cross-sector ranking happens after individual segment calculations complete.

## 88. Caching

Cache:

- latest segment snapshot
- rankings
- ETF holdings
- constituent technical metrics

Do not cache canonical history as sole source of truth.

Redis MAY be used for latest-state acceleration.

## 89. Failure Handling

If technical score missing: mark unavailable and renormalize remaining score weights.

If options unavailable: zero options weight.

If constituent coverage is below threshold: degrade diffusion confidence.

If holdings stale: continue if within acceptable tolerance and emit `STALE_HOLDINGS`.

## 90. Data Completeness Threshold

Recommended:

```text
constituent_coverage >= 80%
```

for full-confidence breadth.

If 50–79%, calculate with warning.

If <50%, do not publish high-confidence diffusion score.

## 91. Algorithm Versioning

Every output SHALL preserve:

```text
sri_algorithm_version
technical_engine_version
options_engine_version
holdings_snapshot_version
```

Example:

```json
{
  "sri_version": "1.0",
  "technical_version": "3.1",
  "options_version": "2.0",
  "holdings_version": "2026-08-07"
}
```

## 92. Backtesting

SRI SHALL support historical replay.

Replay requires:

- historical ETF prices
- historical benchmark prices
- historical constituent prices
- historical holdings where available
- historical Technical Engine outputs
- historical Options Engine outputs if available

Use identical scoring code for historical and live runs.

## 93. SAF Validation

SRI output validation SHALL be delegated to SAF.

SAF should answer:

```text
Does ROTATION_IN precede excess returns?
How long does it take?
Does high diffusion improve reliability?
Does rising diffusion outperform high but falling diffusion?
Does flow confirmation improve materialization?
Does early rotation candidate outperform established leadership?
```

## 94. Core SAF Research Questions

Priority:

1. Does `ROTATION_IN` predict benchmark-relative outperformance?
2. What is median TTM?
3. Does diffusion improve signal quality?
4. Does broad participation outperform concentrated ETF leadership?
5. Does acceleration precede ranking improvement?
6. Does flow confirmation improve outcomes?
7. Does secondary ETF agreement improve reliability?
8. Which state transitions are most predictive?

## 95. Observability

Metrics:

```text
sri_segments_processed_total
sri_segments_failed_total
sri_daily_run_duration_seconds
sri_missing_holdings_total
sri_stale_holdings_total
sri_low_constituent_coverage_total
sri_state_changes_total
sri_rotation_in_total
sri_rotation_out_total
sri_rank_changes_total
```

## 96. Logging

Log context:

```text
segment_id
primary_etf
as_of
state_before
state_after
composite_score
rotation_in_score
rotation_out_score
diffusion
algorithm_version
```

## 97. Testing Strategy

### Unit Tests

Test:

- relative strength
- momentum
- acceleration
- breadth
- weighted breadth
- diffusion
- concentration
- flow score
- rotation in
- rotation out
- state transitions
- pairwise spreads
- score renormalization

### Integration Tests

Test:

- ETF registry
- holdings lookup
- technical integration
- options integration
- daily pipeline
- ranking
- event publication
- SAF assertion registration

### Replay Tests

Ensure same historical inputs yield identical snapshots.

## 98. Golden Fixtures

### Broad Rotation In

Expected:

```text
high relative strength
rising acceleration
broad breadth
high diffusion
positive flow
ROTATION_IN
```

### Narrow Leadership

Expected:

```text
ETF strong
median constituent weak
high concentration
moderate/low diffusion
LATE_LEADERSHIP_RISK or LEADERSHIP with warning
```

### Distribution

Expected:

```text
ETF still near highs
breadth falling
flow negative
acceleration negative
DISTRIBUTION
```

### Recovery

Expected:

```text
prior weak state
diffusion improves
technical scores improve
relative weakness narrows
RECOVERY
```

## 99. Acceptance Criteria

SRI v1 is complete when:

1. ETF registry exists.
2. Segment registry exists.
3. Primary ETFs are mapped.
4. Secondary ETF support exists.
5. Daily ETF snapshots are loaded.
6. Technical scores are consumed.
7. Options scores are consumed where available.
8. Relative strength is calculated.
9. Momentum is calculated.
10. Momentum acceleration is calculated.
11. Breadth is calculated.
12. Weighted breadth is calculated.
13. Sector Diffusion Index is calculated.
14. Leadership concentration is calculated.
15. Flow proxy is calculated.
16. RotationInScore is calculated.
17. RotationOutScore is calculated.
18. RotationDifferential is calculated.
19. CompositeSectorScore is calculated.
20. State classification is produced.
21. Rank is produced.
22. Rank change is produced.
23. Pairwise rotation is produced.
24. State-change events are published.
25. Asset-level sector context API works.
26. Historical snapshots are persisted.
27. Algorithm versions are preserved.
28. Point-in-time holdings are supported.
29. SAF validation hooks work.
30. Outputs are explainable.

## 100. Suggested Repository Layout

```text
marketops/
  services/
    sector-rotation-intelligence/
      cmd/
        sri-api/
        sri-worker/
      internal/
        registry/
        holdings/
        relative_strength/
        momentum/
        breadth/
        diffusion/
        leadership/
        flows/
        scoring/
        states/
        ranking/
        pairwise/
        integrations/
        events/
        persistence/
        api/
      configs/
      migrations/
      tests/
      fixtures/
      Dockerfile
      go.mod
      README.md
```

## 101. Configuration Example

```yaml
sri:
  version: "1.0"
  benchmarks:
    market: "SPY"
    growth: "QQQ"
    equal_weight: "RSP"
  weights:
    composite:
      relative_strength: 0.20
      technical: 0.15
      momentum: 0.15
      diffusion: 0.15
      flow: 0.15
      acceleration: 0.10
      options: 0.10
    rotation_in:
      relative_strength: 0.20
      momentum: 0.15
      acceleration: 0.15
      diffusion: 0.20
      flow: 0.15
      technical: 0.10
      options: 0.05
  breadth:
    minimum_constituent_coverage: 0.80
  state:
    persistence_days: 2
  holdings:
    max_staleness_days: 10
```

## 102. Implementation Sequence

### Phase 1

Build:

- ETF registry
- segment registry
- price integration
- technical integration
- relative strength
- momentum
- basic ranking

### Phase 2

Add:

- holdings ingestion
- constituent breadth
- weighted breadth
- Sector Diffusion Index
- leadership concentration

### Phase 3

Add:

- flow proxies
- options integration
- rotation scoring
- state model
- pairwise rotation

### Phase 4

Add:

- asset context
- UI APIs
- SAF validation
- backtesting
- early rotation candidate logic
- late leadership risk logic

## 103. Engineering Principle for Code Agent

Do not build SRI as an ETF dashboard.

Do not couple SRI to frontend state.

Do not put technical indicator code into SRI.

Do not put options calculations into SRI.

SRI is a deterministic market-structure orchestration engine.

It consumes reusable analytical outputs.

It produces:

```text
segment state
segment strength
segment breadth
segment diffusion
rotation state
rotation opportunity
asset context
```

## 104. Canonical Analytical Questions

SRI must ultimately answer:

```text
Which segments are leading?
Which segments are weakening?
Which segments are entering accumulation?
Which segments are distributing?
Where is breadth expanding?
Where is leadership becoming concentrated?
Which segments are improving fastest?
Which segments are falling fastest?
Is relative strength improving before headline price leadership?
Is there enough cross-signal evidence to classify rotation in?
Is there enough cross-signal evidence to classify rotation out?
Which individual assets have supportive or adverse segment context?
```

## 105. Final Architectural Principle

The ETF is a measurement instrument.

The segment is the analytical object.

SRI should convert ETF and constituent market data into evidence about market structure.

The intended flow is:

```text
ETF / Constituent Data
        ↓
Market Signals
        ↓
Breadth + Diffusion + Relative Strength + Flow
        ↓
Segment Evidence
        ↓
Rotation State
        ↓
Market Context
        ↓
Asset Opportunity Context
```

The most important differentiator is that SRI SHALL NOT equate ETF appreciation with healthy sector leadership.

A strong rotation signal requires evidence that strength is:

```text
relative
broad
accelerating
supported by participation
supported by flow proxies
and persistent enough to survive noise
```

That is the basis for Sector Rotation Intelligence.

---

# End of Specification
