from __future__ import annotations

from dataclasses import dataclass, field
from typing import Mapping, Protocol, Sequence


@dataclass(frozen=True)
class RuntimeContext:
    environment: str
    worker_id: str


@dataclass(frozen=True)
class FeatureContext:
    features: Mapping[str, object] = field(default_factory=dict)


@dataclass(frozen=True)
class DetectionResult:
    detector_id: str
    detector_version: str
    matched: bool
    score: float
    reason: str
    metadata: Mapping[str, object] = field(default_factory=dict)


@dataclass(frozen=True)
class Explanation:
    summary: str
    evidence: Sequence[Mapping[str, object]] = field(default_factory=tuple)


@dataclass(frozen=True)
class EmittedSignal:
    signal_id: str
    signal_type: str
    confidence: float
    severity: str
    payload: Mapping[str, object]


class DetectorPlugin(Protocol):
    detector_id: str
    detector_version: str
    model_version: str

    def initialize(
        self,
        config: Mapping[str, object],
        model_registry: object | None,
        runtime_context: RuntimeContext,
    ) -> None:
        ...

    def detect(
        self,
        normalized_events: Sequence[Mapping[str, object]],
        feature_context: FeatureContext,
    ) -> DetectionResult:
        ...

    def explain(self, detection_result: DetectionResult) -> Explanation:
        ...

    def emit_signal(
        self,
        detection_result: DetectionResult,
        explanation: Explanation,
    ) -> EmittedSignal | None:
        ...
