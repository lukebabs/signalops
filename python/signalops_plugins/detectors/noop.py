from __future__ import annotations

from typing import Mapping, Sequence

from signalops_plugins.detectors.base import (
    DetectionResult,
    DetectorPlugin,
    EmittedSignal,
    Explanation,
    FeatureContext,
    RuntimeContext,
)


class NoopDetector(DetectorPlugin):
    detector_id = "signalops.noop"
    detector_version = "0.1.0"
    model_version = "none"

    def __init__(self) -> None:
        self.initialized = False
        self.runtime_context: RuntimeContext | None = None

    def initialize(
        self,
        config: Mapping[str, object],
        model_registry: object | None,
        runtime_context: RuntimeContext,
    ) -> None:
        self.initialized = True
        self.runtime_context = runtime_context

    def detect(
        self,
        normalized_events: Sequence[Mapping[str, object]],
        feature_context: FeatureContext,
    ) -> DetectionResult:
        return DetectionResult(
            detector_id=self.detector_id,
            detector_version=self.detector_version,
            matched=False,
            score=0.0,
            reason="noop detector does not emit signals",
            metadata={"event_count": len(normalized_events)},
        )

    def explain(self, detection_result: DetectionResult) -> Explanation:
        return Explanation(summary=detection_result.reason, evidence=())

    def emit_signal(
        self,
        detection_result: DetectionResult,
        explanation: Explanation,
    ) -> EmittedSignal | None:
        return None
