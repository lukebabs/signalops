# SignalOps MarketOps Engineering Specification
## Financial Derivation, Technical Derivation, VC, DOSM, AIV, and Ranking Sequence

**Version:** 1.0  
**Status:** Implementation Specification  
**Target:** Code Agent  
**Subsystem:** SignalOps → MarketOps  
**Primary Market Data Source:** Massive.com  
**Primary Financial Statement Source:** Financial Modeling Prep quarterly endpoints (current TTM profile: four quarters; 16-quarter CAGR target on roadmap; seven-year retained immutable snapshots)  
**Purpose:** Define the engineering sequence, contracts, responsibilities, and acceptance criteria required to implement VC and DOSM reliably inside the existing SignalOps MarketOps subsystem.

---

# 1. Executive Summary

MarketOps already integrates with Massive.com for market data. The missing capability is a deterministic financial intelligence pipeline that converts raw market and quarterly financial statement data into normalized snapshots, then applies the VC and DOSM algorithms.

The implementation SHALL be sequenced as follows:

1. Financial Derivation Engine
2. Technical Derivation Engine
3. Peer Comparison Engine
4. VC Engine
5. DOSM Engine
6. Absolute Intrinsic Value Engine
7. Composite Ranking Engine
8. SignalOps Event and Knowledge Graph Integration

The code agent SHALL NOT begin with VC or DOSM before completing and validating the Financial Derivation Engine.

The models must never consume vendor-computed ratios directly. All ratios must be derived locally from raw statements and normalized market data.

---

# 2. Design Principles

## 2.1 Deterministic

Identical inputs and configuration versions must always produce identical outputs.

No LLM, analyst target, sentiment score, or heuristic override may alter numeric calculations.

## 2.2 Explainable

Every output must include:

- source inputs,
- derived metrics,
- scoring contributions,
- penalties,
- adjustments,
- model version,
- calculation timestamp.

## 2.3 Replayable

Historical evaluations must use only data that was available as of the evaluation date.

## 2.4 Modular

Each engine must own one responsibility and emit a versioned snapshot.

## 2.5 GAAP-only

VC and DOSM must use GAAP financials only.

The following are prohibited:

- adjusted EPS,
- non-GAAP EPS,
- management-adjusted EBITDA,
- pro forma margins,
- analyst consensus ratios.

## 2.6 Provider isolation

Only ingestion adapters may know Massive.com or FMP endpoint details.

Scoring engines must consume normalized internal models only.

---

# 3. High-Level Architecture

```text
Massive.com
  ├── Adjusted daily OHLCV
  ├── Corporate actions
  ├── Shares outstanding / market capitalization
  └── Historical prices
            │
            ▼
Financial Modeling Prep
  ├── Quarterly Income Statement
  ├── Quarterly Balance Sheet
  ├── Quarterly Cash Flow Statement
  └── Enterprise Value
            │
            ▼
Financial Derivation Engine
            │
            ├── FinancialSnapshot
            ▼
Technical Derivation Engine
            │
            ├── TechnicalSnapshot
            ▼
Peer Comparison Engine
            │
            ├── PeerSnapshot
            ▼
VC Engine
            │
            ├── VCSnapshot
            ▼
DOSM Engine
            │
            ├── DOSMSnapshot
            ▼
AIV Engine
            │
            ├── AIVSnapshot
            ▼
Composite Ranking Engine
            │
            ├── OpportunitySnapshot
            ▼
SignalOps Events / Knowledge Graph / Dashboard / Alerts
```

---

# 4. Engineering Work Sequence

## Phase 1 — Financial Derivation Engine

### Objective

Create a reliable, testable, historical financial snapshot for each ticker and evaluation date.

### Responsibilities

- fetch quarterly statements from FMP,
- fetch market data from Massive,
- normalize field names,
- validate filing dates,
- select statements available as of the evaluation date,
- derive TTM values,
- derive all financial ratios,
- persist one immutable FinancialSnapshot,
- support historical replay.

### Exit Criteria

Do not begin Phase 2 until:

- all required TTM calculations pass unit tests,
- filing-date replay is correct,
- no vendor ratios are consumed,
- NVDA test data reproduces expected totals,
- at least five additional tickers from different sectors pass validation.

