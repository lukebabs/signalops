from __future__ import annotations

import base64
import json
import logging
from dataclasses import dataclass
from typing import Mapping, Protocol

from signalops_workers.worker import BrokerMessage


logger = logging.getLogger(__name__)


class RetryReplayError(Exception):
    """Base error for retry replay failures."""


class InvalidRetryEventError(RetryReplayError):
    """Raised when a retry topic record cannot be replayed."""


class RetryAttemptsExhausted(RetryReplayError):
    """Raised when a retry record has reached its configured attempt limit."""


class RetryEventConsumer(Protocol):
    def poll(self, timeout_seconds: float) -> BrokerMessage | None:
        ...

    def commit(self, message: BrokerMessage) -> None:
        ...

    def close(self) -> None:
        ...


class RawEventPublisher(Protocol):
    def publish(self, message: BrokerMessage) -> None:
        ...

    def close(self) -> None:
        ...


class DeadLetterPublisher(Protocol):
    def publish(self, message: BrokerMessage, error: Exception) -> None:
        ...

    def close(self) -> None:
        ...


@dataclass(frozen=True)
class ReplayDecision:
    action: str
    retry_attempt: int
    source_message: BrokerMessage | None = None


def run_retry_replayer(
    consumer: RetryEventConsumer,
    raw_publisher: RawEventPublisher,
    dead_letter_publisher: DeadLetterPublisher,
    *,
    poll_timeout_seconds: float,
    max_retry_attempts: int,
    max_messages: int = 0,
) -> int:
    handled_count = 0
    try:
        while max_messages <= 0 or handled_count < max_messages:
            message = consumer.poll(poll_timeout_seconds)
            if message is None:
                continue

            try:
                decision = retry_replay_decision(message, max_retry_attempts)
                if decision.action == "replay":
                    assert decision.source_message is not None
                    raw_publisher.publish(decision.source_message)
                    logger.info(
                        "replayed retry event",
                        extra={
                            "retry_topic": message.topic,
                            "retry_partition": message.partition,
                            "retry_offset": message.offset,
                            "source_topic": decision.source_message.topic,
                            "source_partition": decision.source_message.partition,
                            "source_offset": decision.source_message.offset,
                            "retry_attempt": decision.retry_attempt,
                        },
                    )
                elif decision.action == "dlq":
                    assert decision.source_message is not None
                    dead_letter_publisher.publish(
                        decision.source_message,
                        RetryAttemptsExhausted(
                            f"retry attempts exhausted at {decision.retry_attempt}"
                        ),
                    )
                    logger.warning(
                        "sent exhausted retry event to dlq",
                        extra={
                            "retry_topic": message.topic,
                            "retry_partition": message.partition,
                            "retry_offset": message.offset,
                            "source_topic": decision.source_message.topic,
                            "source_partition": decision.source_message.partition,
                            "source_offset": decision.source_message.offset,
                            "retry_attempt": decision.retry_attempt,
                            "max_retry_attempts": max_retry_attempts,
                        },
                    )
                else:
                    raise InvalidRetryEventError(f"unknown replay action: {decision.action}")
            except InvalidRetryEventError as exc:
                dead_letter_publisher.publish(message, exc)
                logger.warning(
                    "sent invalid retry event to dlq",
                    extra={
                        "error": str(exc),
                        "retry_topic": message.topic,
                        "retry_partition": message.partition,
                        "retry_offset": message.offset,
                    },
                )

            consumer.commit(message)
            handled_count += 1
    finally:
        consumer.close()
        raw_publisher.close()
        dead_letter_publisher.close()

    return handled_count


def retry_replay_decision(message: BrokerMessage, max_retry_attempts: int) -> ReplayDecision:
    if max_retry_attempts < 1:
        raise ValueError("max_retry_attempts must be at least 1")

    retry_event = _decode_retry_event(message.value)
    retry_attempt = _int_field(retry_event, "retry_attempt")
    source = retry_event.get("source")
    if not isinstance(source, Mapping):
        raise InvalidRetryEventError("retry event source must be an object")

    source_message = _source_message(source, retry_attempt)
    if retry_attempt >= max_retry_attempts:
        return ReplayDecision(
            action="dlq", retry_attempt=retry_attempt, source_message=source_message
        )
    return ReplayDecision(
        action="replay", retry_attempt=retry_attempt, source_message=source_message
    )


def _decode_retry_event(value: bytes) -> Mapping[str, object]:
    try:
        decoded = json.loads(value.decode("utf-8"))
    except (UnicodeDecodeError, json.JSONDecodeError) as exc:
        raise InvalidRetryEventError("retry event value must be valid JSON") from exc
    if not isinstance(decoded, Mapping):
        raise InvalidRetryEventError("retry event value must be a JSON object")
    schema_id = decoded.get("schema_id")
    if schema_id != "signalops.retry.raw_event.v1":
        raise InvalidRetryEventError("retry event schema_id is not supported")
    return decoded


def _source_message(source: Mapping[str, object], retry_attempt: int) -> BrokerMessage:
    topic = _required_string(source, "topic")
    partition = _int_field(source, "partition")
    offset = _int_field(source, "offset")
    key = _optional_string(source, "key")
    headers = _headers(source.get("headers"))
    headers["signalops_retry_attempt"] = str(retry_attempt)
    headers["signalops_replayed_from_retry"] = "true"
    value = _source_value(source)

    return BrokerMessage(
        topic=topic,
        partition=partition,
        offset=offset,
        key=key,
        value=value,
        headers=headers,
    )


def _source_value(source: Mapping[str, object]) -> bytes:
    encoded = _required_string(source, "value_base64")
    try:
        return base64.b64decode(encoded.encode("ascii"), validate=True)
    except Exception as exc:
        raise InvalidRetryEventError("source value_base64 must be valid base64") from exc


def _headers(value: object) -> dict[str, str]:
    if not isinstance(value, Mapping):
        raise InvalidRetryEventError("source headers must be an object")
    headers: dict[str, str] = {}
    for key, item in value.items():
        if isinstance(key, str) and isinstance(item, str):
            headers[key] = item
        else:
            raise InvalidRetryEventError("source headers must be string pairs")
    return headers


def _required_string(payload: Mapping[str, object], key: str) -> str:
    value = payload.get(key)
    if not isinstance(value, str) or not value.strip():
        raise InvalidRetryEventError(f"{key} must be a non-empty string")
    return value.strip()


def _optional_string(payload: Mapping[str, object], key: str) -> str:
    value = payload.get(key)
    if value is None:
        return ""
    if not isinstance(value, str):
        raise InvalidRetryEventError(f"{key} must be a string")
    return value


def _int_field(payload: Mapping[str, object], key: str) -> int:
    value = payload.get(key)
    if not isinstance(value, int) or value < 0:
        raise InvalidRetryEventError(f"{key} must be a non-negative integer")
    if key == "retry_attempt" and value < 1:
        raise InvalidRetryEventError("retry_attempt must be at least 1")
    return value
