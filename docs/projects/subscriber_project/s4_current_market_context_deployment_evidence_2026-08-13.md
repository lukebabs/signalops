# S4 Current Market Context UI Deployment Evidence — 2026-08-13

## Releases

- API and projection: `6e6e6ce`.
- Asset-detail presentation: `25075ae`.
- Deployment source: clean archives of the respective commits; unrelated SRI working-tree changes were excluded.

## Delivered experience

In **MarketOps → Assets → select asset → Overview**, the **Current verified EOD** card displays the latest usable global provider revision when one is available. It identifies the provider, session date, OHLC, VWAP, volume, as-of timestamp, `global_reobservation` role, and selection-policy version.

The card explicitly states that it is live context and separate from historical assurance and backtests. If the asset has no usable re-observation, it displays that condition rather than replacing it with historical data.

## Verification

- Migration `000106_subscriber_gateway_current_eod_context` applied successfully from a clean archive.
- The narrow view resolved two current-context records, AAPL and NVDA, each with usable `global_reobservation` provenance.
- Focused Go API/storage tests passed, and the clean gateway build ran `go test ./...` successfully.
- Frontend TypeScript/Vite production build passed.
- Public `GET /healthz` returned `200`; the public application shell returned `200` after the web release.

The full frontend suite currently has five unrelated failures in existing navigation, theme-preference, and default asset-filter tests. The new card is not referenced by those failures, and the production typecheck/build passed.