---

## Phase 2 — Technical Derivation Engine

### Objective

Produce technical indicators from Massive daily adjusted price data.

### Responsibilities

- calculate RSI(14),
- calculate SMA20,
- calculate SMA50,
- calculate SMA200,
- calculate MACD,
- calculate ATR,
- calculate rolling volatility,
- persist one TechnicalSnapshot per ticker and evaluation date.

### Exit Criteria

- indicators match a trusted reference within defined tolerance,
- historical replay uses only bars available through the evaluation date,
- corporate-action-adjusted price history is used.

---

## Phase 3 — Peer Comparison Engine

### Objective

Create a deterministic peer-relative valuation snapshot.

### Responsibilities

- resolve sector and industry,
- assign a maintained peer group,
- exclude invalid or stale peers,
- calculate peer medians,
- calculate relative multiples,
- calculate peer adjustment,
- persist PeerSnapshot.

### Exit Criteria

- minimum peer count enforcement works,
- outlier handling is tested,
- peer adjustment is deterministic,
- sector-specific peer lists are versioned.

---

## Phase 4 — VC Engine

### Objective

Calculate the Valuation Composite from FinancialSnapshot and PeerSnapshot.

### Responsibilities

- calculate P/S score,
- calculate GAAP P/E score,
- calculate EV/EBITDA score,
- calculate weighted VC base,
- apply peer adjustment once,
- apply high-valuation/low-growth penalty once,
- calculate VC fair value,
- classify the signal,
- persist full explanation trace.

### Exit Criteria

- component scoring tests pass,
- unprofitable-company behavior is correct,
- peer and penalty double counting is impossible,
- fair value reproduces the canonical formula exactly.

---

## Phase 5 — DOSM Engine

### Objective

Combine valuation, fundamentals, and technical condition.

### Responsibilities

- consume final VC score,
- calculate six-part fundamental score,
- apply RSI and SMA technical adjustments,
- calculate DOSM score,
- calculate DOSM fair value,
- classify result,
- persist explanation trace.

### Exit Criteria

- no peer adjustment is re-applied,
- no growth penalty is re-applied,
- technical adjustment is bounded,
- result is reproducible from stored snapshots.

---

## Phase 6 — Absolute Intrinsic Value Engine

### Objective

Add a price-independent valuation anchor.

### Responsibilities

- calculate conservative, base, and optimistic DCF scenarios,
- use locally derived free cash flow,
- version assumptions,
- persist AIVSnapshot.

AIV must remain separate from VC and DOSM.

---

## Phase 7 — Composite Ranking Engine

### Objective

Rank opportunities using outputs from VC, DOSM, and AIV.

### Responsibilities

- combine normalized model outputs,
- retain model-specific scores,
- support filtering by sector and risk,
- produce OpportunitySnapshot,
- never alter original model calculations.

---

## Phase 8 — SignalOps Integration

### Objective

Expose opportunities as first-class SignalOps artifacts.

### Responsibilities

- publish events,
- create Knowledge Graph entities and relationships,
- support dashboards,
- support alerts,
- expose model explanations through API.

---

# 5. Source Data Contracts

## 5.1 Massive Market Data

Required fields:

```json
{
  "ticker": "NVDA",
  "evaluation_date": "2026-07-30",
  "adjusted_close": 0.0,
  "price_timestamp_utc": "2026-07-30T20:00:00Z",
  "adjusted_for_splits": true,
  "shares_outstanding": 0.0,
  "market_cap": 0.0,
  "currency": "USD"
}
```

Rules:

- use regular-session adjusted close,
- reject after-hours and pre-market values,
- store exact timestamp,
- preserve corporate-action version.

## 5.2 FMP Quarterly Income Statement

Required latest four quarterly records:

- date,
- filingDate,
- acceptedDate,
- calendarYear,
- period,
- revenue,
- grossProfit,
- operatingIncome,
- incomeBeforeTax,
- incomeTaxExpense,
- netIncome,
- ebitda,
- depreciationAndAmortization where available.

## 5.3 FMP Quarterly Balance Sheet

Required latest quarter available as of evaluation date:

