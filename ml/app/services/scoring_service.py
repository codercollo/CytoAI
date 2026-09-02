"""Feature engineering hooks + inference orchestration for both models.

The Go API currently owns raw-data feature engineering (internal/domain/features.go)
and sends numeric feature dicts here; this module orchestrates inference. It is
also the single Python source of truth for the Feature A swap-cadence feature so
the training script (ml/training/train_repayment_model.py) and serving stay on
the exact same derivation.
"""
from __future__ import annotations

import math
from datetime import date, datetime

from ml.app.models.battery_health_model import BatteryHealthModel
from ml.app.models.repayment_risk_model import RepaymentRiskModel


def _compute_cadence_proxy(rows, time_key: str, distance_key: str) -> float:
    """Shared cadence/regularity computation for telemetry and swap events.

    Derivation (documented in spec.md §5):
      1. Sort rows by the time field; skip rows with a missing time or distance.
      2. For consecutive rows i >= 1:
           gap_days = (time[i] - time[i-1]).days
           distance = distance value of row i
      3. timing_irregularity = sample CV of gap_days (std ddof=1 / mean)
         distance_irregularity = sample CV of distance
      4. cadence = 0.5 * timing_irregularity + 0.5 * distance_irregularity

    Higher => more irregular activity. 0.0 when there are fewer than two usable
    intervals (not enough signal; neutral, never a penalty).
    """
    parsed: list[tuple[date, float]] = []
    for r in rows or []:
        if not isinstance(r, dict):
            continue
        t = r.get(time_key)
        d = r.get(distance_key)
        if t is None or d is None:
            continue
        try:
            day = _coerce_date(t)
            km = float(d)
        except (TypeError, ValueError):
            continue
        if math.isnan(km) or km < 0:
            continue
        parsed.append((day, km))

    if len(parsed) < 2:
        return 0.0

    parsed.sort(key=lambda x: x[0])

    gaps: list[float] = []
    distances: list[float] = []
    for i in range(1, len(parsed)):
        gap = (parsed[i][0] - parsed[i - 1][0]).days
        if gap <= 0:
            continue  # duplicate/replayed timestamp; cadence undefined
        gaps.append(float(gap))
        distances.append(parsed[i][1])

    if len(gaps) < 2:
        return 0.0

    return round(0.5 * _cv(gaps) + 0.5 * _cv(distances), 6)


def compute_telemetry_cadence_proxy(readings: list[dict]) -> float:
    """RRI swap/usage-cadence feature (Feature A) from telemetry_readings."""
    return _compute_cadence_proxy(readings, "reading_at", "distance_km_since_last")


def compute_swap_cadence_proxy(swaps: list[dict]) -> float:
    """RRI cadence feature for swap_network loans: same regularity idea, sourced
    from swap_events timestamps instead of per-reading telemetry."""
    return _compute_cadence_proxy(swaps, "swapped_at", "distance_km_since_last_swap")


def _coerce_date(value) -> date:
    if isinstance(value, datetime):
        return value.date()
    if isinstance(value, date):
        return value
    return date.fromisoformat(str(value)[:10])


def _cv(values: list[float]) -> float:
    """Sample coefficient of variation (std ddof=1 / mean). 0 if mean <= 0."""
    if len(values) < 2:
        return 0.0
    mean = sum(values) / len(values)
    if mean <= 0:
        return 0.0
    var = sum((v - mean) ** 2 for v in values) / (len(values) - 1)
    return (var ** 0.5) / mean


def _mean(values: list[float]) -> float:
    if not values:
        return 0.0
    return sum(values) / len(values)


def _sample_std(values: list[float]) -> float:
    if len(values) < 2:
        return 0.0
    m = _mean(values)
    var = sum((v - m) ** 2 for v in values) / (len(values) - 1)
    return math.sqrt(var)


