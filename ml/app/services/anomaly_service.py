"""Rules-based fraud/anomaly detection for ingested telemetry and repayment data.

Design choice (Feature B): rules, not a statistical model. spec.md §9 commits to
"explainability over accuracy", and every anomaly must be auditable by a lender
or regulator ("state of charge was 140%", "cycle count went backwards"). A model
(e.g. isolation forest) would return a score but not an auditable reason without
extra explanation machinery, so it is out of scope for the MVP.

These rules deliberately DO NOT reject ingestion — callers surface flags as a
confidence signal alongside the score (spec.md §9: no autonomous decisions).
"""
from __future__ import annotations

import math
from datetime import date, datetime

SEVERITY_HIGH = "high"
SEVERITY_MEDIUM = "medium"

# Swap-network stress thresholds (documented, not magic). A rider is flagged
# when their most recent swaps average more than this many fleet standard
# deviations above the fleet mean on temperature or depth-of-discharge,
# sustained across at least SWAP_STRESS_MIN_SWAPS swaps.
SWAP_STRESS_Z_CUTOFF = 1.5
SWAP_STRESS_MIN_SWAPS = 3
SWAP_STRESS_RECENT_LIMIT = 10

# Physical/plausibility bounds. Temperature matches score_schema.py.
SOC_MIN, SOC_MAX = 0.0, 100.0
TEMP_MIN_C, TEMP_MAX_C = -40.0, 100.0
# A single reading-to-reading jump larger than this is treated as implausible.
SOC_JUMP_THRESHOLD = 80.0


def _flag(entity_type, entity_id, rule, severity, reason) -> dict:
    return {
        "entity_type": entity_type,
        "entity_id": entity_id,
        "rule_or_model": rule,
        "severity": severity,
        "reason": reason,
    }


def _num(value):
    if value is None:
        return None
    try:
        v = float(value)
    except (TypeError, ValueError):
        return None
    if math.isnan(v):
        return None
    return v


def _mean(values):
    if not values:
        return 0.0
    return sum(values) / len(values)


def _sample_std(values):
    if len(values) < 2:
        return 0.0
    m = _mean(values)
    var = sum((v - m) ** 2 for v in values) / (len(values) - 1)
    return math.sqrt(var)


def _coerce_date(value):
    if isinstance(value, datetime):
        return value.date()
    if isinstance(value, date):
        return value
    try:
        return date.fromisoformat(str(value)[:10])
    except ValueError:
        return None


def _telemetry_entity_id(record: dict, idx: int) -> str:
    if record.get("id") is not None:
        return str(record["id"])
    if record.get("battery_id") is not None:
        return f"{record['battery_id']}:{record.get('reading_at')}"
    return f"telemetry[{idx}]"


def _repayment_entity_id(record: dict, idx: int) -> str:
    if record.get("id") is not None:
        return str(record["id"])
    if record.get("loan_id") is not None:
        return f"{record['loan_id']}:{record.get('due_date')}"
    return f"repayment[{idx}]"


