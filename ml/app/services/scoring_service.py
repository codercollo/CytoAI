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


def compute_telemetry_cadence_proxy(readings: list[dict]) -> float:
    """Compute the RRI "swap-cadence" feature (Feature A).

    Derivation (documented in spec.md §5):
      1. For a battery, sort telemetry by reading_at; skip records with a
         missing reading_at or distance_km_since_last.
      2. For consecutive readings i >= 1:
           gap_days = (reading_at[i] - reading_at[i-1]).days
           distance = distance_km_since_last of reading i
      3. timing_irregularity = sample CV of gap_days (std ddof=1 / mean)
         distance_irregularity = sample CV of distance
      4. telemetry_cadence_proxy =
           0.5 * timing_irregularity + 0.5 * distance_irregularity

    Higher => more irregular battery usage/swaps. 0.0 when there are fewer than
    two usable intervals (not enough signal; neutral, never a penalty).
    """
    rows: list[tuple[date, float]] = []
    for r in readings or []:
        if not isinstance(r, dict):
            continue
        at = r.get("reading_at")
        dist = r.get("distance_km_since_last")
        if at is None or dist is None:
            continue
        try:
            d = _coerce_date(at)
            km = float(dist)
        except (TypeError, ValueError):
            continue
        if math.isnan(km) or km < 0:
            continue
        rows.append((d, km))

    if len(rows) < 2:
        return 0.0

    rows.sort(key=lambda x: x[0])

    gaps: list[float] = []
    distances: list[float] = []
    for i in range(1, len(rows)):
        gap = (rows[i][0] - rows[i - 1][0]).days
        if gap <= 0:
            continue  # duplicate/replayed timestamp; cadence undefined
        gaps.append(float(gap))
        distances.append(rows[i][1])

    if len(gaps) < 2:
        return 0.0

    return round(0.5 * _cv(gaps) + 0.5 * _cv(distances), 6)


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


class ScoringService:
    def __init__(self, battery_model: BatteryHealthModel, repayment_model: RepaymentRiskModel, bhi_weight: float, rri_weight: float):
        self.battery_model = battery_model
        self.repayment_model = repayment_model
        self.bhi_weight = bhi_weight
        self.rri_weight = rri_weight

    def score_battery(self, features: dict[str, float]) -> dict:
        bhi = self.battery_model.predict_bhi(features)
        return {
            "battery_health_index": bhi,
            "model_version": self.battery_model.model_version,
        }

    def score_repayment(self, features: dict[str, float]) -> dict:
        rri = self.repayment_model.predict_rri(features)
        return {
            "repayment_risk_index": rri,
            "model_version": self.repayment_model.model_version,
        }

    def combined_score(self, battery_features: dict[str, float], repayment_features: dict[str, float]) -> dict:
        bhi = self.battery_model.predict_bhi(battery_features)
        rri = self.repayment_model.predict_rri(repayment_features)
        cyto_score = round(self.rri_weight * (100 - rri) + self.bhi_weight * bhi, 2)
        return {
            "battery_health_index": bhi,
            "repayment_risk_index": rri,
            "cyto_score": cyto_score,
            "battery_model_version": self.battery_model.model_version,
            "repayment_model_version": self.repayment_model.model_version,
        }