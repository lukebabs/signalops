# K8S-3 MarketOps OpenBao Runtime Environment Template

This file documents the protected operator-owned environment expected by:

```bash
/etc/signalops/openbao-signalops-marketops-staging-runtime.env
```

The file is intentionally not committed. It is read only by the guarded MarketOps staging runtime writer and must contain non-production Kubernetes service DNS values.

## Required approval flags

```dotenv
SIGNALOPS_K8S_MARKETOPS_RUNTIME_NON_PRODUCTION_APPROVED=true
SIGNALOPS_K8S_MARKETOPS_DRY_RUN_APPROVED=true
```

## OpenBao authority

Use exactly one bounded administrative token source:

```dotenv
OPENBAO_ADMIN_TOKEN=<redacted>
```

Equivalent supported names are `BAO_TOKEN`, `VAULT_TOKEN`, or `OPENBAO_TOKEN`.

## Required database URLs

All hostnames must be Kubernetes service DNS names containing `.svc`. Placeholder `.invalid`, Compose service names, localhost, and production-like hostnames are rejected.

```dotenv
SIGNALOPS_DATABASE_URL=postgres://<user>:<password>@<signalops-postgres-service>.<namespace>.svc.cluster.local:5432/signalops?sslmode=require
SIGNALOPS_TEMPORAL_DATABASE_URL=postgres://<user>:<password>@<signalops-timescaledb-service>.<namespace>.svc.cluster.local:5432/signalops_temporal?sslmode=require
SIGNALOPS_MARKETOPS_DATABASE_URL=postgres://<user>:<password>@<marketops-postgres-service>.<namespace>.svc.cluster.local:5432/marketops?sslmode=require
SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL=postgres://<user>:<password>@<marketops-timescaledb-service>.<namespace>.svc.cluster.local:5432/marketops_temporal?sslmode=require
SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_DATABASE_URL=postgres://<user>:<password>@<marketops-postgres-service>.<namespace>.svc.cluster.local:5432/marketops?sslmode=require
SIGNALOPS_SUBSCRIBER_GLOBAL_EOD_TEMPORAL_DATABASE_URL=postgres://<user>:<password>@<marketops-timescaledb-service>.<namespace>.svc.cluster.local:5432/marketops_temporal?sslmode=require
```

## Optional provider keys

Provider values may be included for future worker parity, but the first K8S-3 pod execution gate must remain dry-run/non-provider.

```dotenv
SIGNALOPS_MASSIVE_BASE_URL=<redacted>
SIGNALOPS_MASSIVE_API_KEY=<redacted>
SIGNALOPS_FMP_BASE_URL=<redacted>
SIGNALOPS_FMP_API_KEY=<redacted>
```

## Safety contract

- Writes only to `signalops/k8s/marketops/marketops-worker-runtime-staging`.
- Verifies the `signalops-marketops` Kubernetes auth role can read that path.
- Verifies the `signalops-app` role cannot read that path.
- Does not authorize production DNS, production traffic cutover, provider polling, or unsuspended CronJobs.
