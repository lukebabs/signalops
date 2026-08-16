# FMP Annual Financial Enrichment and VC/DOSM v4

**Status:** the additive evidence schema is applied; the central-capture and
scoring foundation is disabled. It is not yet scheduled, populated,
reader-exposed, or promoted over v3.

## Decision

FMP Starter is used for a centrally governed, **annual-only** financial
enrichment of the warm EOD cohort. It is not a tenant feature, a watchlist
trigger, or a browser/provider request path. The initial cohort is the enabled
global warm EOD set, bounded to 1,000 assets. The worker reads its canonical
global asset IDs once and stores one immutable provider capture per asset.

The existing `vc-dosm-3.0` TTM result remains the live default. The new
`vc-dosm-4.0-annual` profile is parallel research evidence until its own
coverage, replay, calibration, reader, and analyst gates pass.

## Provider and capture contract

For each selected global asset, the central worker makes these five FMP calls:

1. annual income statement (`limit=5`);
2. annual balance sheet (`limit=5`);
3. annual cash-flow statement (`limit=5`);
4. annual ratios (`limit=5`); and
5. annual key metrics (`limit=5`).

Only periods with all three raw statements are retained. Ratios and key
metrics are preserved as provider-reference facts; all model ratios are
derived locally from the raw statements. Every evidence run and record carries
the source, endpoint list, capture time, fingerprint, validation contract, and
immutable baseline reference. Failed captures are recorded as invalid evidence
instead of silently becoming zero-valued financials.

The process enforces a minimum 250 ms spacing between *individual* FMP calls:
at most 240 calls/minute. A 1,000-symbol full pass is at most 5,000 calls and
normally takes about 21 minutes before bounded transient retries. It runs only
from the central scheduler after an entitlement preflight; it has no
user-triggered execution mode and no daily 240-call ceiling.

## Annual v4 scoring contract

`vc-dosm-4.0-annual` is an explainable annual profile, not a recommendation or
a daily timing model. It uses six independently exposed 0–10 dimensions:

| Dimension | Weight | Locally derived annual evidence |
| --- | ---: | --- |
| Valuation / VC | 40% | price-to-sales, GAAP P/E where positive, EV/EBITDA where positive |
| Profitability | 12% | operating, net, and free-cash-flow margins |
| Growth | 12% | three-year revenue CAGR from annual statements |
| Liquidity | 12% | current and quick ratios |
| Leverage | 12% | debt/equity, net debt/EBITDA, interest coverage |
| Capital efficiency | 12% | ROIC and ROE |

An unavailable dimension is excluded and the available dimensions are
reweighted; it is never scored as a factual zero. The result records a
confidence deduction and reason for every missing dimension. Canonical price
is only used to render fair-value anchors; absence withholds those anchors and
reduces confidence. No technical adjustment, alert, trade instruction, or
Signal Assurance assertion is created by this profile.

## Required activation gates

1. Migration `000136_subscriber_global_annual_financial_evidence` was applied
   to the dedicated MarketOps primary on 2026-08-16. It changes only allowed
   evidence kinds/execution modes; it creates no provider capture or reader.
2. Verify the configured FMP credential against all five annual endpoints with
   one bounded non-writing preflight. A 402/403 or incomplete response leaves
   the worker disabled.
3. Run one bounded central dry run and retain its count, failure classes,
   endpoint provenance, and rate-limit evidence.
4. Execute one approved full warm-cohort capture and validate immutable row
   counts, absence of tenant copies, and no FMP access from browser routes.
5. The append-only v4 result writer is implemented, but remains inactive until
   annual evidence exists. It writes distinct annual VC/DOSM algorithm IDs and
   discloses `data_profile=annual_5y`, model version, financial as-of date,
   coverage/confidence, source scope, and the input evidence references. A
   restricted global reader remains a separate gate.
6. Run deterministic replay and a time-aware historical calibration before any
   change to the UI default or any comparison with v3.
7. After the above gates, enable the Saturday 02:30 America/New_York central
   schedule with bounded retries and the existing operational monitoring.

## Rollback and continuity

The capture is append-only and v3 stays untouched. Disabling the schedule or
the v4 reader stops future use immediately; no captured financial record or
v3 result is deleted. Dedicated MarketOps PostgreSQL backups and restore
rehearsals cover the evidence ledger. A provider outage records explicit
invalid capture evidence and leaves the latest validated annual evidence
available with its freshness disclosed.
