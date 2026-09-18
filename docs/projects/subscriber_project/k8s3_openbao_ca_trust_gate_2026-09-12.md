# K8S-3 OpenBao CA Trust Gate — 2026-09-12

Status: closed for SignalOps staging app and MarketOps Kubernetes worker paths. No production scheduler authority changed.

## Scope

This gate removes the staging-only `vault.hashicorp.com/tls-skip-verify=true` posture from SignalOps staging OpenBao injector annotations and proves workload injection through an explicit CA trust bundle.

The gate applies to:

- `signalops-app` staging gateway OpenBao injection;
- `signalops-marketops` suspended CronJob OpenBao injection;
- the one-shot MarketOps non-provider dry-run harness.

## CA source and validation

The existing platform trust bundle `syncratic-connect/syncratic-openbao-ca` was verified against the current OpenBao server certificate before distribution.

Important finding: `openbao/openbao-root-ca` contains only `Syncratic Core Root CA` and does not validate the current OpenBao server certificate by itself. The current OpenBao server certificate is issued by `Syncratic Core Intermediate CA`, so the workload trust bundle must include the intermediate chain.

Verified source bundle:

```text
syncratic-connect/syncratic-openbao-ca:ca.crt
```

The chain verification passed:

```text
/tmp/.../tls.crt: OK
subject=CN = Syncratic Core Intermediate CA
issuer=CN = Syncratic Core Root CA
subject=CN = Syncratic Core Root CA
issuer=CN = Syncratic Core Root CA
```

## Source controls added

- `scripts/provision_signalops_openbao_ca_trust.sh`

The script:

1. reads the public CA bundle from `syncratic-connect/syncratic-openbao-ca`;
2. reads the current public OpenBao server certificate from `openbao/openbao-server-tls`;
3. verifies the server certificate against the CA bundle using `openssl verify`;
4. writes a `signalops-openbao-ca` Kubernetes Secret containing only `ca.crt` into `signalops-app` and `signalops-marketops`;
5. emits no certificate body, private key, token, or runtime secret value.

Passing marker:

```text
signalops_openbao_ca_trust_verified
source_configmap=syncratic-connect/syncratic-openbao-ca
source_key=ca.crt
target_secret=signalops-openbao-ca
target_namespaces=signalops-app,signalops-marketops
server_certificate_validated=true
secret_values=public_ca_bundle_only
production_cutover_allowed=false
```

## Annotation change

The staging gateway, suspended MarketOps CronJobs, and one-shot dry-run harness now use:

```yaml
vault.hashicorp.com/tls-secret: "signalops-openbao-ca"
vault.hashicorp.com/ca-cert: "/vault/tls/ca.crt"
```

They no longer use:

```yaml
vault.hashicorp.com/tls-skip-verify: "true"
```

The app and MarketOps manifest verifiers now fail if `tls-skip-verify` reappears.

## Passing validation

Static/server-side manifest gates passed:

```text
signalops_k8s_staging_app_manifests_verified
services=2
deployments=2
ingresses=0
cronjobs=0
jobs=0
statefulsets=0
secrets=0
configmaps=1
networkpolicies=4
production_cutover_allowed=false
applied=false
```

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
production_cutover_allowed=false
applied=false
```

Live no-provider proof was rerun against the dedicated non-production MarketOps staging databases using the immutable job-runner image:

```text
signalops_k8s_marketops_non_provider_dry_run_job_verified
namespace=signalops-marketops
image=ghcr.io/syncratic-inc/signalops-marketops-k8s-job-runner:6488950e98df
job=signalops-marketops-non-provider-dry-run
job_id=marketops-fmp-annual-financial
dry_run=true
max_assets=1
run_id=k8s-ca-trust-parity-20260912T212930Z
scheduler_status_parity=verified
provider_polling=false
production_cutover_allowed=false
signalops_k8s_marketops_dedicated_staging_db_gate_verified
namespace=signalops-data
primary_service=marketops-postgres-staging
temporal_service=marketops-timescaledb-staging
run_id=k8s-ca-trust-parity-20260912T212930Z
scheduler_status_parity=verified
provider_polling=false
production_cutover_allowed=false
```

Post-run cleanup/state checks:

- `signalops-app/signalops-openbao-ca` contains only `ca.crt`;
- `signalops-marketops/signalops-openbao-ca` contains only `ca.crt`;
- the temporary dry-run Job/Pod was removed;
- Docker Compose/systemd remains production scheduler authority.

## Boundary

This gate did not:

- enable or unsuspend any CronJob;
- poll FMP, Massive, State Street, or any other provider;
- change production DNS;
- migrate production secrets into Kubernetes;
- change Docker Compose/systemd production authority.

The local `kubectl` warning about `/etc/rancher/k3s/config.yaml.d/90-syncratic-cilium.yaml` persisted and remains non-blocking for these gates.

## Remaining K8S-3 work

The remaining K8S-3 gate is one separately approved CronJob unsuspend/resuspend smoke with no provider polling. Provider-enabled Kubernetes schedule testing remains a separate future named approval.
