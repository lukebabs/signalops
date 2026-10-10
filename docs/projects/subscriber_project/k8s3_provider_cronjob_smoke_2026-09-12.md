# K8S-3 provider-enabled MarketOps CronJob smoke — 2026-09-12

Status: passed for the approved one-asset provider-enabled staging gate.

## Named approval

Luke approved one provider-enabled Kubernetes MarketOps schedule smoke for the staged `marketops-fmp-annual-financial` CronJob, limited to one asset and one Job execution, with immediate CronJob resuspension, no retries, no production traffic cutover, and automatic cleanup after evidence capture.

## Scope and controls

The smoke used the staged `marketops-fmp-annual-financial` CronJob in namespace `signalops-marketops` and the dedicated non-production MarketOps databases in `signalops-data`.

Controls enforced:

- One staged CronJob only: `marketops-fmp-annual-financial`.
- One Kubernetes Job execution created through the CronJob controller.
- Immediate CronJob resuspension after Job creation.
- One warm asset: synthetic staging AAPL row `k8s-staging-aapl`.
- No retry behavior: `MARKETOPS_FMP_ANNUAL_MAX_RETRIES=0`, producing `max_attempts=1`.
- One FMP provider call verified by worker output and DB task evidence.
- No production traffic cutover.
- OpenBao-injected runtime path remained scoped to `signalops/data/k8s/marketops/marketops-worker-runtime-staging`.
- Private GHCR image was pinned to immutable committed tag `ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner:5012610ac808`.

## Implementation notes

Before the successful smoke, two controlled staging-only issues were exposed and corrected:

1. The lower-level annual refresh worker already accepted provider execution but the Kubernetes path calls `subscriber-global-annual-financial-task-worker`. The task worker now accepts `--max-retries` and `--correlation-id`, and the K8S entrypoint passes the run id through as provider-evidence correlation.
2. Existing staging tables from prior dry-run gates were minimal. The provider smoke harness now idempotently prepares the workflow/task/evidence tables and missing warm-cohort columns required by the task-worker path.

Normal production behavior is preserved: task-worker default retry posture remains equivalent to the existing setting unless an operator explicitly overrides `--max-retries`.

## Image and test evidence

Committed source used for the passing provider smoke:

```text
5012610 — Fix K8S FMP provider smoke controls
```

Published image:

```text
signalops_k8s_marketops_job_runner_publish_verified
registry=ghcr.io
user=lukebabs
image=ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner
source_tag=5012610ac808
staging_tag=staging
permission=write
```

DB-backed evidence captured after the approved provider execution:

```text
scheduler|succeeded|kubernetes-provider-cronjob-smoke|0|false
task|AAPL|succeeded|1|1
records|1
```

Interpretation:

- Scheduler status row succeeded.
- Runner was `kubernetes-provider-cronjob-smoke`.
- Exit code was `0`.
- Dry-run was `false`, as expected for this provider-enabled smoke.
- AAPL task succeeded.
- `attempt_count=1` and `max_attempts=1`, proving no retry execution.
- Exactly one correlated FMP annual evidence record was written.

Post-smoke cleanup was verified: all staged MarketOps CronJobs restored to `suspend=true`, and no Jobs remained in `signalops-marketops`.

## Result

K8S-3 now has provider-enabled scheduler proof for one tightly bounded MarketOps worker path: OpenBao runtime injection, private GHCR pull, Kubernetes CronJob controller execution, one provider call, no retries, immutable evidence append, DB-backed scheduler status, immediate resuspend, and cleanup.

Docker Compose/systemd remains the production scheduler authority until the broader production cutover gates close.
