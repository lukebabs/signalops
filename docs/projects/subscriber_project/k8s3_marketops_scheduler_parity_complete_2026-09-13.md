# K8S-3 MarketOps scheduler parity complete — 2026-09-13

Status: closed for workload representation and staging dry-run execution. Production scheduler authority has not been moved yet.

## Outcome

All Admin MarketOps scheduled jobs now have both:

1. a suspended Kubernetes CronJob in `signalops-marketops`; and
2. a matching `scripts/k8s_marketops_job_entrypoint.sh` handler.

The source-controlled parity report now returns:

```text
signalops_k8s_marketops_scheduler_parity_coverage_report
status=complete
admin_marketops_jobs=12
k8s_cronjobs=13
k8s_entrypoint_jobs=13
missing_cronjob=none
missing_entrypoint=none
extra_cronjob=marketops-saf-benchmark
provider_polling=false
production_cutover_allowed=false
```

The extra CronJob is intentional: `marketops-saf-benchmark` is an analytical SAF support job and is not currently listed in the Admin scheduler catalog.

## Newly represented jobs

This slice added K3s workload representation for the final four Admin MarketOps jobs:

- `marketops-daily-postclose`
- `marketops-postclose-recovery`
- `marketops-risk-reward`
- `marketops-fmp-continuation`

All four are staged as suspended CronJobs with:

- `concurrencyPolicy: Forbid`;
- bounded deadlines;
- OpenBao runtime injection;
- private GHCR image pull;
- `production-cutover-allowed=false`;
- dry-run mode enforced until a separate production scheduler cutover approval.

## Image evidence

Committed source state:

```text
1aecab4 Complete K8s MarketOps scheduler parity
```

Published private GHCR image:

```text
signalops_k8s_marketops_job_runner_publish_verified
registry=ghcr.io
user=lukebabs
image=ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner
source_tag=1aecab496a42
staging_tag=staging
permission=write
```

The no-cache Docker build executed `go test ./...` successfully.

## Live K3s dry-run evidence

Each newly represented job was executed once through the bounded staging dry-run harness against the dedicated non-production MarketOps primary/temporal services in `signalops-data`. No provider polling and no production traffic cutover occurred.

### marketops-daily-postclose

```text
signalops_k8s_marketops_non_provider_dry_run_job_verified
namespace=signalops-marketops
image=ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner:staging
job=signalops-marketops-daily-postclose-dry-run
job_id=marketops-daily-postclose
dry_run=true
run_id=k8s-marketops-daily-postclose-parity-20260913T043335Z
scheduler_status_parity=verified
provider_polling=false
production_cutover_allowed=false
```

### marketops-postclose-recovery

```text
signalops_k8s_marketops_non_provider_dry_run_job_verified
job=signalops-marketops-postclose-recovery-dry-run
job_id=marketops-postclose-recovery
dry_run=true
run_id=k8s-marketops-postclose-recovery-parity-20260913T043344Z
scheduler_status_parity=verified
provider_polling=false
production_cutover_allowed=false
```

### marketops-risk-reward

```text
signalops_k8s_marketops_non_provider_dry_run_job_verified
job=signalops-marketops-risk-reward-dry-run
job_id=marketops-risk-reward
dry_run=true
run_id=k8s-marketops-risk-reward-parity-20260913T043349Z
scheduler_status_parity=verified
provider_polling=false
production_cutover_allowed=false
```

### marketops-fmp-continuation

```text
signalops_k8s_marketops_non_provider_dry_run_job_verified
job=signalops-marketops-fmp-continuation-dry-run
job_id=marketops-fmp-continuation
dry_run=true
run_id=k8s-marketops-fmp-continuation-parity-20260913T043354Z
scheduler_status_parity=verified
provider_polling=false
production_cutover_allowed=false
```

## Readiness report

The non-cutover production-readiness report now shows scheduler parity complete while correctly preserving production authority outside K3s:

```text
production_cutover_allowed=false
compose_systemd_production_authority=true
k8s_marketops_scheduler_parity_coverage=complete
k8s_marketops_scheduler_missing_cronjob=none
k8s_marketops_scheduler_missing_entrypoint=none
pending_signal_connect_ingestion_shadow=true
pending_service_mesh_ingress_dns_cutover_plan=true
pending_capacity_load_validation=true
```

## Remaining before K3s becomes production authority

Scheduler parity is no longer the blocker. The remaining production-migration gates are now:

1. Signal-Connect ingestion shadow/parity in K3s.
2. Istio/Gateway API production ingress and DNS rollback plan.
3. Stripe webhook parity through the K3s route.
4. Capacity/load validation, including database connection saturation and scheduler queue lag.
5. Explicit production scheduler cutover approval to unsuspend K3s CronJobs and disable equivalent systemd timers in a controlled window.
