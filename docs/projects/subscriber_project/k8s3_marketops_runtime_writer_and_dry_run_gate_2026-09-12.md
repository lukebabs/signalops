# K8S-3 MarketOps Runtime Writer and Dry-Run Gate — 2026-09-12

Status: closed for one non-provider Kubernetes Job dry-run against the non-production runtime-smoke database. Production cutover remains blocked.

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

## Refreshed job-runner image publication

After the dry-run entrypoint change was committed, the dedicated MarketOps K8S job-runner image was republished to private GHCR:

```text
signalops_k8s_marketops_job_runner_publish_verified
registry=ghcr.io
user=lukebabs
image=ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner
source_tag=b4e653dd1cce
staging_tag=staging
permission=write
```

The `signalops-marketops` namespace then pulled the refreshed `staging` image through the private `signalops-ghcr-pull` secret without executing a worker:

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


## Deployment-agent action prepared

The deployment agent now has a narrow action for this gate:

```bash
sudo -n signalops-deploy-agent k8s-marketops-non-provider-dry-run
```

The action runs only `scripts/run_k8s_marketops_non_provider_dry_run_job.sh`. It is not exposed through the Admin run-now bridge because it is an infrastructure staging gate, not a user-facing operations-health action.

Local fail-closed proof before reprovisioning/installing the action:

```text
signalops_k8s_marketops_non_provider_dry_run_job_failed: runtime env file is missing or unreadable: /etc/signalops/openbao-signalops-marketops-staging-runtime.env
```

No Kubernetes Job was created in this failure path.


## Live dry-run gate passed

With explicit permission to create the runtime file, the gate used a temporary operator-owned runtime file at:

```text
/tmp/signalops-openbao-marketops-runtime-staging.env
```

The file was populated from the existing non-production `syncratic-runtime-smoke` PostgreSQL secret, rewritten to the Kubernetes service DNS host:

```text
syncratic-refactor-smoke-syncratic-phase1-postgres.syncratic-runtime-smoke.svc.cluster.local
```

Secret values were not printed. The temporary file was removed after the gate completed.

The runtime-smoke database was prepared with only the minimum dry-run objects needed by the FMP annual worker dry-run path:

- role `signalops_subscriber_global_eod`;
- table `subscriber_global_warm_eod_assets`;
- one synthetic warm asset row for `AAPL`;
- `SELECT` grant for the dry-run role.

OpenBao runtime write passed:

```text
openbao_signalops_marketops_runtime_staging_verified
mount=signalops
marketops_role=signalops-marketops
marketops_namespace=signalops-marketops
marketops_service_account=signalops-marketops-provider-worker
secret_path=signalops/k8s/marketops/marketops-worker-runtime-staging
deny_role=signalops-app
deny_namespace=signalops-app
cross_plane_denied=true
secret_values=non_production_runtime_supplied
production_cutover_allowed=false
```

The first pod run exposed the same OpenBao injector protocol issue seen earlier in K8S-2: the injected agent attempted HTTP against an HTTPS-only OpenBao service. The dry-run harness now pins the injector to:

```text
vault.hashicorp.com/service=https://openbao.openbao.svc:8200
vault.hashicorp.com/tls-skip-verify=true
```

This is staging-only and must be replaced with proper CA trust before production workload cutover.

The second run showed the MarketOps worker completed successfully while the OpenBao sidecar kept the Kubernetes Job active. The harness now treats the `marketops-job` container exit code plus dry-run log marker as authoritative.

Final successful gate output:

```text
signalops_k8s_marketops_non_provider_dry_run_job_verified
namespace=signalops-marketops
image=ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner:staging
job=signalops-marketops-non-provider-dry-run
job_id=marketops-fmp-annual-financial
dry_run=true
max_assets=1
provider_polling=false
production_cutover_allowed=false
```

Cleanup evidence:

```text
No resources found in signalops-marketops namespace.
temporary_policy_absent=signalops-marketops/allow-marketops-k8s-dry-run-runtime-smoke-postgres-egress
temporary_policy_absent=syncratic-runtime-smoke/allow-marketops-k8s-dry-run-to-postgres
```

## Current blocker

The default protected runtime-file path remains unresolved for future repeatability:

```text
/etc/signalops/openbao-signalops-marketops-staging-runtime.env
```

could not run in this session because passwordless sudo was not available for that path:

```text
sudo: a password is required
```

For this gate, an explicit temporary runtime file under `/tmp` was used and removed after success. The default `/etc` path still requires either passwordless deployment-agent handling or operator creation for future repeatability.

## Next gate

Move from the runtime-smoke database to dedicated non-production SignalOps/MarketOps database services, then run:

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

The image has already been republished with the dry-run entrypoint change, and the dry-run Job passed. The staged CronJob path can now move toward DB-backed scheduler-completion parity.
