import unittest
from pathlib import Path

from signalops_workers.schema_validation import JsonSchemaValidator, SchemaValidationError


class SchemaValidationTests(unittest.TestCase):
    def setUp(self) -> None:
        self.validator = JsonSchemaValidator(Path("contracts/events"))

    def test_validates_signal_event(self) -> None:
        self.validator.validate(valid_signal_event(), "signal.v1.schema.json")

    def test_rejects_missing_required_signal_field(self) -> None:
        event = valid_signal_event()
        del event["tenant_id"]

        with self.assertRaisesRegex(SchemaValidationError, "tenant_id"):
            self.validator.validate(event, "signal.v1.schema.json")

    def test_rejects_invalid_signal_confidence(self) -> None:
        event = valid_signal_event()
        event["confidence"] = 1.5

        with self.assertRaisesRegex(SchemaValidationError, "confidence"):
            self.validator.validate(event, "signal.v1.schema.json")

    def test_rejects_unexpected_signal_field(self) -> None:
        event = valid_signal_event()
        event["unexpected"] = True

        with self.assertRaisesRegex(SchemaValidationError, "unexpected"):
            self.validator.validate(event, "signal.v1.schema.json")


def valid_signal_event() -> dict[str, object]:
    return {
        "signal_id": "sig-1",
        "tenant_id": "tenant-1",
        "source_id": "source-1",
        "source_domain": "market_data",
        "source_adapter": "market_data.massive",
        "ingestion_mode": "scheduled_pull",
        "dataset": "daily_options",
        "event_ids": ["evt-1"],
        "artifact_ids": [],
        "signal_type": "test_signal",
        "detector_id": "detector-1",
        "detector_version": "1.0.0",
        "model_version": "none",
        "timestamp": "2026-07-07T00:00:00Z",
        "observation_time": "2026-07-07T00:00:00Z",
        "effective_time": "2026-07-07T00:00:00Z",
        "processing_time": "2026-07-07T00:00:00Z",
        "window_start": "2026-07-07T00:00:00Z",
        "window_end": "2026-07-07T00:00:00Z",
        "confidence": 0.5,
        "severity": "low",
        "entities": [],
        "supporting_metrics": {},
        "graph_targets": [],
        "semantic_evidence": [],
        "evidence": [{"type": "raw_event", "ref": "evt-1"}],
        "recommendation": None,
        "correlation_id": "corr-1",
        "trace_id": "trace-1",
        "causation_id": "evt-1",
        "replay_job_id": "",
    }


if __name__ == "__main__":
    unittest.main()
