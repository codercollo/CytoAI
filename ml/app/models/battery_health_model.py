"""Loads and serves the Battery Health Index (BHI) model artifact."""
from __future__ import annotations

import logging
from pathlib import Path
from typing import Any

logger = logging.getLogger(__name__)


class ModelNotLoadedError(RuntimeError):
    """Raised when a prediction is requested before/without a valid artifact."""


class BatteryHealthModel:
    def __init__(self, artifact_path: Path):
        self.artifact_path = artifact_path
        self._artifact: dict[str, Any] | None = None

    def load(self) -> None:
        if not self.artifact_path.exists():
            logger.warning("Battery model artifact not found at %s", self.artifact_path)
            self._artifact = None
            return
        import joblib

        self._artifact = joblib.load(self.artifact_path)
        logger.info(
            "Loaded battery model %s (version=%s)",
            self.artifact_path,
            self._artifact.get("model_version", "unknown"),
        )

    @property
    def is_loaded(self) -> bool:
        return self._artifact is not None

    @property
    def feature_names(self) -> list[str]:
        if not self.is_loaded:
            raise ModelNotLoadedError("Battery model artifact not loaded")
        return self._artifact["feature_names"]

    @property
    def model_version(self) -> str:
        if not self.is_loaded:
            return "unloaded"
        return self._artifact.get("model_version", "unknown")

    def predict_bhi(self, features: dict[str, float]) -> float:
        """Predict Battery Health Index (0-100, higher = healthier)."""
        if not self.is_loaded:
            raise ModelNotLoadedError("Battery model artifact not loaded — train it first (see ml/training/)")

        import pandas as pd

        row = pd.DataFrame([[features[name] for name in self.feature_names]], columns=self.feature_names)
        soh = float(self._artifact["model"].predict(row)[0])
        bhi = max(0.0, min(100.0, soh * 100.0))
        return round(bhi, 2)