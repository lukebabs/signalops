# MarketOps VC/DOSM TTM Operational Profile

**Version:** 1.0  
**Status:** Current production-research profile  
**Effective:** 2026-07-31  
**Scope:** Four-quarter TTM VC/DOSM only; no investment recommendation.

## 1. Purpose and analyst boundary

Valuation Composite (VC) is a deterministic relative-valuation score. It answers: given the latest reported trailing financials, how expensive is an equity on P/S, GAAP P/E, and EV/EBITDA?

Distressed Opportunity Scoring Model (DOSM) combines VC with operating quality, cash generation, balance-sheet capacity, and a bounded technical condition. It answers: which names warrant research because the available TTM evidence is relatively more attractive or distressed?

Both outputs are research artifacts. Fair-value anchors are mathematical translations of score, not price targets. They do not recommend buying, selling, or allocating capital.

## 2. Current source and cadence contract

- Massive is authoritative for completed-session adjusted close, market capitalization, and shares outstanding. Massive remains the source for options and market data.
- FMP is used only for quarterly income statement, balance sheet, and cash-flow statement rows.
- The current FMP entitlement returns four current quarterly rows per statement. The runner makes three FMP calls per newly refreshed symbol and is capped at 240 daily calls.
- FMP polling is **explicit**: only the weekly post-close financial-refresh job and the 02:00 ET continuation pass `--refresh-financials`. Routine valuation/tactical recalculation reuses retained financial snapshots, including stale-but-provenanced snapshots, and must never consume FMP quota.
- RSI-14, SMA-50, and SMA-200 are Massive provider indicators for the same completed session. They are persisted as `technical_provenance`; local EOD history is not used to reconstruct them for VC/DOSM or Tactical Market Posture.
- The job is weekly, post-close. A result is persisted with source statement IDs, accepted filing times, derived values, score trace, model version, and data profile.
- Raw and normalized statements plus derived snapshots are retained for seven years. This supports audit of what was actually calculated; it does not yet create a complete point-in-time historical universe.

## 3. TTM derivation contract

For each completed evaluation session, select the latest four accepted income-statement rows and latest four accepted cash-flow rows whose accepted_at is no later than the evaluation boundary. Select the newest eligible balance-sheet row.

The engine derives locally:

- revenue, gross profit, operating income, pretax income, tax expense, GAAP net income, and EBITDA TTM: sum four income quarters;
- operating cash flow TTM: sum four cash-flow quarters;
- capital expenditure TTM: sum absolute Capex values, because FMP may report cash outflow as negative;
- free cash flow: operating cash flow minus positive Capex;
- enterprise value: market cap + total debt - cash;
- invested capital: total debt + shareholders equity - cash;
- effective tax rate: tax expense / pretax income when pretax income is positive, otherwise 25%, clamped to [0%, 40%];
- NOPAT: operating income × (1 - effective tax rate).

Inputs with fewer than four usable income or cash-flow quarters, no eligible balance row, non-positive canonical price, or non-positive market cap are rejected. No vendor-computed ratio is used.

## 4. Current algorithms

### 4.1 Valuation Composite

The base VC score is:

0.40 x P/S score + 0.30 x GAAP P/E score + 0.30 x EV/EBITDA score.

Each ratio is mapped deterministically to the canonical 0-10 score curve in the v3 specification. When at least three same-sector/same-industry peers exist, a peer-relative adjustment is applied once. The score is bounded to 0-10.

### 4.2 DOSM

DOSM is:

0.50 x final VC + 0.50 x fundamental quality + bounded technical adjustment.

In the TTM profile, fundamental quality is the equally weighted average of five available dimensions:

1. operating-margin score;
2. GAAP profitability score;
3. free-cash-flow score;
4. debt-profile score;
5. capital-efficiency/ROIC score.

The RSI/SMA adjustment is bounded to [-1, +1]. Missing technical input does not invent a neutral signal; it produces no adjustment and reduces confidence.

## 5. Deliberate TTM-only behavior

Three-year revenue CAGR requires sixteen rolling quarters. It is not derivable from four quarters and must never be approximated by annualizing one quarter or treating missing growth as zero.

Therefore every TTM-only result records:

- data_profile = ttm_only;
- growth_status = unavailable_requires_16_quarters;
- evaluation_status = complete_ttm_only;
- a confidence deduction of 15 points and an explanatory reason.

The following are withheld, not zeroed:

- the revenue-growth component score;
- the high-valuation/low-growth penalty;
- any conclusion that depends on 3-year revenue CAGR.

The remaining five DOSM fundamental dimensions are reweighted equally. The UI must show the TTM badge and explain the withheld growth logic in the calculation trace.

## 6. Known gaps and safeguards

| Gap | Current safeguard | Consequence |
|---|---|---|
| FMP access is limited to four current quarters | TTM uses exactly four rows | CAGR and growth penalty unavailable |
| No historical pagination/backfill | accepted-at selection on retained rows | not a complete historical replay capability |
| Provider technical input unavailable | no adjustment plus confidence deduction | score is less complete, not falsely precise |
| Routine rerun could exhaust FMP allowance | FMP polling requires `--refresh-financials`; cached financial snapshots are default | technical and tactical refreshes stay within the FMP allowance |
| Small peer groups | no peer adjustment below three peers | confidence deduction |
| Market cap and statements may have different update timing | provider IDs and timestamps are persisted | analyst must inspect trace for freshness |
| Research scores are unvalidated | no signal, alert, ranking, or execution promotion | no automated action |

A result with insufficient required TTM inputs is insufficient_data and ineligible. A complete_ttm_only result may be visible for research, but its confidence and data-profile caveat remain part of the record.

## 7. Operations and validation

Before a weekly financial refresh, ensure a completed regular session exists. Run only after market close and retain the session date. The runner should not exceed three FMP calls per fresh ticker or the configured daily ceiling. Do not pass `--refresh-financials` for a technical-only or tactical rerun.

For each pilot, verify:

1. FMP requests use period=quarter and limit=4;
2. four distinct accepted quarters exist for income and cash flow;
3. Capex is normalized before free-cash-flow calculation;
4. trace contains ttm_only, an unavailable growth status, zero growth penalty, and the confidence reason;
5. no CAGR value or growth conclusion appears in API/UI output;
6. snapshot statement IDs and source timestamps are persisted.

## 8. Relationship to canonical specifications

The v3 deterministic specification remains the full target algorithm. This document governs deployed behavior while only four quarters are available. If the documents conflict on CAGR or its penalty, this TTM operational profile controls live execution until the roadmap acceptance gate is met.