- cashAndCashEquivalents,
- shortTermInvestments where relevant,
- totalCurrentAssets,
- totalAssets,
- shortTermDebt,
- longTermDebt,
- totalDebt,
- totalLiabilities,
- totalStockholdersEquity,
- commonStockSharesOutstanding.

## 5.4 FMP Quarterly Cash Flow

Required latest four quarterly records:

- netCashProvidedByOperatingActivities,
- operatingCashFlow,
- capitalExpenditure,
- acquisitionsNet where relevant,
- stockBasedCompensation,
- freeCashFlow if returned, used only for validation.

## 5.5 Enterprise Value

Required:

- date,
- stockPrice,
- numberOfShares,
- marketCapitalization,
- minusCashAndCashEquivalents,
- addTotalDebt,
- enterpriseValue.

The internal engine must validate enterprise value using:

```text
Enterprise Value =
Market Capitalization
+ Total Debt
- Cash and Cash Equivalents
```

Tolerance:

```text
absolute difference <= 2%
```

If vendor EV differs by more than 2%, mark it as disputed and use locally calculated EV.

---

# 6. Financial Derivation Engine

## 6.1 Statement Selection

For an evaluation date:

1. fetch sufficient quarterly history,
2. exclude statements with `acceptedDate` after the evaluation date,
3. sort by fiscal period end descending,
4. select the latest four non-duplicate quarters,
5. verify there are no missing fiscal quarters,
6. select the latest balance sheet available as of the evaluation date.

## 6.2 Duplicate Handling

Duplicate statements may exist due to amendments.

Use:

1. latest accepted filing for the same fiscal period,
2. prefer amended values when accepted before evaluation date,
3. retain provenance of superseded records.

## 6.3 TTM Flow Metrics

Sum the latest four quarters:

```text
Revenue TTM = Σ quarterly revenue
Gross Profit TTM = Σ quarterly gross profit
Operating Income TTM = Σ quarterly operating income
Pretax Income TTM = Σ quarterly income before tax
Tax Expense TTM = Σ quarterly income tax expense
GAAP Net Income TTM = Σ quarterly net income
EBITDA TTM = Σ quarterly EBITDA
Operating Cash Flow TTM = Σ quarterly operating cash flow
Capital Expenditure TTM = Σ quarterly capital expenditure
Free Cash Flow TTM = Operating Cash Flow TTM - Capital Expenditure TTM
```

Capital expenditure normalization:

FMP may return capital expenditure as a negative cash-flow value. The adapter must normalize Capex to a positive spending amount before subtraction.

```text
Normalized Capex = absolute(capitalExpenditure)
FCF = OCF - Normalized Capex
```

## 6.4 Balance Sheet Metrics

Use the latest quarter only:

```text
Cash = cash and cash equivalents
Total Debt = short-term debt + long-term debt
Equity = total stockholders' equity
Total Assets = total assets
Shares Outstanding = common shares outstanding
```

Never sum balance-sheet values across quarters.

## 6.5 Revenue CAGR

Preferred:

```text
Revenue CAGR 3Y =
(Revenue TTM / Revenue TTM 3 Years Ago)^(1/3) - 1
```

Require at least sixteen quarterly records to reconstruct comparable TTM periods.

If insufficient history exists:

- set CAGR status to missing,
- reduce confidence,
- do not infer from annualized latest-quarter growth.

## 6.6 Effective Tax Rate

```text
Effective Tax Rate =
Tax Expense TTM / Pretax Income TTM
```

Rules:

- valid only when pretax income > 0,
- clamp observed rate to 0–40%,
- fallback to 25% if invalid,
- record fallback use.

## 6.7 NOPAT

```text
NOPAT =
Operating Income TTM × (1 - Effective Tax Rate)
```

## 6.8 Invested Capital

Default formula:

```text
Invested Capital =
Total Debt
+ Total Stockholders' Equity
- Cash and Cash Equivalents
```

## 6.9 Derived Ratios

```text
P/S =
Market Capitalization / Revenue TTM
```

```text
GAAP P/E =
Market Capitalization / GAAP Net Income TTM
```

If net income <= 0:

```text
status = NOT_MEANINGFUL
```

```text
EV/EBITDA =
Enterprise Value / EBITDA TTM
```

If EBITDA <= 0:

