from __future__ import annotations

import json
import logging
from dataclasses import dataclass
from datetime import datetime, timezone
from typing import Mapping, Protocol, Sequence

from signalops_plugins.detectors.base import (
    DetectorPlugin,
    EmittedSignal,
    Explanation,
    FeatureContext,
)


logger = logging.getLogger(__name__)


class WorkerError(Exception):
    """Base error for worker processing failures."""


class InvalidRawEventError(WorkerError):
    """Raised when a consumed raw event is not a JSON object."""


class RetryableWorkerError(WorkerError):
    """Raised when processing should be retried through the durable retry topic."""


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


class FailurePublisher(Protocol):
    def publish(self, message: BrokerMessage, error: Exception) -> None:
        ...

    def close(self) -> None:
        ...


class SignalPublisher(Protocol):
    def publish(
        self, signal_event: Mapping[str, object], source_message: BrokerMessage
    ) -> None:
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
    retry_publisher: FailurePublisher | None = None,
    dead_letter_publisher: FailurePublisher | None = None,
    signal_publisher: SignalPublisher | None = None,
    detector: DetectorPlugin | None = None,
) -> int:
    processed_count = 0
    try:
        while max_messages <= 0 or processed_count < max_messages:
            message = consumer.poll(poll_timeout_seconds)
            if message is None:
                continue

            try:
                processed = handler.handle(message)
            except RetryableWorkerError as exc:
                if retry_publisher is None:
                    raise
                retry_publisher.publish(message, exc)
                logger.warning(
                    "sent raw event to retry",
                    extra={
                        "error": str(exc),
                        "error_type": type(exc).__name__,
                        "topic": message.topic,
                        "partition": message.partition,
                        "offset": message.offset,
                    },
                )
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
                try:
                    signal = None
                    if detector is not None:
                        detection = detector.detect([processed.payload], FeatureContext())
                        explanation = detector.explain(detection)
                        signal = detector.emit_signal(detection, explanation)
                        if signal is not None:
                            if signal_publisher is None:
                                raise RetryableWorkerError(
                                    "signal publisher is required for emitted signals"
                                )
                            signal_event = build_signal_event(
                                processed,
                                detector,
                                signal,
                                explanation,
                            )
                            try:
                                signal_publisher.publish(signal_event, message)
                            except Exception as exc:
                                raise RetryableWorkerError(
                                    "failed to publish emitted signal"
                                ) from exc
                        logger.info(
                            "detector evaluated raw event",
                            extra={
                                "event_id": processed.event_id,
                                "detector_id": detection.detector_id,
                                "detector_version": detection.detector_version,
                                "matched": detection.matched,
                                "score": detection.score,
                                "signal_emitted": signal is not None,
                                "signal_id": signal.signal_id if signal is not None else "",
                            },
                        )
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
                except RetryableWorkerError as exc:
                    if retry_publisher is None:
                        raise
                    retry_publisher.publish(message, exc)
                    logger.warning(
                        "sent raw event to retry",
                        extra={
                            "error": str(exc),
                            "error_type": type(exc).__name__,
                            "topic": message.topic,
                            "partition": message.partition,
                            "offset": message.offset,
                        },
                    )
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
            consumer.commit(message)
            processed_count += 1
    finally:
        consumer.close()
        if retry_publisher is not None:
            retry_publisher.close()
        if dead_letter_publisher is not None:
            dead_letter_publisher.close()
        if signal_publisher is not None:
            signal_publisher.close()

    return processed_count


def build_signal_event(
    processed: ProcessedRawEvent,
    detector: DetectorPlugin,
    signal: EmittedSignal,
    explanation: Explanation,
    *,
    now: datetime | None = None,
) -> Mapping[str, object]:
    raw = processed.payload
    signal_payload = signal.payload
    now_text = _timestamp(now or datetime.now(timezone.utc))
    observation_time = _first_non_empty(
        _string_field(raw, "observation_time"),
        _string_field(raw, "observed_at"),
        _string_field(raw, "occurred_at"),
        now_text,
    )
    effective_time = _first_non_empty(
        _string_field(raw, "effective_time"),
        _string_field(raw, "occurred_at"),
        observation_time,
    )
    processing_time = _first_non_empty(_string_field(raw, "processing_time"), now_text)
    window_start = _first_non_empty(
        _string_field(signal_payload, "window_start"), observation_time
    )
    window_end = _first_non_empty(
        _string_field(signal_payload, "window_end"), effective_time
    )

    evidence = _object_sequence(signal_payload.get("evidence"))
    if not evidence:
        evidence = [
            {
                "type": "raw_event",
                "ref": processed.event_id,
                "summary": explanation.summary,
            }
        ]

    return {
        "signal_id": signal.signal_id,
        "tenant_id": _required_string(raw, "tenant_id"),
        "source_id": _required_string(raw, "source_id"),
        "source_domain": _required_string(raw, "source_domain"),
        "source_adapter": _required_string(raw, "source_adapter"),
        "ingestion_mode": _required_string(raw, "ingestion_mode"),
        "dataset": _required_string(raw, "dataset"),
        "event_ids": _string_sequence(signal_payload.get("event_ids"))
        or [processed.event_id],
        "artifact_ids": _string_sequence(signal_payload.get("artifact_ids")),
        "signal_type": signal.signal_type,
        "detector_id": detector.detector_id,
        "detector_version": detector.detector_version,
        "model_version": detector.model_version,
        "timestamp": now_text,
        "observation_time": observation_time,
        "effective_time": effective_time,
        "processing_time": processing_time,
        "window_start": window_start,
        "window_end": window_end,
        "confidence": signal.confidence,
        "severity": signal.severity,
        "entities": _object_sequence(signal_payload.get("entities")),
        "supporting_metrics": _object_mapping(signal_payload.get("supporting_metrics")),
        "graph_targets": _object_sequence(signal_payload.get("graph_targets")),
        "semantic_evidence": _object_sequence(signal_payload.get("semantic_evidence")),
        "evidence": evidence,
        "recommendation": _nullable_object(signal_payload.get("recommendation")),
        "correlation_id": _first_non_empty(
            processed.correlation_id,
            _string_field(raw, "correlation_id"),
            processed.event_id,
        ),
        "trace_id": _string_field(raw, "trace_id"),
        "causation_id": _first_non_empty(
            _string_field(raw, "causation_id"), processed.event_id
        ),
        "replay_job_id": _string_field(raw, "replay_job_id"),
    }


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


def _required_string(payload: Mapping[str, object], key: str) -> str:
    value = _string_field(payload, key)
    if not value:
        raise InvalidRawEventError(f"signal emission requires raw field: {key}")
    return value


def _string_sequence(value: object) -> list[str]:
    if not isinstance(value, Sequence) or isinstance(value, (str, bytes, bytearray)):
        return []
    strings = []
    for item in value:
        if isinstance(item, str) and item.strip():
            strings.append(item.strip())
    return strings


def _object_sequence(value: object) -> list[Mapping[str, object]]:
    if not isinstance(value, Sequence) or isinstance(value, (str, bytes, bytearray)):
        return []
    objects: list[Mapping[str, object]] = []
    for item in value:
        if isinstance(item, Mapping):
            objects.append(dict(item))
    return objects


def _object_mapping(value: object) -> Mapping[str, object]:
    if isinstance(value, Mapping):
        return dict(value)
    return {}


def _nullable_object(value: object) -> Mapping[str, object] | None:
    if isinstance(value, Mapping):
        return dict(value)
    return None


def _timestamp(value: datetime) -> str:
    return value.astimezone(timezone.utc).isoformat().replace("+00:00", "Z")
