# S4 Provider Revision Policy

Status: implemented for the S4 canary evidence; expansion remains blocked pending review.

## Policy

Massive can revise completed-session aggregates after an initial capture. SignalOps therefore treats a provider response as a versioned observation, not an in-place correction.

| Concern | Current policy |
|---|---|
| Initial tenant-local capture | Immutable evidence; retained exactly as used by historical tenant-local processing. |
| Later global re-observation | Immutable second observation with its own provider response, run, fingerprint, quality, algorithm version, and timestamp. |
| Existing tenant-local output | Remains authoritative during review; it is not overwritten by the global response. |
| Revision deltas | Stored field-by-field for OHLCV; changed volume/VWAP are `review_required`, unchanged OHLC fields are `informational`. |
| Future canonical selection | Explicit later decision: `initial`, `latest_provider`, or dual-version consumer choice. It cannot be inferred from a parity mismatch. |

This implements `hold_initial_pending_review`. It prevents silent historical restatement while preserving current provider data for controlled evaluation.

## Transparency contract

Every revision comparison must identify the global asset, session, provider, two immutable observation IDs, source event/run, payload fingerprints, algorithm version, quality state, and per-field before/after values. A revision can be explained without changing any MarketOps signal, historical outcome, or user watchlist.

The S4 AAPL/NVDA comparison is backfilled into this ledger from the already persisted live baseline and tenant-local parity evidence. It makes no additional Massive request.
