# K8S-2 staging app manifest evidence

Status: manifest gate complete; workloads not applied.

Recorded: 2026-09-12.

## Scope

K8S-2 creates staging-only Kubernetes manifests for the stateless app plane:

- `signalops-web` Deployment and Service;
- `signalops-gateway` Deployment and Service;
- OpenBao Agent Injector annotations for the gateway runtime secret file;
- explicit readiness/liveness probes;
- explicit gateway database pool cap environment variables;
- staging-only labels with `production-cutover-allowed=false`.

The manifests are intentionally not included in `deploy/kubernetes/base/kustomization.yaml`. Applying the base scaffold still does not start app pods.

## Boundary

This gate did not apply the staging app workloads. It did not create Ingress, DNS, Kubernetes Secrets, Jobs, CronJobs, StatefulSets, provider polling, or production traffic changes.

The gateway manifest expects OpenBao to inject this staging-only path:

```text
signalops/data/k8s/app/signalops-gateway-runtime-staging
```

That path must contain non-production or explicitly approved staging values before the Deployments are applied.

## Validation evidence

The staging manifests rendered successfully with Kustomize and passed server-side dry run.

Reusable verifier result:

```text
signalops_k8s_staging_app_manifests_verified
services=2
deployments=2
ingresses=0
cronjobs=0
jobs=0
statefulsets=0
secrets=0
openbao_secret_path=signalops/data/k8s/app/signalops-gateway-runtime-staging
production_cutover_allowed=false
applied=false
```

The server-side dry run would create only:

```text
service/signalops-gateway
deployment.apps/signalops-gateway
service/signalops-web
deployment.apps/signalops-web
```

## Known warnings

`kubectl` continues to emit the existing k3s Cilium config permission warning for `/etc/rancher/k3s/config.yaml.d/90-syncratic-cilium.yaml`. It did not block render or server-side dry-run validation.

## Next step

Before applying K8S-2 workloads, create and verify the OpenBao staging app role/path using non-production test values:

- OpenBao Kubernetes auth role: `signalops-app`;
- bound service accounts: `signalops-gateway`, `signalops-web`, and `signalops-app-secret-reader` in `signalops-app`;
- readable path: `signalops/data/k8s/app/signalops-gateway-runtime-staging`;
- cross-plane denial proof from another plane role.

Only after that should the staging `web` and `gateway` pods be applied and tested through `/healthz`, `/readyz`, and Playwright smokes against a non-production hostname or port-forward.
