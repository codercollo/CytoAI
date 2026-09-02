"""
Request validation for the scoring endpoints.

Deliberately dependency-free (no marshmallow/pydantic) to keep the sidecar's
footprint small per spec.md — a hand-rolled validator is enough for five
numeric fields per model.
"""
from __future__ import annotations

from dataclasses import dataclass


class ValidationError(ValueError):
    def __init__(self, errors: dict[str, str]):
        self.errors = errors
        super().__init__(str(errors))


@dataclass
class FieldSpec:
    name: str
    min_value: float | None = None
    max_value: float | None = None
    required: bool = True


BATTERY_FIELDS = [
    FieldSpec("cycle_count", min_value=0),
    FieldSpec("avg_depth_of_discharge", min_value=0, max_value=100),
    FieldSpec("avg_temperature_c", min_value=-40, max_value=100),
    FieldSpec("age_days", min_value=0),
    FieldSpec("charge_rate_variance", min_value=0),
]

REPAYMENT_FIELDS = [
    FieldSpec("on_time_ratio", min_value=0, max_value=1),
    FieldSpec("avg_days_late", min_value=0),
    FieldSpec("payment_cadence_proxy", min_value=0),
    FieldSpec("loan_to_battery_value_ratio", min_value=0),
    FieldSpec("tenure_days", min_value=0),
    FieldSpec("telemetry_cadence_proxy", min_value=0),
    # Swap-network-only stress profile. Now a trained feature (Phase 6 retrain):
    # required, and 0.0 for leased_fixed callers (Go always sends it).
    FieldSpec("battery_stress_profile", min_value=0),
]


def validate_payload(payload: dict, fields: list[FieldSpec]) -> dict[str, float]:
    if not isinstance(payload, dict):
        raise ValidationError({"_": "request body must be a JSON object"})

    errors: dict[str, str] = {}
    cleaned: dict[str, float] = {}

    for field in fields:
        if field.name not in payload:
            if field.required:
                errors[field.name] = "required"
            continue
        value = payload[field.name]
        try:
            value = float(value)
        except (TypeError, ValueError):
            errors[field.name] = "must be a number"
            continue
        if field.min_value is not None and value < field.min_value:
            errors[field.name] = f"must be >= {field.min_value}"
            continue
        if field.max_value is not None and value > field.max_value:
            errors[field.name] = f"must be <= {field.max_value}"
            continue
        cleaned[field.name] = value

    unknown = set(payload.keys()) - {f.name for f in fields}
    if unknown:
        errors["_"] = f"unexpected field(s): {', '.join(sorted(unknown))}"

    if errors:
        raise ValidationError(errors)
    return cleaned


def validate_battery_payload(payload: dict) -> dict[str, float]:
    return validate_payload(payload, BATTERY_FIELDS)


def validate_repayment_payload(payload: dict) -> dict[str, float]:
    return validate_payload(payload, REPAYMENT_FIELDS)