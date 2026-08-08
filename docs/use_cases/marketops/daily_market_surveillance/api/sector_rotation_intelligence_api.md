# Sector Rotation Intelligence API

Status: implemented research-only API surface.

All SRI responses are tenant-scoped and require the same gateway authentication as other MarketOps endpoints when authentication is enabled. A request without a bearer token should return 401 in an auth-enabled environment.

## Endpoints

| Endpoint | Purpose |
|---|---|
| GET /v1/marketops/sectors?tenant_id=TENANT | active segment registry |
| GET /v1/marketops/sectors/rankings?tenant_id=TENANT | latest usable cross-sectional sector and industry snapshots |
| GET /v1/marketops/sectors/{segment_id}?tenant_id=TENANT | latest usable snapshot for one segment |
| GET /v1/marketops/sectors/{segment_id}/history?tenant_id=TENANT | persisted SRI history for one segment |
| GET /v1/marketops/assets/{symbol}/sector-context?tenant_id=TENANT | informational mapped SRI context for a MarketOps asset |

The rankings endpoint accepts optional segment_type=sector or segment_type=industry and state=LEADING, IMPROVING, NEUTRAL, WEAKENING, or LAGGING filters.

## Snapshot semantics

A snapshot includes:

- segment identity and session_date
- state, composite score, relative-strength score, momentum score, acceleration, and rank
- evidence quality and quality_state
- quality flags, calculation components, and input provenance
- algorithm and configuration versions

quality_state=usable means the primary ETF and all three benchmarks satisfied the 61-session minimum. quality_state=partial means the result is a non-ranked availability record, not a low score.

When a more recent partial snapshot coexists with a prior usable snapshot, the rankings, single-segment, and asset-context endpoints select the latest usable snapshot. History remains complete and includes both records for audit.

Every rankings response carries research_only=true and an evidence note stating that Foundation results do not claim sector rotation, breadth, diffusion, flows, or a trade recommendation.

## Asset context mapping

The asset-context endpoint maps an active MarketOps asset by industry first, then sector, to the implemented SRI segment registry. It responds with:

- informational and a snapshot when a usable or partial mapped context exists
- not_ready when a mapped segment has no snapshot
- unmapped when the asset metadata does not match an implemented segment

The mapping adds research context only. It does not change asset state, produce a signal, or modify another algorithm.

## Example

~~~bash
curl -H "Authorization: Bearer TOKEN" \
  "http://localhost:18000/v1/marketops/sectors/rankings?tenant_id=tenant-local&segment_type=sector"
~~~

Do not infer a buy/sell, rotation-in/rotation-out, or future-performance assertion from any response.
