# Syncratic User API Boundary

Status: integration contract indexed
Use case: MarketOps Daily Market Surveillance

## Purpose

`docs/syncratic_user_api_v1.yaml` is the canonical OpenAPI contract for the Syncratic user-facing facade. It covers Search, Ask, ingestion, graph context, Explorer document pivots, Insights, DDE, Documents, and Threads.

SignalOps should treat this as an external Syncratic API boundary, not as an internal SignalOps route set.

## Environment Contract

Use namespaced environment variables for this boundary:

- `SYNCRATIC_API_BASE_URL`: base URL for the Syncratic user facade, such as `https://portal.syncratic.co`.
- `SYNCRATIC_AUTH_ISSUER`: identity issuer used for Syncratic user-facade tokens.
- `SYNCRATIC_AUTH_MODE`: `api_key` for direct API-key bearer auth, or `token` for token-endpoint acquisition.
- `SYNCRATIC_TOKEN_URL`: token endpoint for obtaining a bearer JWT when token mode is used.
- `SYNCRATIC_TOKEN_GRANT`: configured token grant, for example `password` only when explicitly approved for CLI use.
- `SYNCRATIC_CLIENT_ID`: token client id.
- `SYNCRATIC_CLIENT_SECRET`: Syncratic API key used as the secret credential for the configured non-browser token flow.
- `SYNCRATIC_USERNAME`: user credential for the configured token flow.
- `SYNCRATIC_PASSWORD`: user credential for the configured token flow.

Do not use generic `USERNAME` or `PASSWORD` variables. They collide with common shell, database, and service conventions.

## Auth Boundary

The OpenAPI contract declares `BearerAuth` with JWT bearer tokens. It does not define a login route. Therefore, integration code must not infer token acquisition from the OpenAPI file alone. Token acquisition must come from explicit environment configuration. In this repo, `SYNCRATIC_CLIENT_SECRET` is the Syncratic API key for the configured non-browser token flow. Keep API keys and token material out of logs, docs, committed files, and URLs.

## Initial SignalOps Usage

The first safe MarketOps integration should be read-oriented:

- Search or Ask over operator-accessible Syncratic knowledge when explaining a persisted Syncratic context window.
- Fetch compact Syncratic insights if needed for cross-reference.
- Use graph context only as explicit opt-in supporting evidence.

Do not ingest MarketOps data into Syncratic, write graph state, call privacy-token reveal, or generate operator-facing narratives until a dedicated gate approves the data boundary and retention rules.
## Implemented Client Boundary

SignalOps includes a small internal Go client package at `internal/syncratic/userapi`. It owns:

- loading `SYNCRATIC_*` environment configuration;
- obtaining a bearer JWT from the configured token endpoint;
- sending the Syncratic API key as `client_secret` during token acquisition;
- caching the bearer token in process until shortly before expiry;
- attaching `Authorization: Bearer <api key>` in `api_key` mode or `Authorization: Bearer <token>` in `token` mode;
- read-oriented calls for Search, Ask, and compact Insights listing.

The package must not log API keys, bearer tokens, usernames, passwords, raw retrieval payloads, or long document text. Higher-level MarketOps gates should call this package instead of constructing Syncratic HTTP requests directly.

## Live Auth Probe

Live credential probes on 2026-07-13 showed that the current Syncratic user facade accepts `SYNCRATIC_CLIENT_SECRET` directly as an API key using `Authorization: Bearer <api key>` and `X-API-Key`. The token endpoint variants tested with the configured password/client-credentials shapes returned HTTP `401`.

Recommended current mode: `SYNCRATIC_AUTH_MODE=api_key`.
