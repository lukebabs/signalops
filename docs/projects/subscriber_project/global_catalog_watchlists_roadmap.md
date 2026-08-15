# Global Catalog and Watchlist Roadmap

Status: target architecture and delivery record for the Subscriber Project.

## 1. Target model

The target separates three concerns that are currently too close together:

| Concern | Target owner | Examples | Must not contain |
|---|---|---|---|
| Global market catalog | Platform | asset identity, Massive reference metadata, eligibility, coverage state, corporate-action lifecycle | tenant preferences or user membership |
| Shared market intelligence | Platform | EOD prices, normalized events, features, algorithm outputs, quality and provenance | duplicate per-tenant copies |
| Subscriber experience | Tenant and user | entitlement, tenant default list, user-private lists, saved views, alert preferences | a second asset master or direct provider data |

A global asset has one durable platform identifier and one normalized symbol identity. A user’s list entry stores that global ID, list ID, and list-specific presentation metadata only. It never copies the global identity, pricing history, or algorithm result.

## 2. Current implementation and migration intent

Today, marketops_asset_universe is tenant-owned. The current universal view projects selected tenant groups, and the direct asset onboarding path validates a ticker with Massive before writing it into the tenant analyst_watchlist group. This is correct for the present single-tenant operating model, but it is not the source-of-truth model for a subscription product.

The migration objective is compatibility first:

1. Introduce a global catalog and coverage registry without changing existing asset reads.
2. Seed it from the existing MarketOps universe and Massive reference metadata.
3. Map current tenant universe rows to tenant-default or migrated user-list membership as appropriate.
4. Change collection and calculation planners to resolve the global active coverage set.
5. Change tenant read routes to authorize and project global results through memberships and entitlements.
6. Retire direct tenant-owned asset creation as the normal user path. Adding an eligible cold global asset to a tenant-default or private list creates one idempotent, governed global coverage-activation request; it never creates a tenant-local asset or provider pull from the browser.

No destructive migration is allowed until the global catalog, membership mapping, and result parity are proved.

## 3. Global catalog and coverage lifecycle

The catalog begins with US common stocks that Massive identifies as active and eligible under the platform’s provider agreement. Catalog inclusion and data coverage are different states.

| State | Meaning | User effect |
|---|---|---|
| discovered | Reference identity was found but has not passed governance checks. | Not searchable to subscribers. |
| eligible | Identity, market, security type, and provider policy passed. | May be made searchable when coverage is active. |
| backfilling | Bounded EOD history is being established. | Searchable only if the product permits partial readiness; coverage is labelled. |
| queued | A list membership requested activation and the global planner has accepted it, subject to the coverage budget. | The asset remains visible in the list with an explicit pending reason. |
| warming_up | The first central collection has begun but the selected analysis does not yet have its required evidence. | Current coverage is shown truthfully; no missing evidence is rendered as a signal. |
| active | Daily EOD collection and baseline algorithms are scheduled. | Available for entitled lists and research views. |
| degraded | A required collection or quality condition failed. | Still identifiable; state/date/coverage are shown truthfully. |
| suspended | Delisted, ineligible, or provider-blocked. | Cannot be newly added; retained history/provenance remains auditable. |

Global identity should carry source ID, provider symbol, name, exchange, security type, sector and industry where supplied, effective timestamps, and immutable reference provenance. Symbol changes and corporate actions must use a lifecycle relationship instead of overwriting identity or losing prior evidence.

A discovery process may use issuer constituent snapshots, indexes, or Massive reference enumeration as inputs, but Massive eligibility and governance validation decide whether an asset enters the global active set.

## 4. EOD processing plane

The initial continuous service is EOD-first:

1. The global coverage planner selects active assets for the completed market session.
2. A bounded reconciliation queue fetches each symbol from Massive, applies retry and quota controls, and writes the canonical raw and normalized evidence once.
3. The global feature, market-state, and price-based algorithm pipeline processes that canonical evidence once per symbol/session.
4. Completeness, quality, session date, provider use, and failure reasons are persisted as platform operational facts.
5. Tenant-facing read APIs authorize access, then project those shared persisted facts without collecting again.

This must be batchable and observable. A full US common-stock catalog is materially larger than the current MarketOps universe, so the scheduler needs explicit batch size, deadline, retry budget, provider-request budget, queue age, and coverage-completion metrics. A failed or delayed asset must be recorded as degraded or deferred, never converted into a synthetic neutral algorithm result.

