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

    for schema_path in schemas:
        with schema_path.open("r", encoding="utf-8") as handle:
            schema = json.load(handle)
        for key in ("$schema", "$id", "title"):
            if key not in schema:
                raise SystemExit(f"{schema_path}: missing {key}")
        if schema.get("type") != "object" and "$defs" not in schema:
            raise SystemExit(f"{schema_path}: expected object schema or definitions")
        print(f"ok {schema_path.relative_to(root)}")

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
