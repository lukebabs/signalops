from __future__ import annotations

from signalops_plugins.detectors.base import DetectorPlugin, RuntimeContext
from signalops_plugins.detectors.noop import NoopDetector, StaticSignalDetector


class UnknownDetectorError(ValueError):
    pass


def load_detector(detector_id: str, *, environment: str, worker_id: str) -> DetectorPlugin:
    detector_id = detector_id.strip() or "signalops.noop"
    detectors = {
        "signalops.noop": NoopDetector,
        "signalops.static_test": StaticSignalDetector,
    }
    detector_type = detectors.get(detector_id)
    if detector_type is None:
        raise UnknownDetectorError(f"unknown detector: {detector_id}")

    detector = detector_type()
    detector.initialize(
        config={},
        model_registry=None,
        runtime_context=RuntimeContext(environment=environment, worker_id=worker_id),
    )
    return detector
