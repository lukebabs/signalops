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


## Private pull credential attempt

A `GHCR_KEY` value was supplied through the local `.env` file and used to create `signalops-ghcr-pull` in the `signalops-app` namespace. The staging manifests were updated to reference that pull secret.

The token owner lookup returned `lukebabs`, so the username was not the blocker. A direct host-side manifest check using the supplied token returned `denied` for both images, and Kubernetes returned authenticated `403 Forbidden` while pulling both private GHCR images.

Current blocker: the supplied `GHCR_KEY` does not have package-read access to the private Syncratic GHCR packages, or it has not been authorized for the `syncratic-inc` organization/packages. The token must be replaced or re-authorized before staging pods can pull private images.

Validation helper added:

```bash
scripts/verify_ghcr_pull_token.sh .env
```

Expected success marker after the token is corrected:

```text
signalops_ghcr_pull_token_verified
registry=ghcr.io
user=<token-owner>
packages=signalops-web,signalops-gateway
permission=read
```
