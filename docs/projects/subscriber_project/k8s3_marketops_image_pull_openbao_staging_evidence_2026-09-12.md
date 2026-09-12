# K8S-3 MarketOps Image Pull and OpenBao Staging Evidence — 2026-09-12

Status: closed for private GHCR image pull in `signalops-marketops` and placeholder-only OpenBao MarketOps staging role/path. No MarketOps worker executed. No provider polling occurred.

## Image pull smoke

Added repeatable helper:

```bash
scripts/run_k8s_marketops_job_runner_pull_smoke.sh .env
```

The helper refreshes the `signalops-ghcr-pull` Docker registry secret in the `signalops-marketops` namespace from the local `GHCR_KEY`, validates manifest-read access, runs a temporary pod with the job-runner image, overrides the image entrypoint to `echo`, and deletes the pod after success.

Result:

```text
signalops_k8s_marketops_job_runner_pull_smoke_verified
namespace=signalops-marketops
registry=ghcr.io
image=ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner:staging
secret=signalops-ghcr-pull
pod_succeeded=true
worker_executed=false
provider_polling=false
production_cutover_allowed=false
```

Kubernetes evidence showed the image was pulled by digest and the container exited `0` after printing the marker. A follow-up pod listing returned no `signalops-marketops-k8s-job-runner` pods, confirming cleanup.

## OpenBao MarketOps staging role/path

Added repeatable helper:

```bash
scripts/provision_openbao_signalops_marketops_staging.sh .env
```

The helper configures the staging-only MarketOps read policy and Kubernetes auth role, writes placeholder-only values to the MarketOps staging path, proves the MarketOps role can read that path, and proves the app role cannot read it.

Result:

```text
openbao_signalops_marketops_staging_verified
mount=signalops
marketops_role=signalops-marketops
marketops_namespace=signalops-marketops
marketops_service_accounts=signalops-marketops-provider-worker,signalops-marketops-analytics-worker,signalops-marketops-saf-worker,signalops-marketops-syncratic-worker,signalops-marketops-secret-reader
secret_path=signalops/k8s/marketops/marketops-worker-runtime-staging
deny_role=signalops-app
deny_namespace=signalops-app
cross_plane_denied=true
secret_values=placeholder_only
production_cutover_allowed=false
```

## Scheduler object state

The five staged MarketOps CronJobs remain suspended and inactive after this gate:

```text
marketops-fmp-annual-financial   SUSPEND=True   ACTIVE=0   LAST SCHEDULE=<none>
marketops-intraday               SUSPEND=True   ACTIVE=0   LAST SCHEDULE=<none>
marketops-saf-benchmark          SUSPEND=True   ACTIVE=0   LAST SCHEDULE=<none>
marketops-sri-holdings-refresh   SUSPEND=True   ACTIVE=0   LAST SCHEDULE=<none>
marketops-sri-refresh            SUSPEND=True   ACTIVE=0   LAST SCHEDULE=<none>
```

## Remaining K8S-3 gates

1. Replace placeholder-only OpenBao MarketOps runtime values with approved non-production Kubernetes service DNS values.
2. Run the guarded runtime writer added in `scripts/provision_openbao_signalops_marketops_runtime_staging.sh`; it fails closed unless the runtime file contains non-production `.svc` database hosts and explicit approval.
3. Run one explicit non-provider dry-run Job with `scripts/run_k8s_marketops_non_provider_dry_run_job.sh`.
4. Republish the job-runner image after dry-run entrypoint changes are committed.
5. Add DB-backed scheduler-completion parity before any CronJob can be unsuspended.
6. Only after those gates, request named approval for one suspended-to-active staging CronJob test with no provider polling.