def detect_telemetry_anomalies(records) -> list[dict]:
    """Run telemetry sanity/consistency rules; never raises, never rejects."""
    records = list(records or [])
    flags: list[dict] = []

    # Per-record physical bounds + duplicate detection.
    seen: set = set()
    for i, r in enumerate(records):
        if not isinstance(r, dict):
            continue
        eid = _telemetry_entity_id(r, i)
        soc = _num(r.get("state_of_charge"))
        temp = _num(r.get("temperature_c"))

        if soc is not None and (soc < SOC_MIN or soc > SOC_MAX):
            flags.append(_flag(
                "telemetry_reading", eid, "soc_out_of_bounds", SEVERITY_HIGH,
                f"state_of_charge={soc} is outside physical bounds [{SOC_MIN}, {SOC_MAX}]",
            ))
        if temp is not None and (temp < TEMP_MIN_C or temp > TEMP_MAX_C):
            flags.append(_flag(
                "telemetry_reading", eid, "temperature_out_of_bounds", SEVERITY_HIGH,
                f"temperature_c={temp} is outside physical bounds [{TEMP_MIN_C}, {TEMP_MAX_C}]",
            ))

        key = (r.get("battery_id"), r.get("reading_at"))
        if r.get("battery_id") is not None and r.get("reading_at") is not None:
            if key in seen:
                flags.append(_flag(
                    "telemetry_reading", eid, "duplicate_telemetry", SEVERITY_MEDIUM,
                    f"duplicate telemetry reading for battery_id={r.get('battery_id')} reading_at={r.get('reading_at')}",
                ))
            else:
                seen.add(key)

    # Per-battery ordering checks: cycle-count regression + SoC jumps.
    by_battery: dict = {}
    for i, r in enumerate(records):
        if isinstance(r, dict) and r.get("battery_id") is not None:
            by_battery.setdefault(r["battery_id"], []).append((i, r))

    for battery_id, rows in by_battery.items():
        def sort_key(item):
            d = _coerce_date(item[1].get("reading_at"))
            return (d is None, d)

        ordered = sorted(rows, key=sort_key)
        max_cycle = None
        prev_soc = None
        for i, r in ordered:
            cycle = _num(r.get("cycle_count"))
            soc = _num(r.get("state_of_charge"))
            if cycle is not None:
                if max_cycle is not None and cycle < max_cycle:
                    eid = _telemetry_entity_id(r, i)
                    flags.append(_flag(
                        "telemetry_reading", eid, "cycle_count_regression", SEVERITY_HIGH,
                        f"cycle_count={cycle} regressed below prior max {max_cycle} for battery_id={battery_id}",
                    ))
                else:
                    max_cycle = cycle if max_cycle is None else max(max_cycle, cycle)
            if soc is not None and prev_soc is not None and abs(soc - prev_soc) > SOC_JUMP_THRESHOLD:
                eid = _telemetry_entity_id(r, i)
                flags.append(_flag(
                    "telemetry_reading", eid, "soc_jump_implausible", SEVERITY_MEDIUM,
                    f"state_of_charge jumped from {prev_soc} to {soc} (> {SOC_JUMP_THRESHOLD} points) for battery_id={battery_id}",
                ))
            if soc is not None:
                prev_soc = soc

    return flags


def detect_repayment_anomalies(records) -> list[dict]:
    """Run repayment sanity/consistency rules; never raises, never rejects."""
    records = list(records or [])
    flags: list[dict] = []
    seen: set = set()

    for i, r in enumerate(records):
        if not isinstance(r, dict):
            continue
        eid = _repayment_entity_id(r, i)
        amount_due = _num(r.get("amount_due_kes"))
        amount_paid = _num(r.get("amount_paid_kes"))
        status = r.get("status")
        due = _coerce_date(r.get("due_date"))
        paid = _coerce_date(r.get("paid_date"))

        if amount_due is not None and amount_due < 0:
            flags.append(_flag(
                "repayment_event", eid, "amount_due_negative", SEVERITY_HIGH,
                f"amount_due_kes={amount_due} is negative",
            ))
        if amount_paid is not None and amount_paid < 0:
            flags.append(_flag(
                "repayment_event", eid, "amount_paid_negative", SEVERITY_HIGH,
                f"amount_paid_kes={amount_paid} is negative",
            ))

        if status == "missed" and paid is not None:
            flags.append(_flag(
                "repayment_event", eid, "missed_with_paid_date", SEVERITY_HIGH,
                f"status=missed but paid_date={r.get('paid_date')} is present",
            ))
        if status == "on_time" and due is not None and paid is not None and paid > due:
            flags.append(_flag(
                "repayment_event", eid, "on_time_with_late_paid", SEVERITY_MEDIUM,
                f"status=on_time but paid_date={r.get('paid_date')} is after due_date={r.get('due_date')}",
            ))
        if status == "partial" and amount_due is not None and amount_paid is not None and amount_paid >= amount_due:
            flags.append(_flag(
                "repayment_event", eid, "partial_not_less_than_due", SEVERITY_MEDIUM,
                f"status=partial but amount_paid_kes={amount_paid} >= amount_due_kes={amount_due}",
            ))

        key = (r.get("loan_id"), r.get("due_date"))
        if r.get("loan_id") is not None and r.get("due_date") is not None:
            if key in seen:
                flags.append(_flag(
                    "repayment_event", eid, "duplicate_repayment", SEVERITY_MEDIUM,
                    f"duplicate repayment event for loan_id={r.get('loan_id')} due_date={r.get('due_date')}",
                ))
            else:
                seen.add(key)

    return flags


