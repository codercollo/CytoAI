"""Verify the lease/swap RRI model split (Phase 6 follow-up).

The two artifacts must be distinct: the lease model is the 6-feature set (no
``battery_stress_profile``) so leased_fixed scores are mathematically unaffected
by the swap-network stress feature; the swap model is the 7-feature set. A fixed
lease feature vector maps to a stable golden RRI, and the ``battery_stress_profile``
feature changes the swap model's output (proving it is actually used) without
changing the lease model's output.
"""
from __future__ import annotations

from pathlib import Path

import joblib
import pandas as pd
import pytest

ARTIFACTS = Path("ml/artifacts")

LEASE_VEC = {
    "on_time_ratio": 0.82,
    "avg_days_late": 1.4,
    "payment_cadence_proxy": 0.9,
    "loan_to_battery_value_ratio": 1.1,
    "tenure_days": 300.0,
    "telemetry_cadence_proxy": 0.35,
}

# Golden RRI for LEASE_VEC under the 6-feature lease model (rri-v0.4-lease). This is
# the pre-pivot lease behaviour: derived only from the 6 lease features, with
# battery_stress_profile contributing nothing.
LEASE_GOLDEN_RRI = 39.3


def _load(name: str) -> dict:
    p = ARTIFACTS / name
    if not p.exists():
        pytest.skip(f"{p} not present — train models first (make train)")
    return joblib.load(p)


def _predict(artifact: dict, features: dict) -> float:
    row = pd.DataFrame(
        [[features[n] for n in artifact["feature_names"]]],
        columns=artifact["feature_names"],
    )
    scaled = artifact["scaler"].transform(row)
    return round(float(artifact["model"].predict_proba(scaled)[0][1] * 100), 2)


def test_lease_artifact_has_six_features():
    a = _load("repayment_risk_model.joblib")
    assert a["feature_names"] == [
        "on_time_ratio",
        "avg_days_late",
        "payment_cadence_proxy",
        "loan_to_battery_value_ratio",
        "tenure_days",
        "telemetry_cadence_proxy",
    ]
    assert a["model_version"] == "rri-v0.4-lease"


def test_swap_artifact_has_seven_features():
    a = _load("repayment_risk_model_swap.joblib")
    assert a["feature_names"][-1] == "battery_stress_profile"
    assert a["model_version"] == "rri-v0.4-swap"


def test_lease_golden_rri():
    a = _load("repayment_risk_model.joblib")
    assert _predict(a, LEASE_VEC) == LEASE_GOLDEN_RRI


def test_swap_stress_changes_rri_but_not_lease():
    lease = _load("repayment_risk_model.joblib")
    swap = _load("repayment_risk_model_swap.joblib")

    lease_rri = _predict(lease, LEASE_VEC)
    swap_unstressed = _predict(swap, dict(LEASE_VEC, battery_stress_profile=0.0))
    swap_stressed = _predict(swap, dict(LEASE_VEC, battery_stress_profile=3.36))

    # The swap feature moves the swap model's output...
    assert swap_stressed != swap_unstressed
    # ...and the lease model's output is untouched by the swap feature.
    assert lease_rri == LEASE_GOLDEN_RRI
    assert _predict(lease, LEASE_VEC) == LEASE_GOLDEN_RRI
