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

Run the broker integration test against the local Redpanda listener:

```bash
make docker-test-broker-integration
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

Runtime config currently reads:

- `SIGNALOPS_HTTP_ADDR`
- `SIGNALOPS_BROKER_PROVIDER`
- `SIGNALOPS_BROKER_BROKERS`
- `SIGNALOPS_ENV`

Broker configuration is loaded now; concrete broker clients will be wired in a
later gate. The shared Go broker boundary and topic constants live under
`pkg/broker`.

## Broker Client

The concrete Kafka-compatible Go client lives under `internal/broker/kafka`.
It uses the shared `pkg/broker` interfaces and preserves SignalOps metadata in
Kafka record headers:

- `correlation_id`
- `causation_id`
- `trace_id`

Dockerized integration tests use host networking because the local Redpanda
compose listener advertises `localhost:19092`.
