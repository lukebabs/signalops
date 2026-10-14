# S4 Revision Review Deployment Evidence — 2026-08-13

## Releases

- API/projection commit: `5a1c683`.
- Asset-detail UI commit: `5d75af6`.
- Migration: `000107_subscriber_gateway_eod_revision_review`.
- Deployment source: clean archives of the committed revisions; unrelated SRI working-tree changes were excluded.

## Verification

- Migration `000107` applied successfully.
- The gateway-safe review projection returned all 12 immutable delta rows: six per AAPL and NVDA.
- Each symbol has exactly two `review_required` provider revisions (VWAP and volume); OHLC rows remain informational/unchanged.
- Focused Go API/storage tests passed and the clean gateway image ran `go test ./...` successfully.
- Frontend TypeScript/Vite production build passed.
- Public `/healthz` and the application shell each returned `200` after release.

The existing full frontend suite still has five unrelated failures in navigation, theme-preference, and default asset-filter tests. The revision-review panel is not involved; the production build typechecked it successfully.
