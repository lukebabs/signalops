# K8S-3 MarketOps Scheduled Jobs Scaffold — 2026-09-12

Status: source-ready and validated; no live Kubernetes schedule was enabled.

## Purpose

This slice begins the MarketOps worker/CronJob conversion after K8S web/gateway app parity. It does not replace the current host/systemd scheduler. The goal is to create the safe Kubernetes shape first, then promote through later gates only after OpenBao HA, runtime secret paths, data-plane parity, and scheduler completion evidence are approved.

## Why a job-runner image is required

The existing host scheduler scripts are Docker Compose orchestration wrappers. They call Compose services, host-side scripts, and host runtime paths. That model should not be copied into Kubernetes CronJobs.

The Kubernetes path needs a workload-native entrypoint that:

- starts inside the CronJob pod;
- sources only the approved OpenBao-injected runtime file;
- runs compiled MarketOps worker binaries directly;
- keeps data-boundary enforcement enabled;
- avoids broad Compose `.env` injection;
- avoids shelling out to Docker or systemd.

## Source changes

Added a `marketops-k8s-job-runner` Dockerfile target. It includes a minimal Debian runtime with bash, certificates, and the compiled worker binaries needed for the first scheduled-job scaffold:

- `signalops-marketops-intraday-monitor`
- `signalops-marketops-sri-runner`
- `signalops-marketops-sri-holdings-runner`
- `signalops-subscriber-global-annual-financial-task-worker`
- `signalops-subscriber-global-saf-benchmark-materializer`

Added entrypoint:

- `scripts/k8s_marketops_job_entrypoint.sh`

Added suspended staging CronJob overlay:

- `deploy/kubernetes/staging/marketops-jobs/kustomization.yaml`
- `deploy/kubernetes/staging/marketops-jobs/scheduled-cronjobs.yaml`
- `deploy/kubernetes/staging/marketops-jobs/marketops-jobs-openbao-egress.yaml`

Added verifier:

- `scripts/verify_k8s_marketops_scheduled_jobs_manifests.sh`

## CronJobs scaffolded

All CronJobs are explicitly `suspend: true` and use `concurrencyPolicy: Forbid`.

| CronJob | Schedule | Service account | Status |
| --- | --- | --- | --- |
| `marketops-intraday` | every 15 minutes, Mon-Fri 09:00-20:00 ET | `signalops-marketops-provider-worker` | suspended |
| `marketops-sri-refresh` | Mon-Fri 17:05 ET | `signalops-marketops-analytics-worker` | suspended |
| `marketops-sri-holdings-refresh` | Mon-Fri 17:20 ET | `signalops-marketops-provider-worker` | suspended |
| `marketops-fmp-annual-financial` | Saturday 02:30 ET | `signalops-marketops-provider-worker` | suspended |
| `marketops-saf-benchmark` | Mon-Fri 18:45 ET | `signalops-marketops-saf-worker` | suspended |

## Safety controls

- No production DNS change.
- No provider polling.
- No live schedule enabled.
- No Kubernetes Secret committed.
- OpenBao path is staging-only: `signalops/data/k8s/marketops/marketops-worker-runtime-staging`.
- `production-cutover-allowed=false` label is applied.
- CronJobs are suspended by default.
- Each CronJob forbids concurrency.
- Each CronJob has a deadline and backoff policy.

## Validation evidence

Docker target build:

```text
docker build --target marketops-k8s-job-runner -t signalops-marketops-k8s-job-runner:staging .
...
naming to docker.io/library/signalops-marketops-k8s-job-runner:staging
```

Manifest verifier:

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

## Remaining gates before K8S scheduler activation

1. Publish `ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner:staging` and verify private GHCR pull. Current attempt is blocked because the available GHCR token can read existing packages but cannot push/create this package.
2. Provision the OpenBao `signalops-marketops` staging runtime path with non-production values and prove app-plane denial.
3. Suspended-CronJob apply/list smoke is closed: five CronJobs are present in `signalops-marketops`, all suspended, with zero Jobs and zero Pods created.
4. Add a non-provider dry-run Job for one worker where supported.
5. Build DB-backed scheduler-completion parity so Admin Operations Health can read K8S job status exactly as it reads host/systemd status today.
6. Only after the above, request a named approval to unsuspend one staging CronJob with no provider polling.
