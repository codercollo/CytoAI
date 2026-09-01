from __future__ import annotations

from ml.app.services.scoring_service import compute_telemetry_cadence_proxy


def test_compute_telemetry_cadence_proxy_regular_is_zero():
    readings = [
        {"reading_at": "2026-01-01", "distance_km_since_last": 50},
        {"reading_at": "2026-01-02", "distance_km_since_last": 50},
        {"reading_at": "2026-01-03", "distance_km_since_last": 50},
    ]
    assert compute_telemetry_cadence_proxy(readings) == 0.0


def test_compute_telemetry_cadence_proxy_irregular_is_positive():
    readings = [
        {"reading_at": "2026-01-01", "distance_km_since_last": 50},
        {"reading_at": "2026-01-03", "distance_km_since_last": 10},
        {"reading_at": "2026-01-10", "distance_km_since_last": 200},
    ]
    assert compute_telemetry_cadence_proxy(readings) > 0.0


def test_compute_telemetry_cadence_proxy_insufficient_returns_zero():
    readings = [{"reading_at": "2026-01-01", "distance_km_since_last": 50}]
    assert compute_telemetry_cadence_proxy(readings) == 0.0
