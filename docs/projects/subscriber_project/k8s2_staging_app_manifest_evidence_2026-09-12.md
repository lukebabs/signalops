# K8S-2 staging app manifest evidence

Status: manifest gate complete; staging workload apply mechanics exercised; GHCR images published; private image pulls verified; web runtime readiness verified; gateway runtime readiness blocked on placeholder OpenBao database values; deployments scaled back to zero.

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
configmaps=1
openbao_secret_path=signalops/data/k8s/app/signalops-gateway-runtime-staging
production_cutover_allowed=false
applied=false
```

The server-side dry run would create only:

```text
configmap/signalops-web-nginx-staging
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
configmaps=1
openbao_secret_path=signalops/data/k8s/app/signalops-gateway-runtime-staging
production_cutover_allowed=false
applied=false
```

## Web runtime readiness evidence

After private GHCR pull access was verified, the staging web Deployment still failed availability because the Docker Compose nginx configuration was being reused in Kubernetes and kubelet HTTP probes were timing out through the host/network-policy path. The staging overlay now mounts a Kubernetes-native nginx ConfigMap and uses in-container localhost exec probes:

```text
configmap/signalops-web-nginx-staging configured
deployment "signalops-web" successfully rolled out
```

Direct in-pod smoke evidence confirmed the SPA is served from the private GHCR image:

```text
<title>SignalOps</title>
<script type="module" crossorigin src="/assets/index-BZAC9GHh.js"></script>
```

The web ConfigMap keeps `/v1/`, `/healthz`, and `/readyz` pointed at the Kubernetes Service DNS name `signalops-gateway.signalops-app.svc.cluster.local:8080`, with lazy DNS resolution through the cluster resolver `10.43.0.10`. This closes the web runtime/probe portion of K8S-2.

The staging Deployments were scaled back to zero after the bounded smoke.

## Gateway runtime blocker

The gateway image pull and OpenBao injection path are verified, but the injected runtime file still contains placeholder-only database endpoints such as `postgres.staging.invalid`. When scaled above zero, the gateway exits while trying to connect to that unresolved database host. This is the remaining K8S-2 blocker. Closing it requires an approved OpenBao runtime update with reachable staging database dependencies or explicit non-production database endpoints. No production workload authority, DNS, provider polling, or production secret migration is allowed by the current evidence.

## Known warnings

`kubectl` continues to emit the existing k3s Cilium config permission warning for `/etc/rancher/k3s/config.yaml.d/90-syncratic-cilium.yaml`. It did not block render or server-side dry-run validation.

## Next step

Before running K8S-2 workload readiness, replace placeholder-only OpenBao runtime values with approved staging dependencies. The replacement classic `GHCR_KEY` verified package-read access, and Kubernetes successfully pulled both private GHCR images through `signalops-ghcr-pull`. The OpenBao app role/path prerequisite is complete, and images now point to `ghcr.io/syncratic-inc/signalops-*`:

- OpenBao Kubernetes auth role: `signalops-app`;
- bound service accounts: `signalops-gateway`, `signalops-web`, and `signalops-app-secret-reader` in `signalops-app`;
- readable path: `signalops/data/k8s/app/signalops-gateway-runtime-staging`;
- cross-plane denial proof from another plane role.

Only after runtime values are approved should the staging `gateway` pod be scaled above zero and tested through `/healthz`, `/readyz`, and Playwright smokes against a non-production hostname or port-forward. The `web` pod static-runtime path has been verified, but full app parity still depends on the gateway runtime path. See [K8S-2 GHCR image publication evidence](k8s2_ghcr_image_publication_evidence_2026-09-12.md).

## Gateway runtime smoke action prepared

A constrained gateway runtime smoke path has been added for the approved operation:

```text
sudo -n signalops-deploy-agent k8s-staging-gateway-runtime-smoke
```

The action delegates to `scripts/run_k8s_staging_gateway_runtime_smoke.sh`, which:

- reads `/etc/signalops/openbao-signalops-app-staging-runtime.env`;
- writes only the existing OpenBao app staging path `signalops/data/k8s/app/signalops-gateway-runtime-staging`;
- rejects missing approval marker `SIGNALOPS_K8S_STAGING_RUNTIME_NON_PRODUCTION_APPROVED=true`;
- rejects placeholder, localhost, Docker Compose, and production-like database hosts;
- requires database hosts to be Kubernetes service DNS names containing `.svc`;
- applies the staging app overlay;
- scales `signalops-gateway` only for a bounded readiness smoke;
- verifies gateway `/healthz` and `/readyz` from inside the gateway container;
- scales staging workloads back to zero on exit.

The current `.env` contains `OPENBAO_ADMIN_TOKEN`, but it does not contain the required non-production SignalOps temporal and MarketOps primary/temporal database URLs or the explicit non-production approval marker. Therefore the runtime writer correctly fails closed until the protected staging runtime env file is supplied. See [K8S-2 gateway staging runtime environment template](k8s2_gateway_runtime_env_template.md).
