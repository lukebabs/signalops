# K8S-3 MarketOps Runtime Writer and Dry-Run Gate — 2026-09-12

Status: source package prepared and validated. Live OpenBao runtime write and pod dry-run execution remain blocked until the protected non-production runtime file is present and readable to the operator context.

## What changed

Added:

```bash
scripts/provision_openbao_signalops_marketops_runtime_staging.sh
scripts/run_k8s_marketops_non_provider_dry_run_job.sh
docs/projects/subscriber_project/k8s3_marketops_runtime_env_template.md
```

Updated:

```bash
scripts/k8s_marketops_job_entrypoint.sh
```

The Kubernetes MarketOps job entrypoint now supports dry-run dispatch for jobs whose worker binaries already expose a dry-run contract:

- `marketops-intraday`
- `marketops-fmp-annual-financial`
- `marketops-saf-benchmark`

SRI refresh jobs remain unchanged because those runners do not yet expose an equivalent non-provider dry-run contract.

## Runtime writer safety boundary

The new MarketOps OpenBao runtime writer writes only to:

```text
signalops/k8s/marketops/marketops-worker-runtime-staging
```

It fails closed unless all of the following are true:

- `SIGNALOPS_K8S_MARKETOPS_RUNTIME_NON_PRODUCTION_APPROVED=true` is present.
- Required SignalOps and MarketOps database URLs are present.
- Every required PostgreSQL URL uses a Kubernetes service DNS hostname containing `.svc`.
- Placeholder `.invalid` hosts are absent.
- Docker Compose service names such as `marketops-postgres` and `marketops-timescaledb` are absent.
- Localhost and production-like host fragments are absent.
- The `signalops-marketops` Kubernetes auth role can read the path.
- The `signalops-app` role cannot read the path.

No secret values are printed by the script.

## Dry-run Job safety boundary

The new pod-level dry-run harness is constrained to:

- one explicit Kubernetes Job;
- namespace `signalops-marketops`;
- service account `signalops-marketops-provider-worker`;
- published image `ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner:staging`;
- job id `marketops-fmp-annual-financial`;
- `MARKETOPS_K8S_DRY_RUN=true`;
- `MARKETOPS_FMP_ANNUAL_MAX_ASSETS=1`;
- `production_cutover_allowed=false`.

The harness calls the runtime writer first. If the runtime writer fails, the Job is not created.

## Validation completed

```text
bash -n scripts/k8s_marketops_job_entrypoint.sh scripts/provision_openbao_signalops_marketops_runtime_staging.sh scripts/run_k8s_marketops_non_provider_dry_run_job.sh scripts/provision_openbao_signalops_marketops_staging.sh
git diff --check
scripts/verify_k8s_marketops_scheduled_jobs_manifests.sh
```

Manifest verification passed with:

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

## Current blocker

The protected runtime-file probe for:

```text
/etc/signalops/openbao-signalops-marketops-staging-runtime.env
```

could not run in this session because passwordless sudo was not available for that path:

```text
sudo: a password is required
```

The file contents were not printed or inferred. The dry-run Job was not started.

## Next gate

Provide the protected runtime file using the template in `k8s3_marketops_runtime_env_template.md`, then run:

```bash
scripts/run_k8s_marketops_non_provider_dry_run_job.sh /etc/signalops/openbao-signalops-marketops-staging-runtime.env
```

The expected successful result is:

```text
signalops_k8s_marketops_non_provider_dry_run_job_verified
dry_run=true
provider_polling=false
production_cutover_allowed=false
```

Only after that passes should the image be republished with the dry-run entrypoint change and the staged CronJob path move toward DB-backed scheduler-completion parity.
