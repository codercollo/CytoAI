from __future__ import annotations

import csv
from pathlib import Path

from ml.app.services.scoring_service import (
    compute_battery_stress_profile,
    compute_swap_battery_bhi_features,
    compute_swap_cadence_proxy,
    compute_telemetry_cadence_proxy,
)


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


def test_compute_swap_cadence_proxy_regular_is_zero():
    swaps = [
        {"swapped_at": "2026-01-01", "distance_km_since_last_swap": 40},
        {"swapped_at": "2026-01-02", "distance_km_since_last_swap": 40},
        {"swapped_at": "2026-01-03", "distance_km_since_last_swap": 40},
    ]
    assert compute_swap_cadence_proxy(swaps) == 0.0


def test_compute_swap_cadence_proxy_irregular_is_positive():
    swaps = [
        {"swapped_at": "2026-01-01", "distance_km_since_last_swap": 40},
        {"swapped_at": "2026-01-04", "distance_km_since_last_swap": 10},
        {"swapped_at": "2026-01-12", "distance_km_since_last_swap": 200},
    ]
    assert compute_swap_cadence_proxy(swaps) > 0.0


def test_compute_battery_stress_profile_zero_when_at_fleet_average():
    fleet = [
        {"returned_temperature_c": 27, "returned_depth_of_discharge": 80},
        {"returned_temperature_c": 29, "returned_depth_of_discharge": 70},
        {"returned_temperature_c": 31, "returned_depth_of_discharge": 90},
    ]
    rider = [{"returned_temperature_c": 29, "returned_depth_of_discharge": 80}]
    assert compute_battery_stress_profile(rider, fleet) == 0.0


def test_compute_battery_stress_profile_positive_when_hotter_and_deeper():
    fleet = [
        {"returned_temperature_c": 27, "returned_depth_of_discharge": 80},
        {"returned_temperature_c": 29, "returned_depth_of_discharge": 70},
        {"returned_temperature_c": 31, "returned_depth_of_discharge": 90},
    ]
    rider = [{"returned_temperature_c": 40, "returned_depth_of_discharge": 95}]
    assert compute_battery_stress_profile(rider, fleet) > 0.0


def test_compute_swap_battery_bhi_features():
    swaps = [
        {"swapped_at": "2026-01-01", "returned_cycle_count": 100, "returned_depth_of_discharge": 60, "returned_temperature_c": 27},
        {"swapped_at": "2026-01-11", "returned_cycle_count": 110, "returned_depth_of_discharge": 80, "returned_temperature_c": 29},
    ]
    f = compute_swap_battery_bhi_features(swaps)
    assert f is not None
    assert f["cycle_count"] == 110
    assert f["avg_depth_of_discharge"] == 70
    assert f["avg_temperature_c"] == 28
    assert f["age_days"] == 10
    assert f["charge_rate_variance"] == 0.0


def test_compute_swap_battery_bhi_features_empty_returns_none():
    assert compute_swap_battery_bhi_features([]) is None


def _load_sample_swaps() -> list[dict]:
    path = Path("ml/data/sample/swap_events.csv")
    with path.open(newline="") as f:
        return list(csv.DictReader(f))


def test_battery_stress_profile_matches_sample_fixture():
    swaps = _load_sample_swaps()
    assert swaps, "sample swap fixture should be non-empty"
    rider = [r for r in swaps if r["rider_id"] == "rider_s"]
    assert rider, "sample fixture should contain rider_s"
    profile = compute_battery_stress_profile(rider, swaps)
    assert profile > 0.0


def test_swap_cadence_proxy_matches_sample_fixture():
    swaps = _load_sample_swaps()
    assert compute_swap_cadence_proxy(swaps) >= 0.0