```text
status = NOT_MEANINGFUL
```

```text
Operating Margin =
Operating Income TTM / Revenue TTM
```

```text
Net Margin =
GAAP Net Income TTM / Revenue TTM
```

```text
FCF Margin =
Free Cash Flow TTM / Revenue TTM
```

```text
Debt to Equity =
Total Debt / Total Stockholders' Equity
```

If equity <= 0:

```text
status = DISTRESSED_BALANCE_SHEET
```

```text
Net Debt =
Total Debt - Cash
```

```text
Net Debt / EBITDA =
Net Debt / EBITDA TTM
```

```text
ROIC =
NOPAT / Invested Capital
```

```text
Market Cap / Revenue =
Market Capitalization / Revenue TTM
```

---

# 7. FinancialSnapshot Contract

```json
{
  "snapshot_id": "uuid",
  "snapshot_version": "financial-v1.0",
  "ticker": "NVDA",
  "evaluation_date": "2026-07-30",
  "price": 0.0,
  "market_cap": 0.0,
  "enterprise_value": 0.0,
  "revenue_ttm": 0.0,
  "gross_profit_ttm": 0.0,
  "operating_income_ttm": 0.0,
  "net_income_gaap_ttm": 0.0,
  "ebitda_ttm": 0.0,
  "operating_cash_flow_ttm": 0.0,
  "capex_ttm": 0.0,
  "free_cash_flow_ttm": 0.0,
  "cash": 0.0,
  "total_debt": 0.0,
  "equity": 0.0,
  "invested_capital": 0.0,
  "revenue_cagr_3y": 0.0,
  "ps_ttm": 0.0,
  "pe_gaap_ttm": 0.0,
  "pe_status": "VALID",
  "ev_ebitda_ttm": 0.0,
  "ev_ebitda_status": "VALID",
  "operating_margin": 0.0,
  "net_margin": 0.0,
  "fcf_margin": 0.0,
  "debt_to_equity": 0.0,
  "net_debt_to_ebitda": 0.0,
  "roic": 0.0,
  "effective_tax_rate": 0.0,
  "confidence": 100,
  "source_provenance": {
    "market": "massive",
    "financials": "fmp-quarterly",
    "enterprise_value": "local-or-fmp"
  },
  "statement_ids": [],
  "created_at": "timestamp"
}
```

VC and DOSM must read only from this snapshot and the TechnicalSnapshot / PeerSnapshot.

---

# 8. Technical Derivation Engine

## 8.1 Inputs

At least 252 adjusted daily bars through the evaluation date.

## 8.2 Indicators

- RSI(14), Wilder method
- SMA20
- SMA50
- SMA200
- EMA12
- EMA26
- MACD
- MACD signal 9
- ATR14
- 20-day annualized volatility

## 8.3 TechnicalSnapshot Contract

```json
{
  "snapshot_version": "technical-v1.0",
  "ticker": "NVDA",
  "evaluation_date": "2026-07-30",
  "rsi_14": 0.0,
  "sma_20": 0.0,
  "sma_50": 0.0,
  "sma_200": 0.0,
  "macd": 0.0,
  "macd_signal": 0.0,
  "atr_14": 0.0,
  "volatility_20d": 0.0,
  "confidence": 100
}
```

---

# 9. Peer Comparison Engine

## 9.1 Peer Group Rules

Peer groups must be explicitly configured.

Example categories:

- semiconductors,
- enterprise software,
- cybersecurity,
- fintech,
- retail,
- logistics,
- healthcare insurers,
- banks,
- industrials.

## 9.2 Exclusions

Exclude peers with:

- stale financial snapshots,
- mismatched currency without normalization,
- invalid revenue,
- market cap outside configured peer range,
- materially different business model.

## 9.3 Metrics

Calculate medians for:

- P/S,
- valid GAAP P/E,
- valid EV/EBITDA,
- revenue CAGR,
- operating margin,
- FCF margin.

## 9.4 PeerSnapshot Contract

