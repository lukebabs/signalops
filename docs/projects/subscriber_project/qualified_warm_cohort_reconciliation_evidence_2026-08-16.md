# Qualified warm-cohort reconciliation evidence — 2026-08-16

## Purpose

Fill the centrally governed warm EOD cohort to its approved capacity of 1,000
active, exchange-listed US common stocks. This is distinct from the
watchlist-driven hot intraday tier and does not broaden the deferred
ADR/preferred/special-security policy.

## Controlled result

| Measure | Result |
| --- | ---: |
| Retained ranked candidate pool | 1,500 distinct symbols |
| Source rows examined | 1,501 |
| Duplicate source symbols skipped | 1 |
| Source checksum | `3eeabf25c0fedc02bceb0435da301631ddc92f64cf38f68d772fc8623939b272` |
| Qualified warm assets activated | **1,000** |
| Lowest source rank in active plan | 1 |
| Highest source rank in active plan | 1,129 |
| Activated plan | `subeodplan_8aba2bb1bc865f626ee5a55e` |
| Activation policy | `subscriber-warm-eod-v2` |
| Activation correlation | `subscriber-qualified-warm-20260816T042844Z` |

The plan was recorded at 2026-08-16 04:32:29 UTC and the new warm-set
activation at 04:32:29 UTC. The active `subscriber_global_warm_eod_assets`
projection then returned exactly 1,000 assets.

## Scope and safety

- Massive was used only for ranked reference/eligibility validation; the action
  did not request EOD prices, options, intraday quotes, or FMP fundamentals.
- Each reference lookup uses the worker's single-attempt path. A failed lookup
  is retained as `deferred`; it is not inferred, normalized, or retried.
- The new activation affects central warm EOD coverage only. It does not alter
  any tenant watchlist or enable any hot intraday asset.
- The selected plan remains restricted to the current US-common-stock policy.
  ADRs, preferred shares, special security classes, and unresolved provider
  symbols remain deferred roadmap work.

## Execution incident and remediation

The deployment-log relay detached the first long-running agent invocation
without returning its final result. A second invocation was then started while
the first was still qualifying candidates. The second container was stopped as
soon as it was observed. The first invocation, correlation
`subscriber-qualified-warm-20260816T042844Z`, completed the plan and activation.
The stopped duplicate recorded partial append-only eligibility evidence under
`subscriber-qualified-warm-20260816T042935Z`; it did not create a plan,
activation, EOD pull, intraday pull, or FMP request.

The launcher now holds an exclusive root-owned `flock` for the full validation,
planning, and activation transaction. Any future concurrent invocation exits
before provider work with
`subscriber_qualified_warm_cohort_reconciliation_already_running`.
