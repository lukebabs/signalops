# K8S-3 task-retry and warm-EOD dry-run evidence — 2026-09-13

Status: closed for this scheduler-parity slice.

This gate validated the two newly ported non-provider MarketOps scheduled jobs against the dedicated Kubernetes staging MarketOps databases and OpenBao-injected runtime path. No production traffic was cut over and no provider polling was performed.

## Scope

Validated jobs:

- `marketops-task-retry`
- `marketops-warm-eod`

Controls preserved:

- K8S CronJobs remain suspended.
- One-shot Jobs used `backoffLimit: 0`.
- OpenBao runtime injection was used for the worker runtime path.
- The dedicated staging primary and temporal MarketOps services were used.
- `provider_polling=false` and `production_cutover_allowed=false` remained enforced.

## Source/image evidence

Committed fix:

```text
de5c983 Fix K8s MarketOps dry-run status checks
```

Published private GHCR image:

```text
signalops_k8s_marketops_job_runner_publish_verified
registry=ghcr.io
user=lukebabs
image=ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner
source_tag=de5c9833fd04
staging_tag=staging
permission=write
```

Build-time validation included `go test ./...` during the no-cache Docker image build.

## Staging bootstrap evidence

The dedicated staging schema bootstrap passed before the one-shot Jobs ran:

```text
signalops_k8s_staging_enrollment_schema_bootstrap_verified
namespace=signalops-data
pod=marketops-postgres-staging-0
database=marketops
tables=13
provider_polling=false
production_cutover_allowed=false
```

## Task-retry dry-run evidence

```text
signalops_k8s_marketops_non_provider_dry_run_job_verified
namespace=signalops-marketops
image=ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner:staging
job=signalops-marketops-task-retry-dry-run
job_id=marketops-task-retry
dry_run=true
max_assets=1
run_id=k8s-task-retry-parity-20260913T042157Z
scheduler_status_parity=verified
provider_polling=false
production_cutover_allowed=false

signalops_k8s_marketops_dedicated_staging_db_gate_verified
namespace=signalops-data
primary_service=marketops-postgres-staging
temporal_service=marketops-timescaledb-staging
run_id=k8s-task-retry-parity-20260913T042157Z
scheduler_status_parity=verified
provider_polling=false
production_cutover_allowed=false
```

## Warm-EOD dry-run evidence

```text
signalops_k8s_marketops_non_provider_dry_run_job_verified
namespace=signalops-marketops
image=ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner:staging
job=signalops-marketops-warm-eod-dry-run
job_id=marketops-warm-eod
dry_run=true
max_assets=1
run_id=k8s-warm-eod-parity-20260913T042216Z
scheduler_status_parity=verified
provider_polling=false
production_cutover_allowed=false

signalops_k8s_marketops_dedicated_staging_db_gate_verified
namespace=signalops-data
primary_service=marketops-postgres-staging
temporal_service=marketops-timescaledb-staging
run_id=k8s-warm-eod-parity-20260913T042216Z
scheduler_status_parity=verified
provider_polling=false
production_cutover_allowed=false
```

## Notes

The first task-retry attempt exposed two useful hardening fixes before the final pass:

1. The task-retry dry-run command needed SQL dollar quoting so shell `set -u` could not expand `session_date` in the wrong scope.
2. The dry-run verifier now prefers in-cluster status verification for Kubernetes `.svc` database URLs instead of trying to use a host-local `psql` client against cluster DNS.

Both fixes are included in commit `de5c983`. The manifest verifier was also hardened after the broader readiness report exposed a shell `pipefail`/`grep -q` false negative against large rendered YAML; the rendered guard was present, and the corrected verifier now passes.

## Remaining scheduler parity gap

After this closure, the broader K8S scheduler parity gap is reduced to the remaining provider/composite production jobs:

- `marketops-daily-postclose`
- `marketops-fmp-continuation`
- `marketops-postclose-recovery`
- `marketops-risk-reward`

Docker Compose/systemd remains the live production scheduler authority until those remaining parity gates and cutover controls are approved.