def detect_swap_anomalies(records) -> list[dict]:
    """Flag riders with sustained swap-network thermal or DoD stress.

    Fleet statistics are computed over the uploaded batch (a true fleet
    baseline would come from persisted history). A rider is flagged only when
    their most recent swaps average more than SWAP_STRESS_Z_CUTOFF fleet
    standard deviations above the fleet mean, sustained across at least
    SWAP_STRESS_MIN_SWAPS swaps. Non-gating, same as all anomaly flags.
    """
    records = list(records or [])
    if not records:
        return []

    def _values(rows, key):
        out: list[float] = []
        for r in rows:
            if not isinstance(r, dict):
                continue
            v = _num(r.get(key))
            if v is not None:
                out.append(v)
        return out

    fleet_temp = _values(records, "returned_temperature_c")
    fleet_dod = _values(records, "returned_depth_of_discharge")
    fleet_temp_mean = _mean(fleet_temp)
    fleet_temp_std = _sample_std(fleet_temp)
    fleet_dod_mean = _mean(fleet_dod)
    fleet_dod_std = _sample_std(fleet_dod)

    by_rider: dict = {}
    for r in records:
        if isinstance(r, dict) and r.get("rider_id") is not None:
            by_rider.setdefault(r["rider_id"], []).append(r)

    flags: list[dict] = []
    for rider_id, rows in by_rider.items():
        # Most recent swaps last; None/unparseable dates treated as very old.
        ordered = sorted(rows, key=lambda item: _coerce_date(item.get("swapped_at")) or date.min)
        recent = ordered[-SWAP_STRESS_RECENT_LIMIT:]

        recent_temp = _values(recent, "returned_temperature_c")
        recent_dod = _values(recent, "returned_depth_of_discharge")

        if len(recent_temp) >= SWAP_STRESS_MIN_SWAPS and fleet_temp_std > 0:
            if (_mean(recent_temp) - fleet_temp_mean) > SWAP_STRESS_Z_CUTOFF * fleet_temp_std:
                flags.append(_flag(
                    "rider", str(rider_id), "swap_thermal_stress", SEVERITY_MEDIUM,
                    f"rider's returned batteries run hotter than fleet average across last {len(recent_temp)} swaps — possible overloading/misuse",
                ))
        if len(recent_dod) >= SWAP_STRESS_MIN_SWAPS and fleet_dod_std > 0:
            if (_mean(recent_dod) - fleet_dod_mean) > SWAP_STRESS_Z_CUTOFF * fleet_dod_std:
                flags.append(_flag(
                    "rider", str(rider_id), "swap_deep_discharge_stress", SEVERITY_MEDIUM,
                    f"rider's returned batteries come back more deeply discharged than fleet average across last {len(recent_dod)} swaps — possible over-discharge/misuse",
                ))

    return flags


def detect(telemetry=None, repayments=None, swaps=None) -> list[dict]:
    """Run all anomaly rules and return a list of flag dicts (never rejects)."""
    return detect_telemetry_anomalies(telemetry) + detect_repayment_anomalies(repayments) + detect_swap_anomalies(swaps)
