from __future__ import annotations

import json
from datetime import datetime
from pathlib import Path
from typing import Mapping, Sequence
from urllib.parse import urldefrag


class SchemaValidationError(ValueError):
    pass


class JsonSchemaValidator:
    def __init__(self, schema_root: Path) -> None:
        self._schema_root = schema_root
        self._cache: dict[str, Mapping[str, object]] = {}

    def validate(self, instance: object, schema_name: str) -> None:
        schema = self._load(schema_name)
        self._validate(instance, schema, path="$", base_schema_name=schema_name)

    def _load(self, schema_name: str) -> Mapping[str, object]:
        if schema_name not in self._cache:
            schema_path = self._schema_root / schema_name
            with schema_path.open("r", encoding="utf-8") as handle:
                loaded = json.load(handle)
            if not isinstance(loaded, Mapping):
                raise SchemaValidationError(f"{schema_name}: schema must be an object")
            self._cache[schema_name] = loaded
        return self._cache[schema_name]

    def _resolve_ref(
        self, ref: str, base_schema_name: str
    ) -> tuple[Mapping[str, object], str]:
        ref_file, fragment = urldefrag(ref)
        schema_name = ref_file or base_schema_name
        schema = self._load(schema_name)
        target: object = schema
        if fragment:
            for raw_part in fragment.lstrip("/").split("/"):
                part = raw_part.replace("~1", "/").replace("~0", "~")
                if not isinstance(target, Mapping) or part not in target:
                    raise SchemaValidationError(f"unresolvable schema ref: {ref}")
                target = target[part]
        if not isinstance(target, Mapping):
            raise SchemaValidationError(f"schema ref does not resolve to object: {ref}")
        return target, schema_name

    def _validate(
        self,
        instance: object,
        schema: Mapping[str, object],
        *,
        path: str,
        base_schema_name: str,
    ) -> None:
        ref = schema.get("$ref")
        if isinstance(ref, str):
            target, target_schema_name = self._resolve_ref(ref, base_schema_name)
            self._validate(instance, target, path=path, base_schema_name=target_schema_name)
            return

        if "const" in schema and instance != schema["const"]:
            raise SchemaValidationError(f"{path}: expected const {schema['const']!r}")

        enum = schema.get("enum")
        if isinstance(enum, Sequence) and not isinstance(enum, (str, bytes, bytearray)):
            if instance not in enum:
                raise SchemaValidationError(f"{path}: value {instance!r} is not in enum")

        expected_type = schema.get("type")
        if expected_type is not None:
            if not self._matches_type(instance, expected_type):
                raise SchemaValidationError(f"{path}: expected type {expected_type!r}")

        if isinstance(instance, Mapping):
            self._validate_object(instance, schema, path=path, base_schema_name=base_schema_name)
        elif isinstance(instance, list):
            self._validate_array(instance, schema, path=path, base_schema_name=base_schema_name)
        elif isinstance(instance, str):
            self._validate_string(instance, schema, path=path)
        elif isinstance(instance, (int, float)) and not isinstance(instance, bool):
            self._validate_number(instance, schema, path=path)

    def _validate_object(
        self,
        instance: Mapping[object, object],
        schema: Mapping[str, object],
        *,
        path: str,
        base_schema_name: str,
    ) -> None:
        required = schema.get("required", [])
        if isinstance(required, Sequence) and not isinstance(required, (str, bytes, bytearray)):
            for key in required:
                if isinstance(key, str) and key not in instance:
                    raise SchemaValidationError(f"{path}: missing required field {key!r}")

        properties = schema.get("properties", {})
        if not isinstance(properties, Mapping):
            properties = {}

        additional = schema.get("additionalProperties", True)
        if additional is False:
            for key in instance:
                if key not in properties:
                    raise SchemaValidationError(f"{path}: unexpected field {key!r}")

        for key, value in instance.items():
            if not isinstance(key, str):
                raise SchemaValidationError(f"{path}: object keys must be strings")
            child_schema = properties.get(key)
            if isinstance(child_schema, Mapping):
                self._validate(
                    value,
                    child_schema,
                    path=f"{path}.{key}",
                    base_schema_name=base_schema_name,
                )
            elif isinstance(additional, Mapping):
                self._validate(
                    value,
                    additional,
                    path=f"{path}.{key}",
                    base_schema_name=base_schema_name,
                )

    def _validate_array(
        self,
        instance: list[object],
        schema: Mapping[str, object],
        *,
        path: str,
        base_schema_name: str,
    ) -> None:
        min_items = schema.get("minItems")
        if isinstance(min_items, int) and len(instance) < min_items:
            raise SchemaValidationError(f"{path}: expected at least {min_items} items")

        items = schema.get("items")
        if isinstance(items, Mapping):
            for index, value in enumerate(instance):
                self._validate(
                    value,
                    items,
                    path=f"{path}[{index}]",
                    base_schema_name=base_schema_name,
                )

    def _validate_string(
        self, instance: str, schema: Mapping[str, object], *, path: str
    ) -> None:
        min_length = schema.get("minLength")
        if isinstance(min_length, int) and len(instance) < min_length:
            raise SchemaValidationError(f"{path}: expected length at least {min_length}")
        if schema.get("format") == "date-time":
            value = instance.replace("Z", "+00:00")
            try:
                datetime.fromisoformat(value)
            except ValueError as exc:
                raise SchemaValidationError(f"{path}: expected date-time") from exc

    def _validate_number(
        self, instance: int | float, schema: Mapping[str, object], *, path: str
    ) -> None:
        minimum = schema.get("minimum")
        if isinstance(minimum, (int, float)) and instance < minimum:
            raise SchemaValidationError(f"{path}: expected >= {minimum}")
        maximum = schema.get("maximum")
        if isinstance(maximum, (int, float)) and instance > maximum:
            raise SchemaValidationError(f"{path}: expected <= {maximum}")

    def _matches_type(self, instance: object, expected: object) -> bool:
        if isinstance(expected, list):
            return any(self._matches_type(instance, item) for item in expected)
        if expected == "object":
            return isinstance(instance, Mapping)
        if expected == "array":
            return isinstance(instance, list)
        if expected == "string":
            return isinstance(instance, str)
        if expected == "integer":
            return isinstance(instance, int) and not isinstance(instance, bool)
        if expected == "number":
            return isinstance(instance, (int, float)) and not isinstance(instance, bool)
        if expected == "boolean":
            return isinstance(instance, bool)
        if expected == "null":
            return instance is None
        return True


_DEFAULT_VALIDATOR: JsonSchemaValidator | None = None


def validate_signal_event(signal_event: Mapping[str, object]) -> None:
    global _DEFAULT_VALIDATOR
    if _DEFAULT_VALIDATOR is None:
        _DEFAULT_VALIDATOR = JsonSchemaValidator(_default_schema_root())
    _DEFAULT_VALIDATOR.validate(signal_event, "signal.v1.schema.json")


def _default_schema_root() -> Path:
    current = Path(__file__).resolve()
    for parent in current.parents:
        candidate = parent / "contracts" / "events"
        if candidate.exists():
            return candidate
    raise SchemaValidationError("contracts/events schema directory was not found")
