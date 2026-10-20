# SignalOps Kubernetes image build and release

Production runtime is Kubernetes. Docker Compose is not a production runtime or a normal release authority.

## Build authority

`.github/workflows/build-k8s-images.yml` is the normal image-build path. On a `main` push, or a manual workflow dispatch,
GitHub Actions BuildKit builds and publishes immutable images to the private GHCR packages:

```text
ghcr.io/syncratic-inc/signalops-gateway:sha-<commit>
ghcr.io/syncratic-inc/signalops-web:sha-<commit>
```

The workflow uses `GITHUB_TOKEN` with package-write permission. It does not require, read, or publish any `.env` secret and it
does not deploy, alter DNS, enable schedulers, or move traffic.

## Deployment authority

After the image workflow succeeds, the approved deployment process updates the production Kubernetes manifests to the immutable
`sha-<commit>` references and applies the app Kustomization. Rollout readiness is verified before the change is considered
complete. The deployment manifest must never point at a local Docker tag.

The live production workloads are:

```text
signalops-app/signalops-gateway-production
signalops-app/signalops-web-production
```

## Legacy host publisher

`scripts/publish_k8s_signalops_app_images.sh` remains available for emergency recovery or isolated development when GitHub
Actions is unavailable. It is not the production build authority. It requires an explicit operator invocation and should not be
used as part of routine K8s releases.

## Rollback

Rollback is an image-reference change to the previously verified immutable tag, followed by a Kubernetes rollout-status check.
Mutable `production` or `latest` tags are not used for production deployment references.