def compute_battery_stress_profile(rider_swaps, fleet_swaps) -> float:
    """Rider battery-stress profile (swap_network RRI input).

    Aggregates, per rider across their swap_events, how the batteries they
    return compare to the fleet average on temperature and depth-of-discharge at
    return time.

    Hypothesis (flag in spec.md; validate against real repayment data in pilot):
    a rider who consistently returns batteries hotter and more deeply discharged
    than the fleet average is a candidate for heavier operational abuse, which
    may correlate with income-maximizing risk-taking behaviour. This is a real
    but UNVALIDATED signal, not a proven correlation.

    Returns 0.0 at/below fleet average, positive above it:
      0.5 * max(0, (rider_mean_temp - fleet_mean_temp) / fleet_std_temp)
    + 0.5 * max(0, (rider_mean_dod  - fleet_mean_dod)  / fleet_std_dod)
    """
    def _field(rows, key):
        out: list[float] = []
        for r in rows or []:
            if not isinstance(r, dict):
                continue
            v = r.get(key)
            if v is None:
                continue
            try:
                f = float(v)
            except (TypeError, ValueError):
                continue
            if not math.isnan(f):
                out.append(f)
        return out

    rider_temp = _field(rider_swaps, "returned_temperature_c")
    rider_dod = _field(rider_swaps, "returned_depth_of_discharge")
    fleet_temp = _field(fleet_swaps, "returned_temperature_c")
    fleet_dod = _field(fleet_swaps, "returned_depth_of_discharge")

    if not rider_temp and not rider_dod:
        return 0.0

    def _stress(rider_vals, fleet_vals):
        if not rider_vals or len(fleet_vals) < 2:
            return 0.0
        sd = _sample_std(fleet_vals)
        if sd <= 0:
            return 0.0
        return max(0.0, (_mean(rider_vals) - _mean(fleet_vals)) / sd)

    return round(0.5 * _stress(rider_temp, fleet_temp) + 0.5 * _stress(rider_dod, fleet_dod), 6)


def compute_swap_battery_bhi_features(swaps) -> dict | None:
    """Fleet-level BHI features for one battery from its swap_events.

    Serves the OPERATOR (which units to pull from rotation), not a rider's
    collateral value. `charge_rate_variance` is not derivable from swap_events
    (no voltage snapshot), so it is returned as 0.0 (neutral) — swap-BHI
    training may revisit this feature set in Phase 6.
    """
    cycles: list[float] = []
    dods: list[float] = []
    temps: list[float] = []
    times: list[date] = []

    for s in swaps or []:
        if not isinstance(s, dict):
            continue
        t = s.get("swapped_at")
        if t is not None:
            try:
                times.append(_coerce_date(t))
            except (TypeError, ValueError):
                pass
        for key, out in (
            ("returned_cycle_count", cycles),
            ("returned_depth_of_discharge", dods),
            ("returned_temperature_c", temps),
        ):
            v = s.get(key)
            if v is None:
                continue
            try:
                f = float(v)
            except (TypeError, ValueError):
                continue
            if not math.isnan(f):
                out.append(f)

    if not cycles and not dods and not temps:
        return None

    features = {
        "cycle_count": max(cycles) if cycles else 0.0,
        "avg_depth_of_discharge": _mean(dods),
        "avg_temperature_c": _mean(temps),
        "age_days": 0.0,
        "charge_rate_variance": 0.0,
    }
    if len(times) >= 2:
        features["age_days"] = float((max(times) - min(times)).days)
    return features


class ScoringService:
    def __init__(self, battery_model: BatteryHealthModel, repayment_model: RepaymentRiskModel, repayment_swap_model: RepaymentRiskModel, bhi_weight: float, rri_weight: float):
        self.battery_model = battery_model
        self.repayment_model = repayment_model
        self.repayment_swap_model = repayment_swap_model
        self.bhi_weight = bhi_weight
        self.rri_weight = rri_weight

    def _repayment_model_for(self, financing_model: str | None) -> RepaymentRiskModel:
        # Lease riders use the 6-feature lease model (battery_stress_profile is
        # ignored); swap_network riders use the 7-feature swap model. This keeps
        # leased_fixed scores unaffected by the swap-network stress feature.
        return self.repayment_swap_model if financing_model == "swap_network" else self.repayment_model

    def score_battery(self, features: dict[str, float]) -> dict:
        bhi = self.battery_model.predict_bhi(features)
        return {
            "battery_health_index": bhi,
            "model_version": self.battery_model.model_version,
        }

    def score_repayment(self, features: dict[str, float], financing_model: str | None = None) -> dict:
        model = self._repayment_model_for(financing_model)
        rri = model.predict_rri(features)
        return {
            "repayment_risk_index": rri,
            "model_version": model.model_version,
        }

    def combined_score(self, battery_features: dict[str, float], repayment_features: dict[str, float], financing_model: str | None = None) -> dict:
        bhi = self.battery_model.predict_bhi(battery_features)
        rri = self._repayment_model_for(financing_model).predict_rri(repayment_features)
        cyto_score = round(self.rri_weight * (100 - rri) + self.bhi_weight * bhi, 2)
        return {
            "battery_health_index": bhi,
            "repayment_risk_index": rri,
            "cyto_score": cyto_score,
            "battery_model_version": self.battery_model.model_version,
            "repayment_model_version": self._repayment_model_for(financing_model).model_version,
        }