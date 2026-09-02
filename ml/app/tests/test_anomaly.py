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


def test_anomaly_clean_telemetry(client):
    resp = client.post("/predict/anomaly", json={
        "telemetry": [
            {"battery_id": "b1", "reading_at": "2026-01-01", "state_of_charge": 80, "temperature_c": 25, "cycle_count": 1, "distance_km_since_last": 50},
            {"battery_id": "b1", "reading_at": "2026-01-02", "state_of_charge": 70, "temperature_c": 26, "cycle_count": 2, "distance_km_since_last": 55},
        ],
    })
    assert resp.status_code == 200
    assert resp.get_json()["flags"] == []


def test_anomaly_soc_out_of_bounds(client):
    resp = client.post("/predict/anomaly", json={
        "telemetry": [{"battery_id": "b1", "reading_at": "2026-01-01", "state_of_charge": 140}],
    })
    assert resp.status_code == 200
    flags = resp.get_json()["flags"]
    assert any(f["rule_or_model"] == "soc_out_of_bounds" for f in flags)


def test_anomaly_cycle_count_regression(client):
    resp = client.post("/predict/anomaly", json={
        "telemetry": [
            {"battery_id": "b1", "reading_at": "2026-01-01", "cycle_count": 5, "state_of_charge": 80, "distance_km_since_last": 10},
            {"battery_id": "b1", "reading_at": "2026-01-02", "cycle_count": 3, "state_of_charge": 75, "distance_km_since_last": 10},
        ],
    })
    flags = resp.get_json()["flags"]
    assert any(f["rule_or_model"] == "cycle_count_regression" for f in flags)


def test_anomaly_duplicate_telemetry(client):
    resp = client.post("/predict/anomaly", json={
        "telemetry": [
            {"battery_id": "b1", "reading_at": "2026-01-01", "state_of_charge": 80},
            {"battery_id": "b1", "reading_at": "2026-01-01", "state_of_charge": 80},
        ],
    })
    flags = resp.get_json()["flags"]
    assert any(f["rule_or_model"] == "duplicate_telemetry" for f in flags)


def test_anomaly_repayment_missed_with_paid_date(client):
    resp = client.post("/predict/anomaly", json={
        "repayments": [{"loan_id": "l1", "due_date": "2026-01-01", "paid_date": "2026-01-02", "status": "missed"}],
    })
    flags = resp.get_json()["flags"]
    assert any(f["rule_or_model"] == "missed_with_paid_date" for f in flags)


def test_anomaly_requires_input(client):
    resp = client.post("/predict/anomaly", json={})
    assert resp.status_code == 400
    assert resp.get_json()["error"] == "invalid_request"


def test_anomaly_swap_thermal_stress(client):
    swaps = []
    for i in range(10):
        swaps.append({"rider_id": f"r{i}", "swapped_at": f"2026-01-0{i % 9 + 1}", "returned_temperature_c": 27 + (i % 3), "returned_depth_of_discharge": 80})
    for i in range(3):
        swaps.append({"rider_id": "hot_rider", "swapped_at": f"2026-01-{20 + i}", "returned_temperature_c": 45, "returned_depth_of_discharge": 80})

    resp = client.post("/predict/anomaly", json={"swaps": swaps})
    assert resp.status_code == 200
    flags = resp.get_json()["flags"]
    assert any(f["rule_or_model"] == "swap_thermal_stress" and f["entity_id"] == "hot_rider" for f in flags)


def test_anomaly_swap_deep_discharge_stress(client):
    swaps = []
    for i in range(10):
        swaps.append({"rider_id": f"r{i}", "swapped_at": f"2026-01-0{i % 9 + 1}", "returned_temperature_c": 28, "returned_depth_of_discharge": 70 + (i % 3)})
    for i in range(3):
        swaps.append({"rider_id": "deep_rider", "swapped_at": f"2026-01-{20 + i}", "returned_temperature_c": 28, "returned_depth_of_discharge": 95})

    resp = client.post("/predict/anomaly", json={"swaps": swaps})
    assert resp.status_code == 200
    flags = resp.get_json()["flags"]
    assert any(f["rule_or_model"] == "swap_deep_discharge_stress" and f["entity_id"] == "deep_rider" for f in flags)


def test_anomaly_swap_no_stress_when_uniform(client):
    swaps = [{"rider_id": f"r{i}", "swapped_at": f"2026-01-0{i + 1}", "returned_temperature_c": 28, "returned_depth_of_discharge": 80} for i in range(6)]
    resp = client.post("/predict/anomaly", json={"swaps": swaps})
    assert resp.status_code == 200
    flags = resp.get_json()["flags"]
    assert not any(f["rule_or_model"].startswith("swap_") for f in flags)
