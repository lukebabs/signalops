# SignalOps Broker Boundary

The broker package defines the durable internal communication boundary used by
Go services, Python algorithm workers, and future stream-oriented subsystems.

The package intentionally contains no concrete Kafka client yet. It owns:

- durable topic naming;
- publish and consume interfaces;
- message metadata fields required for traceability;
- acknowledgement and offset shapes.

Concrete Redpanda/Kafka clients must implement these interfaces without leaking
client-specific types into application code.

## Topic Naming

Durable topics use:

```text
signalops.<environment>.<topic>.v1
```

The default local environment is `local`.

## Durable Topics

- `signalops.<environment>.raw.v1`
- `signalops.<environment>.normalized.v1`
- `signalops.<environment>.signal.v1`
- `signalops.<environment>.artifact.v1`
- `signalops.<environment>.graph_mutation.v1`
- `signalops.<environment>.insight_candidate.v1`
- `signalops.<environment>.retry.algorithm.v1`
- `signalops.<environment>.dlq.algorithm.v1`
