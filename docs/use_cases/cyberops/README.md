# CyberOps Use Cases

CyberOps is a planned SignalOps app profile for evidence-driven security
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

The concrete CyberOps `domain` and `use_case` values will be defined with the
first implemented security workflow. That workflow should receive its own
folder at `docs/use_cases/cyberops/{use_case}/`.

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
