# SignalOps Internal Communication Protocols

SignalOps uses explicit protocol boundaries between the Go core platform and
the Python algorithm system.

## Durable Path

Use Kafka or Redpanda for durable asynchronous work.

Use this path for:

- Go-to-Python algorithm jobs.
- Python-to-Go algorithm results.
- replayable processing.
- retries and DLQ routing.
- batch, windowed, or long-running processing.
- work that must survive process restarts.

Payload format v1:

- JSON.

Schema contract v1:

- JSON Schema files under `contracts/`.

Go implementation boundary:

- shared topic names and broker interfaces live under `pkg/broker`.
- application code depends on `pkg/broker` interfaces, not concrete
  Redpanda/Kafka client types.
- concrete broker clients must preserve `correlation_id`, `causation_id`, and
  `trace_id` metadata when publishing derived work.

## Fast Sync Path

Use gRPC with Protobuf for bounded synchronous internal calls.

Use this path only for:

- low-latency scoring.
- health, status, and control APIs.
- small internal lookups.
- calls that are safe to retry from the caller.

gRPC responses from Python are not canonical truth by themselves. The Go core
platform must persist or republish the resulting `Signal`, `EventArtifact`,
`GraphMutationProposal`, or `InsightCandidate` before it is treated as durable
SignalOps output.

## Decision Rule

- Durable, replayable, retryable, auditable, or DLQ-routable work uses
  Kafka/Redpanda.
- Short-lived, bounded, request/response work that is safe to retry may use
  gRPC.
- Public APIs, tenant/admin APIs, and compatibility boundaries use REST.
- Go services must not import, embed, or directly execute Python libraries.
