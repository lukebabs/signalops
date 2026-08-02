# SignalOps MarketOps
# Deterministic VC and DOSM Algorithm Specification

**Version:** 3.0  
**Status:** Canonical implementation specification  
**Scope:** Algorithm only  
**Target system:** Existing SignalOps MarketOps subsystem  
**Market data sources:** Massive.com for canonical prices/options; FMP for VC/DOSM financial statements; Massive.com for market capitalization.  
**Primary objective:** Reproduce VC and DOSM locally with identical results for identical inputs

---

## 1. Purpose

This document defines the complete deterministic implementation of:

1. **VC — Valuation Composite**
2. **DOSM — Distressed Opportunity Scoring Model**

The existing MarketOps subsystem handles canonical prices, options, normalization, and historical market-data retrieval from Massive.com. VC/DOSM financial snapshots are acquired from FMP on the weekly post-close cadence, persisted with FMP provenance, and reused by the deterministic engine. The FMP runner enforces a 240-request daily safety ceiling against the 250-request plan allowance; the current TTM profile requires three FMP calls per refreshed symbol.

This document begins with a normalized equity snapshot and defines exactly how to:

- validate inputs,
- calculate raw valuation ratios,
- convert ratios into 0–10 metric scores,
- calculate peer adjustments,
- apply overvaluation penalties,
- calculate the VC score,
- calculate the fundamental score,
- calculate technical adjustments,
- calculate the DOSM score,
- derive VC and DOSM fair-value anchors,
- classify the resulting signals,
- handle missing or invalid data,
- persist an explainable calculation trace,
- replay historical evaluations deterministically.

All calculations are deterministic. No LLM, probabilistic model, or discretionary override may alter the numeric output.

---

# 2. Terminology

## 2.1 VC

**Valuation Composite** is a valuation-only score based on:

- Price-to-Sales,
- GAAP Price-to-Earnings,
- Enterprise Value-to-EBITDA,
- peer-relative valuation,
- an explicit high-valuation / low-growth penalty.

VC intentionally excludes momentum and broader business quality.

## 2.2 DOSM

**Distressed Opportunity Scoring Model** combines:

- VC valuation score,
- fundamental quality,
- technical condition,
- peer-relative valuation,
- high-valuation / low-growth penalty.

DOSM identifies companies whose valuation and operating condition may represent an opportunity, while penalizing weak profitability, poor balance-sheet quality, and technically deteriorating price action.

## 2.3 Evaluation date

The trading date for which the model is executed.

## 2.4 Canonical price

The split-adjusted regular-session closing price for the evaluation date.

## 2.5 GAAP-only rule

Only GAAP earnings and financial-statement values may be used in VC and DOSM.

The following are prohibited for scoring:

- adjusted EPS,
- non-GAAP EPS,
- adjusted EBITDA,
- management-defined free cash flow,
- pro forma earnings,
- forward consensus earnings,
- analyst price targets.

---

# 3. Required Input Contract

The engine SHALL consume one normalized `EquityEvaluationInput` object.

```json
{
  "ticker": "TGT",
  "evaluation_date": "2025-12-22",
  "currency": "USD",
  "price": {
    "regular_session_close": 101.25,
    "timestamp_utc": "2025-12-22T21:00:00Z",
    "adjusted_for_splits": true,
    "adjusted_for_dividends": false,
    "source": "massive"
  },
  "market": {
    "shares_outstanding": 455000000,
    "market_cap": 46068750000,
    "enterprise_value": 62000000000
  },
  "financials": {
    "revenue_ttm": 106500000000,
    "net_income_gaap_ttm": 4200000000,
    "ebitda_gaap_ttm": 7500000000,
    "operating_income_ttm": 5600000000,
    "operating_cash_flow_ttm": 7200000000,
    "capital_expenditures_ttm": 3500000000,
    "free_cash_flow_ttm": 3700000000,
    "total_debt": 18500000000,
    "cash_and_equivalents": 3500000000,
    "shareholders_equity": 13500000000,
    "invested_capital": 28500000000
  },
  "growth": {
    "revenue_3y_ago": 109100000000,
    "revenue_cagr_3y": -0.008
  },
  "technicals": {
    "rsi_14": 33.5,
    "sma_50": 105.75,
    "sma_200": 118.60
  },
  "classification": {
    "sector": "Consumer Defensive",
    "industry": "Discount Stores",
    "peer_group_id": "US_DISCOUNT_RETAIL"
  },
  "peer_snapshot": {
    "peer_count": 5,
    "ps_median": 0.85,
    "pe_median": 18.0,
    "ev_ebitda_median": 10.5
  }
}
```

---

# 4. Input Validation

The engine MUST validate all inputs before scoring.

