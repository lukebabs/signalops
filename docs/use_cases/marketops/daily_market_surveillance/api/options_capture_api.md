# Options Capture API

G142 exposes read-only, tenant-scoped inspection of prospective point-in-time options capture quality.

## List Captures

`GET /v1/tenants/{tenant_id}/marketops/options/captures`

Optional query parameters:

- `symbol`
- `session_start` inclusive, `YYYY-MM-DD`
- `session_end` exclusive, `YYYY-MM-DD`
- `status`: `analytics_ready`, `partial`, `no_data`, or `failed`
- `analytics_ready`: `true` or `false`
- `limit`

Response envelope: `{ "options_captures": [] }`.

## Capture Detail

`GET /v1/tenants/{tenant_id}/marketops/options/captures/{capture_id}`

Response envelope: `{ "options_capture": {} }`.

Each record includes deterministic identity, symbol/session, provider lineage, terminal status, analytics readiness, usable-field counts, the number of required surface cells found, quality reasons, bounded execution metrics, attempts, timestamps, and any terminal error.

These endpoints do not invoke Massive, mutate capture state, or trigger G141.