Options, intraday quotes, and fundamentals are not prerequisites for the baseline global EOD pipeline. An algorithm requiring unavailable enrichment must retain a quality/coverage reason and must not invent inputs.

## 5. Tenant and user experience

### Authorization

A JWT tenant claim and tenant access grants remain the boundary for tenant data. The subscriber model adds authorization checks for access to global catalog projections and coverage products; it does not give one tenant visibility into another tenant’s list memberships, settings, or usage.

### Subscriber access-control readiness gate

The current platform already supplies the required foundation: Keycloak/OIDC authentication; gateway verification of JWT issuer, signature, audience, expiry, and tenant claim; immutable subject identity; registered-use-case read/write grants; super-admin grant management; and grant audit records. The Subscriber Project extends those primitives; it does not introduce another identity provider or authorization service.

The enhancement must add:

- canonical tenant resolution from the authenticated principal, with rejection of any query, path, or JSON-body tenant value that disagrees with it;
- subject-owned private-list and tenant-administrator default-list permissions, enforced server-side for every list and membership read or write;
- entitlement, product-tier, and quota decisions for catalog search, EOD activation, Options demand, and other gated enrichments, separate from basic MarketOps read/write grants;
- tenant-authorized projections from shared global records, which expose neither another tenant's memberships nor their usage or demand;
- scoped service identities for schedulers and workers that process shared data without impersonating a browser user; and
- an explicit defense-in-depth decision for database row-level security, including its operational and migration model if adopted.

The project may exit this gate only when:

1. Production-like OIDC/JWKS configuration is enabled and a valid browser session reaches protected APIs with the expected grant.
2. Every tenant-bearing request source—path, query, and JSON body—is bound to the principal or rejected; client-supplied actor identity cannot override the principal.
3. List, membership, catalog-projection, entitlement, and quota authorization is server-enforced and covered by direct-API negative tests.
4. Cross-tenant read, write, enumeration, and privilege-escalation attempts are rejected and exercised in automated integration tests.
5. Grant, entitlement, list-administration, and quota-decision audit records retain actor, tenant, action, prior/new state where applicable, and correlation/provenance.
6. Worker identities have least-privilege scopes and cannot use subscriber-facing administration paths.
7. Feature flags and rollback preserve the current tenant-owned experience without deleting shared evidence or subscriber preferences.

### Coverage activation

The platform maintains a centrally governed hot EOD coverage set of the top 1,000 eligible US common stocks. Those assets are normally `active` before any subscriber selects them. The broader governed catalog may contain eligible but cold assets.

When a tenant administrator adds an eligible cold asset to a tenant-default list, or a user adds one to a private list, SignalOps must create an idempotent global coverage-activation request keyed by the global asset ID. The request records the requesting tenant and subject for audit and entitlement/quota evaluation, but the worker resolves a single global demand record and never exposes another tenant's membership or demand.

The central coverage service—initially interoperating with the current `tenant-local` MarketOps operating plane during migration—admits the unique asset into global coverage and moves it through `queued`, `warming_up`, and `active` (or a truthful deferred/degraded state). It must not clone assets, prices, features, or algorithm results into `tenant-pilot-b` or any other subscriber tenant. The browser never calls Massive directly.

### Lists

Each tenant has one or more administrator-managed default lists. Each user may have one or more private lists. At launch, the UI shows the tenant default and the authenticated user’s private lists only.

Membership operations are idempotent:

- Adding an asset references its global asset ID. If the asset is cold, the membership writes successfully and creates the idempotent global activation request; it does not wait for a browser provider call.
- A cold or warming asset displays its coverage state, reason, first successful EOD date, and available evidence instead of an invented score.
- Adding an existing membership returns the existing item without a duplicate.
- Removing a membership never deletes the global asset or shared intelligence.
- Deactivating or suspending a global asset preserves existing memberships but shows its coverage state and blocks new additions under policy.
- Tenant administrators may manage tenant default lists; users may manage only their private lists.

The normal user flow is catalog search, select asset, choose list, and add. A hot asset is immediately projected from central coverage. A cold eligible asset is added to the selected list and queued for globally deduplicated activation. Neither flow calls Massive synchronously from the browser or creates a tenant-local market-data asset.

## 6. Dynamic EOD options-demand planner