## 4.1 Price validation

The canonical price is valid only when:

```text
regular_session_close > 0
adjusted_for_splits = true
timestamp date = evaluation_date
source = massive or normalized MarketOps store
```

Reject:

- intraday price,
- pre-market price,
- after-hours price,
- previous-close fallback with a mismatched date,
- unadjusted price after a split,
- price without a timestamp,
- price from a future date relative to the financial snapshot.

## 4.2 Financial data age

Financial data must be the latest filing available as of the evaluation date.

The model MUST NOT use a filing published after the evaluation date during historical replay.

## 4.3 Peer data validation

Peer medians are usable when:

```text
peer_count >= 3
```

If fewer than three valid peers remain after filtering, peer adjustment is set to `0.0`.

## 4.4 Numeric validation

The following must be finite numbers where supplied:

- revenue,
- market capitalization,
- enterprise value,
- net income,
- EBITDA,
- free cash flow,
- debt,
- equity,
- invested capital,
- RSI,
- moving averages.

NaN and infinity are prohibited.

---

# 5. Derived Raw Metrics

The engine SHALL calculate ratios locally from normalized data.

## 5.1 Price-to-Sales

```text
P/S = Market Capitalization / Revenue TTM
```

Requirements:

```text
Market Capitalization > 0
Revenue TTM > 0
```

## 5.2 GAAP Price-to-Earnings

```text
P/E = Market Capitalization / GAAP Net Income TTM
```

P/E is valid only when:

```text
GAAP Net Income TTM > 0
```

If GAAP net income is zero or negative, P/E is classified as `NOT_MEANINGFUL`.

## 5.3 EV/EBITDA

```text
EV/EBITDA = Enterprise Value / GAAP EBITDA TTM
```

EV/EBITDA is valid only when:

```text
Enterprise Value > 0
GAAP EBITDA TTM > 0
```

If GAAP EBITDA is zero or negative, EV/EBITDA is classified as `NOT_MEANINGFUL`.

## 5.4 Market-cap-to-revenue

```text
MarketCapToRevenue = Market Capitalization / Revenue TTM
```

This is mathematically identical to P/S but retained as a named field for penalty evaluation.

## 5.5 Revenue CAGR

When not supplied by the normalized data layer:

```text
Revenue CAGR 3Y =
(Revenue TTM / Revenue 3Y Ago)^(1/3) - 1
```

Both revenue values must be positive.

## 5.6 Operating margin

```text
Operating Margin = Operating Income TTM / Revenue TTM
```

## 5.7 Net margin

```text
Net Margin = GAAP Net Income TTM / Revenue TTM
```

## 5.8 Free-cash-flow margin

```text
FCF Margin = Free Cash Flow TTM / Revenue TTM
```

Where:

```text
Free Cash Flow TTM =
Operating Cash Flow TTM - Capital Expenditures TTM
```

If a normalized GAAP-derived free-cash-flow field already exists, the engine SHALL verify that it matches the above within a configurable tolerance of 1%.

## 5.9 Debt-to-equity

```text
Debt-to-Equity = Total Debt / Shareholders' Equity
```

If equity is zero or negative, debt-to-equity is `NOT_MEANINGFUL` and receives the distressed score defined later.

## 5.10 Return on invested capital

```text
ROIC = NOPAT / Invested Capital
```

Where:

```text
NOPAT = Operating Income TTM × (1 - Effective Tax Rate)
```

Default effective tax rate when unavailable:

```text
25%
```

The tax-rate fallback must be recorded in the explanation trace.

---

# 6. Score Scale

All component scores use a continuous or table-interpolated scale from `0.0` to `10.0`.

Interpretation:

| Score | Meaning |
|---:|---|
| 0–2 | Severely unattractive |
| 2–4 | Weak |
| 4–6 | Neutral |
| 6–8 | Attractive |
| 8–10 | Exceptional |

All scores SHALL be rounded to four decimal places internally and two decimal places for presentation.

Final VC and DOSM values SHALL be clamped to `[0.0, 10.0]`.

---

# 7. Absolute Valuation Score Tables

These are the default cross-sector tables.

Sector-specific overrides MAY be added later through configuration, but the default algorithm must remain available and versioned.

Interpolation is linear between adjacent breakpoints.

For lower-is-better metrics, values below the first breakpoint receive `10.0`; values above the final breakpoint receive `0.0`.

## 7.1 P/S score

| P/S | Score |
|---:|---:|
| 0.50 | 10.0 |
| 1.00 | 9.0 |
| 2.00 | 7.5 |
| 3.00 | 6.0 |
| 5.00 | 4.0 |
| 8.00 | 2.0 |
| 12.00 | 1.0 |
| 15.00 | 0.5 |
| 20.00 | 0.0 |

