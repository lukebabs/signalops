# Mesh-1 Istio readiness evidence — 2026-09-13

Status: closed for control-plane readiness; no SignalOps production traffic cutover.

Recorded: 2026-09-13 UTC.

## Scope

Mesh-1 records that the Kubernetes service-mesh control plane is available for the next SignalOps staging-route parity gate. This gate does not move `signalops.syncratic.io`, does not change Keycloak redirect behavior, does not enroll SignalOps namespaces into sidecar or ambient dataplane mode, and does not enable provider polling.

## Live cluster evidence

Read-only cluster inspection showed:

- `istio-base` Helm release deployed in `istio-system`;
- `istiod` Helm release deployed in `istio-system`;
- `istiod` pod running;
- Istio CRDs present;
- Gateway API CRDs present;
- `GatewayClass/istio` accepted;
- `GatewayClass/istio-remote` accepted for unmanaged/remote use;
- `istio-system/public-ingress` programmed with address `192.168.2.233`;
- `istio-system/public-ingress-istio` service exposed as `LoadBalancer|192.168.2.233|31574|32284`;
- SignalOps namespaces exist but have no `istio-injection` label and no `istio.io/dataplane-mode` label.

The observed Gateway routes currently belong to other Syncratic domains (`portal.syncratic.co`, `strategiclabs.io`, and `onemanarmy.blog`). No SignalOps route was moved during this gate.

## Repeatable verifier

Added:

```bash
scripts/verify_k8s_mesh1_istio_readiness.sh
```

Passing output:

```text
signalops_k8s_mesh1_istio_readiness_verified
istio_base=deployed
istiod=deployed
istiod_pod=running
gateway_api_crds=present
istio_crds=present
gateway_class_istio=accepted
public_gateway=istio-system/public-ingress
public_gateway_programmed=true
public_gateway_address=192.168.2.233
public_gateway_service=LoadBalancer|192.168.2.233|31574|32284
signalops_namespace_auto_injection=false
signalops_dataplane_enrollment=false
production_cutover_allowed=false
recommended_next_gate=mesh2_signalops_staging_route_parity
```

## Interpretation

Istio is now the selected mesh path for the SignalOps Kubernetes production-readiness program. Cilium remains the cluster CNI and NetworkPolicy foundation. Docker Compose/systemd remains the production authority for SignalOps until staging route parity, authenticated Keycloak parity, scheduler parity, Signal-Connect shadow parity, rollback, and capacity gates pass.

The next safe gate is Mesh-2: create a SignalOps staging-only route through the Istio Gateway and validate `/healthz`, `/readyz`, SPA serving, and eventually authenticated Keycloak callback behavior without production DNS movement.
