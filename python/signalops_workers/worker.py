from __future__ import annotations

import json
import logging
from dataclasses import dataclass
from typing import Mapping, Protocol


logger = logging.getLogger(__name__)


class WorkerError(Exception):
    """Base error for worker processing failures."""


class InvalidRawEventError(WorkerError):
    """Raised when a consumed raw event is not a JSON object."""


@dataclass(frozen=True)
class BrokerMessage:
    topic: str
    partition: int
    offset: int
    key: str
    value: bytes
    headers: Mapping[str, str]


@dataclass(frozen=True)
class ProcessedRawEvent:
    event_id: str
    idempotency_key: str
    correlation_id: str
    payload: Mapping[str, object]


class RawEventConsumer(Protocol):
    def poll(self, timeout_seconds: float) -> BrokerMessage | None:
        ...

    def commit(self, message: BrokerMessage) -> None:
        ...

    def close(self) -> None:
        ...


class DeadLetterPublisher(Protocol):
    def publish(self, message: BrokerMessage, error: Exception) -> None:
        ...

    def close(self) -> None:
        ...


class RawEventHandler:
    def handle(self, message: BrokerMessage) -> ProcessedRawEvent:
        payload = _decode_json_object(message.value)
        event_id = _first_non_empty(
            message.headers.get("signalops_event_id", ""),
            _string_field(payload, "event_id"),
        )
        idempotency_key = _first_non_empty(
            message.headers.get("signalops_idempotency", ""),
            _string_field(payload, "idempotency_key"),
            message.key,
            event_id,
        )
        correlation_id = _first_non_empty(
            message.headers.get("correlation_id", ""),
            _string_field(payload, "correlation_id"),
        )

        if not event_id:
            raise InvalidRawEventError("raw event is missing event_id")
        if not idempotency_key:
            raise InvalidRawEventError("raw event is missing idempotency key")

        return ProcessedRawEvent(
            event_id=event_id,
            idempotency_key=idempotency_key,
            correlation_id=correlation_id,
            payload=payload,
        )


def run_worker(
    consumer: RawEventConsumer,
    handler: RawEventHandler,
    *,
    poll_timeout_seconds: float,
    max_messages: int = 0,
    dead_letter_publisher: DeadLetterPublisher | None = None,
) -> int:
    processed_count = 0
    try:
        while max_messages <= 0 or processed_count < max_messages:
            message = consumer.poll(poll_timeout_seconds)
            if message is None:
                continue

            try:
                processed = handler.handle(message)
            except Exception as exc:
                if dead_letter_publisher is None:
                    raise
                dead_letter_publisher.publish(message, exc)
                logger.warning(
                    "sent raw event to dlq",
                    extra={
                        "error": str(exc),
                        "error_type": type(exc).__name__,
                        "topic": message.topic,
                        "partition": message.partition,
                        "offset": message.offset,
                    },
                )
            else:
                logger.info(
                    "processed raw event",
                    extra={
                        "event_id": processed.event_id,
                        "idempotency_key": processed.idempotency_key,
                        "correlation_id": processed.correlation_id,
                        "topic": message.topic,
                        "partition": message.partition,
                        "offset": message.offset,
                    },
                )
            consumer.commit(message)
            processed_count += 1
    finally:
        consumer.close()
        if dead_letter_publisher is not None:
            dead_letter_publisher.close()

    return processed_count


def _decode_json_object(value: bytes) -> Mapping[str, object]:
    try:
        decoded = json.loads(value.decode("utf-8"))
    except (UnicodeDecodeError, json.JSONDecodeError) as exc:
        raise InvalidRawEventError("raw event value must be valid JSON") from exc

    if not isinstance(decoded, dict):
        raise InvalidRawEventError("raw event value must be a JSON object")
    return decoded


def _string_field(payload: Mapping[str, object], key: str) -> str:
    value = payload.get(key)
    if isinstance(value, str):
        return value.strip()
    return ""


def _first_non_empty(*values: str) -> str:
    for value in values:
        value = value.strip()
        if value:
            return value
    return ""
