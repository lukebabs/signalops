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

## SAF-V2b governed legacy-sector normalization — 2026-08-17

The 17 `sector_unmapped` observations are attributable to 11 global catalog
assets. A bounded FMP `/stable/profile` lookup returned a profile sector,
industry, and exchange for every one. The classifications are normalized only
to the governed SRI sector vocabulary:

| Canonical sector | Symbols |
|---|---|
| Industrials | FDX |
| Healthcare | GILD |
| Materials | LIN |
| Consumer Discretionary | MCD, NKE |
| Communication Services | META, TMUS |
| Consumer Staples | MO, PEP |
| Utilities | NEE |
| Technology | QCOM |

Migration `000150_subscriber_global_sector_normalization` creates append-only
provider-backed classification evidence, applies these eleven catalog references
with FMP authority/provenance, and prevents later legacy-catalog seeding from
overwriting that authority. It retains the v1 benchmark projection until the
v2 calculation is present for each observation.

The separate, one-time `saf_benchmark.v2` materialization repeats all 92
broad-market and sector comparisons from immutable initial-capture EOD data.
It appends rather than updates: v1 stays available as audit history, while the
projection prefers v2 per observation only after it exists. The guarded launchers
are `apply_subscriber_global_sector_normalization_migration.sh` and
`run_subscriber_global_saf_benchmark_v2_materialization.sh`.

## SAF-V2c historical-identity reconciliation — 2026-08-17

The first v2 calculation exposed a catalog lineage issue: some immutable historical evidence records referenced an older global-asset identity for the same canonical symbol. The FMP-backed classification had been recorded against the newer identity, so the v2 computation correctly preserved an unresolved sector rather than silently crossing identities.

Migration `000151_subscriber_global_sector_classification_reconciliation` reconciles reference metadata for those equivalent canonical-symbol catalog identities. It does **not** merge, delete, or rewrite global assets, evidence records, historical outcomes, or v1/v2 benchmark rows. The durable catalog consolidation work remains a separate roadmap item; this narrow repair allows the fixed legacy cohort to use its governed sector classification now.

`saf_benchmark.v3` is a new append-only calculation version. Its projection preference is v3, then v2, then v1, so every calculation remains available for audit. The controlled launcher is `run_subscriber_global_saf_benchmark_v3_materialization.sh`.

### Deployment evidence

- `000150_subscriber_global_sector_normalization` applied at `2026-08-17 17:11:57 UTC`.
- `000151_subscriber_global_sector_classification_reconciliation` applied at `2026-08-17 17:16:58 UTC`.
- The v3 run used only already-stored, immutable initial-capture EOD data: 92 legacy observations, 184 benchmark rows, 92 broad-market matches, and 92 sector matches. It made no FMP or market-data request.
- The live gateway projection now reports `matched=92` and no `sector_unmapped` rows.
- `python/tests/test_signal_assurance_benchmark_ui_smoke.py` is the read-only authenticated browser regression: it verifies the live Signal Assurance view exposes only `broad=matched; sector=matched` for the legacy benchmark-coverage cohort. The 2026-08-17 run passed.

This establishes complete benchmark coverage for the declared historical cohort. It does not establish algorithm viability by itself; the next gate is the predeclared score/confidence and frozen-policy evaluation described in SAF-V2/SAF-V3 above.

## SAF-V3a direction-normalized viability baseline — 2026-08-17

Before freezing a viability interpretation, the benchmark calculation was corrected to express excess return in the asserted direction. This matters because 85 of the 92 historical outcomes are downside opportunities: an asset declining more than its benchmark is favorable evidence and must be positive after normalization.

Migration `000152_subscriber_global_saf_directional_benchmark_projection` makes `saf_benchmark.v4` the preferred projection when present, while v1-v3 remain append-only audit evidence. The v4 materializer uses only the same immutable initial-capture EOD observations; it made no provider request.

The read-only `saf_viability.v1` gate is therefore **not demonstrated** for this cohort. Its complete benchmark coverage is a prerequisite, not proof of effectiveness:

- 92 matured legacy observations; 46 directional matches (50.0%).
- Mean direction-normalized broad-market excess: 0.041%.
- Mean direction-normalized sector excess: 0.061%.
- Mean favorable excursion: 2.310%; mean adverse excursion: -3.858%.

The cohort does not clear the predeclared directional-confidence or favorable-versus-adverse-excursion conditions. This is an actionable research result: do not promote or tune from it automatically. The next governed step is a formal frozen-policy record and an independently matured forward cohort, with score/confidence and algorithm-version attribution captured prospectively.

## SAF-V3b frozen policy record — 2026-08-17

Migration `000153_subscriber_global_saf_frozen_viability_baseline` persists the append-only `saf_viability.v1` policy and frozen baseline: the immutable 132-member legacy-default scope, 92 completed observations through the 2026-08-14 cutoff, `saf_benchmark.v4` direction-normalized evidence, predeclared thresholds, and the research-only `not_demonstrated` result with its exact metric snapshot. This baseline is evidence, not a parameter change; a later forward cohort or revised policy must be a new append-only record.
