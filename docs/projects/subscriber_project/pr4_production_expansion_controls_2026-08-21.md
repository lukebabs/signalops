# PR-4 Production Expansion Controls — 2026-08-21

Status: started.

## Decision boundary

PR-3 backup/restore rehearsal is intentionally deferred by product decision. Prior dedicated MarketOps pgBackRest backup and isolated restore rehearsal evidence remains useful, but it is not current after the latest PR-1/PR-2 changes. This is accepted as a known readiness risk, not closed recovery evidence.

PR-4 now focuses on production expansion controls that can be improved without widening provider polling or tenant access.

## Scope

PR-4 covers four areas:

1. **FMP annual financial lifecycle**
   - Keep the annual enrichment centrally governed.
   - Ensure it is visible as an Admin job/freshness row.
   - Keep it failure-isolated so one bad symbol or endpoint does not fail the full workflow.
   - Respect the current FMP Starter 300-calls/minute entitlement while retaining a conservative worker throttle unless changed by explicit release decision.

2. **Trading-calendar correctness**
   - Move from weekend-only guards toward a canonical US market-session calendar.
   - Use the same calendar for job eligibility and UI 10/20 trading-day interpretation.
   - Explicitly prevent EOD/intraday jobs on weekends and market holidays, except maintenance jobs.

3. **Subscriber administration**
   - PR-2 already closed the governance-surface smoke for configured production QA identities.
   - Remaining PR-4 work is polish/operationalization, not core access-control closure.

4. **Incident runbooks**
   - Maintain concise runbooks for stale dashboard data, failed post-close, provider outage, access-control regression, failed deployment smoke, and backup/restore.
   - Each runbook must include detection, owner, first response, recovery action, verification, and rollback criteria.

## Current source state

- `marketops-fmp-annual-financial` exists as an Admin-visible scheduled job and deployment-agent run-now target.
- The FMP annual worker records per-asset task status and workflow coverage, and marks the workflow `degraded` instead of failing the entire run when individual tasks fail/defer.
- The installer supports `--enable-fmp-annual` and would enable the Saturday 02:30 America/New_York timer.
- `scripts/lib/marketops_trading_calendar.sh` currently provides a weekend guard only. It allows maintenance jobs and FMP annual on weekends.

## First implementation target

Add a reusable MarketOps calendar primitive with a static US market-holiday list for the current operational range, then wire scheduler guards and tests around it. This is safer than immediately enabling new provider schedules.

Acceptance for the first slice:

- Calendar helper returns non-trading day for weekends and configured US market holidays.
- EOD/intraday/SRI/post-close jobs skip on non-trading days.
- Maintenance and FMP annual jobs remain permitted on weekends by explicit allowlist.
- Documentation records how to update the holiday list.

## First slice implementation — trading-calendar guard

Implemented an explicit MarketOps trading-calendar helper in `scripts/lib/marketops_trading_calendar.sh`:

- supports weekend detection;
- includes static 2026 and 2027 US market-holiday closures;
- exposes `marketops_is_trading_day`;
- exposes `marketops_non_trading_reason`;
- preserves the explicit maintenance/FMP weekend allowlist.

`scripts/marketops_scheduled_job.sh` now uses the unified non-trading-day predicate. EOD, intraday, SRI, warm-EOD, retry, and post-close jobs skip on weekends and configured market holidays. Maintenance jobs and `marketops-fmp-annual-financial` remain explicitly permitted on weekends.

Added `scripts/test_marketops_trading_calendar.sh` to prove:

- 2026-08-21 is a trading day;
- 2026-08-22 is a weekend non-trading day;
- 2026-09-07, 2026-11-26, and 2027-07-05 are market holidays;
- FMP annual is weekend-permitted;
- daily post-close is not weekend-permitted.

Verification:

```text
bash -n scripts/lib/marketops_trading_calendar.sh scripts/marketops_scheduled_job.sh scripts/test_marketops_trading_calendar.sh
scripts/test_marketops_trading_calendar.sh
marketops_trading_calendar_tests_passed
scripts/run_subscription_admin_ui_smoke.sh
...                                                                      [100%]
3 passed in 3.20s
```

Operational note: the holiday list is intentionally static and must be updated annually from the official exchange calendar before enabling a new production year.
