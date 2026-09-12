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

Image-pull strategy selected and verified: keep GHCR private and use a scoped pull credential. Next readiness decision:

1. Replace placeholder-only OpenBao runtime values with approved staging endpoints/credentials, or create dedicated non-production dependencies.
2. Scale staging web/gateway above zero and validate `/healthz`, `/readyz`, and browser smokes through a staging hostname or port-forward.

For production SaaS posture, GHCR packages remain private with a narrowly scoped pull credential.


## Private pull credential attempt

A `GHCR_KEY` value was supplied through the local `.env` file and used to create `signalops-ghcr-pull` in the `signalops-app` namespace. The staging manifests were updated to reference that pull secret.

The token owner lookup returned `lukebabs`, so the username was not the blocker. A direct host-side manifest check using the supplied token returned `denied` for both images, and Kubernetes returned authenticated `403 Forbidden` while pulling both private GHCR images.

Resolved: the replacement classic `GHCR_KEY` has package-read access to the private Syncratic GHCR packages and Kubernetes successfully pulled both images through `signalops-ghcr-pull`.

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


## Private pull credential verified

A new classic token was supplied through `.env` as `GHCR_KEY`. The validator passed:

```text
signalops_ghcr_pull_token_verified
registry=ghcr.io
user=lukebabs
packages=signalops-web,signalops-gateway
permission=read
```

The Kubernetes `signalops-ghcr-pull` secret in `signalops-app` was refreshed from that token and the staging pull smoke was rerun. Kubernetes successfully pulled both private GHCR images:

```text
Successfully pulled image "ghcr.io/syncratic-inc/signalops-web:staging"
Successfully pulled image "ghcr.io/syncratic-inc/signalops-gateway:staging"
```

This closes the private-registry pull-credential gate. The remaining K8S-2 blocker is runtime readiness: the web pod started but failed probes, and the gateway pod advanced through OpenBao injection but entered container restart/backoff. This is expected while the OpenBao runtime values remain placeholder-only and no production/staging database cutover has been authorized.

The staging Deployments were scaled back to zero after the smoke:

```text
deployment.apps/signalops-gateway   0/0
deployment.apps/signalops-web       0/0
```
