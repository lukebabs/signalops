# API Notes

Use this folder for MarketOps Daily Market Surveillance API notes that supplement `docs/api.md`.

Current MarketOps-specific endpoints:

- `GET /v1/tenants/{tenant_id}/marketops/assets`
- `GET /v1/marketops/dsm/artifacts`
- `GET /v1/marketops/dsm/artifacts/{artifact_id}`

MarketOps signal, alert, insight, raw-event, and normalized-event views use the shared `/v1/*` APIs with metadata filters:

- `app_id=marketops`
- `domain=market_data`
- `use_case=daily_market_surveillance`

Authentication is enforced by the gateway when enabled. Positive live API validation requires a real bearer token; unauthenticated probes should return `401 unauthorized`.

Proposed notes:

- `graph_proposal_api.md`: proposed G079 graph proposal list/detail/decision API boundary.