Options are scarce, provider-sensitive enrichment. User watchlists can influence demand, but each user must not cause a separate option-chain capture.

### History and readiness policy

The options pipeline captures chains prospectively. It does not reconstruct historical option chains when a global asset is first selected. The first successful EOD capture establishes the asset's shared options-history start; every entitled watcher subsequently reads that same centrally retained history.

The current runner uses a **10-calendar-day** lookback by default and may be configured from 1 through 60 calendar days. Its persisted window name is currently `10_trade_days`, but that label must not be interpreted as a guarantee of 10 trading sessions: the effective evidence consists only of captured session dates inside the calendar-day lookback. Holidays, failed runs, quota deferrals, late activation, and provider gaps can all make the available history shorter.

The subscriber interface and coverage API must therefore return, per global asset:

- `history_start_date`, latest successful capture date, captured-session count in the selected window, configured lookback days, and window-label/version;
- a readiness state of `warming_up` when the requested view needs more captured sessions than are available, rather than treating sparse history as a signal;
- a truthful reason such as `newly_activated`, `deferred_quota`, `provider_gap`, or `collection_failed` where applicable; and
- immutable capture/run provenance sufficient to explain which chain snapshots contributed to an analysis.

`analytics_ready` means the minimum evidence requirement of the selected analysis is met; it does not claim a fully backfilled options history. A newly selected asset may therefore show current chain facts before longer-horizon comparison statistics are ready. No missing or warm-up condition may be rendered as bullish, bearish, neutral, or zero.

At a defined EOD cutoff:

1. Snapshot active tenant-default and private-list memberships for tenants/users with the options entitlement.
2. Filter out non-optionable, suspended, and otherwise ineligible global assets.
3. Union and deduplicate requests by global asset ID.
4. Persist demand evidence: session date, asset ID, eligible tenant count, eligible watcher count, entitlement tier, and membership snapshot/version.
5. Apply the daily symbol/contract/page/request budgets deterministically.
6. Run one bounded options capture per selected global asset.
7. Persist a central coverage outcome: selected, warming_up, analytics_ready, partial, no_data, deferred_quota, blocked_entitlement, or failed.
8. Authorize the same stored coverage/result to every entitled subscriber who watches the asset.

If demand exceeds budget, prioritization is deterministic:

1. Higher subscribed product tier.
2. Guaranteed per-tenant allocation for an active entitled tenant.
3. Higher cross-tenant and watcher demand.
4. Prior deferred age to prevent starvation.
5. Stable global asset ID as the final tie-breaker.

The planner reports demand count, selected count, deferred count, requests, contracts, quality distribution, history/readiness distribution, and per-tier allocation. The UI shows whether an asset was selected, warming up, deferred by quota, not optionable, or had incomplete provider data; missing coverage is never presented as bullish, bearish, neutral, or zero.

## 7. Public interface direction

New subscriber-facing API surfaces should be tenant-scoped projections over global records:

| Capability | Direction |
|---|---|
| Catalog search | Search globally governed eligible/active assets; return coverage state and entitlement-aware availability. |
| Tenant default lists | Read/write for tenant administrators; membership refers to global asset IDs. |
| Personal lists | Read/write only for the authenticated subject in the request tenant. |
| List assets | Project global identity and latest shared EOD intelligence through authorized membership. |
| Coverage | Return global EOD, options, intraday, and enrichment coverage with truthful status/date/reason. For options, include prospective-history start, latest capture, captured-session count, lookback, window label/version, readiness, and provenance. |
| Options demand | Administrative, auditable read of the tenant’s contribution and applicable outcome; do not disclose other tenants’ memberships. |

Existing MarketOps asset APIs require a compatibility period. The direct onboard endpoint becomes an administrator catalog-admission request only after the new catalog/search/list path is available and users have been migrated.

## 8. Delivery phases

### Sprint operating rules

Sprints are additive, reversible, and independently releasable. The current tenant-owned MarketOps Assets APIs, scheduled jobs, and analyst workflows remain the production path until a sprint exit gate is approved.

Every sprint must preserve these controls:

- additive migrations only; no destructive schema or ownership change;
- feature flags and tenant-scoped rollout controls for every new write or read path;
- read-only parity reports before a new projection becomes selectable;
- idempotent writes, immutable provenance, bounded provider budgets, and observable failure reasons; and
- a documented rollback that disables the new path without deleting shared evidence or member preferences.

