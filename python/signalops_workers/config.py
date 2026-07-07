from __future__ import annotations

import os
from dataclasses import dataclass


DEFAULT_ENVIRONMENT = "local"
DEFAULT_BROKERS = "redpanda:9092"
DEFAULT_GROUP_ID = "signalops.raw-worker.v1"
DEFAULT_POLL_TIMEOUT_SECONDS = 1.0


@dataclass(frozen=True)
class WorkerConfig:
    brokers: str
    environment: str
    input_topic: str
    retry_topic: str
    dlq_topic: str
    group_id: str
    poll_timeout_seconds: float
    max_messages: int
    detector_id: str
    log_level: str


def load_config() -> WorkerConfig:
    environment = _env("SIGNALOPS_ENV", DEFAULT_ENVIRONMENT)
    return WorkerConfig(
        brokers=_env("SIGNALOPS_BROKER_BROKERS", DEFAULT_BROKERS),
        environment=environment,
        input_topic=_env("SIGNALOPS_WORKER_INPUT_TOPIC", raw_topic(environment)),
        retry_topic=_env("SIGNALOPS_WORKER_RETRY_TOPIC", retry_topic(environment)),
        dlq_topic=_env("SIGNALOPS_WORKER_DLQ_TOPIC", dlq_topic(environment)),
        group_id=_env("SIGNALOPS_WORKER_GROUP_ID", DEFAULT_GROUP_ID),
        poll_timeout_seconds=_float_env(
            "SIGNALOPS_WORKER_POLL_TIMEOUT_SECONDS", DEFAULT_POLL_TIMEOUT_SECONDS
        ),
        max_messages=_int_env("SIGNALOPS_WORKER_MAX_MESSAGES", 0),
        detector_id=_env("SIGNALOPS_WORKER_DETECTOR_ID", "signalops.noop"),
        log_level=_env("SIGNALOPS_WORKER_LOG_LEVEL", "INFO"),
    )


def raw_topic(environment: str) -> str:
    environment = environment.strip() or DEFAULT_ENVIRONMENT
    return f"signalops.{environment}.raw.v1"


def retry_topic(environment: str) -> str:
    environment = environment.strip() or DEFAULT_ENVIRONMENT
    return f"signalops.{environment}.retry.algorithm.v1"


def dlq_topic(environment: str) -> str:
    environment = environment.strip() or DEFAULT_ENVIRONMENT
    return f"signalops.{environment}.dlq.algorithm.v1"


def _env(key: str, fallback: str) -> str:
    value = os.getenv(key, "").strip()
    return value or fallback


def _int_env(key: str, fallback: int) -> int:
    value = os.getenv(key, "").strip()
    if not value:
        return fallback
    return int(value)


def _float_env(key: str, fallback: float) -> float:
    value = os.getenv(key, "").strip()
    if not value:
        return fallback
    return float(value)
