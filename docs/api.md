# SignalOps API

## Raw Event Ingestion

`POST /v1/events/raw`

Accepts a JSON object and publishes it to the durable raw event topic:

```text
signalops.<environment>.raw.v1
```

The endpoint returns `202 Accepted` only after the broker publish is
acknowledged.

### Request Headers

- `Content-Type: application/json`
- `X-SignalOps-Event-ID`: optional event identifier override.
- `X-Idempotency-Key`: optional broker key override.
- `X-Correlation-ID`: optional correlation identifier.
- `X-Causation-ID`: optional causation identifier.
- `X-Trace-ID`: optional trace identifier.

If identifiers are omitted:

- `event_id` is read from the JSON body, or generated.
- `idempotency_key` is read from the JSON body, or falls back to `event_id`.
- `correlation_id` is read from the JSON body, or generated.

### Broker Mapping

- topic: `signalops.<environment>.raw.v1`
- key: idempotency key
- value: original request JSON bytes
- headers:
  - `content_type`
  - `signalops_event_id`
  - `signalops_idempotency`
  - `signalops_ingest_route`
  - `signalops_ingest_format`
  - `correlation_id`
  - `causation_id`
  - `trace_id`

### Success Response

```json
{
  "status": "accepted",
  "event_id": "evt-123",
  "idempotency_key": "idem-123",
  "correlation_id": "corr-123",
  "topic": "signalops.local.raw.v1",
  "partition": 0,
  "offset": 1
}
```

### Error Responses

- `400 invalid_json`: request body is not a JSON object.
- `502 publish_failed`: broker publish failed.
- `503 broker_unavailable`: ingestion route is not wired with a publisher.