| Sprint | Scope | Safe delivery boundary and exit gate |
|---|---|---|
| S0 — Baseline and controls | Inventory current asset, scheduler, API, authorization, and retention contracts; define metrics and flags. | Read-only. The [S0 baseline utility and rollback posture](s0_baseline_and_controls.md) are reviewed before any schema change. |
| S0-A — Access-control hardening | Bind every tenant-bearing input to the principal; add list ownership, entitlement/quota policy, service identities, audit, and negative integration tests. | Local implementation is complete; the [S0-A exit checklist](s0a_exit_checklist.md) records the required production workload-login and browser/cross-tenant evidence. Current tenant-owned reads and writes remain the fallback under feature flags until formal exit. |
| S1 — Global catalog shadow | Add global identity, eligibility, reference provenance, and coverage registry; seed from existing assets and reference data. | Existing reads and jobs are unchanged. Every current active asset maps to exactly one global asset; no duplicate identities. |
| S2 — Catalog breadth and EOD planner shadow | Admit exchange-listed US common stocks to the catalog; configure a centrally governed top-1,000 hot EOD baseline and a cold-watchlist-triggered activation queue. | Shadow plan only. It produces budget, queue-age, eligibility, duplicate-demand, and expected-coverage evidence without changing collection. |
| S3 — Lists and authorization projection | Add tenant-default lists, private lists, and global-ID memberships behind feature flags. | Opt-in tenant pilot. Authorization tests prove list isolation; the existing Assets view remains the default projection. |
| S4 — Shared EOD canary | Enable global EOD collection and baseline calculation for a small approved cohort, then expand within the top-1,000 budget. | Dual-run parity confirms one canonical result per symbol/session, no duplicate provider pulls, and safe rollback to current scheduling. |
| S5 — Subscriber read experience | Add catalog search, list management, coverage truth, and compatibility projections. | Pilot users can compare new and current views; no current endpoint is removed or changes ownership. |
| S6 — Options demand union | Add entitled demand snapshots, deterministic deduplication, central capture ownership, and prospective-history readiness. | Start with a bounded cohort. One selected symbol produces one capture; quota and warm-up outcomes are auditable. |
| S7 — Controlled adoption | Expand eligible tenants and coverage capacity; migrate existing tenant universe rows only after parity is sustained. | Formal cutover review approves each migration cohort; legacy writes remain available until post-cutover reconciliation succeeds. |

Sprint 0 establishes the release controls for every later sprint. S0-A must complete before any subscriber-data schema, list, global-projection, or entitlement path is enabled. Sprints 1 through 6 may proceed only when their own exit gate and all prior gates remain satisfied; Sprint 7 is a deliberate adoption decision, not an automatic consequence of implementation.

### Delivery phase detail

### Phase A — Catalog foundation

Create global asset identity and coverage lifecycle records, reference provenance, platform operational ownership, and catalog search. Seed the current MarketOps symbols. Deliver read-only parity reporting between current tenant universe and global catalog.

Exit criterion: each active current asset maps to exactly one global record; no identity duplicates or silent drops.

### Phase B — Subscriber lists and access projection

Create tenant-default lists, user-private lists, memberships, subject-aware authorization, and list UX. Migrate the current analyst watchlist into the applicable tenant default. Keep the existing Assets experience as a compatibility projection.

Exit criterion: two users in one tenant can have separate private lists; a user in another tenant cannot read either membership set; shared asset details and EOD results are identical.

### Phase C — Global EOD coverage

Move the completed-session reconciliation queue and price-based algorithm planners to global active coverage. Maintain the top-1,000 hot set, then admit eligible cold symbols when one or more watchlist memberships create a deduplicated activation request. Run bounded pilot batches, measure provider requests, completeness, queue age, quality, activation-to-first-evidence time, and storage growth, then expand progressively.

Exit criterion: the same globally covered symbol is ingested and calculated once per session even when used by multiple tenants.

### Phase D — Options demand union

Add the daily demand snapshot, entitlement filter, deterministic quota planner, central capture ownership, and tenant-authorized coverage UI. Keep current bounded runner controls for max symbols, pages, candidate contracts, distribution limits, and the 10-calendar-day default lookback (configurable up to 60 days). Expose the prospective-history and readiness policy rather than backfilling newly selected assets by default.

