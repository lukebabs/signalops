from __future__ import annotations

import argparse
import json
import sys
from datetime import datetime, timezone
from typing import Iterable, Mapping

from signalops_plugins.detectors.base import FeatureContext
from signalops_workers.detectors import load_detector
from signalops_workers.worker import ProcessedRawEvent, build_signal_event, validate_built_signal_event


def run_batch(
    events: Iterable[Mapping[str, object]],
    *,
    detector_id: str,
    environment: str = "local",
    worker_id: str = "marketops-backtest",
    now: datetime | None = None,
) -> list[Mapping[str, object]]:
    detector = load_detector(detector_id, environment=environment, worker_id=worker_id)
    emitted: list[Mapping[str, object]] = []
    for event in events:
        event_id = _string(event.get("event_id"))
        processed = ProcessedRawEvent(
            event_id=event_id,
            idempotency_key=_string(event.get("idempotency_key")) or event_id,
            correlation_id=_string(event.get("correlation_id")) or event_id,
            payload=dict(event),
        )
        detection = detector.detect([processed.payload], FeatureContext())
        explanation = detector.explain(detection)
        signal = detector.emit_signal(detection, explanation)
        if signal is None:
            continue
        signal_event = build_signal_event(processed, detector, signal, explanation, now=now)
        validate_built_signal_event(signal_event)
        emitted.append(signal_event)
    return emitted


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Run SignalOps MarketOps detector over normalized-event JSONL for back-tests.")
    parser.add_argument("--detector-id", default="marketops.dsm.taxonomy_v1")
    parser.add_argument("--environment", default="local")
    parser.add_argument("--worker-id", default="marketops-backtest")
    args = parser.parse_args(argv)

    events = []
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        decoded = json.loads(line)
        if not isinstance(decoded, dict):
            raise ValueError("each input line must be a JSON object")
        events.append(decoded)

    for signal_event in run_batch(
        events,
        detector_id=args.detector_id,
        environment=args.environment,
        worker_id=args.worker_id,
        now=datetime.now(timezone.utc),
    ):
        sys.stdout.write(json.dumps(signal_event, separators=(",", ":"), sort_keys=True) + "\n")
    return 0


def _string(value: object) -> str:
    return value.strip() if isinstance(value, str) else ""


if __name__ == "__main__":
    raise SystemExit(main())
