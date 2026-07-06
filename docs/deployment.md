# SignalOps Deployment

SignalOps uses Redpanda as the default broker runtime while keeping the
implementation Kafka API compatible.

## Local Docker Compose

Start the local stack:

```bash
make compose-up
```

Validate the compose file:

```bash
make compose-validate
```

Show services:

```bash
make compose-ps
```

Stop the stack:

```bash
make compose-down
```

## Services

- `redpanda`: default Kafka-compatible broker.
- `redpanda-console`: local broker UI on `http://localhost:18080`.
- `topic-bootstrap`: one-shot topic creation job.
- `gateway`: SignalOps gateway on `http://localhost:18000`.

## Local Ports

- Gateway: `18000` host port mapped to container port `8080`
- Redpanda Kafka external listener: `19092`
- Redpanda Schema Registry: `18081`
- Redpanda HTTP Proxy: `18082`
- Redpanda Admin/metrics: `19644`
- Redpanda Console: `18080`

## Default Topics

- `signalops.local.raw.v1`
- `signalops.local.normalized.v1`
- `signalops.local.signal.v1`
- `signalops.local.artifact.v1`
- `signalops.local.graph_mutation.v1`
- `signalops.local.insight_candidate.v1`
- `signalops.local.retry.algorithm.v1`
- `signalops.local.dlq.algorithm.v1`

## Broker Decision

- Redpanda is the default local and production broker target.
- SignalOps code must depend on Kafka-compatible broker abstractions.
- Apache Kafka remains a deployment alternative for environments that already
  standardize on Kafka.

## Environment

Copy `.env.example` when local overrides are needed.

```bash
cp .env.example .env
```

The gateway currently reads:

- `SIGNALOPS_HTTP_ADDR`

Broker environment variables are documented now and will be wired into the
broker abstraction implementation in a later gate:

- `SIGNALOPS_BROKER_PROVIDER`
- `SIGNALOPS_BROKER_BROKERS`
- `SIGNALOPS_ENV`
