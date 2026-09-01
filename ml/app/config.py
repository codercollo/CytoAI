"""Config for the ML sidecar. Env-driven, sane defaults for local dev."""
from __future__ import annotations

import os
from pathlib import Path

BASE_DIR = Path(__file__).resolve().parent.parent  # ml/


class Config:
    ARTIFACTS_DIR = Path(os.environ.get("CYTOAI_ARTIFACTS_DIR", BASE_DIR / "artifacts"))
    BATTERY_MODEL_PATH = ARTIFACTS_DIR / "battery_health_model.joblib"
    REPAYMENT_MODEL_PATH = ARTIFACTS_DIR / "repayment_risk_model.joblib"

    # CytoScore combination weights (spec.md section 5) — configurable per
    # partner risk appetite; Go's internal/risk/scoring.go is the source of
    # truth for what a partner is actually billed/shown, this sidecar's
    # default is just for the /predict/score convenience endpoint.
    BHI_WEIGHT = float(os.environ.get("CYTOAI_BHI_WEIGHT", "0.5"))
    RRI_WEIGHT = float(os.environ.get("CYTOAI_RRI_WEIGHT", "0.5"))

    HOST = os.environ.get("CYTOAI_ML_HOST", "0.0.0.0")
    PORT = int(os.environ.get("CYTOAI_ML_PORT", "5001"))
    DEBUG = os.environ.get("CYTOAI_ML_DEBUG", "false").lower() == "true"