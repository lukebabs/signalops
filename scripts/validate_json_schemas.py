#!/usr/bin/env python3
"""Validate that repository JSON Schema files parse and expose required metadata."""

from __future__ import annotations

import json
from pathlib import Path


def main() -> int:
    root = Path(__file__).resolve().parents[1]
    schemas = sorted(root.glob("contracts/**/*.schema.json"))
    if not schemas:
        raise SystemExit("no schema files found")

    loaded = {}
    for schema_path in schemas:
        with schema_path.open("r", encoding="utf-8") as handle:
            schema = json.load(handle)
        for key in ("$schema", "$id", "title"):
            if key not in schema:
                raise SystemExit(f"{schema_path}: missing {key}")
        if schema.get("type") != "object" and "$defs" not in schema:
            raise SystemExit(f"{schema_path}: expected object schema or definitions")
        loaded[schema_path.name] = schema
        print(f"ok {schema_path.relative_to(root)}")

    common = loaded["common.defs.v1.schema.json"]
    quality = common["$defs"].get("quality_envelope")
    expected_states = ["usable", "degraded", "partial", "stale", "missing", "invalid", "contradictory", "not_applicable", "suppressed"]
    if quality is None or quality.get("required") != ["quality_state", "quality_policy_id", "quality_policy_version"]:
        raise SystemExit("common quality envelope is incomplete")
    if common["$defs"].get("quality_state", {}).get("enum") != expected_states:
        raise SystemExit("common quality state enum is invalid")
    for name in ("raw_signal_event.v1.schema.json", "normalized_signal_event.v1.schema.json"):
        quality_ref = loaded[name]["properties"].get("metadata", {}).get("properties", {}).get("quality", {}).get("$ref")
        if quality_ref != "common.defs.v1.schema.json#/$defs/quality_envelope":
            raise SystemExit(f"{name}: metadata.quality must reference the common quality envelope")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
