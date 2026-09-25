# K8S-1 OpenBao staging foundation evidence

Status: completed under the approved single-node OpenBao staging exception; no production workload cutover approved.

Recorded: 2026-09-12.

## Scope

K8S-1 applied only the non-secret Kubernetes foundation for SignalOps staging mechanics:

- namespaces;
- service accounts;
- default-deny and first-pass allow NetworkPolicies;
- non-secret OpenBao policy/secret-class ConfigMaps.

No SignalOps Deployments, StatefulSets, Jobs, CronJobs, Ingresses, production Secrets, provider polling, production DNS changes, or production workload cutover were applied.

## Preflight evidence

OpenBao staging-exception verifier passed:

```text
openbao_readiness_surface_verified
mode=single_node_staging_exception
production_cutover_allowed=false
namespace=openbao
server_ready_pods=1
required_production_server_pods=3
injector_available_replicas=1
services=openbao,openbao-active,openbao-standby,openbao-agent-injector-svc
webhook=vault.hashicorp.com
annotation_prefix=vault.hashicorp.com
```

The default production HA gate remains blocked until at least three ready non-injector OpenBao server pods, or an explicitly documented equivalent fault-tolerant topology, are proven.

## Apply evidence

The initial server-side dry run failed because Kubernetes server dry-run validates namespaced resources before dry-run-created namespaces exist. The seven namespaces were therefore applied first, then the full server-side dry run passed, then the full base scaffold was applied.

Applied resources:

```text
namespace/signalops-app
namespace/signalops-connect
namespace/signalops-cyberops
namespace/signalops-data
namespace/signalops-identity
namespace/signalops-marketops
namespace/signalops-observability
```

The full base apply then created the expected per-plane service accounts, NetworkPolicies, and non-secret policy ConfigMaps.

## Verification evidence

`scripts/verify_k8s_base_scaffold.sh` passed:

```text
signalops_k8s_base_scaffold_verified
namespaces=7
service_accounts=29
network_policies=15
policy_configmaps=2
workload_pods=0
production_cutover_allowed=false
```

A direct namespace inventory confirmed all seven SignalOps namespaces are active and labeled by plane.

A direct pod inventory confirmed no pods exist in the newly created `signalops-*` namespaces as part of this gate.

## Known warnings

`kubectl` continues to emit permission warnings for `/etc/rancher/k3s/config.yaml.d/90-syncratic-cilium.yaml`. These warnings did not block OpenBao verification, Kustomize render, server-side dry run, apply, or post-apply verification. They should be cleaned up before relying on unattended CI/deployment-agent K8s checks.

The cluster also contains unrelated pre-existing non-SignalOps pod failures in other namespaces. They were not created by this K8S-1 gate.

## Next gate

Proceed to K8S-2 only under the same non-production staging boundary:

- create staging-only `web` and `gateway` workload manifests;
- use OpenBao-injected non-production test secrets only;
- expose through non-production hostnames only;
- run `/readyz` and Playwright parity smokes against staging;
- keep Docker Compose as production authority.
