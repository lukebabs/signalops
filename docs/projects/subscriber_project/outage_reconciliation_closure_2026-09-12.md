# September 2026 outage reconciliation closure

Status: closed with accepted provider-evidence gap.

Recorded: 2026-09-12.

## Scope

This closure covers the host outage reconciliation work for the missed September 2026 MarketOps trading sessions, including the explicit September 9, 2026 reconciliation path.

The purpose was to restore deterministic MarketOps freshness after host disruption without fabricating market evidence or reintroducing shared/dedicated database drift.

## Root cause addressed

The outage exposed two operational issues:

- Some scheduled scripts still used fixed `/tmp` lock files. When recovery was executed through a different operator context, stale lock ownership could block the governed deployment-agent path.
- Some historical projections were not session-aware enough after the dedicated MarketOps database split, which made recovered data look stale or absent in user-facing views.

The source fix moved MarketOps scheduled scripts to the shared runtime lock helper and restored session-aware projection behavior for recovered MarketOps evidence.

## Reconciliation outcome

The outage reconciliation path is now constrained to:

- validate that the requested date is a completed trading day;
- reject weekends, holidays, future sessions, and same-day runs before the close guard;
- run only the governed MarketOps recovery sequence;
- keep all status/evidence in the dedicated MarketOps operations tables;
- preserve provider provenance instead of inventing replacement rows.

The September 2026 closure accepts that some missed options-chain evidence cannot be reconstructed from the persisted source tables. Those gaps must remain visible as `provider_evidence_missing`. They must not be reclassified as recovered, current, bullish, bearish, neutral, or zero.

Historical options reconstruction from per-contract endpoints is intentionally out of scope and would require a separate sprint and named approval.

## Closure evidence

Verified evidence captured during closure:

- `sudo -n signalops-deploy-agent scheduler-status` returned active MarketOps timers and success results for the tracked scheduled services.
- `sudo -n signalops-deploy-agent operations-monitor-run` exited successfully.
- `scripts/run_marketops_dashboard_freshness_ui_smoke.sh` passed: `1 passed`.
- `scripts/run_subscription_admin_ui_smoke.sh` passed: `3 passed, 1 skipped`.
- The current session could not run a compact production DB status query because passwordless `sudo docker exec` was unavailable after session refresh. This does not change the closure state because scheduler, operations monitor, and browser validation evidence passed; operators can still re-run the read-only DB evidence query from the host if needed.

## Operator verification query

If a compact database confirmation is needed, run:

```bash
sudo -n docker exec signalops-marketops-postgres-1 \
  psql -U signalops -d marketops -P pager=off -c "
SELECT job_id,status,COALESCE(NULLIF(reason,''),'—') AS reason,
       to_char(completed_at AT TIME ZONE 'UTC','YYYY-MM-DD HH24:MI:SS') AS completed_utc,
       exit_code
FROM marketops_scheduled_job_statuses
WHERE job_id IN (
  'marketops-warm-eod',
  'marketops-daily-postclose',
  'marketops-postclose-recovery',
  'marketops-sri-refresh',
  'marketops-sri-holdings-refresh',
  'marketops-fmp-annual-financial',
  'marketops-intraday'
)
ORDER BY job_id;"
```

## Closure decision

The outage item is closed for the current production-readiness checklist. Remaining work is not outage-specific; it rolls into the broader production-readiness backlog:

- keep monitoring the next natural post-close cycle;
- keep options-chain gaps visible as provider-evidence gaps;
- continue hardening K8s/OpenBao deployment architecture;
- continue mobile and subscription journey regression coverage.
