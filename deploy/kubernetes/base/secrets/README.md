# SignalOps Kubernetes Secret-Class Strategy

Status: OpenBao selected; planning scaffold only; no runtime secrets are stored here.

The Compose `.env` model is intentionally not carried into Kubernetes. Kubernetes must use per-plane and per-workload-class secret boundaries so app traffic, ingestion, analytical jobs, identity, data, and observability do not share broad credentials.

## Current artifact

`secret-class-policy.yaml` is a policy ConfigMap only. It is not a secret and not a deployment credential. It documents the required secret classes for the selected OpenBao Agent Injector model.

## Required backend decision

OpenBao is selected as the production secret backend. The live cluster currently has OpenBao Agent Injector but not External Secrets Operator CRDs, so the initial scaffold uses injector annotations and per-plane secret-reader service accounts. The OpenBao scaffold is under `secrets/openbao/`.
