# SignalOps OpenBao Secret Scaffold

Status: OpenBao selected; planning scaffold only; not applied to production by this repo.

SignalOps will use OpenBao as the Kubernetes secret backend. The live cluster check on 2026-09-09 confirmed the `openbao` namespace and `openbao-agent-injector-svc` exist. External Secrets Operator CRDs were not present, so the current scaffold uses the OpenBao Agent Injector model rather than `SecretStore` / `ExternalSecret` resources.

## Assumptions

- OpenBao is installed in namespace `openbao`.
- OpenBao Agent Injector is installed and reachable through `openbao-agent-injector-svc`.
- OpenBao exposes a Vault-compatible endpoint through the in-cluster `openbao` services.
- OpenBao has a KV v2 mount for SignalOps secrets.
- Kubernetes auth is enabled in OpenBao.
- OpenBao roles are created per SignalOps plane: `signalops-app`, `signalops-connect`, `signalops-marketops`, `signalops-cyberops`, `signalops-identity`, `signalops-data`, and `signalops-observability`.

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

## Later option

External Secrets Operator can still be introduced later if we want Kubernetes Secret projection instead of sidecar/file injection. That is a separate platform choice and should not block the initial OpenBao-backed workload conversion.
