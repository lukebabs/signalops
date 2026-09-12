# K8S-2 staging app manifest evidence

Status: manifest gate complete; staging workload apply mechanics exercised and scaled back to zero pending image publication/import.

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

This gate initially prepared the staging app workloads, then exercised the staging apply mechanics after OpenBao app-role verification. It did not create Ingress, DNS, Kubernetes Secrets, Jobs, CronJobs, StatefulSets, provider polling, or production traffic changes.

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


## Live apply evidence

After `openbao_signalops_app_staging_verified` was captured, the staging overlay was applied to the cluster:

```text
service/signalops-gateway created
service/signalops-web created
deployment.apps/signalops-gateway created
deployment.apps/signalops-web created
```

The first gateway pod proved the OpenBao injector path was active, but the injected agent initially attempted HTTP against the HTTPS-only OpenBao service. The base scaffold was updated with a narrow `signalops-app` gateway-to-OpenBao egress policy, and the staging gateway overlay now pins the injector service to `https://openbao.openbao.svc:8200` with `vault.hashicorp.com/tls-skip-verify: "true"` for staging only. Production must replace this with proper CA trust before any cutover.

The corrected gateway pod advanced past OpenBao agent startup. The remaining rollout blocker was not OpenBao policy authorization; it was image availability:

```text
Failed to pull image "signalops-web:staging": pull access denied for docker.io/library/signalops-web:staging
Failed to pull image "signalops-gateway:staging": pull access denied for docker.io/library/signalops-gateway:staging
```

The local Docker images were tagged as `signalops-web:staging` and `signalops-gateway:staging`, but importing them into k3s containerd requires an interactive host sudo password. To avoid noisy non-production `ImagePullBackOff` pods, both staging Deployments were scaled to zero. Final cluster state:

```text
deployment.apps/signalops-gateway   0/0
deployment.apps/signalops-web       0/0
service/signalops-gateway           ClusterIP 8080/TCP
service/signalops-web               ClusterIP 8080/TCP
```

Final verifier results:

```text
signalops_k8s_base_scaffold_verified
namespaces=7
service_accounts=29
network_policies=16
policy_configmaps=2
workload_pods=0
production_cutover_allowed=false

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

## Known warnings

`kubectl` continues to emit the existing k3s Cilium config permission warning for `/etc/rancher/k3s/config.yaml.d/90-syncratic-cilium.yaml`. It did not block render or server-side dry-run validation.

## Next step

Before running K8S-2 workload readiness, publish/import staging images into the Kubernetes runtime or point the manifests at an approved staging registry image. The OpenBao app role/path prerequisite is complete:

- OpenBao Kubernetes auth role: `signalops-app`;
- bound service accounts: `signalops-gateway`, `signalops-web`, and `signalops-app-secret-reader` in `signalops-app`;
- readable path: `signalops/data/k8s/app/signalops-gateway-runtime-staging`;
- cross-plane denial proof from another plane role.

Only after image publication/import should the staging `web` and `gateway` pods be scaled above zero and tested through `/healthz`, `/readyz`, and Playwright smokes against a non-production hostname or port-forward.
