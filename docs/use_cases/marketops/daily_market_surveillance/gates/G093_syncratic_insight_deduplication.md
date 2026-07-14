# G093 Syncratic Insight De-duplication And Ask State Clarity

Status: specification proposed
Use case: MarketOps Daily Market Surveillance

## Goal

Clarify how SignalOps should present and manage multiple Syncratic context windows and insights for the same subject without deleting historical evidence or confusing deterministic materialization with Syncratic Ask enrichment.

G093 should produce a small, auditable lifecycle policy for overlapping Syncratic rows. It should not change detector behavior, materialization evidence rules, Ask prompt behavior, graph state, alert lifecycle state, or production policy deployment.

## Problem

G091/G092 validation showed that AAPL can have multiple persisted Syncratic insight rows across overlapping context windows, and several older rows may already include `metrics.syncratic_ask`. This is expected from previous validation runs, but it creates operator ambiguity:

- materialization may be deterministic and not trigger Ask, while the list still shows older Ask-enriched rows;
- multiple active context windows for the same subject can look like duplicate insights even when they came from different windows, builder versions, or evidence digests;
- unchanged-evidence skips are correctly idempotent, but overlapping windows can still accumulate active rows;
- operators need to know which row is current, historical, Ask-enriched, or superseded.

The immediate issue is not data correctness. The issue is lifecycle clarity.

## Product Semantics

G093 should preserve these distinctions:

- **Context window**: deterministic evidence boundary built from SignalOps/MarketOps ledgers.
- **Syncratic insight**: durable explanation row attached to a context window.
- **Materialization state**: whether deterministic context/insight rows were created, skipped, or unchanged.
- **Ask state**: whether an operator later generated LLM synthesis over a persisted context window.
- **Lifecycle state**: whether a row is current, historical, superseded, reviewed, dismissed, or archived.

Ask state must not determine whether a row is current. A deterministic row can be current without Ask. An Ask-enriched row can become historical if a newer evidence digest supersedes it.

## Non-Goals

Do not implement in G093 unless explicitly approved after this specification is reviewed:

- deleting context windows or insights;
- changing G091 materialization thresholds;
- changing G090 Ask prompt or calling Ask automatically;
- creating scheduled materialization jobs;
- changing alert lifecycle semantics;
- graph writes or proposal decisions;
- detector threshold edits;
- policy deployment;
- migrations that rewrite historical rows without an explicit backfill plan.

## Recommended Currentness Model

A Syncratic insight should be considered current when it is the preferred active row for this tuple:

```text
tenant_id + app_id + domain + use_case + subject_symbol + context_strategy + context_builder_version
```

Within that tuple, order candidates by:

1. latest `window_end`;
2. latest `updated_at`;
3. deterministic tie-breaker by `context_window_id` or `syncratic_insight_id`.

A row should be eligible to supersede older rows only when:

- it has a different `evidence_digest` or a materially newer `window_end`;
- it passed existing subject/evidence purity checks;
- it is not below threshold;
- it was persisted successfully through the normal materialization path.

Rows with the same idempotency key and evidence digest should remain idempotent unchanged skips, not new rows.

## Recommended Lifecycle Policy

Use existing statuses first:

- `active`: row is available for operator review.
- `reviewed`: operator reviewed it; not necessarily current.
- `dismissed`: operator intentionally dismissed it.
- `archived`: row is retained but hidden from default operational views.
- `superseded`: row is retained but replaced by a newer/current row for the same subject/strategy tuple.

G093 should prefer `superseded` for older overlapping rows that are automatically displaced by a newer deterministic context. Use `archived` for manual or housekeeping hide-from-default-list behavior. Do not overload `dismissed`; dismissal is an operator judgment, not a currentness algorithm.

## Ask State Clarity

Ask state should remain metadata under `metrics.syncratic_ask`.

Recommended display categories:

- `Deterministic current`: current row, no Ask metadata.
- `Ask-enriched current`: current row, Ask completed for this evidence digest/prompt digest.
- `Historical deterministic`: non-current row, no Ask metadata.
- `Historical Ask-enriched`: non-current row with Ask metadata.
- `Data-quality blocked`: Ask or context evidence flagged unsupported subject evidence.