Example:

```text
P/S = 2.50
Between 2.00 (7.5) and 3.00 (6.0)
Score = 6.75
```

## 7.2 GAAP P/E score

| GAAP P/E | Score |
|---:|---:|
| 5 | 10.0 |
| 8 | 9.0 |
| 12 | 8.0 |
| 16 | 7.0 |
| 20 | 6.0 |
| 25 | 5.0 |
| 30 | 4.0 |
| 40 | 2.5 |
| 60 | 1.0 |
| 80 | 0.0 |

Special rule for unprofitable companies:

```text
If GAAP Net Income TTM <= 0:
PE_score = 0.0
```

This is deliberate. An unprofitable company must not receive a neutral P/E score simply because P/E is unavailable.

## 7.3 EV/EBITDA score

| EV/EBITDA | Score |
|---:|---:|
| 3 | 10.0 |
| 5 | 9.0 |
| 7 | 8.0 |
| 9 | 7.0 |
| 12 | 5.5 |
| 15 | 4.0 |
| 20 | 2.5 |
| 30 | 1.0 |
| 40 | 0.0 |

Special rule:

```text
If GAAP EBITDA TTM <= 0:
EVEBITDA_score = 0.0
```

---

# 8. Linear Interpolation Function

The scoring engine SHALL implement a reusable interpolation function.

```python
def interpolate_descending(value: float, points: list[tuple[float, float]]) -> float:
    '''
    points: sorted ascending by metric value.
    score is expected to decline as value rises.
    '''
    if value <= points[0][0]:
        return points[0][1]
    if value >= points[-1][0]:
        return points[-1][1]

    for i in range(len(points) - 1):
        x1, y1 = points[i]
        x2, y2 = points[i + 1]
        if x1 <= value <= x2:
            ratio = (value - x1) / (x2 - x1)
            return y1 + ratio * (y2 - y1)

    raise ValueError("Interpolation failed")
```

The implementation must not use nearest-neighbor bucket assignment because that creates discontinuities.

---

# 9. Peer-relative Valuation Adjustment

The peer adjustment is based on the company's three valuation multiples relative to peer medians.

## 9.1 Relative ratio

For each valid metric:

```text
Relative Ratio = Company Multiple / Peer Median Multiple
```

## 9.2 Metric-level peer adjustment

| Relative ratio | Adjustment |
|---:|---:|
| <= 0.60 | +0.20 |
| 0.60–0.80 | linearly +0.20 to +0.10 |
| 0.80–0.95 | linearly +0.10 to 0.00 |
| 0.95–1.05 | 0.00 |
| 1.05–1.20 | linearly 0.00 to -0.10 |
| 1.20–1.50 | linearly -0.10 to -0.20 |
| >= 1.50 | -0.20 |

## 9.3 Aggregate peer adjustment

```text
Peer Adjustment =
mean(valid metric-level peer adjustments)
```

Then clamp:

```text
-0.50 <= Peer Adjustment <= +0.50
```

Because each metric contributes at most ±0.20, the expected ordinary range is approximately ±0.20. The wider clamp allows future sector overrides without changing the contract.

## 9.4 Invalid metrics

If company P/E or EV/EBITDA is not meaningful, that metric is excluded from peer comparison.

P/S remains valid as long as revenue is positive.

If fewer than two valid peer-relative metric adjustments remain:

```text
Peer Adjustment = 0.0
```

## 9.5 No double counting in DOSM

The peer adjustment is applied once inside VC.

DOSM SHALL consume the final VC score and SHALL NOT add peer adjustment again.

This corrects the earlier informal formula that risked double counting peer valuation.

---

# 10. High-Valuation / Low-Growth Penalty

Apply when both conditions are true:

```text
MarketCapToRevenue > 15.0
Revenue CAGR 3Y < 30%
```

## 10.1 Penalty magnitude

The penalty is tiered:

| MarketCap / Revenue | Revenue CAGR | Penalty |
|---:|---:|---:|
| >15 and <=20 | <30% | 0.50 |
| >20 and <=30 | <30% | 0.75 |
| >30 | <30% | 1.00 |

Additional profitability penalty:

```text
If GAAP Net Income TTM <= 0 and the high-valuation penalty applies:
add 0.50
```

Maximum combined high-valuation penalty:

```text
1.50
```

## 10.2 Penalty sign convention

Persist penalty as a positive magnitude:

```text
growth_valuation_penalty = 0.50
```

Subtract it in formulas.

---

# 11. VC Algorithm

## 11.1 Base formula

```text
VC Base =
0.40 × P/S Score
+ 0.30 × GAAP P/E Score
+ 0.30 × EV/EBITDA Score
```

