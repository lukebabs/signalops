# K8s EOD reconciliation and SAF worker grant evidence — 2026-09-15

## Outcome

The 2026-09-14 EOD session was reconciled into the dedicated K8s MarketOps databases without replacing either database or restoring a Docker volume. The live UI was then validated with the read-only Playwright subscriber smoke: 3 tests passed.

K8s verification showed current 2026-09-14 state for normalized EOD history, Market State, Risk/Reward, Valuation, Options, and SRI. The SRI refresh completed successfully after the dedicated MarketOps database URL aliases were added to the production OpenBao worker secret.

## SAF remediation

The production OpenBao path `signalops/k8s/marketops/marketops-worker-runtime-production` was missing the dedicated global-EOD aliases consumed by the SAF materializer. The aliases now point to the existing dedicated MarketOps primary, temporal, and status URLs. No credentials were rotated or exposed.

K8s SAF validation initially exposed missing read grants on the canonical SAF view and its source tables, followed by the append-only benchmark writer grant. Migration `000175_subscriber_global_saf_worker_runtime_grants` records the least-privilege grants for both dedicated global-EOD roles. The final bounded K8s SAF job completed successfully:

`observations=500 benchmark_rows=1000 inserted=1000 matched=1000 sector_unmapped=0 price_unavailable=0 calculation_version=saf_benchmark.k8s_staging`

Manual validation Jobs were removed after evidence capture. Docker/systemd MarketOps schedulers remain disabled; K8s CronJobs remain the scheduler authority.

## Remaining operational note

The one-shot session reconciliation is currently an operator workflow (`marketops-outage-reconcile:<date>` plus the additive K8s data sync). A follow-up deployment-agent action should make that additive session sync repeatable and auditable; it is not required for the validated 2026-09-14 closure.
