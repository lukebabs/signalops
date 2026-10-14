# S2 Top-1,000 Ranking Source Preflight

Date: 2026-08-12 UTC
Environment: live provider credentials; no persistent market-data change

## Massive discovery preflight

Massive All Tickers supports active US common-stock discovery through locale, market, type, and active filters. A read-only live request with the intended market-cap ranking parameters returned HTTP 400:

```text
Invalid sort field: market_cap
```

Therefore this endpoint can govern identity/eligibility discovery but cannot supply a demonstrated market-cap-ranked top-1,000 snapshot in one bounded request.

## FMP ranking preflight

The existing configured FMP credential was tested read-only against the documented Company Screener endpoint with US, active, non-ETF filters. The endpoint returned HTTP 402:

```text
Restricted Endpoint: This endpoint is not available under your current subscription
```

No FMP subscription or entitlement was changed. The FMP screener must not be treated as an available ranking source.

## Required decision

To fill the approved 1,000-security hot-set capacity, provide one of:

1. an approved licensed market-cap/liquidity ranking feed and its workload credential;
2. a provenance-retained ranked snapshot file/dataset from the platform owner; or
3. explicit approval to purchase/enable an FMP screener entitlement.

Until then, the 125-security compatibility cohort remains a correct S2 shadow plan, not a top-1,000 claim.
