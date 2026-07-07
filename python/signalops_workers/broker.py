from __future__ import annotations

from collections.abc import Mapping

from confluent_kafka import Consumer, KafkaError, KafkaException, TopicPartition

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


def _headers_to_mapping(headers: list[tuple[str, bytes | None]] | None) -> Mapping[str, str]:
    mapped: dict[str, str] = {}
    for key, value in headers or []:
        mapped[key] = _decode_optional(value)
    return mapped


def _decode_optional(value: bytes | None) -> str:
    if value is None:
        return ""
    return value.decode("utf-8")

