# Legacy Hot Cohort Shadow Evidence — 2026-08-16

Status: shadow-only foundation applied; scheduler and UI cutover blocked pending a real explicit watchlist selection and a matching dual-run.

## Temporary compatibility cohort

Migration `000144_subscriber_legacy_hot_cohort_shadow` introduces a platform-governed temporary cohort seeded from `MarketOps Legacy Default` (`sublist-tenant-local-legacy-default`). It preserves all 132 memberships without changing that durable tenant default.

The cohort does not override the product rule that hot intraday work requires an eligible global asset. At seed time:

| State | Count |
| --- | ---: |
| preserved legacy members | 132 |
| active, globally eligible grandfathered members | 125 |
| deferred catalog-ineligible members | 7 |

Deferred members are `ARM`, `ASML`, `BABA`, `HSBC`, `NVS`, `TCEHY`, and `TSM`. They remain visible in the legacy default and retain their historical evidence; they are not silently reclassified or scheduled for central intraday polling.

## Immutable dual-run evidence

The existing `subscriber_global_hot_intraday_assets` watchlist selector remains unmodified. The new comparison function reads only aggregate asset membership and watcher counts, records no tenant/user/list identity, and does not call a provider or enqueue work.

First live comparison:

- shadow run: `subhotshadow-3e6191389db7d0fc9795e2bc6a2b710a`
- correlation: `legacy-hot-132-selector-shadow-2026-08-16`
- fingerprint: `fd478f7ce9ee2bd5f67da9bf9e01cece`
- status: `mismatch`
- grandfathered eligible cohort: 125
- watchlist-derived selector: 0
- shared/cohort-only/selector-only: `0 / 125 / 0`

This mismatch is expected and is evidence, not a failure to preserve data. There are no saved `subscriber_watchlist_context_preferences` selections. The Asset experience can resolve a display fallback, but the hot policy intentionally requires a saved explicit selection before it permits intraday demand.

## Exit condition

An authorized `tenant-local` analyst must explicitly choose **MarketOps Legacy Default** as their MarketOps watchlist. That saved selection will contribute its 125 eligible assets to the selector. Re-run the shadow comparison with `require_match=true`; it must report `125 / 125 / 125 / 0 / 0` before any scheduler or reader cutover is considered.

No selection is fabricated by this migration, because doing so would violate the explicit-user-selection hot-tier rule. No scheduler, task, API, UI behavior, provider request, or legacy membership changed.