```json
{
  "snapshot_version": "peer-v1.0",
  "ticker": "NVDA",
  "evaluation_date": "2026-07-30",
  "peer_group_id": "SEMICONDUCTORS_LARGE_CAP",
  "peer_count": 0,
  "ps_median": 0.0,
  "pe_median": 0.0,
  "ev_ebitda_median": 0.0,
  "revenue_growth_median": 0.0,
  "operating_margin_median": 0.0,
  "fcf_margin_median": 0.0,
  "peer_adjustment": 0.0,
  "confidence": 100
}
```

---

# 10. VC Engine

The VC engine must implement the deterministic algorithm already defined in the canonical VC/DOSM algorithm specification.

## Inputs

- FinancialSnapshot
- PeerSnapshot
- VC configuration version

## Outputs

- P/S component score
- GAAP P/E component score
- EV/EBITDA component score
- VC base
- peer adjustment
- high-valuation/low-growth penalty
- final VC score
- fair value
- upside/downside
- classification
- confidence
- explanation trace

## Important Rule

Peer adjustment and growth penalty are applied exactly once.

---

# 11. DOSM Engine

## Inputs

- FinancialSnapshot
- TechnicalSnapshot
- VCSnapshot
- DOSM configuration version

## Outputs

- revenue growth score
- operating margin score
- GAAP profitability score
- FCF score
- debt profile score
- capital efficiency score
- fundamental composite
- RSI adjustment
- moving-average adjustment
- technical adjustment
- final DOSM score
- fair value
- classification
- confidence
- explanation trace

## Important Rule

DOSM consumes the final VC score.

It must not independently reapply:

- peer adjustment,
- growth valuation penalty.

---

# 12. Persistence Model

Recommended tables:

```text
market_price_snapshots
financial_statement_raw
financial_snapshots
technical_snapshots
peer_snapshots
vc_snapshots
dosm_snapshots
aiv_snapshots
opportunity_snapshots
model_events
model_config_versions
```

All snapshots are immutable.

Corrections create a new version rather than overwriting previous output.

---

# 13. API Endpoints

Recommended internal endpoints:

```text
POST /marketops/derive/financial/{ticker}
POST /marketops/derive/technical/{ticker}
POST /marketops/derive/peers/{ticker}
POST /marketops/evaluate/vc/{ticker}
POST /marketops/evaluate/dosm/{ticker}
POST /marketops/evaluate/full/{ticker}
GET  /marketops/snapshots/{ticker}/{date}
POST /marketops/replay/{ticker}/{date}
GET  /marketops/explain/{ticker}/{date}
```

Batch:

```text
POST /marketops/evaluate/batch
```

---

# 14. Job Orchestration

Recommended daily sequence after market close:

1. ingest Massive adjusted daily bars,
2. update corporate actions,
3. refresh FMP statements only for companies with new filings,
4. derive FinancialSnapshot,
5. derive TechnicalSnapshot,
6. refresh PeerSnapshot,
7. run VC,
8. run DOSM,
9. run AIV if enabled,
10. update ranking,
11. publish events.

All jobs must be idempotent.

Use snapshot keys:

```text
ticker + evaluation_date + engine_version + input_hash
```

---

# 15. Observability

Each engine must emit:

- execution duration,
- input record count,
- missing field count,
- confidence score,
- warning count,
- snapshot ID,
- engine version,
- source latency,
- replay status.

Recommended metrics:

```text
marketops_fde_success_total
marketops_fde_failure_total
marketops_snapshot_confidence
marketops_vc_score
marketops_dosm_score
marketops_replay_mismatch_total
marketops_fmp_402_total
marketops_fmp_429_total
```

---

# 16. Error Handling

## Retriable

- HTTP 429
- HTTP 5xx
- network timeout
- temporary source failure

## Non-retriable

- HTTP 401
- HTTP 402 entitlement failure
- invalid ticker
- insufficient quarterly history
- malformed statement payload

## Degraded Mode

If technical data is missing:

- VC may still run,
- DOSM may run with reduced confidence and zero technical adjustment.

If peer data is missing:

- peer adjustment = 0,
- confidence reduced.

If fewer than four quarters exist:

- FinancialSnapshot is rejected,
- VC and DOSM do not run.

---

# 17. Testing Strategy

## Unit Tests

- all TTM sums,
- capex sign normalization,
- enterprise value validation,
- every ratio,
- CAGR,
- tax-rate fallback,
- ROIC,
- interpolation tables,
- penalties,
- peer adjustment,
- RSI,
- moving-average logic,
- fair-value formulas.

