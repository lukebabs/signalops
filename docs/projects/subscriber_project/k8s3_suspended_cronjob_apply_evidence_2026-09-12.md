# K8S-3 Suspended MarketOps CronJob Apply Evidence — 2026-09-12

Status: suspended in-cluster CronJob apply/list smoke passed. GHCR job-runner publication remains blocked by token/package write scope.

## Scope

This gate applies the staging MarketOps scheduled-job overlay to Kubernetes while all CronJobs are suspended. It proves that the K8S scheduler objects can exist in-cluster without executing provider polling, creating Jobs, or creating Pods.

Docker Compose/systemd remains the production scheduler authority.

## Publication attempt

A repeatable publication helper was added:

```bash
scripts/publish_k8s_marketops_job_runner_image.sh .env
```

The local image build succeeded for the current source commit, but GHCR push failed:

```text
denied: permission_denied: The token provided does not match expected scopes.
```

Interpretation: the current `GHCR_KEY` can read existing private packages but cannot create or push the new `ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner` package. The next publication attempt needs a token/account with package write permission for `syncratic-inc`, typically `write:packages` plus the required organization/package permission/SSO authorization.

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

1. Publish and verify `ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner:staging` with a package-write capable token.
2. Refresh the Kubernetes `signalops-ghcr-pull` secret in `signalops-marketops` if the package-read credential changes.
3. Provision `signalops/data/k8s/marketops/marketops-worker-runtime-staging` in OpenBao with non-production values and cross-plane denial evidence.
4. Run one explicit non-provider dry-run Job after the image and runtime path are available.
5. Build scheduler-completion parity before any CronJob is unsuspended.
