# SignalOps Integration — NarrativeOps Workspace

## Public workspace

NarrativeOps is mounted as an independently deployed SignalOps workspace at:

```text
https://signalops.syncratic.io/narrativeops
```

The SignalOps shell owns discovery and navigation. NarrativeOps owns narrative formation, evidence admission, replay, market validation, and operations APIs.

## Request paths

| Public path | Internal target |
|---|---|
| `/narrativeops/` | NarrativeOps frontend SPA |
| `/narrativeops/healthz` | NarrativeOps `/healthz` |
| `/narrativeops/readyz` | NarrativeOps `/readyz` |
| `/narrativeops/api/v1/...` | NarrativeOps `/v1/...` |

The frontend uses `/narrativeops/api` as its production API base path and uses `/v1` directly through the Vite proxy in local development.

## Authentication

Production uses the shared SignalOps OIDC session and `signalops-web` client. NarrativeOps validates the bearer token against the Syncratic issuer, `signalops-api` audience, JWKS, tenant claim, and administrator role. The frontend redirects unauthenticated users to SignalOps login and performs one silent-renew retry after `401`.

Local development remains mock/header based. `X-Tenant-ID` is accepted only in development/test mode; it is not a production identity mechanism.

## Deployment

The Kubernetes package in `deploy/k8s/` provides the API, frontend, configuration, services, namespace, and HTTPRoute. Images are intended to be published as:

- `ghcr.io/syncratic-inc/narrativeops-api:<tag>`
- `ghcr.io/syncratic-inc/narrativeops-frontend:<tag>`

The current backend remains an in-memory/local qualification implementation. Durable persistence, production backup/restore, and external artifact publication remain separate readiness gates.

## StreamRecorder boundary

StreamRecorder remains upstream. Its immutable `TranscriptionArtifact` output is admitted by NarrativeOps as an `EvidenceArtifact` through the producer/artifact boundary. NarrativeOps must not import StreamRecorder storage or depend on its database schema.
