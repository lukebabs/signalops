# Signal Assurance Viability Sprint

Status: active. This is a read-only research-governance sprint. It does not create investment recommendations, alter algorithm thresholds, or enable automated actions.

## Primary cohort

The primary SAF viability cohort is the preserved `tenant-local` **MarketOps Legacy Default** list of 132 assets. It is deliberately not the wider 1,000-symbol warm-EOD universe.

This cohort is the correct first baseline because it preserves the already-matured MarketOps history, has stable canonical membership, and represents the current tenant-local operating universe. A later cohort may be compared with it, but must never be silently blended into it.

Every viability run must retain:

- tenant: `tenant-local`;
- list identity and membership snapshot;
- algorithm and version;
- signal type, direction, score/confidence band, and horizon;
- immutable historical-assurance selection/provenance; and
- the metric and viability-policy versions.

## Baseline inventory - 2026-08-17

Read-only inventory of the tenant-local default list:

| Measure | Result |
|---|---:|
| Selected assets | 132 |
| Matured historical LEGACY observations | 92 |
| Assets with an observation | 41 |
| Directional matches | 46 |
| First origin session | 2026-07-31 |
| Latest maturity | 2026-08-14 |

The available observations are concentrated in 1-, 5-, and 10-session horizons and are predominantly downside outcomes. They contain no confirmed SAF assertions and no benchmark-relative return. These limitations are evidence, not errors: the initial scorecard must show them as coverage and comparability gaps.

## Decision contract

The unit of assessment is never an algorithm name alone:

```text
algorithm + version + signal type + direction + score/confidence band
+ horizon + market/sector regime + data-quality contract
```

The scorecard will use conservative states:

| State | Meaning |
|---|---|
| `insufficient_evidence` | Fewer than 30 complete, matured directional observations. No ranking or viability claim. |
| `benchmark_pending` | Matured sample exists but matched benchmark-relative evidence is absent. |
| `outcome_profile_pending` | Matched return exists but favorable/adverse excursion evidence is incomplete. |
| `not_demonstrated` | Predeclared in-sample directional and/or excess-return conditions are not met. |
| `research_supported_in_sample` | The completed in-sample cohort clears the predeclared conditions. It remains research-only. |
| `out_of_sample_pending` | An immutable in-sample baseline is frozen; independent evidence has not matured. |
| `research_validated` | The independent cohort also clears the same conditions. It remains research-only. |
| `degraded` | A rolling cohort materially deteriorates against the frozen baseline. |

No state authorizes trading, alerting, or automatic parameter changes.

## Work sequence

### SAF-V1 - operational scorecard

1. Add the conservative viability state/reasons to current effectiveness rows.
2. Make the 132-list identity, observation coverage, evidence class, and benchmark gap explicit.
3. Preserve existing cohort drill-down, immutable baseline/provenance, and improvement candidates.
4. Add API/UI tests for insufficient evidence and benchmark-pending truthfulness.

### SAF-V2 - comparable cohorts

Add matched broad-market and sector-relative returns, score bands, data-quality dimensions, and sector/SRI-regime slices. Retain selection provenance and keep algorithm versions separate.

### SAF-V3 - frozen evaluation

Persist a versioned viability policy, freeze an in-sample cutoff, and evaluate a held-out or forward cohort. No threshold can change without a new versioned replay/calibration record.

### SAF-V4 - ongoing governance

Monitor rolling drift after completed sessions and surface degraded cohorts in the analyst/admin workbench.

## SAF-V1 acceptance

- The 132-member tenant-local default is the declared primary cohort.
- Historical `LEGACY` outcomes remain visibly distinct from confirmed `SAF` assertions.
- Fewer than 30 matured observations cannot be presented as viable.
- Missing benchmark-relative evidence cannot be interpreted as zero or as a pass.
- The slice is read-only and cannot change an algorithm, validation contract, provider schedule, or signal outcome.

SAF-V2a is now implemented as an additive benchmark materializer. For each matured historical observation in the immutable 132-member legacy-default cohort, it records:

- a broad-market comparison against `SPY`;
- a sector comparison against the governed SRI primary ETF (for example `XLK`, `XLF`, or `XLV`) when the current global catalog has a resolvable sector;
- the first available normalized EOD record for both the origin and maturity session, rather than a later provider revision;
- source event identifiers, content fingerprints, selection-policy version, calculation version, and run correlation; and
- explicit `sector_unmapped`, `origin_price_unavailable`, or `maturity_price_unavailable` states rather than a fabricated return.

The materializer writes only `subscriber_global_saf_benchmark_observations`; it cannot update or delete a legacy outcome, outcome payload, baseline, or confirmation. The scorecard treats incomplete broad-market **or** sector coverage as `benchmark_pending`.

The initial catalog inspection shows that only 40 of the 132 legacy members currently carry a canonical top-level sector label. This is an input-quality gap, not a license to infer a sector silently. The remaining unmapped rows remain visible and block a complete sector-relative viability conclusion until governed catalog normalization is completed.

## SAF-V2a operational benchmark diagnostic — 2026-08-17

The analyst drill-down now supports a **Benchmark coverage** cohort dimension.
It separates observations by their immutable broad-market and sector resolution
states (for example, `broad=matched; sector=matched` and
`broad=matched; sector=sector_unmapped`) and shows both states on each included
observation. This is diagnostic only: it does not infer sectors, fill missing
returns, revise outcomes, or alter any viability gate.

This makes the current sector-normalization backlog directly inspectable while
retaining the existing rule that incomplete sector coverage remains
`benchmark_pending`.