Exit criterion: multiple entitled watchers of the same asset produce one capture; quota deferrals and warm-up states are durable, reproducible, and visible; every options view identifies the capture history it actually used.

### Phase E — Commercial and operational hardening

Add account provisioning, product-tier administration, usage/audit reporting, capacity alarms, data retention policies, and billing integration only after the data/authorization model is operating reliably.

Exit criterion: a new tenant can be provisioned with an entitlement tier and a default list without an operational data-copy workflow.

## 9. Acceptance measures

The project is ready to progress when it can demonstrate:

- One global asset identity and one global EOD result per covered symbol/session.
- The centrally governed hot set contains the approved top 1,000 eligible assets; a cold eligible asset selected by any authorized watchlist creates one auditable, deduplicated global activation request and transitions through truthful coverage states.
- No cross-tenant list membership disclosure.
- Personal lists plus tenant default render correctly for authenticated users.
- Every list membership points to a governed global asset.
- Provider requests scale with unique covered symbols, not users or tenants.
- Global EOD coverage is measured by completed session, quality, failures, retries, and stale/deferred age.
- Options selected/deferred outcomes are explainable from a retained demand snapshot and policy version.
- Options history is explicitly prospective: captured-session counts, lookback semantics, readiness, gaps, and source-capture provenance are visible per asset.
- Existing current MarketOps users retain access to their migrated assets without unintended list or data loss.

## 10. Explicitly deferred decisions

The following need a separate product and commercial decision before implementation:

- Subscription pricing, payments, invoicing, trials, and cancellation policy.
- Exact product-tier limits for catalog search, private-list count, list size, options allocation, and intraday availability.
- The provider contract and budget needed to make the full Massive US common-stock catalog active.
- Whether to fund any optional historical options backfill beyond the prospective-capture policy, and its cost, entitlement, retention, and disclosure rules.
- Fundamental-data provider scope and whether that enrichment is product-tiered.
- Any recommendation, alerting, or execution product built from MarketOps research outputs.

Until these are decided, the architecture must preserve entitlement-aware interfaces and measurable usage without fabricating a commercial policy.


## Production blocker — global analytical data plane — 2026-08-14

**Status: critical; blocks Subscriber Project production readiness.**

The deployed subscriber catalogue currently centralizes identity, list membership, coverage state, and a narrow current-EOD projection. It does not yet centralize the historical MarketOps evidence plane. The pilot browser therefore correctly resolves globally shared AAPL/NVDA EOD context, but it cannot resolve global Market State, EROC, valuation, EEOM, earnings-event, or historical price/feature evidence. Those ledgers remain legacy `tenant-local` records and must not be presented as a subscriber data source.

Production exit requires one platform-owned, immutable global evidence path keyed by `global_asset_id` and market session. It must retain provider/source/run provenance and power global EOD history, features, Market State, EROC, VC/DOSM, EEOM, material events, outcomes, and SRI. Central jobs process each covered asset exactly once; authorized tenant/default/private lists only project those results. A verified legacy-history import may seed the global path only after identity and provenance parity checks. It must never copy historical records per user or silently treat tenant-local results as globally canonical.

Until that work passes migration, backfill, scheduler, authorization, parity, and browser acceptance gates, the UI must use truthful global coverage states. It must not render invalid placeholder timestamps, "Open Market State", or implied algorithm availability for a symbol without globally materialized evidence.

### Evidence-ledger foundation — applied, inactive — 2026-08-14

Migration `000118_subscriber_global_analytical_evidence_foundation` is the first additive remediation slice. It introduces an append-only platform ledger keyed by global asset, session, evidence kind, algorithm/version, fingerprint, validation-contract reference, immutable-baseline reference, and source/run provenance. Its writer contract is constrained to `shadow_read_only` reconciliation runs under the existing no-login `signalops_subscriber_global_eod` role. The only new gateway surface is a coverage-count view; it cannot serve scores, states, signals, or legacy data as canonical.

Migration `000118` was applied to the dedicated MarketOps primary database on 2026-08-14 17:29 UTC. Ownership is `signalops_subscriber_migrator`; `signalops_subscriber_global_eod` has `SELECT, INSERT` only on the two base tables; the gateway has `SELECT` only on the coverage view; `PUBLIC` has no access. The ledger contains zero runs and zero records after migration. It did not import history, reroute existing jobs, enable a scheduler, restart a service, or change an API response.

