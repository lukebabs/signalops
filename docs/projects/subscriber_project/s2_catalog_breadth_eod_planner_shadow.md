# Sprint S2 ºw^~)Þt Catalog Breadth and EOD Planner Shadow

Status: migration and initial zero-selection shadow plan deployed; governed reference admission and activation-queue event evidence remain outstanding. No provider collection, scheduler, browser route, tenant projection, or list membership write is enabled.

## Scope delivered

Migration `000092_subscriber_global_eod_planner_shadow` adds four platform-owned, additive records:

- immutable global-asset eligibility decisions;
- auditable top-1,000 EOD hot-set plan runs;
- deterministic selected-member records; and
- deduplicated future coverage-activation requests.

The worker gateway has no grant to any of these tables. The catalog-sync workload may write eligibility decisions; the future global-EOD workload may write shadow plans and activation-request state. Neither is an enabled service.

## Governed eligibility

An `eligible` decision is rejected unless its retained provider evidence confirms all of:

- `country_code = US`;
- `security_type = common_stock`;
- `exchange_listed = true`;
- `provider_eligible = true`; and
- `is_active = true`.

The decision is immutable. It records its policy version, provider-reference time, reason, evidence, provenance, actor, and decision time. A deferred decision leaves the global asset as `discovered`; an ineligible decision changes it to `ineligible`; only verified evidence promotes it to `eligible`.

The S1 seed has 178 identities, all intentionally `discovered`. Its compatibility metadata is insufficient to declare them US common stocks. S2 therefore does not infer eligibility from ticker, asset type, or historical processing.

## Deterministic hot-set shadow

The pure planner considers up to 10,000 global candidates, selects only eligible assets with an active source link, then sorts by:

1. lowest active compatibility-source rank;
2. stable global asset ID.

Capacity is bounded from 1 through 1,000. Every exclusion is counted as `not_eligible`, `no_active_source`, or `capacity`. The persisted plan is always `execution_mode = shadow`; it cannot modify the coverage registry, call Massive, enqueue work, or start a worker.

After migration and future global-EOD workload preflight, an operator may record a plan:

```sh
go run ./cmd/subscriber-global-eod-shadow-planner --execute \
  --capacity 1000 \
  --actor subscriber-global-eod-reconciler \
  --correlation-id s2-shadow-<change-id>
```

A plan with zero selected members is correct until governed reference evidence admits eligible assets.

## Cold activation queue

The S2 queue is global and deduplicated by global asset ID while a request is `queued` or `warming_up`. It preserves the origin kind, tenant/subject/list coordinates where a later authorized S3 membership supplies them, request key, policy version, provenance, and state. No API or browser writer exists in S2. S3 may create an idempotent request only after server-side list authorization and entitlement/quota evaluation succeed.

## S2 exit evidence

1. Reference-import evidence shows every admitted asset has valid US-common-stock provider evidence.
2. The planner report records candidate, eligible, selected, excluded, and reason counts at capacity 1,000.
3. Selected ranks and IDs can be replayed exactly from retained input data.
4. No coverage row or plan is in `enabled` mode; no provider request or scheduled job changed.
5. Duplicate cold-activation requests coalesce to one global active request.

## Rollback

Do not invoke the manual planner or any future admission import. Existing MarketOps reads, jobs, coverage, and provider behavior remain untouched. Preserve the immutable eligibility and plan evidence; later S3/S4 work must be disabled before any production fallback considers removal.
