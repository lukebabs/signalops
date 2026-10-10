# K8S Stripe webhook route parity — 2026-09-13

Status: closed. Synthetic signed-webhook validation passed through the K3s Istio staging route. The smoke did not call Stripe, did not move production DNS, did not move production traffic, and returned staging app workloads to zero.

## Purpose

Before public SignalOps traffic can move to K3s, the subscription commerce path must prove that Stripe webhook delivery can reach the Kubernetes Gateway and be processed by the same signature-verified Gateway code used in Docker Compose production.

## Security assumptions

- Stripe webhook signatures must be verified using `STRIPE_WEBHOOK_SECRET` before processing.
- Secret values must come from runtime secret management / OpenBao injection, not source code.
- Invalid signatures must fail closed with no ledger mutation.
- Valid synthetic events may write only to the staging runtime database during this gate.
- No Stripe provider API call, payment action, refund, or customer-visible subscription change is performed by this smoke.
- The staging route currently uses a self-signed certificate, so the smoke uses an explicit staging-only TLS exception. Production traffic must use trusted public TLS before DNS cutover.

## Smoke behavior

`scripts/run_k8s_stripe_webhook_route_parity_smoke.sh` performs the following bounded sequence:

1. verifies Istio Mesh-1 readiness, staging app manifests, Mesh-2 route manifests, and Mesh-4 ingress/DNS rollback plan;
2. applies the staging app and mesh-route overlays;
3. provisions OpenBao staging runtime from the local env without printing secrets;
4. bootstraps the staging enrollment/subscription schema, including the `subscriber_checkout_sessions` ledger required by migration `000161`;
5. seeds one synthetic checkout reference in the staging MarketOps database so the webhook follows the real webhook-authoritative activation path;
6. scales staging `signalops-gateway` to one while keeping `signalops-web` at zero because the webhook path only needs the API gateway;
7. posts one invalid-signature synthetic webhook through `https://signalops-staging.syncratic.co/v1/billing/stripe/webhook` using `curl --resolve` against the Istio Gateway address and requires HTTP 400;
8. posts one valid HMAC-signed synthetic `checkout.session.completed` event through the same K3s route and requires HTTP 200;
9. scales the staging gateway workload back to zero.

## Non-authorizations

This gate does not authorize production DNS changes, production traffic movement, live Stripe event replay, provider polling, scheduler authority transfer, or database deletion.

## Passing evidence

```text
signalops_k8s_stripe_webhook_route_parity_verified
namespace=signalops-app
staging_hostname=signalops-staging.syncratic.co
gateway=istio-system/public-ingress
gateway_ip=192.168.2.233
invalid_signature_rejected=true
valid_signature_processed=true
event_type=checkout.session.completed
run_id=k8s-stripe-webhook-20260913T172841Z
stripe_provider_called=false
production_dns_changed=false
production_traffic_moved=false
production_cutover_allowed=false
staging_self_signed_tls_allowed=true
scaled_back_to_zero=true
```

## Implementation notes

The first attempts exposed two useful staging hardening items:

- single-node K3s capacity was too tight for staging web plus gateway plus OpenBao-injected sidecars, so this route-specific smoke keeps web at zero and lowers staging-only gateway/OpenBao resource requests;
- a subscription-update event without an existing checkout ledger can validly be unmatched or fail depending on schema state, so the accepted parity path now mirrors the real activation flow: seed `checkout_ref`, then send signed `checkout.session.completed`.

Both changes are staging-only and preserve `production_cutover_allowed=false`.