### Legacy parity manifests — applied, canary evidence recorded — 2026-08-14

Migration `000119_subscriber_global_analytical_parity_manifests` adds a fixed, security-barrier source view over six asset-scoped tenant-local evidence types: feature vectors, Market State, valuation, EEOM, Signal Assurance assertions, and outcomes. The controlled `subscriber-global-marketops-parity-manifest` runner is bounded to 1–50,000 rows, requires `--execute`, and writes an immutable manifest that reports each source row as `mapped`, `unmapped`, or `ambiguous` against the canonical global identity. A mapped entry remains `pending_global_materialization`; the runner does not write `subscriber_global_marketops_evidence_records`, change a legacy row, invoke a provider, or make a gateway projection selectable.

Migration `000119` was applied to the dedicated MarketOps database on 2026-08-14 17:36 UTC. A new no-inheritance runtime login was granted only `signalops_subscriber_global_eod`; the runner explicitly assumes that group role and fails closed if it can read a raw legacy table or has a privileged identity. Six initial manifests were recorded: 1,075 populated source rows across EEOM (75), feature vectors (250), Market State (250), valuation (250), and outcomes (250), all uniquely mapped; Signal Assurance correctly produced an empty manifest because no assertions currently exist. No global evidence run, evidence record, or gateway coverage row was created.

SRI is segment-scoped rather than asset-scoped and remains a separate parity slice; material-event normalization is likewise retained for its own event-keyed contract. The next slices are: (1) expand bounded manifests deterministically until each source type is complete, (2) add a separately approved immutable import/capture path, and (3) expose each algorithm only through a type-specific parity-approved gateway projection and browser contract. This remains a production blocker until all cited evidence types have completed those gates.

## Watchlist context closure — 2026-08-14

MarketOps resolves a user’s first private list when no saved context exists; the tenant default is used only when that user has no private list. A later explicit selection remains authoritative. Creating a private list now selects it for the shared MarketOps context, and the Watchlists screen exposes a **Use across MarketOps** action for an existing list. The context is persisted per tenant and subject, then invalidates MarketOps queries so Assets, Dashboard, EROC, valuation, EEOM, material events, opportunities, and Market State reads converge on the same selected ticker set. Coverage remains truthful: a selected symbol without usable central EOD evidence is displayed as pending rather than populated with invented values.


### Browser acceptance evidence — 2026-08-14

The pilot user for `tenant-pilot-b` selected `First List` through the Watchlists **Use across MarketOps** action. The retained `signalops.syncratic.io-testsignal-04.har` records a `200` context mutation and subsequent `First List` responses for Assets, Dashboard signal overview, and Market State. Assets rendered usable shared EOD rows for AAPL and NVDA; NOW and SNOW remained pending because they have no usable central EOD evidence. The follow-up `signalops.syncratic.io-testsignal-05.har` records `200` responses carrying `First List` context for EROC, Material Events, Valuation, EEOM, and both Market State list requests. This closes the pilot browser acceptance for shared watchlist-context propagation.


### Global analytical-evidence materialization — next-cycle foundation

The global catalog is not complete merely because a tenant membership resolves to a canonical symbol. Every analytical view must ultimately read a single-copy, platform-owned record by `global_asset_id`; it must not create a tenant-local clone or infer a score from a raw EOD bar. Migration `000122` and the `subscriber-global-marketops-evidence-materializer` establish the first safe transition path for existing evidence:

- The materializer accepts only an immutable, mapped parity-manifest run and its manifest entries.
- It reads the fixed tenant-local parity source through the restricted worker role, appends globally identified evidence with the source fingerprint and provenance, and never calls a provider or changes a legacy row.
- Each appended run declares `legacy_materialized` execution and `legacy_materialization` source scope. The global coverage view may report that provenance, but remains metadata-only.
- Type-specific gateway readers for Dashboard, EROC, Material Events, Valuation, EEOM, Market State, SAF, SRI, intraday, and Risk/Reward remain independently gated. A projection may be enabled only after its parity, freshness, authorization, and browser acceptance evidence is retained.

This sequencing keeps the ownership model explicit: catalogue identity and analytical evidence are global; tenant data is limited to entitlement, membership, list preference, and authorized projection selection. An asset without materialized central evidence remains Pending; it is never rendered with fabricated historical or algorithmic data.
