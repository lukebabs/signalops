# S4 Historical Assurance Deployment Evidence — 2026-08-13

## Release

- Branch: `subscribers`
- Application commit: `75201ea` (`feat(subscribers): bind SAF backtests to historical EOD`)
- Scope: gateway only; no database migration, web rebuild, scheduler change, provider request, historical backfill, or historical restatement.
- Build source: clean `git archive` of the committed revision, isolated from unrelated working-tree changes.

## Controls released

- SAF effectiveness, observation, and recommendation responses expose the fixed `historical_assurance` selection contract.
- Newly created MarketOps backtests persist that same selection contract in immutable filters and parameters.
- Backtests reject `current_market_context`; later provider revisions cannot silently substitute into a historical run.

## Verification

- Focused tests passed: `go test ./internal/marketops/backtest ./internal/api`.
- The clean image build ran `go test ./...` successfully.
- Gateway was recreated from the clean release image.
- `GET http://localhost:15173/healthz` returned `200`.
- `GET https://signalops.syncratic.io/healthz` returned `200`.
- Unauthenticated `GET /v1/marketops/signal-assurance/effectiveness?tenant_id=tenant-local` returned `401`, preserving the authentication boundary.

An authenticated analyst can now observe `data_selection.usage_context=historical_assurance`, `selected_observation_role=initial_tenant_local_capture`, and `restatement=disabled` on each SAF effectiveness endpoint.
