# SignalOps Kubernetes Secret-Class Strategy

Status: planning scaffold; no runtime secrets are stored here.

The Compose `.env` model is intentionally not carried into Kubernetes. Kubernetes must use per-plane and per-workload-class secret boundaries so app traffic, ingestion, analytical jobs, identity, data, and observability do not share broad credentials.

## Current artifact

`secret-class-policy.yaml` is a policy ConfigMap only. It is not a secret, not an ExternalSecret, and not a deployment credential. It documents the required secret classes until a production secret backend is selected.

## Required backend decision

Before workload Deployments/CronJobs are introduced, select one:

1. External Secrets Operator backed by AWS Secrets Manager or another managed secret store.
2. Sealed Secrets for GitOps-friendly encrypted manifests.
3. SOPS with age/KMS for encrypted Git-managed secret values.

For SaaS scale, External Secrets Operator with a managed secret store is the preferred production path because it supports rotation, auditability, and per-workload IAM boundaries.
