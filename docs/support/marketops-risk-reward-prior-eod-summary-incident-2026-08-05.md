# MarketOps Risk/Reward Prior-EOD Summary Incident — 2026-08-05

## Summary

The MarketOps Assets view showed **Awaiting prior EOD** for some active assets, including NVDA, AAPL, and MSFT, even though their completed-session Risk/Reward calculations existed. The message described a missing day-over-day comparison; it did not mean that the current EOD calculation was absent.

## Impact

- Current-session Risk/Reward direction, score, confidence, and risk level remained available.
- The Assets view could not consistently show improvement, regression, or unchanged status for every active asset.
- The issue became visible after the universal active universe grew to 115 assets.

## Root Cause

The Assets summary endpoint selected Risk/Reward history from a single globally limited raw `algorithm_results` query with a limit of 200 rows. A completed session now produces 115 result rows. The query therefore returned all or most of the latest session but only a non-deterministic subset of the prior session. An asset absent from that truncated prior-session subset had no `previous_score` or `score_change`, so the UI rendered **Awaiting prior EOD**.

The authoritative `marketops_risk_reward_snapshots` projection already contained the required records. On 2026-08-05, both 2026-08-04 and 2026-08-03 snapshots existed for all 115 active assets.

## Correction

The Assets Risk/Reward summary endpoint now:

1. Reads `marketops_risk_reward_snapshots` for the active-symbol set rather than a global raw-result page.
2. Requests a bounded 21-day history and a capacity proportional to the active universe (minimum 1,000 rows).
3. Selects the best eligible revision per asset and trading session using usable-input count, confidence, and creation time.
4. Calculates evolution from the two latest persisted trading sessions for that asset.
5. Retains the raw-result path only as a compatibility fallback when a repository does not implement the snapshot projection.

## Prevention and Verification

- A regression test creates 115 active symbols and two sessions per symbol, asserting that every summary receives a prior-session score change.
- The endpoint must not use a global raw-result page for per-asset day-over-day Risk/Reward evolution.
- Operations verification: after a post-close run, query the Assets Risk/Reward endpoint and confirm that all eligible active assets with two persisted snapshots expose `previous_trade_date` and `score_change`.

## Analyst Semantics

**Awaiting prior EOD** is now reserved for an asset that genuinely has only one eligible persisted Risk/Reward session. It does not signal an EOD processing failure by itself.
