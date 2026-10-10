# K8S-3 Scheduler Status Parity Evidence — 2026-09-12

Status: closed for one bounded Kubernetes MarketOps non-provider dry-run. No production scheduler authority changed.

## Scope

This gate proves the dedicated Kubernetes MarketOps job runner can write the same scheduler-completion evidence shape used by the current production operations layer:

- `marketops_scheduled_job_statuses`
- `marketops_scheduled_job_runs`

The gate used a one-shot `marketops-fmp-annual-financial` dry-run against the approved non-production `syncratic-runtime-smoke` PostgreSQL service. It did not poll FMP, Massive, State Street, or any other provider.

## Runtime and image controls

A temporary runtime file was created from the existing non-production runtime-smoke PostgreSQL secret and rewritten to Kubernetes `.svc.cluster.local` service DNS. Secret values were not printed and the file was removed after validation.

The job-runner image was hardened for scheduler parity:

- added `postgresql-client` to the `marketops-k8s-job-runner` image so the worker pod can write status evidence directly;
- changed the Kubernetes job entrypoint to record `running` then terminal `succeeded`/`failed` rows;
- records `runner=kubernetes`, schedule label, timestamps, exit code, and detail JSON;
- builds detail JSON inside PostgreSQL from scalar values to avoid shell/`psql` JSON quoting drift;
- changed the publisher to use a no-cache build for this image target so mutable `staging` cannot retain a stale wrapper layer.

During validation, the cluster showed mutable-tag cache ambiguity, so a unique validation tag was used for the final proof:

```text
ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner:k8s-status-parity-20260912c
```

The stable `staging` tag was also refreshed, but production promotion should prefer immutable commit tags when unsuspending CronJobs.

## Passing evidence

Command class:

```bash
SIGNALOPS_MARKETOPS_K8S_JOB_RUNNER_IMAGE=ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner:k8s-status-parity-20260912c SIGNALOPS_K8S_MARKETOPS_DRY_RUN_RUN_ID=k8s-status-parity-20260912T202200Z scripts/run_k8s_marketops_non_provider_dry_run_job.sh /tmp/signalops-openbao-marketops-runtime-staging.env
```

Result:

```text
signalops_k8s_marketops_non_provider_dry_run_job_verified
namespace=signalops-marketops
image=ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner:k8s-status-parity-20260912c
job=signalops-marketops-non-provider-dry-run
job_id=marketops-fmp-annual-financial
dry_run=true
max_assets=1
run_id=k8s-status-parity-20260912T202200Z
scheduler_status_parity=verified
provider_polling=false
production_cutover_allowed=false
```

The harness verified the persisted run row through the non-production runtime-smoke PostgreSQL pod because the host has an unusable `psql` wrapper without a local client package. Expected DB evidence was observed:

```text
succeeded|kubernetes|0|true
```

## Cleanup evidence

Post-run cleanup verified:

- no dry-run Job or Pod remained in `signalops-marketops`;
- temporary NetworkPolicies `allow-marketops-k8s-dry-run-runtime-smoke-postgres-egress` and `allow-marketops-k8s-dry-run-to-postgres` were removed;
- the temporary `/tmp/signalops-openbao-marketops-runtime-staging.env` file was removed;
- all production MarketOps schedules remain under Docker Compose/systemd authority.

`kubectl` continued to emit the known local warning about `/etc/rancher/k3s/config.yaml.d/90-syncratic-cilium.yaml`. It did not block the Kubernetes apply, image pull, job execution, DB verification, or cleanup checks.

## Remaining K8S-3 work

Scheduler-status parity is no longer the blocker. Before any CronJob can be unsuspended, the next gates are:

1. dedicated non-production MarketOps database services — closed by [K8S-3 dedicated staging database gate](k8s3_dedicated_staging_database_gate_2026-09-12.md);
2. publish job-runner images with immutable commit tags after the dedicated staging database gate is committed;
3. remove staging-only TLS skip-verify by installing/proving proper OpenBao CA trust;
4. run a one-CronJob unsuspend test with no provider polling and automatic resuspension;
5. only after that, request a separate named approval for any provider-enabled Kubernetes schedule test.
