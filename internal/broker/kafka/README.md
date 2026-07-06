# SignalOps Kafka-Compatible Broker Client

This package implements the `pkg/broker` interfaces with franz-go and targets
Redpanda as the default Kafka-compatible runtime.

Application code should depend on `pkg/broker` interfaces. This package stays
under `internal/` so franz-go types do not become part of the public SignalOps
contract.

## Behavior

- `Publish` uses synchronous produce acknowledgement and returns topic,
  partition, and offset.
- `NewConsumer` creates a manual-commit consumer group.
- `Consume` buffers records from fetch batches so records are not dropped when
  a caller processes one message at a time.
- `Commit` commits the consumed record offset through franz-go.
- `correlation_id`, `causation_id`, and `trace_id` are carried as Kafka record
  headers.

## Integration Test

Start the local stack first:

```bash
make compose-up
```

Then run:

```bash
make docker-test-broker-integration
```

The test publishes to `signalops.local.raw.v1`, consumes the same keyed record,
verifies metadata headers, and commits the consumed offset.