## 11.2 Missing metric policy

The weights SHALL NOT be silently redistributed when P/E or EV/EBITDA is not meaningful.

Instead:

```text
P/E Score = 0.0 when GAAP net income <= 0
EV/EBITDA Score = 0.0 when GAAP EBITDA <= 0
```

This ensures lack of profitability materially lowers VC.

If a metric is missing because of data failure rather than economic meaning:

```text
status = INCOMPLETE
model confidence is reduced
metric weight is re-normalized across available valid metrics
```

Economic invalidity and data incompleteness are distinct states.

## 11.3 Final formula

```text
VC Raw =
VC Base
+ Peer Adjustment
- Growth/Valuation Penalty
```

```text
VC Score = clamp(VC Raw, 0, 10)
```

## 11.4 VC fair-value anchor

```text
VC Fair Value =
Canonical Price × exp(0.1 × (VC Score - 5))
```

## 11.5 VC upside/downside

```text
VC Upside =
(VC Fair Value / Canonical Price) - 1
```

## 11.6 Interpretation

Because fair value is anchored to the current price, VC fair value is a relative valuation normalization, not an independent intrinsic value.

---

# 12. Fundamental Score for DOSM

The DOSM fundamental score comprises six equally weighted components:

1. Revenue growth
2. Operating margin
3. GAAP profitability
4. Free cash flow
5. Debt profile
6. Capital efficiency

```text
Fundamental Score =
mean(
Revenue Growth Score,
Operating Margin Score,
GAAP Profitability Score,
FCF Score,
Debt Profile Score,
Capital Efficiency Score
)
```

Each component has weight:

```text
1 / 6 = 16.6667%
```

---

# 13. Revenue Growth Score

Use 3-year revenue CAGR.

| Revenue CAGR | Score |
|---:|---:|
| <= -15% | 0.0 |
| -10% | 1.5 |
| -5% | 3.0 |
| 0% | 5.0 |
| 5% | 6.0 |
| 10% | 7.0 |
| 15% | 8.0 |
| 25% | 9.0 |
| >= 40% | 10.0 |

Use linear interpolation.

---

# 14. Operating Margin Score

Default cross-sector table:

| Operating Margin | Score |
|---:|---:|
| <= -15% | 0.0 |
| -5% | 1.5 |
| 0% | 3.0 |
| 5% | 5.0 |
| 10% | 6.5 |
| 15% | 7.5 |
| 20% | 8.5 |
| 30% | 9.5 |
| >= 40% | 10.0 |

Use linear interpolation.

Sector-specific margin tables may be configured later, but the score-table version must be persisted.

---

# 15. GAAP Profitability Score

This score captures both positive net income and net margin.

## 15.1 If GAAP net income is negative

| Net Margin | Score |
|---:|---:|
| <= -30% | 0.0 |
| -20% | 0.5 |
| -10% | 1.5 |
| -5% | 2.5 |
| just below 0% | 3.5 |

## 15.2 If GAAP net income is positive

| Net Margin | Score |
|---:|---:|
| 0% | 5.0 |
| 3% | 6.0 |
| 5% | 6.5 |
| 8% | 7.5 |
| 12% | 8.5 |
| 20% | 9.5 |
| >= 30% | 10.0 |

A positive but extremely thin margin is not treated as exceptional profitability.

---

# 16. Free-Cash-Flow Score

Use FCF margin and FCF sign.

## 16.1 Negative FCF

| FCF Margin | Score |
|---:|---:|
| <= -20% | 0.0 |
| -10% | 1.0 |
| -5% | 2.0 |
| just below 0% | 3.0 |

## 16.2 Positive FCF

| FCF Margin | Score |
|---:|---:|
| 0% | 5.0 |
| 3% | 6.0 |
| 5% | 6.5 |
| 8% | 7.5 |
| 12% | 8.5 |
| 20% | 9.5 |
| >= 30% | 10.0 |

---

# 17. Debt Profile Score

Use debt-to-equity and net-debt-to-EBITDA where available.

## 17.1 Debt-to-equity score

| Debt / Equity | Score |
|---:|---:|
| 0.0 | 10.0 |
| 0.25 | 9.0 |
| 0.50 | 8.0 |
| 1.00 | 6.5 |
| 1.50 | 5.0 |
| 2.00 | 3.5 |
| 3.00 | 2.0 |
| >= 5.00 | 0.0 |

If equity <= 0:

```text
Debt-to-Equity Score = 0.0
```

## 17.2 Net-debt-to-EBITDA score

```text
Net Debt = Total Debt - Cash and Equivalents
Net Debt / EBITDA = Net Debt / GAAP EBITDA
```