The UI should not imply materialization generated Ask. When a newly materialized row has no `metrics.syncratic_ask`, show it as deterministic even if older rows for the same symbol are Ask-enriched.

## Backend Design Options

### Option A: Read-Time Currentness Only

Add API response metadata that marks rows as current/historical without mutating existing rows.

Potential fields:

```json
{
  "currentness": {
    "is_current": true,
    "currentness_key": "tenant-local|marketops|market_data|daily_market_surveillance|AAPL|symbol_signal_cluster_5d|syncratic.context_builder.v1",
    "superseded_by_context_window_id": "",
    "superseded_by_syncratic_insight_id": "",
    "reason": "latest_window_end"
  }
}
```

Pros:

- no migration;
- no historical mutation risk;
- easy to test and iterate.

Cons:

- lifecycle state in storage remains ambiguous;
- list queries may need more computation.

### Option B: Materialization-Time Supersession

When a new eligible context/insight is persisted, update older active rows in the same currentness tuple to `superseded` and record the replacing ids in metrics/recommendation metadata.

Pros:

- storage reflects lifecycle;
- default lists become cleaner;
- idempotent reruns remain simple.

Cons:

- requires careful transaction semantics;
- needs tenant-scoped repository update methods;
- needs backfill or manual closeout for existing overlapping rows.

### Option C: Hybrid Recommended MVP

Implement read-time currentness first, then add materialization-time supersession only after operator review confirms the rule works.

Recommended G093 MVP: **Option C, first slice = read-time currentness plus UI clarity**.

This avoids rewriting history while giving operators a clear view immediately. A later G094 can persist supersession if needed.

## API Shape Proposal

Avoid a broad API redesign. Extend existing list/detail responses if implementation confirms it is low-risk, or add a narrow summary field inside `metrics`/DTO response only.

For `GET /v1/syncratic/insights` and detail:

- include currentness metadata if backend implementation is chosen;
- continue supporting existing filters;
- add optional filter only if needed:
  - `current_only=true`
  - `include_historical=true`

Default behavior should be conservative. If changing defaults risks hiding rows unexpectedly, keep current list behavior and let frontend visually group current vs historical first.

## UI Behavior Proposal

The Syncratic Insights UI should:

- group or sort current rows above historical rows;
- display current/historical state separately from Ask state;
- show Ask badges only as Ask state, not currentness;
- make overlapping rows understandable by showing window range, evidence digest short id, and context strategy;
- avoid showing multiple AAPL rows as identical duplicates;
- offer a filter/toggle for `Current only` once the backend/API state supports it or frontend can safely derive it.

## Data Safety

G093 must be non-destructive.

- No deletes.
- No automatic dismissal.
- No operator-review loss.
- No removal of Ask metadata.
- No changing evidence digest or idempotency keys.
- No raw prompt or secret exposure.

If later gates persist supersession, they must preserve enough metadata to reconstruct:

- which row superseded which;
- when the supersession occurred;
- what currentness key was used;
- whether the older row had Ask metadata.

## Acceptance Criteria

G093 is accepted when:

- the currentness policy is implemented or explicitly approved for implementation;
- operators can distinguish current vs historical Syncratic rows;
- operators can distinguish deterministic vs Ask-enriched rows independently of currentness;
- no data is deleted;
- Ask is not triggered automatically;
- tests cover overlapping rows for the same symbol, currentness selection, Ask-state independence, and idempotent unchanged rows;
- docs and gate audit record the chosen storage/API/UI behavior.

## Validation Plan

For implementation, use seeded or fake rows covering:

- two AAPL rows with different window ranges and evidence digests;
- one current deterministic row without Ask;
- one older Ask-enriched row;
- an unchanged materialization rerun;
- a row with data-quality warning metadata;
- another symbol to prove grouping is symbol-scoped.

Run:

- targeted backend/API tests if currentness is backend-derived;
- frontend helper/API tests if currentness is frontend-derived;
- full Go and/or frontend suite matching touched code;
- live UI smoke showing current/historical and Ask badges do not collapse into one state.
