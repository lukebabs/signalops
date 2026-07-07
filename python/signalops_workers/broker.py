from __future__ import annotations

import base64
import json
from collections.abc import Mapping
from datetime import datetime, timezone

from confluent_kafka import Consumer, KafkaError, KafkaException, Producer, TopicPartition

from signalops_workers.worker import BrokerMessage


class RedpandaRawEventConsumer:
    def __init__(self, *, brokers: str, group_id: str, input_topic: str) -> None:
        self._consumer = Consumer(
            {
                "bootstrap.servers": brokers,
                "group.id": group_id,
                "auto.offset.reset": "earliest",
                "enable.auto.commit": False,
            }
        )
        self._consumer.subscribe([input_topic])

    def poll(self, timeout_seconds: float) -> BrokerMessage | None:
        message = self._consumer.poll(timeout_seconds)
        if message is None:
            return None
        if message.error():
            if message.error().code() == KafkaError._PARTITION_EOF:
                return None
            raise KafkaException(message.error())

        return BrokerMessage(
            topic=message.topic(),
            partition=message.partition(),
            offset=message.offset(),
            key=_decode_optional(message.key()),
            value=message.value() or b"",
            headers=_headers_to_mapping(message.headers()),
        )

    def commit(self, message: BrokerMessage) -> None:
        self._consumer.commit(
            offsets=[TopicPartition(message.topic, message.partition, message.offset + 1)],
            asynchronous=False,
        )

    def close(self) -> None:
        self._consumer.close()


class RedpandaRawEventPublisher:
    def __init__(self, *, brokers: str, raw_topic: str) -> None:
        self._topic = raw_topic
        self._producer = Producer({"bootstrap.servers": brokers})

    def publish(self, message: BrokerMessage) -> None:
        headers = [(key, value) for key, value in message.headers.items()]
        self._producer.produce(
            self._topic,
            key=message.key,
            value=message.value,
            headers=headers,
        )
        remaining = self._producer.flush(timeout=10)
        if remaining:
            raise TimeoutError("timed out publishing raw replay message")

    def close(self) -> None:
        self._producer.flush(timeout=10)


class RedpandaDeadLetterPublisher:
    def __init__(self, *, brokers: str, dlq_topic: str) -> None:
        self._topic = dlq_topic
        self._producer = Producer({"bootstrap.servers": brokers})

    def publish(self, message: BrokerMessage, error: Exception) -> None:
        payload = {
            "schema_id": "signalops.dlq.raw_event.v1",
            "failed_at": datetime.now(timezone.utc).isoformat(),
            "error_type": type(error).__name__,
            "error_message": str(error),
            "source": {
                "topic": message.topic,
                "partition": message.partition,
                "offset": message.offset,
                "key": message.key,
                "headers": dict(message.headers),
                "value_base64": base64.b64encode(message.value).decode("ascii"),
            },
        }
        headers = [
            ("content_type", "application/json"),
            ("signalops_dlq_reason", type(error).__name__),
            ("signalops_source_topic", message.topic),
            ("signalops_source_partition", str(message.partition)),
            ("signalops_source_offset", str(message.offset)),
        ]
        correlation_id = message.headers.get("correlation_id")
        if correlation_id:
            headers.append(("correlation_id", correlation_id))

        self._producer.produce(
            self._topic,
            key=message.key,
            value=json.dumps(payload, separators=(",", ":")).encode("utf-8"),
            headers=headers,
        )
        remaining = self._producer.flush(timeout=10)
        if remaining:
            raise TimeoutError("timed out publishing DLQ message")

    def close(self) -> None:
        self._producer.flush(timeout=10)


class RedpandaRetryPublisher:
    def __init__(self, *, brokers: str, retry_topic: str) -> None:
        self._topic = retry_topic
        self._producer = Producer({"bootstrap.servers": brokers})

    def publish(self, message: BrokerMessage, error: Exception) -> None:
        payload = {
            "schema_id": "signalops.retry.raw_event.v1",
            "failed_at": datetime.now(timezone.utc).isoformat(),
            "error_type": type(error).__name__,
            "error_message": str(error),
            "retry_attempt": _next_retry_attempt(message.headers),
            "source": {
                "topic": message.topic,
                "partition": message.partition,
                "offset": message.offset,
                "key": message.key,
                "headers": dict(message.headers),
                "value_base64": base64.b64encode(message.value).decode("ascii"),
            },
        }
        headers = [
            ("content_type", "application/json"),
            ("signalops_retry_reason", type(error).__name__),
            ("signalops_retry_attempt", str(payload["retry_attempt"])),
            ("signalops_source_topic", message.topic),
            ("signalops_source_partition", str(message.partition)),
            ("signalops_source_offset", str(message.offset)),
        ]
        correlation_id = message.headers.get("correlation_id")
        if correlation_id:
            headers.append(("correlation_id", correlation_id))

        self._producer.produce(
            self._topic,
            key=message.key,
            value=json.dumps(payload, separators=(",", ":")).encode("utf-8"),
            headers=headers,
        )
        remaining = self._producer.flush(timeout=10)
        if remaining:
            raise TimeoutError("timed out publishing retry message")

    def close(self) -> None:
        self._producer.flush(timeout=10)


class RedpandaSignalPublisher:
    def __init__(self, *, brokers: str, signal_topic: str) -> None:
        self._topic = signal_topic
        self._producer = Producer({"bootstrap.servers": brokers})

    def publish(
        self, signal_event: Mapping[str, object], source_message: BrokerMessage
    ) -> None:
        headers = [
            ("content_type", "application/json"),
            ("signalops_schema_id", "signalops.signal.v1"),
            ("signalops_source_topic", source_message.topic),
            ("signalops_source_partition", str(source_message.partition)),
            ("signalops_source_offset", str(source_message.offset)),
        ]
        correlation_id = signal_event.get("correlation_id")
        if isinstance(correlation_id, str) and correlation_id:
            headers.append(("correlation_id", correlation_id))
        trace_id = signal_event.get("trace_id")
        if isinstance(trace_id, str) and trace_id:
            headers.append(("trace_id", trace_id))

        key = signal_event.get("signal_id")
        self._producer.produce(
            self._topic,
            key=key if isinstance(key, str) else source_message.key,
            value=json.dumps(signal_event, separators=(",", ":")).encode("utf-8"),
            headers=headers,
        )
        remaining = self._producer.flush(timeout=10)
        if remaining:
            raise TimeoutError("timed out publishing signal message")

    def close(self) -> None:
        self._producer.flush(timeout=10)


def _headers_to_mapping(headers: list[tuple[str, bytes | None]] | None) -> Mapping[str, str]:
    mapped: dict[str, str] = {}
    for key, value in headers or []:
        mapped[key] = _decode_optional(value)
    return mapped


def _decode_optional(value: bytes | None) -> str:
    if value is None:
        return ""
    return value.decode("utf-8")


def _next_retry_attempt(headers: Mapping[str, str]) -> int:
    value = headers.get("signalops_retry_attempt", "").strip()
    if not value:
        return 1
    try:
        return int(value) + 1
    except ValueError:
        return 1
