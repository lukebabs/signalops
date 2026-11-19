# K8S-3 MarketOps CronJob unsuspend/resuspend smoke — 2026-09-12

Status: passed for the constrained no-provider staging gate.

## Scope

This gate exercised one staged MarketOps Kubernetes CronJob through the controller path without enabling production schedules. The selected CronJob was `marketops-fmp-annual-financial` in namespace `signalops-marketops`.

The test was intentionally bounded:

- `suspend=true` was required before the smoke started.
- The CronJob was temporarily patched to `* * * * *` and `suspend=false`.
- The worker image was pinned to `ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner:6488950e98df`.
- The worker ran with `MARKETOPS_K8S_DRY_RUN=true` and `MARKETOPS_FMP_ANNUAL_MAX_ASSETS=1`.
- Provider polling remained disabled.
- `signalops.syncratic.io/production-cutover-allowed=false` remained in effect.
- The CronJob was resuspended immediately after the controller created one Job.
- The smoke Job was deleted after validation.

## Initial failure and permanent fix

The first CronJob smoke created a pod but the OpenBao init container stayed in `PodInitializing`. Its logs showed repeated connection resets while authenticating to `openbao.openbao.svc:8200`.

Root cause: the staged CronJob manifests had CA trust annotations but did not explicitly set `vault.hashicorp.com/service`. The injector therefore defaulted to `http://openbao.openbao.svc:8200`, while the OpenBao endpoint requires HTTPS.

Permanent fix:

- Added `vault.hashicorp.com/service: "https://openbao.openbao.svc:8200"` to all five staged MarketOps CronJobs.
- Updated `scripts/verify_k8s_marketops_scheduled_jobs_manifests.sh` to fail if the HTTPS service annotation is absent.
- Retained the prior guard that rejects `vault.hashicorp.com/tls-skip-verify` in staged MarketOps CronJobs.

A follow-up smoke then reached the worker container and exposed a test-harness issue: the temporary patch replaced the container spec without preserving the CronJob job argument. The smoke script was corrected to pass the selected CronJob name as the worker argument.

## Passing evidence

Final command:

```bash
SIGNALOPS_K8S_MARKETOPS_CRONJOB_RUN_ID=k8s-cronjob-unsuspend-20260912T214635Z \
  scripts/run_k8s_marketops_cronjob_unsuspend_smoke.sh .env
```

Final evidence:

```text
signalops_k8s_marketops_cronjob_unsuspend_smoke_verified
namespace=signalops-marketops
cronjob=marketops-fmp-annual-financial
image=ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner:6488950e98df
run_id=k8s-cronjob-unsuspend-20260912T214635Z
job_created=true
jobs_created=1
resuspended=true
scheduler_status_parity=verified
provider_polling=false
production_cutover_allowed=false
```

Manifest verifier after the fix:

```text
signalops_k8s_marketops_scheduled_jobs_manifests_verified
cronjobs=5
networkpolicies=1
secrets=0
services=0
deployments=0
statefulsets=0
suspended=true
concurrency_policy=Forbid
openbao_secret_path=signalops/data/k8s/marketops/marketops-worker-runtime-staging
production_cutover_allowed=false
applied=false
```

Post-smoke cluster cleanup was verified: all staged MarketOps CronJobs were restored to `suspend=true`, and no smoke Jobs remained in `signalops-marketops`.

## Result

K8S-3 no-provider scheduler mechanics are now verified through the Kubernetes CronJob controller, OpenBao-injected runtime, private GHCR image pull, dedicated non-production MarketOps database routing, and DB-backed scheduler status parity.

Docker Compose/systemd remains the production scheduler authority. Provider-enabled Kubernetes schedules still require a separate named approval and production cutover gate.
