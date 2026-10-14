# K8S-3 Dedicated Staging Database Gate — 2026-09-12

Status: closed for one bounded Kubernetes MarketOps non-provider dry-run against dedicated non-production MarketOps database services. No production scheduler authority changed.

## Scope

This gate replaces the earlier `syncratic-runtime-smoke` database dependency for MarketOps worker validation with dedicated staging-only MarketOps data services in `signalops-data`:

- `marketops-postgres-staging` for the MarketOps primary database;
- `marketops-timescaledb-staging` for the MarketOps temporal database.

The services are ClusterIP-only, have no Ingress, and use runtime-generated Kubernetes Secrets. No database passwords or OpenBao tokens are committed to source.

## Source controls added

- `deploy/kubernetes/staging/marketops-data/kustomization.yaml`
- `deploy/kubernetes/staging/marketops-data/marketops-staging-databases.yaml`
- `scripts/verify_k8s_marketops_dedicated_staging_data_manifests.sh`
- `scripts/run_k8s_marketops_dedicated_staging_db_gate.sh`

The manifest verifier asserts:

- exactly 2 Services and 2 StatefulSets;
- 0 committed Secrets;
- 0 Jobs/CronJobs;
- 0 Ingress resources;
- namespace `signalops-data`;
- staging labels including `production-cutover-allowed=false`;
- disposable `local-path` storage for the non-production gate.

Passing marker:

```text
signalops_k8s_marketops_dedicated_staging_data_manifests_verified
namespace=signalops-data
primary_service=marketops-postgres-staging
temporal_service=marketops-timescaledb-staging
services=2
statefulsets=2
committed_secrets=0
production_cutover_allowed=false
```

## Runtime controls

The gate runner:

1. creates `marketops-postgres-staging-auth` and `marketops-timescaledb-staging-auth` only if missing;
2. applies the staging data manifests;
3. waits for both StatefulSets to roll out;
4. seeds only the minimum non-production scheduler-parity tables and one synthetic warm EOD asset row;
5. writes a temporary runtime env file under `/tmp` with Kubernetes `.svc.cluster.local` database URLs;
6. provisions the OpenBao MarketOps staging runtime path without printing secret values;
7. runs the existing one-shot non-provider dry-run harness;
8. verifies DB-backed scheduler status parity;
9. deletes the temporary dry-run Job/Pod and temporary NetworkPolicies;
10. removes the temporary runtime env file.

## Passing evidence

Command class:

```bash
scripts/run_k8s_marketops_dedicated_staging_db_gate.sh .env
```

Result:

```text
signalops_k8s_marketops_non_provider_dry_run_job_verified
namespace=signalops-marketops
image=ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner:staging
job=signalops-marketops-non-provider-dry-run
job_id=marketops-fmp-annual-financial
dry_run=true
max_assets=1
run_id=k8s-dedicated-db-parity-20260912T202846Z
scheduler_status_parity=verified
provider_polling=false
production_cutover_allowed=false
signalops_k8s_marketops_dedicated_staging_db_gate_verified
namespace=signalops-data
primary_service=marketops-postgres-staging
temporal_service=marketops-timescaledb-staging
run_id=k8s-dedicated-db-parity-20260912T202846Z
scheduler_status_parity=verified
provider_polling=false
production_cutover_allowed=false
```

Post-run state:

```text
statefulset.apps/marketops-postgres-staging      1/1
statefulset.apps/marketops-timescaledb-staging   1/1
service/marketops-postgres-staging               ClusterIP 5432/TCP
service/marketops-timescaledb-staging            ClusterIP 5432/TCP
pod/marketops-postgres-staging-0                 Running
pod/marketops-timescaledb-staging-0              Running
```

Temporary dry-run Job/Pod lookup returned no resources.

## Defects found and fixed during the gate

- The first attempt exposed that the staging seed step used a here-doc through `kubectl exec` without `-i`; the command exited without loading SQL. The gate runner now uses `kubectl exec -i` for SQL seed input.
- The MarketOps OpenBao runtime writer now includes `SIGNALOPS_K8S_STATUS_DATABASE_URL`, preventing scheduler-status recording from falling back to an unintended database.
- The staging scheduler table seed now matches the Kubernetes job entrypoint columns used by `marketops_scheduled_job_statuses` and `marketops_scheduled_job_runs`.

## Boundary

This was a staging-only Kubernetes validation. It did not:

- enable or unsuspend any CronJob;
- poll FMP, Massive, State Street, or any market-data provider;
- change production DNS;
- change Docker Compose/systemd production scheduler authority;
- migrate production secrets into Kubernetes.

`kubectl` continued to emit the known local warning about `/etc/rancher/k3s/config.yaml.d/90-syncratic-cilium.yaml`. It did not block apply, rollout, job execution, DB verification, or cleanup.

## Immutable image publication follow-up

After this gate was committed, the job-runner image was published with the immutable commit-derived tag:

```text
ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner:6488950e98df
```

Publication evidence:

```text
signalops_k8s_marketops_job_runner_publish_verified
registry=ghcr.io
user=lukebabs
image=ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner
source_tag=6488950e98df
staging_tag=staging
permission=write
```

Kubernetes pull smoke against the immutable tag also passed:

```text
signalops_k8s_marketops_job_runner_pull_smoke_verified
namespace=signalops-marketops
registry=ghcr.io
image=ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner:6488950e98df
secret=signalops-ghcr-pull
pod_succeeded=true
worker_executed=false
provider_polling=false
production_cutover_allowed=false
```

## Remaining K8S-3 work

Dedicated non-production MarketOps data services are no longer the blocker. The remaining K8S-3 gates are:

1. OpenBao CA trust — closed by [K8S-3 OpenBao CA trust gate](k8s3_openbao_ca_trust_gate_2026-09-12.md);
2. run one approved CronJob unsuspend/resuspend smoke with no provider polling;
3. only after that, request a separate named approval for any provider-enabled Kubernetes schedule test.
