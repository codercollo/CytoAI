"""Loads and serves the Repayment Risk Index (RRI) model artifact.

The feature list is read from the artifact's ``feature_names`` key and includes
on_time_ratio, avg_days_late, payment_cadence_proxy,
loan_to_battery_value_ratio, tenure_days, telemetry_cadence_proxy (the Feature A
swap-cadence signal), and battery_stress_profile (swap_network-only, added in
the Phase 6 retrain).
"""
from __future__ import annotations

import logging
from pathlib import Path
from typing import Any

from ml.app.models.battery_health_model import ModelNotLoadedError

logger = logging.getLogger(__name__)


class RepaymentRiskModel:
    def __init__(self, artifact_path: Path):
        self.artifact_path = artifact_path
        self._artifact: dict[str, Any] | None = None

    def load(self) -> None:
        if not self.artifact_path.exists():
            logger.warning("Repayment model artifact not found at %s", self.artifact_path)
            self._artifact = None
            return
        import joblib

        self._artifact = joblib.load(self.artifact_path)
        logger.info(
            "Loaded repayment model %s (version=%s)",
            self.artifact_path,
            self._artifact.get("model_version", "unknown"),
        )

    @property
    def is_loaded(self) -> bool:
        return self._artifact is not None

    @property
    def feature_names(self) -> list[str]:
        if not self.is_loaded:
            raise ModelNotLoadedError("Repayment model artifact not loaded")
        return self._artifact["feature_names"]

    @property
    def model_version(self) -> str:
        if not self.is_loaded:
            return "unloaded"
        return self._artifact.get("model_version", "unknown")

    def predict_rri(self, features: dict[str, float]) -> float:
        """Predict Repayment Risk Index (0-100, lower = safer)."""
        if not self.is_loaded:
            raise ModelNotLoadedError("Repayment model artifact not loaded — train it first (see ml/training/)")

        import pandas as pd

        row = pd.DataFrame([[features[name] for name in self.feature_names]], columns=self.feature_names)
        scaled = self._artifact["scaler"].transform(row)
        prob_default = float(self._artifact["model"].predict_proba(scaled)[0][1])
        rri = max(0.0, min(100.0, prob_default * 100.0))
        return round(rri, 2)