## Contract Tests

- FMP quarterly endpoint payloads,
- FMP enterprise value payload,
- Massive adjusted bars,
- schema compatibility.

## Integration Tests

Use at least:

- NVDA — profitable high-growth semiconductor
- NET — GAAP-unprofitable high valuation
- TGT — mature low-multiple retailer
- JPM — financial institution edge case
- HUM — insurer edge case
- CRWV — newly public, negative earnings
- UPS — mature industrial/logistics company

## Replay Tests

Historical run must reproduce:

```text
VC score tolerance <= 0.0001
DOSM score tolerance <= 0.0001
fair value tolerance <= $0.01
```

---

# 18. Delivery Milestones

## Milestone 1 — Adapters and Raw Persistence

Deliver:

- Massive adapter reuse
- FMP quarterly adapters
- FMP EV adapter
- raw statement storage
- provenance and filing metadata

## Milestone 2 — Financial Derivation Engine

Deliver:

- TTM calculations
- balance-sheet normalization
- ratios
- FinancialSnapshot
- unit tests
- NVDA validation report

## Milestone 3 — Technical Engine

Deliver:

- indicator calculations
- TechnicalSnapshot
- reference comparison tests

## Milestone 4 — Peer Engine

Deliver:

- peer configuration
- medians
- adjustment
- PeerSnapshot

## Milestone 5 — VC Engine

Deliver:

- deterministic VC scoring
- fair value
- explanation trace
- API endpoint
- persistence

## Milestone 6 — DOSM Engine

Deliver:

- fundamental composite
- technical adjustment
- deterministic DOSM
- explanation trace
- API endpoint
- persistence

## Milestone 7 — Replay and Events

Deliver:

- historical replay
- SignalOps events
- Knowledge Graph artifact mapping
- dashboard-ready output

## Milestone 8 — AIV and Ranking

Deliver:

- price-independent AIV
- ranking layer
- scenario support

---

# 19. Acceptance Criteria

The implementation is accepted when:

1. FMP quarterly endpoints are used successfully without TTM endpoint dependency.
2. All ratios are derived locally.
3. No vendor ratio is consumed by VC or DOSM.
4. FinancialSnapshot is reproducible for a ticker/date pair.
5. VC and DOSM consume only versioned internal snapshots.
6. Historical replay respects filing availability dates.
7. Peer and penalty logic are applied once only.
8. Explanation traces reconstruct every score.
9. All integration tickers pass.
10. SignalOps events include model and input versions.
11. No LLM can alter numeric outputs.
12. All milestone test suites pass.

---

# 20. Recommended First Implementation Task

The code agent should begin with NVDA and implement only:

1. FMP quarterly income statement adapter
2. FMP quarterly balance sheet adapter
3. FMP quarterly cash-flow adapter
4. FMP enterprise-value adapter
5. TTM derivation
6. FinancialSnapshot
7. Validation tests

Do not implement VC or DOSM until the NVDA FinancialSnapshot is correct and manually reconciled.

After NVDA passes, validate NET and TGT to cover:

- unprofitable growth company,
- mature profitable retailer.

Only then proceed to scoring engines.



---

# Current implementation constraint: TTM-only phase

Phase 1 currently operates as a four-quarter TTM slice. The FMP entitlement has been verified to return four current quarterly rows and to reject larger requests. Accordingly, Phase 1 accepts four eligible income rows, four cash-flow rows, and one balance row; it derives TTM values and persists their provenance. It does not derive three-year revenue CAGR.

The VC/DOSM engines must mark results ttm_only, withhold the growth score and high-valuation/low-growth penalty, and reweight the five remaining DOSM fundamental dimensions equally. This is an explicit model profile, not a missing value substituted with zero.

The original 16-quarter/CAGR requirement remains a roadmap gate. It may be enabled only after the provider can supply sixteen distinct quarterly rows with accepted filing times and replay validation passes. See [TTM Operational Profile](SignalOps_MarketOps_VC_DOSM_TTM_Operational_Profile_v1.md) and [VC/DOSM Roadmap](VC_DOSM_Roadmap.md).
