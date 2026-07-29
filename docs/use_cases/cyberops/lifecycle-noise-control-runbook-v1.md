# CyberOps lifecycle noise-control v1 runbook

## Initial rollout

Migration `000062_cyberops_lifecycle_noise_control` seeds the three `cyberops-lifecycle-v1` policies for `tenant-local` in `shadow` mode. Shadow mode writes durable Signals, episodes, and immutable policy decisions, but creates no policy-driven Insight or Alert work item.

The seed covers external firewall denies (record only), public-service exposure (approved service: record only; unapproved service: projected low Insight), and a public-source port scan threshold of ten distinct denied ports in five minutes (projected high Alert).

## Operations

- Use `GET /v1/tenants/{tenant_id}/cyberops/lifecycle/decisions` and `/episodes` to review shadow outcomes.
- Use CyberOps Settings to add or remove an approved public service. These admin-only writes record actor and before/after values in `cyberops_lifecycle_policy_audit`.
- Compare projected dispositions with the prior generic lifecycle volume before changing a policy to `enforced`. Do not enable enforcement without explicit approval.
- To roll back, set policy mode to `disabled` or `shadow`; do not delete historical signals, episodes, decisions, Insights, or Alerts.

## Data guarantees

Policy evaluation is in the same database transaction as Signal persistence. Episode evidence keeps at most 100 signal IDs, and re-delivery is deduplicated by tenant, signal, and policy.
