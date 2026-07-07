from __future__ import annotations

import logging

from signalops_workers.broker import (
    RedpandaDeadLetterPublisher,
    RedpandaRawEventConsumer,
    RedpandaRetryPublisher,
    RedpandaSignalPublisher,
)
from signalops_workers.config import load_config
from signalops_workers.detectors import load_detector
from signalops_workers.worker import RawEventHandler, run_worker


def main() -> int:
    config = load_config()
    logging.basicConfig(
        level=config.log_level,
        format="%(asctime)s %(levelname)s %(name)s %(message)s",
    )

    detector = load_detector(
        config.detector_id,
        environment=config.environment,
        worker_id=config.group_id,
    )

    consumer = RedpandaRawEventConsumer(
        brokers=config.brokers,
        group_id=config.group_id,
        input_topic=config.input_topic,
    )
    retry_publisher = RedpandaRetryPublisher(
        brokers=config.brokers,
        retry_topic=config.retry_topic,
    )
    dead_letter_publisher = RedpandaDeadLetterPublisher(
        brokers=config.brokers,
        dlq_topic=config.dlq_topic,
    )
    signal_publisher = RedpandaSignalPublisher(
        brokers=config.brokers,
        signal_topic=config.signal_topic,
    )
    processed = run_worker(
        consumer,
        RawEventHandler(),
        poll_timeout_seconds=config.poll_timeout_seconds,
        max_messages=config.max_messages,
        retry_publisher=retry_publisher,
        dead_letter_publisher=dead_letter_publisher,
        signal_publisher=signal_publisher,
        detector=detector,
    )
    logging.getLogger(__name__).info("worker stopped", extra={"processed": processed})
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
