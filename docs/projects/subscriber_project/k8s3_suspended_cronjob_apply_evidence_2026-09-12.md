# K8S-3 Suspended MarketOps CronJob Apply Evidence — 2026-09-12

Status: suspended in-cluster CronJob apply/list smoke passed. GHCR job-runner publication is now verified.

## Scope

This gate applies the staging MarketOps scheduled-job overlay to Kubernetes while all CronJobs are suspended. It proves that the K8S scheduler objects can exist in-cluster without executing provider polling, creating Jobs, or creating Pods.

Docker Compose/systemd remains the production scheduler authority.

## Publication attempt

A repeatable publication helper was added:

```bash
scripts/publish_k8s_marketops_job_runner_image.sh .env
```

The initial local image build succeeded but GHCR push failed because the available token could read existing packages but could not create/push the new job-runner package. After the package-write permission was corrected, publication passed:

```text
signalops_k8s_marketops_job_runner_publish_verified
registry=ghcr.io
user=lukebabs
image=ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner
source_tag=826d9d01d6bc
staging_tag=staging
permission=write
```

## Suspended apply/list validation

Command:

```bash
scripts/run_k8s_marketops_suspended_cronjob_apply_smoke.sh
```

Result:

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

signalops_k8s_marketops_suspended_cronjob_apply_smoke_verified
namespace=signalops-marketops
cronjobs=5
suspended=true
concurrency_policy=Forbid
jobs_created=0
pods_created=0
provider_polling=false
production_cutover_allowed=false
```

## In-cluster objects

The following suspended CronJobs are now present in `signalops-marketops`:

- `marketops-intraday`
- `marketops-sri-refresh`
- `marketops-sri-holdings-refresh`
- `marketops-fmp-annual-financial`
- `marketops-saf-benchmark`

They are intentionally not runnable as production replacements yet. The image tag is not published, OpenBao MarketOps runtime staging is not provisioned, and the CronJobs remain suspended.

## Remaining K8S-3 gates

1. GHCR job-runner publication is closed for `ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner:staging`.
2. `signalops-marketops` pull-secret and image pull smoke are closed; the temporary pod succeeded and was removed.
3. Placeholder-only OpenBao MarketOps staging role/path and app-plane denial proof are closed.
4. Replace placeholder values with approved non-production runtime values before any worker dry-run.
5. Run one explicit non-provider dry-run Job after the non-production runtime path is available.
6. Build scheduler-completion parity before any CronJob is unsuspended.
