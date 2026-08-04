# MarketOps Post-Close Universal-Universe Capacity Incident — 2026-08-03

## Summary

The governed MarketOps post-close workflow started on 2026-08-03 at 18:01:55 ET and successfully acquired and normalized equity EOD data for all 115 active universal assets. It stopped before Market State, Risk/Reward, EROC, final convergence, Opportunities, and dashboard publication.

The options capture barrier found only three capture rows when 115 were required:

```
options capture barrier failed: captures=3 expected=115
```

The run is correctly recorded as failed in `runtime/scheduled-jobs/marketops-daily-postclose.json`. No partial post-close analytical state was published.

## Root cause

The post-close script used the database universal universe and correctly passed `--max-symbols 115` to `marketops-options-coverage-runner`. The runner's configuration normalizer still treated every value above 100 as invalid and silently reset it to its legacy default of 3.

This created an internal contract mismatch:

| Component | Expected capacity | Actual behavior before fix |
| --- | ---: | --- |
| Post-close script | 115 active assets | Requested 115 options captures |
| Options coverage runner | 115 requested symbols | Reset count to 3 |
| Capture barrier | 115 captures | Rejected 3 captures and halted the pipeline |

This was not a provider credential, normalization, or scheduler availability failure. The equity stage reported 115 published events and the normalization barrier passed at 115/115.

## Impact

- Dashboards continued to show the last fully completed post-close date, 2026-07-31.
- Assets showed **Awaiting EOD analysis** because the Aug. 3 Market State and Risk/Reward stages did not begin.
- Opportunities did not refresh because final convergence did not run.
- Three option captures were persisted, but they were not allowed to create a partial cross-universe analytical publication.

## Remediation

1. The options runner now preserves requested counts up to an explicit `maxCoverageSymbols` capacity of 200; it no longer rewrites valid universal counts to 3.
2. Counts above 200 are rejected with an explicit validation error rather than silently downgraded.
3. The post-close script now checks the active database universe against the same 200-asset capacity **before** any provider or equity work. A future mismatch fails immediately with an actionable error.
4. A regression test verifies that 115 is preserved and accepted, while 201 is rejected.
5. The options coverage runner and scheduled cohort image must be rebuilt before the governed Aug. 3 recovery run.

## Recovery procedure

After deployment, run the governed workflow once for the missed completed session:

```bash
MARKETOPS_DAILY_ACKNOWLEDGE_WRITES=true \
  scripts/marketops_daily_postclose.sh --date 2026-08-03 --write
```

Do not bypass the capture barrier or manually publish downstream results. The recovery must produce all 115 capture rows, then the normal ten-symbol cohorts, final convergence, outcome maturity sweep, and dashboard refresh.

## Related incident

The 2026-07-31 post-close service also ended non-zero after downstream work because the Risk/Reward retention script was not executable. That issue is resolved: the script is executable and is invoked with `bash`. It is documented separately in `marketops-eod-retention-permission-incident-2026-07-31.md`.

## Follow-up: unified selection remediation (2026-08-04)

A subsequent Exhaustive Reversal reconciliation exposed an additional selection inconsistency: an older job image iterated 161 membership records while the authoritative database projection held 115 unique active tickers. This was membership duplication across universe groups, not 161 unique assets.

The shared `ListMarketOpsAssets(..., "all_active", ...)` selector now scopes and deduplicates by ticker before applying its limit. The post-close and algorithm-corroboration scripts use the same deterministic ticker projection. The affected job images and persistent asset-backfill worker were rebuilt. The post-remediation canonical count is 115, and Exhaustive Reversal verified 115/115 current result rows.
