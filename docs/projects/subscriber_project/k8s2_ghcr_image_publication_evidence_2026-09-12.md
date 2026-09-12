# K8S-2 GHCR image publication evidence

Recorded: 2026-09-12.

## Outcome

The staging `web` and `gateway` images were published to GitHub Container Registry under the Syncratic organization namespace. The source commit used for immutable tagging was `53e773731939`.

Published images:

```text
ghcr.io/syncratic-inc/signalops-web:53e773731939
ghcr.io/syncratic-inc/signalops-web:staging
ghcr.io/syncratic-inc/signalops-gateway:53e773731939
ghcr.io/syncratic-inc/signalops-gateway:staging
```

Published digests:

```text
signalops-web:     sha256:a25a9a504c12526315de4fb97f0381c39f45c8ce0b21aa3ebe9dafcc92957f6a
signalops-gateway: sha256:d387af5dd8d4bece4b1bc33c9cfd4fef034091c7b01dfdb5146705ef422b1034
```

The staging Kubernetes manifests now reference the GHCR registry tags instead of local-only Docker image names.

## Pull-smoke evidence

A bounded non-production pull smoke scaled the staging web/gateway Deployments to one replica and then back to zero. Kubernetes reached GHCR but could not pull the images anonymously:

```text
Failed to pull image "ghcr.io/syncratic-inc/signalops-web:staging": failed to authorize: failed to fetch anonymous token: 401 Unauthorized
Failed to pull image "ghcr.io/syncratic-inc/signalops-gateway:staging": failed to authorize: failed to fetch anonymous token: 401 Unauthorized
```

This confirms the previous local-image blocker is replaced by a production-grade registry access decision.

## Boundary

No production traffic moved. No Ingress, DNS, Kubernetes Secret, provider polling, or production cutover was performed. The staging Deployments were scaled back to zero after the pull smoke. Docker Compose remains the production authority.

## Next decision

Choose one image-pull strategy before K8S-2 readiness can close:

1. Make the GHCR packages public for anonymous cluster pulls.
2. Keep GHCR packages private and provision a scoped Kubernetes `imagePullSecret`/registry credential path, preferably managed through OpenBao or the platform secret pipeline.

For production SaaS posture, the preferred option is private GHCR packages with a narrowly scoped pull credential.