| Net Debt / EBITDA | Score |
|---:|---:|
| <= 0 | 10.0 |
| 1 | 8.5 |
| 2 | 7.0 |
| 3 | 5.5 |
| 4 | 4.0 |
| 5 | 2.5 |
| >= 7 | 0.0 |

If EBITDA <= 0 and net debt > 0:

```text
Net-Debt-to-EBITDA Score = 0.0
```

## 17.3 Combined debt score

When both measures are valid:

```text
Debt Profile Score =
0.50 × Debt-to-Equity Score
+ 0.50 × Net-Debt-to-EBITDA Score
```

When only one measure is valid, use the valid score and reduce model confidence.

---

# 18. Capital Efficiency Score

Use ROIC.

| ROIC | Score |
|---:|---:|
| <= -10% | 0.0 |
| 0% | 3.0 |
| 5% | 5.0 |
| 8% | 6.5 |
| 10% | 7.0 |
| 15% | 8.0 |
| 20% | 9.0 |
| >= 30% | 10.0 |

Use linear interpolation.

---

# 19. Technical Adjustment

Technical adjustment is additive to DOSM and intentionally bounded.

```text
Technical Adjustment =
RSI Adjustment + Trend Adjustment
```

Clamp:

```text
-1.0 <= Technical Adjustment <= +1.0
```

## 19.1 RSI adjustment

| RSI(14) | Adjustment |
|---:|---:|
| < 30 | +0.50 |
| 30–70 | 0.00 |
| > 70 | -0.50 |

No interpolation is required.

## 19.2 Moving-average trend adjustment

| Condition | Adjustment |
|---|---:|
| Price > SMA50 and Price > SMA200 | +0.50 |
| Price < SMA50 and Price < SMA200 | -0.50 |
| Mixed | 0.00 |

Equality is treated as mixed.

## 19.3 Missing technical data

If RSI or moving averages are unavailable:

- missing component adjustment = 0.0,
- confidence is reduced,
- model remains executable.

---

# 20. DOSM Algorithm

## 20.1 Canonical formula

```text
DOSM Raw =
0.50 × VC Score
+ 0.50 × Fundamental Score
+ Technical Adjustment
```

```text
DOSM Score = clamp(DOSM Raw, 0, 10)
```

Peer adjustment and growth penalty are not added again because they are already embedded in VC.

## 20.2 DOSM fair-value anchor

```text
DOSM Fair Value =
Canonical Price × exp(0.1 × (DOSM Score - 5))
```

## 20.3 DOSM upside/downside

```text
DOSM Upside =
(DOSM Fair Value / Canonical Price) - 1
```

---

# 21. Model Classification

Apply to both VC and DOSM.

| Score | Classification |
|---:|---|
| 0.00–1.99 | Avoid |
| 2.00–3.99 | Weak |
| 4.00–5.99 | Neutral |
| 6.00–7.99 | Opportunity |
| 8.00–10.00 | Exceptional |

Boundary implementation:

```python
def classify(score: float) -> str:
    if score < 2:
        return "AVOID"
    if score < 4:
        return "WEAK"
    if score < 6:
        return "NEUTRAL"
    if score < 8:
        return "OPPORTUNITY"
    return "EXCEPTIONAL"
```

---

# 22. Confidence Score

Confidence measures data completeness, not investment certainty.

Start at:

```text
100
```

Deduct:

| Condition | Deduction |
|---|---:|
| Missing peer adjustment | -10 |
| Missing RSI | -5 |
| Missing SMA50 or SMA200 | -5 |
| Missing ROIC | -10 |
| Missing debt submetric | -5 |
| Financial snapshot older than 180 days | -10 |
| Financial snapshot older than 365 days | additional -20 |
| Any valuation metric missing because of data failure | -15 each |

Do not deduct when P/E or EV/EBITDA is economically not meaningful due to losses; that condition is already scored as zero.

Clamp:

```text
0 <= Confidence <= 100
```

Confidence labels:

| Confidence | Label |
|---:|---|
| 90–100 | High |
| 70–89 | Moderate |
| 50–69 | Low |
| <50 | Insufficient |

If confidence < 50:

```text
evaluation_status = INSUFFICIENT_DATA
```

Scores may be stored but must not trigger investment-opportunity events.

---

# 23. Calculation Order

The exact execution order is:

1. Validate evaluation date and canonical price.
2. Select financial snapshot available as of evaluation date.
3. Calculate raw ratios.
4. Calculate absolute valuation scores.
5. Calculate peer-relative adjustments.
6. Calculate growth/valuation penalty.
7. Calculate VC base.
8. Calculate final VC.
9. Calculate six fundamental component scores.
10. Calculate fundamental composite.
11. Calculate RSI adjustment.
12. Calculate moving-average adjustment.
13. Calculate technical adjustment.
14. Calculate DOSM.
15. Calculate VC and DOSM fair values.
16. Calculate upside/downside.
17. Calculate classifications.
18. Calculate confidence.
19. Persist full explanation trace.
20. Publish eligible SignalOps events.

This order must not change without a model-version increment.

---

# 24. Reference Pseudocode

```python
def evaluate_equity(data: EquityEvaluationInput) -> EvaluationResult:
    validate_input(data)

    price = data.price.regular_session_close

    ps = data.market.market_cap / data.financials.revenue_ttm

    pe = None
    if data.financials.net_income_gaap_ttm > 0:
        pe = data.market.market_cap / data.financials.net_income_gaap_ttm

    ev_ebitda = None
    if data.financials.ebitda_gaap_ttm > 0:
        ev_ebitda = (
            data.market.enterprise_value
            / data.financials.ebitda_gaap_ttm
        )

    ps_score = score_ps(ps)
    pe_score = score_pe(pe) if pe is not None else 0.0
    ev_ebitda_score = (
        score_ev_ebitda(ev_ebitda)
        if ev_ebitda is not None
        else 0.0
    )

    vc_base = (
        0.40 * ps_score
        + 0.30 * pe_score
        + 0.30 * ev_ebitda_score
    )

    peer_adjustment = calculate_peer_adjustment(
        ps=ps,
        pe=pe,
        ev_ebitda=ev_ebitda,
        peer_snapshot=data.peer_snapshot,
    )

    valuation_penalty = calculate_growth_valuation_penalty(
        market_cap_to_revenue=ps,
        revenue_cagr_3y=data.growth.revenue_cagr_3y,
        net_income=data.financials.net_income_gaap_ttm,
    )

    vc_score = clamp(
        vc_base + peer_adjustment - valuation_penalty,
        0.0,
        10.0,
    )

    revenue_growth_score = score_revenue_growth(
        data.growth.revenue_cagr_3y
    )

    operating_margin = (
        data.financials.operating_income_ttm
        / data.financials.revenue_ttm
    )
    operating_margin_score = score_operating_margin(
        operating_margin
    )

    net_margin = (
        data.financials.net_income_gaap_ttm
        / data.financials.revenue_ttm
    )
    profitability_score = score_profitability(net_margin)

    fcf_margin = (
        data.financials.free_cash_flow_ttm
        / data.financials.revenue_ttm
    )
    fcf_score = score_fcf_margin(fcf_margin)

    debt_score = score_debt_profile(data)
    capital_efficiency_score = score_roic(data)

    fundamental_score = mean([
        revenue_growth_score,
        operating_margin_score,
        profitability_score,
        fcf_score,
        debt_score,
        capital_efficiency_score,
    ])

    rsi_adjustment = calculate_rsi_adjustment(
        data.technicals.rsi_14
    )

    trend_adjustment = calculate_trend_adjustment(
        price=price,
        sma50=data.technicals.sma_50,
        sma200=data.technicals.sma_200,
    )

    technical_adjustment = clamp(
        rsi_adjustment + trend_adjustment,
        -1.0,
        1.0,
    )

    dosm_score = clamp(
        0.50 * vc_score
        + 0.50 * fundamental_score
        + technical_adjustment,
        0.0,
        10.0,
    )

    vc_fair_value = price * exp(0.1 * (vc_score - 5.0))
    dosm_fair_value = price * exp(0.1 * (dosm_score - 5.0))

    confidence = calculate_confidence(data)

    return EvaluationResult(
        vc_score=round(vc_score, 4),
        dosm_score=round(dosm_score, 4),
        vc_fair_value=round(vc_fair_value, 2),
        dosm_fair_value=round(dosm_fair_value, 2),
        confidence=confidence,
        explanation=build_explanation_trace(...),
    )
```

---

# 25. Worked Example

The following example is illustrative and validates the computation path.

Assume:

```text
Price = 100.00

P/S = 0.80
GAAP P/E = 11.00
EV/EBITDA = 7.00

Peer Adjustment = +0.10
Growth/Valuation Penalty = 0.00
```

Interpolated scores:

```text
P/S Score = 9.40
P/E Score = 8.25
EV/EBITDA Score = 8.00
```

VC base:

```text
VC Base =
0.40 × 9.40
+ 0.30 × 8.25
+ 0.30 × 8.00

= 3.76 + 2.475 + 2.40
= 8.635
```

Final VC:

```text
VC = 8.635 + 0.10 - 0.00
VC = 8.735
```

VC fair value:

```text
100 × exp(0.1 × (8.735 - 5))
= 100 × exp(0.3735)
= 145.27
```

Assume fundamental component scores:

```text
Revenue Growth = 4.50
Operating Margin = 6.20
Profitability = 6.50
FCF = 7.00
Debt Profile = 5.80
Capital Efficiency = 6.60
```

Fundamental score:

```text
(4.50 + 6.20 + 6.50 + 7.00 + 5.80 + 6.60) / 6
= 6.10
```

Assume:

```text
RSI = 28 → +0.50
Price below SMA50 and SMA200 → -0.50
Technical Adjustment = 0.00
```

DOSM:

```text
DOSM =
0.50 × 8.735
+ 0.50 × 6.10
+ 0.00

= 4.3675 + 3.05
= 7.4175
```

DOSM fair value:

```text
100 × exp(0.1 × (7.4175 - 5))
= 100 × exp(0.24175)
= 127.35
```

Result:

| Model | Score | Fair Value | Classification |
|---|---:|---:|---|
| VC | 8.74 | 145.27 | Exceptional |
| DOSM | 7.42 | 127.35 | Opportunity |

---

# 26. Explainability Output Contract

Each run SHALL persist a complete trace.

```json
{
  "model_version": "vc-dosm-3.0",
  "ticker": "TGT",
  "evaluation_date": "2025-12-22",
  "canonical_price": 100.0,
  "raw_metrics": {
    "ps_ttm": 0.8,
    "pe_gaap_ttm": 11.0,
    "ev_ebitda_ttm": 7.0,
    "revenue_cagr_3y": -0.01,
    "operating_margin": 0.052,
    "net_margin": 0.039,
    "fcf_margin": 0.035,
    "debt_to_equity": 1.37,
    "net_debt_to_ebitda": 2.0,
    "roic": 0.14,
    "rsi_14": 28.0,
    "sma_50": 104.0,
    "sma_200": 116.0
  },
  "component_scores": {
    "ps_score": 9.4,
    "pe_score": 8.25,
    "ev_ebitda_score": 8.0,
    "vc_base": 8.635,
    "peer_adjustment": 0.1,
    "growth_valuation_penalty": 0.0,
    "revenue_growth_score": 4.5,
    "operating_margin_score": 6.2,
    "profitability_score": 6.5,
    "fcf_score": 7.0,
    "debt_profile_score": 5.8,
    "capital_efficiency_score": 6.6,
    "fundamental_score": 6.1,
    "rsi_adjustment": 0.5,
    "trend_adjustment": -0.5,
    "technical_adjustment": 0.0
  },
  "results": {
    "vc_score": 8.735,
    "dosm_score": 7.4175,
    "vc_fair_value": 145.27,
    "dosm_fair_value": 127.35,
    "vc_classification": "EXCEPTIONAL",
    "dosm_classification": "OPPORTUNITY",
    "confidence": 95
  }
}
```

---

# 27. SignalOps Event Rules

Events SHALL be emitted only when confidence is at least 70.

## 27.1 Valuation Opportunity

```text
VC Score >= 7.0
AND VC Fair Value > Canonical Price
```

## 27.2 DOSM Opportunity

```text
DOSM Score >= 7.0
AND DOSM Fair Value > Canonical Price
```

## 27.3 Exceptional Opportunity

```text
VC Score >= 8.0
AND DOSM Score >= 8.0
```

## 27.4 Score upgrade

```text
Current DOSM - Previous DOSM >= 0.75
```

## 27.5 Score downgrade

```text
Previous DOSM - Current DOSM >= 0.75
```

## 27.6 Fair-value cross

```text
Previous Price >= Previous DOSM Fair Value
AND Current Price < Current DOSM Fair Value
```

or the inverse for an upward cross.

---

# 28. Historical Replay Requirements

Historical execution MUST use:

- price for the requested trading date,
- only filings published on or before that date,
- technical indicators computed using bars available through that date,
- peer medians calculated from data available through that date,
- the exact model configuration version stored with the original run.

A historical replay is valid when:

```text
absolute(original VC - replayed VC) <= 0.0001
absolute(original DOSM - replayed DOSM) <= 0.0001
```

Fair-value differences must be no greater than one cent after rounding.

---

# 29. Configuration Versioning

All score tables, weights, thresholds, and penalties must be externalized into versioned configuration.

Example:

