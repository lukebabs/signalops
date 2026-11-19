# K8S-2 gateway staging runtime environment template

Status: operator handoff; contains no secret values.

This file defines the root-owned environment file needed to close the K8S-2 gateway runtime smoke. The deployment-agent action reads the file at:

```text
/etc/signalops/openbao-signalops-app-staging-runtime.env
```

The file must contain only non-production/staging endpoints. Do not use Docker Compose production hosts such as `postgres`, `marketops-postgres`, `localhost`, or any `.invalid` placeholder. Database hosts must be Kubernetes service DNS names containing `.svc`.

Required shape:

```bash
OPENBAO_ADMIN_TOKEN=...
SIGNALOPS_K8S_STAGING_RUNTIME_NON_PRODUCTION_APPROVED=true

SIGNALOPS_DATABASE_URL=postgres://<user>:<password>@<signalops-primary>.<namespace>.svc.cluster.local:5432/signalops?sslmode=require
SIGNALOPS_TEMPORAL_DATABASE_URL=postgres://<user>:<password>@<signalops-temporal>.<namespace>.svc.cluster.local:5432/signalops_temporal?sslmode=require
SIGNALOPS_MARKETOPS_DATABASE_URL=postgres://<user>:<password>@<marketops-primary>.<namespace>.svc.cluster.local:5432/marketops?sslmode=require
SIGNALOPS_MARKETOPS_TEMPORAL_DATABASE_URL=postgres://<user>:<password>@<marketops-temporal>.<namespace>.svc.cluster.local:5432/marketops_temporal?sslmode=require
SIGNALOPS_SUBSCRIBER_GATEWAY_DATABASE_URL=postgres://<user>:<password>@<subscriber-primary>.<namespace>.svc.cluster.local:5432/marketops?sslmode=require

SIGNALOPS_NOTIFICATION_ENCRYPTION_KEY=...
STRIPE_RESTRICTED_API_KEY=...
STRIPE_WEBHOOK_SECRET=...
SYNCRATIC_API_BASE_URL=...
SYNCRATIC_CLIENT_SECRET=...
```

Once the file exists with root-readable permissions, the approved bounded action is:

```bash
sudo -n signalops-deploy-agent k8s-staging-gateway-runtime-smoke
```

The action writes only `signalops/data/k8s/app/signalops-gateway-runtime-staging`, applies the staging app overlay, scales the gateway for one readiness smoke, checks `/healthz` and `/readyz` inside the gateway container, and scales staging workloads back to zero.

The action does not authorize production DNS changes, provider polling, production traffic cutover, or production secret migration.
