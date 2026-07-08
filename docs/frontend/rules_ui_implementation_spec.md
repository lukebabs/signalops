# SignalOps Frontend Rules UI Implementation Specification

Status: ready for frontend-agent implementation  
Author: Codex  
Date: 2026-07-08  
Backend baseline: G040 `c2ed060 Add rules catalog foundation`

## Purpose

Implement the first Rules UI page in the existing SignalOps `web/` frontend. This is a read-only
operator view backed by the rules catalog API added in G040.

The frontend agent must not invent rule records, execution state, signal outcomes, or editable
rule management. The current backend exposes catalog metadata only.

## Current Backend Contract

Rules are available through the gateway and web proxy:

```http
GET /v1/tenants/{tenant_id}/catalog/rules?limit=50
```

Local tenant for the current UI:

```text
tenant-local
```

Expected response envelope:

```json
{
  "rules": [
    {
      "tenant_id": "tenant-local",
      "rule_id": "rule-marketdata-eod-price-quality",
      "rule_name": "Market Data EOD Price Quality",
      "description": "Flags Massive EOD equity records with missing or non-positive close prices before downstream signal evaluation.",
      "rule_type": "quality_check",
      "severity": "medium",
      "status": "active",
      "version": 1,
      "source_id": "src-massive",
      "pipeline_id": "pipeline-massive-raw-ingest",
      "dataset_scope": ["equity_eod_prices"],
      "entity_scope": ["ticker"],
      "expression": {
        "language": "json_logic",
        "conditions": [
          {"field": "close", "operator": "exists"},
          {"field": "close", "operator": ">", "value": 0}
        ],
        "mode": "all"
      },
      "actions": ["emit_alert", "mark_event_quality_failed"],
      "metadata": {
        "provider": "massive",
        "formerly": "polygon.io",
        "execution": "catalog_only",
        "streaming": false
      },
      "created_at": "2026-07-08T00:00:00Z",
      "updated_at": "2026-07-08T00:00:00Z"
    }
  ]
}
```

Status values: `active`, `inactive`, `deprecated`.  
Severity values: `info`, `low`, `medium`, `high`, `critical`.

Backend references:

- `docs/api.md`, "Stream Catalog API / Rules"
- `internal/api/router.go`, rules route and DTO
- `internal/storage/storage.go`, `CatalogRuleRecord`
- `migrations/000004_catalog_rules.up.sql`, local seed

## Existing Frontend Pattern

Use the current `web/` implementation. Do not create a new frontend package.

Follow these existing patterns closely:

- Sources page: `web/src/routes/SourcesRoute.tsx`
- Pipelines page: `web/src/routes/PipelinesRoute.tsx`
- API client: `web/src/api/client.ts`
- Query hooks: `web/src/api/queries.ts`
- Router: `web/src/router.tsx`
- Shell navigation: `web/src/components/DashboardShell.tsx`
- Shared UI: `MetricTile`, `StatusBadge`, `JsonViewer`, `States`

The web app is already served through Compose at:

```text
http://localhost:15173
```

The nginx web proxy already forwards `/v1/` to the gateway, including `/v1/tenants/.../catalog/rules`.

## Required Implementation

### 1. Types

Update `web/src/types.ts`:

```ts
export type CatalogRuleStatus = 'active' | 'inactive' | 'deprecated';
export type CatalogRuleSeverity = 'info' | 'low' | 'medium' | 'high' | 'critical';

export interface CatalogRule {
  tenant_id: string;
  rule_id: string;
  rule_name: string;
  description: string;
  rule_type: string;
  severity: string;
  status: string;
  version: number;
  source_id?: string;
  pipeline_id?: string;
  dataset_scope: string[];
  entity_scope: string[];
  expression: unknown;
  actions: string[];
  metadata: unknown;
  created_at: string;
  updated_at: string;
}

export interface CatalogRulesResponse {
  rules: CatalogRule[];
}
```

Keep `source_id` and `pipeline_id` optional in TypeScript because the API omits empty values.

### 2. API Client

Update `web/src/api/client.ts`:

- Import `CatalogRulesResponse`.
- Add:

```ts
listCatalogRules: (tenantId = 'tenant-local', limit = 50) =>
  get<CatalogRulesResponse>(`/v1/tenants/${encodeURIComponent(tenantId)}/catalog/rules`, { limit }),
```

### 3. Query Hook

Update `web/src/api/queries.ts`:

- Add query key:

```ts
catalogRules: (tenantId: string, limit: number) => ['catalog-rules', tenantId, limit] as const,
```

- Add hook:

```ts
export function useCatalogRules(tenantId = 'tenant-local', limit = 50) {
  return useQuery({
    queryKey: queryKeys.catalogRules(tenantId, limit),
    queryFn: () => api.listCatalogRules(tenantId, limit),
  });
}
```

### 4. Route

Create:

```text
web/src/routes/RulesRoute.tsx
```

The page must:

- Use `tenant-local` until auth/tenant selection exists.
- Fetch rules with `useCatalogRules('tenant-local', 50)`.
- Render loading, error, and empty states using existing shared components.
- Render only data returned by the API.
- Avoid any edit/create/delete controls.

Required page metrics:

- Registered Rules: `rules.length`
- Active Rules: count where `status === 'active'`
- Rule Types: distinct `rule_type`
- Critical/High: count where severity is `critical` or `high`

Required table columns:

- Rule: icon + `rule_name`, `rule_id`, description, version
- Type
- Severity
- Scope: source, pipeline, datasets, entities
- Actions
- Status
- Updated

Recommended icon: `ShieldCheck` or `ListChecks` from `lucide-react`.

Use `StatusBadge` for `status`. If there is no existing severity badge component, render severity
as restrained text or a small local badge. Keep styling consistent with Sources/Pipelines and avoid
introducing a new color-heavy visual system.

Render JSON sections below the table:

- "Rule Expressions": array of `{ rule_id, expression }`
- "Rule Metadata": array of `{ rule_id, metadata }`

Use the existing `JsonViewer`.

### 5. Router and Navigation

Update `web/src/router.tsx`:

- Lazy-load `RulesRoute`.
- Add `/rules`.
- Add it to the route tree near Sources/Pipelines.

Update `web/src/components/DashboardShell.tsx`:

- Add a `Rules` nav item near `Sources` and `Pipelines`.
- Use a lucide icon, preferably `ShieldCheck` or `ListChecks`.

## UX Requirements

This is an operational UI, not a marketing screen.

Keep the page dense, scannable, and consistent with the current Sources/Pipelines layouts:

- No hero sections.
- No decorative gradients or oversized cards.
- No nested cards.
- Tables must remain readable at desktop widths.
- On small screens, allow horizontal table overflow instead of squeezing text until it overlaps.
- Long arrays should render as comma-separated text with a fallback `-`.
- Optional `source_id` / `pipeline_id` should render as `-` when absent.
- Timestamps must use existing `formatUtc`.

Do not add visible instructional copy about how to use the app.

## Non-Goals

Do not implement these in this frontend gate:

- Rule creation, editing, deletion, enable/disable toggles, or version publishing.
- Rule execution history.
- Signal/insight result views.
- Rule test runner.
- Rule expression builder.
- Tenant picker.
- Auth or role enforcement.
- Mock data fallback.

The backend currently exposes catalog visibility only.

## Validation Requirements

Run from the repo root unless noted.

Frontend validation:

```bash
cd web && npm test
cd web && npm run build
cd web && npm audit --json
```

Compose validation:

```bash
docker compose build web
docker compose up -d web
curl -fsS http://localhost:15173/rules
curl -fsS 'http://localhost:15173/v1/tenants/tenant-local/catalog/rules?limit=10'
docker compose ps web
```

Expected API verification:

- `http://localhost:15173/rules` returns the SPA shell.
- The proxied rules API returns `tenant-local/rule-marketdata-eod-price-quality`.
- `npm audit --json` reports zero vulnerabilities.

## Documentation Requirements

After implementation, update:

- `docs/build_journal.md`
- `docs/gate_audit.md`

Use a new gate:

```text
G041: Frontend Rules Catalog Page
```

The journal/audit entry must include:

- UTC timestamp.
- Files changed.
- Validation commands run.
- Live verification result.
- Any issues found/resolved.

## Acceptance Criteria

G041 is complete when:

- `/rules` is routable in the web app.
- The Rules nav item appears in the app shell.
- The page fetches `GET /v1/tenants/tenant-local/catalog/rules?limit=50`.
- The seeded rule is rendered without mock data.
- Loading, error, and empty states are present.
- Metrics, table, expression JSON, and metadata JSON render correctly.
- `npm test`, `npm run build`, `docker compose build web`, `docker compose up -d web`, web route curl, proxied API curl, and `npm audit --json` pass.
- `docs/build_journal.md` and `docs/gate_audit.md` record G041 with timestamped evidence.
