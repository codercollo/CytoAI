from __future__ import annotations

import pytest

from ml.app import create_app
from ml.app.config import Config


@pytest.fixture()
def client():
    app = create_app(Config)
    app.testing = True
    with app.test_client() as c:
        yield c


VALID_BATTERY_PAYLOAD = {
    "cycle_count": 120,
    "avg_depth_of_discharge": 65.0,
    "avg_temperature_c": 27.5,
    "age_days": 400,
    "charge_rate_variance": 0.05,
}

VALID_REPAYMENT_PAYLOAD = {
    "on_time_ratio": 0.82,
    "avg_days_late": 1.4,
    "payment_cadence_proxy": 0.9,
    "loan_to_battery_value_ratio": 1.1,
    "tenure_days": 300,
    "telemetry_cadence_proxy": 0.35,
    "battery_stress_profile": 0.0,
}


def test_healthz(client):
    resp = client.get("/healthz")
    assert resp.status_code == 200
    assert resp.get_json()["status"] == "ok"


def test_readyz_when_models_loaded(client):
    resp = client.get("/readyz")
    body = resp.get_json()
    # This assumes ml/artifacts/*.joblib exist (train the models first).
    # If not present, readyz should correctly report 503, not crash.
    assert resp.status_code in (200, 503)
    assert "battery_model_loaded" in body
    assert "repayment_model_loaded" in body


def test_predict_battery_success(client):
    resp = client.post("/predict/battery", json=VALID_BATTERY_PAYLOAD)
    if resp.status_code == 503:
        pytest.skip("battery model artifact not trained in this environment")
    assert resp.status_code == 200
    body = resp.get_json()
    assert 0 <= body["battery_health_index"] <= 100


def test_predict_battery_missing_field(client):
    payload = dict(VALID_BATTERY_PAYLOAD)
    del payload["cycle_count"]
    resp = client.post("/predict/battery", json=payload)
    assert resp.status_code == 400
    assert resp.get_json()["details"]["cycle_count"] == "required"


def test_predict_battery_out_of_range(client):
    payload = dict(VALID_BATTERY_PAYLOAD)
    payload["avg_depth_of_discharge"] = 250  # > 100
    resp = client.post("/predict/battery", json=payload)
    assert resp.status_code == 400
    assert "avg_depth_of_discharge" in resp.get_json()["details"]


def test_predict_repayment_success(client):
    resp = client.post("/predict/repayment", json=VALID_REPAYMENT_PAYLOAD)
    if resp.status_code == 503:
        pytest.skip("repayment model artifact not trained in this environment")
    assert resp.status_code == 200
    body = resp.get_json()
    assert 0 <= body["repayment_risk_index"] <= 100


def test_predict_repayment_accepts_battery_stress_profile(client):
    payload = dict(VALID_REPAYMENT_PAYLOAD)
    payload["battery_stress_profile"] = 3.362
    resp = client.post("/predict/repayment", json=payload)
    if resp.status_code == 503:
        pytest.skip("repayment model artifact not trained in this environment")
    assert resp.status_code == 200
    body = resp.get_json()
    assert 0 <= body["repayment_risk_index"] <= 100


def test_predict_repayment_unexpected_field(client):
    payload = dict(VALID_REPAYMENT_PAYLOAD)
    payload["extra_field"] = 1
    resp = client.post("/predict/repayment", json=payload)
    assert resp.status_code == 400
    assert "_" in resp.get_json()["details"]


def test_predict_repayment_missing_cadence_field(client):
    payload = dict(VALID_REPAYMENT_PAYLOAD)
    del payload["telemetry_cadence_proxy"]
    resp = client.post("/predict/repayment", json=payload)
    assert resp.status_code == 400
    assert resp.get_json()["details"]["telemetry_cadence_proxy"] == "required"


def test_predict_score_combined(client):
    resp = client.post(
        "/predict/score",
        json={"battery": VALID_BATTERY_PAYLOAD, "repayment": VALID_REPAYMENT_PAYLOAD},
    )
    if resp.status_code == 503:
        pytest.skip("model artifacts not trained in this environment")
    assert resp.status_code == 200
    body = resp.get_json()
    assert 0 <= body["cyto_score"] <= 100
    assert body["cyto_score"] == round(
        0.5 * (100 - body["repayment_risk_index"]) + 0.5 * body["battery_health_index"], 2
    )


def test_predict_score_bad_nested_payload(client):
    resp = client.post("/predict/score", json={"battery": {}, "repayment": {}})
    assert resp.status_code == 400
    body = resp.get_json()
    assert body["error"] == "invalid_request"