```yaml
model_version: vc-dosm-3.0

vc:
  weights:
    ps: 0.40
    pe: 0.30
    ev_ebitda: 0.30

  peer_adjustment:
    enabled: true
    minimum_peer_count: 3
    clamp_min: -0.50
    clamp_max: 0.50

  high_valuation_penalty:
    revenue_multiple_thresholds:
      - min: 15
        max: 20
        penalty: 0.50
      - min: 20
        max: 30
        penalty: 0.75
      - min: 30
        max: null
        penalty: 1.00
    growth_threshold: 0.30
    unprofitable_additional_penalty: 0.50
    maximum_penalty: 1.50

dosm:
  vc_weight: 0.50
  fundamental_weight: 0.50
  technical_adjustment_min: -1.00
  technical_adjustment_max: 1.00

fair_value:
  exponent_coefficient: 0.10
  neutral_score: 5.00
```

Any change to numeric behavior requires a new `model_version`.

---

# 30. Minimum Test Suite

## 30.1 VC tests

- exact breakpoint scoring,
- interpolation between breakpoints,
- negative earnings produce P/E score of zero,
- negative EBITDA produces EV/EBITDA score of zero,
- peer adjustment positive,
- peer adjustment negative,
- insufficient peers produce zero adjustment,
- high-valuation penalty tiers,
- unprofitable high-valuation additional penalty,
- score clamping,
- fair-value calculation.

## 30.2 Fundamental tests

- negative and positive revenue growth,
- negative and positive margins,
- negative free cash flow,
- negative equity,
- debt-free company,
- negative EBITDA with debt,
- ROIC interpolation.

## 30.3 Technical tests

- RSI below 30,
- RSI exactly 30,
- RSI exactly 70,
- RSI above 70,
- price above both moving averages,
- below both,
- mixed,
- missing technical values.

## 30.4 DOSM tests

- VC/fundamental weighting,
- additive technical adjustment,
- no peer double counting,
- no penalty double counting,
- clamp at zero,
- clamp at ten.

## 30.5 Replay tests

- split-adjusted historical price,
- filing availability date,
- deterministic peer snapshot,
- exact reproduction tolerance.

---

# 31. Non-Goals

The following are outside this specification:

- Absolute Intrinsic Value / DCF,
- analyst consensus,
- options sentiment,
- insider trading,
- credit spreads,
- macroeconomic overlays,
- portfolio construction,
- trade execution,
- AI-generated recommendations.

These may consume VC and DOSM outputs later but must not alter their calculations.

---

# 32. Canonical Formulas Summary

```text
P/S Score = interpolate(P/S table)
P/E Score = interpolate(P/E table), or 0 if GAAP net income <= 0
EV/EBITDA Score = interpolate(EV/EBITDA table), or 0 if GAAP EBITDA <= 0

VC Base =
0.40 × P/S Score
+ 0.30 × P/E Score
+ 0.30 × EV/EBITDA Score

VC =
clamp(
VC Base
+ Peer Adjustment
- Growth/Valuation Penalty,
0,
10
)

Fundamental Score =
mean(
Revenue Growth Score,
Operating Margin Score,
GAAP Profitability Score,
FCF Score,
Debt Profile Score,
Capital Efficiency Score
)

Technical Adjustment =
RSI Adjustment
+ Moving-Average Adjustment

DOSM =
clamp(
0.50 × VC
+ 0.50 × Fundamental Score
+ Technical Adjustment,
0,
10
)

VC Fair Value =
Price × exp(0.1 × (VC - 5))

DOSM Fair Value =
Price × exp(0.1 × (DOSM - 5))
```

---

# 33. Implementation Acceptance Criteria

The implementation is complete when:

1. A normalized MarketOps equity snapshot can be submitted without any direct Massive.com call from the scoring engine.
2. VC and DOSM are reproduced exactly from configuration and inputs.
3. Every result contains the full explanation trace.
4. Unprofitable companies receive zero for P/E and/or EV/EBITDA where economically not meaningful.
5. Peer adjustment and growth penalty are applied once only.
6. Historical replays match stored runs within specified tolerances.
7. All score tables and thresholds are versioned.
8. The minimum test suite passes.
9. Eligible Insight Events are emitted with model version and confidence.
10. No LLM-generated value can alter a numeric score.



---

## Current operational profile: four-quarter TTM

The full v3 target includes 16 rolling quarters, three-year revenue CAGR, a growth score, and a high-valuation/low-growth penalty. The currently authorized implementation is narrower because the available FMP entitlement exposes only four current quarterly rows per statement.

For deployed behavior, [the TTM Operational Profile](SignalOps_MarketOps_VC_DOSM_TTM_Operational_Profile_v1.md) is normative: derive only TTM inputs from four quarters; do not synthesize CAGR; withhold the growth score and CAGR-dependent penalty; reweight the five available DOSM fundamental dimensions equally; and disclose the ttm_only data profile with its confidence deduction. The full v3 provisions resume only after the roadmap acceptance gate for sixteen distinct, point-in-time eligible quarters is passed.
