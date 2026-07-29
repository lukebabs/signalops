# CyberOps Use Cases

CyberOps is an implemented SignalOps app profile for evidence-driven security
operations. It consumes the shared SignalOps primitive contracts; it does not
own or fork the platform's ingestion, evidence, lineage, quality, replay, or
governed-materialization capabilities.

## Scope Identity

CyberOps uses the following scope rules:

- `app_id`: `cyberops`
- local-development `tenant_id`: `tenant-local`
- production `tenant_id`: the ID of the organization receiving the CyberOps
  service

`tenant_id` represents an authorization and data-isolation boundary. It is not
the name of a SignalOps application or use case. CyberOps therefore shares an
organization's tenant with its other SignalOps applications unless a separate
organization or isolation boundary requires a different tenant.

The implemented workflow is `domain=security`, `use_case=cyberops`. Every parsed public-source firewall allow or deny is retained as durable evidence. The policy lifecycle is seeded for `tenant-local` in shadow mode: it records immutable decisions and grouped episodes without creating new policy-driven Insight or Alert work items.

The policy contract retains external denies as evidence, treats an approved destination IP/protocol/port service as evidence only, projects an unapproved public service as a low-severity Insight, and projects a high-severity port-scan Alert after ten distinct denied destination ports from one public source to one destination in five minutes. An approved service is explicit tenant-owned configuration; absence means unapproved.

The dashboard presents allowed-traffic trends and live ingestion quality. Its live stream reads the primary normalized ledger, which is the current CyberOps source of truth; temporal ledger replication remains historical/backfill support rather than the live display dependency.

## Platform Expectations

CyberOps integrations must preserve the platform rules that apply to every
SignalOps app profile:

- retain provider payloads as Raw Events before deriving durable results;
- normalize provider-specific telemetry before downstream processing;
- retain immutable, versioned lineage for derived data and claims;
- fail closed when input quality is insufficient; and
- materialize knowledge changes only through the governed review path.

See the [foundational platform primitives specification](../../design/syncratic_signalops_foundational_platform_primitives_spec_v1.md)
for the complete contracts and rollout phases.
