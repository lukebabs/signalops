# SignalOps OpenBao Secret Scaffold

Status: OpenBao selected; planning scaffold only; not applied to production by this repo.

SignalOps will use OpenBao as the Kubernetes secret backend. The live cluster check on 2026-09-09 confirmed the `openbao` namespace and `openbao-agent-injector-svc` exist. External Secrets Operator CRDs were not present, so the current scaffold uses the OpenBao Agent Injector model rather than `SecretStore` / `ExternalSecret` resources.

## Assumptions

- OpenBao is installed in namespace `openbao`.
- OpenBao Agent Injector is installed and reachable through `openbao-agent-injector-svc`.
- OpenBao exposes a Vault-compatible endpoint through the in-cluster `openbao` services.
- OpenBao is operated in HA mode before SignalOps production workloads depend on it.
- OpenBao has a KV v2 mount for SignalOps secrets.
- Kubernetes auth is enabled in OpenBao.
- OpenBao roles are created per SignalOps plane: `signalops-app`, `signalops-connect`, `signalops-marketops`, `signalops-cyberops`, `signalops-identity`, `signalops-data`, and `signalops-observability`.

## HA operating contract

OpenBao is platform-critical infrastructure for the Kubernetes migration. SignalOps workloads should not be cut over to OpenBao-backed secret injection until the HA cluster has recorded evidence for:

- at least three OpenBao server pods or an equivalent fault-tolerant topology;
- one active node and healthy standby nodes reachable through the in-cluster service;
- storage backend durability and snapshot/restore procedure;
- unseal or auto-unseal recovery procedure with separated operator authority;
- OpenBao Agent Injector availability after an OpenBao pod restart;
- Kubernetes auth enabled and bound to only the expected SignalOps service accounts;
- per-plane policies that prevent cross-plane secret reads;
- audit logging enabled for secret reads, policy changes, auth changes, and token activity;
- backup/restore or snapshot rehearsal that proves SignalOps secret paths can be recovered without exposing secret values.

If OpenBao is unavailable, existing SignalOps pods should continue running with already-injected material until restart/rotation requires a new injection. New pods that cannot obtain required secrets must fail closed rather than start with empty/default credentials.

## Read-only surface verifier

After building OpenBao HA, run the repository verifier from the SignalOps workspace:

```bash
scripts/verify_openbao_ha_readiness.sh
```

For the approved non-production staging exception only, the verifier may be run as:

```bash
scripts/verify_openbao_ha_readiness.sh --allow-single-node-staging
```

Single-node staging mode requires at least one ready OpenBao server pod and emits `production_cutover_allowed=false`. It is suitable only for staging injector mechanics with non-production test secrets. It does not satisfy the production HA gate.

The verifier checks the Kubernetes namespace, expected OpenBao services, Agent Injector deployment, injector webhook, and ready OpenBao server pod count. It does not authenticate to OpenBao and therefore does not prove seal state, storage snapshots, audit devices, policies, or Kubernetes auth roles.

## Remote secret path convention

Use this convention for OpenBao KV paths:

```text
signalops/data/k8s/<plane>/<secret-name>
```

Examples:

```text
signalops/data/k8s/app/signalops-gateway-runtime
signalops/data/k8s/marketops/signalops-marketops-provider-runtime
signalops/data/k8s/identity/signalops-keycloak-runtime
```

## Injection model

Workload manifests should use OpenBao Agent Injector annotations. The base scaffold only creates per-plane secret-reader service accounts and a policy ConfigMap that documents the annotation pattern. Actual workload-specific annotations belong with the Deployment/CronJob manifest being converted.

## Guardrails

- Do not store raw secret values in Git.
- Do not create one global SignalOps secret. Keep app, ingestion, MarketOps, CyberOps, identity, data, and observability secrets separate.
- OpenBao policies must grant each plane read access only to its own path prefix.
- Database credentials should remain per workload role even when the OpenBao read policy is plane-scoped.
- Rotation should occur in OpenBao. Pods should be restarted or reloaded according to the injector/agent renewal policy.
- OpenBao HA health, seal state, injector health, and audit-log delivery must be observable before production cutover.

## Later option

External Secrets Operator can still be introduced later if we want Kubernetes Secret projection instead of sidecar/file injection. That is a separate platform choice and should not block the initial OpenBao-backed workload conversion.
