from __future__ import annotations

from flask import Blueprint, current_app, jsonify, request

from ml.app.models.battery_health_model import ModelNotLoadedError
from ml.app.schemas.score_schema import ValidationError, validate_battery_payload, validate_repayment_payload

score_bp = Blueprint("score", __name__, url_prefix="/predict")


def _service():
    return current_app.config["SCORING_SERVICE"]


@score_bp.post("/battery")
def predict_battery():
    payload = request.get_json(silent=True) or {}
    try:
        features = validate_battery_payload(payload)
        result = _service().score_battery(features)
    except ValidationError as exc:
        return jsonify({"error": "invalid_request", "details": exc.errors}), 400
    except ModelNotLoadedError as exc:
        return jsonify({"error": "model_unavailable", "message": str(exc)}), 503
    return jsonify(result), 200


@score_bp.post("/repayment")
def predict_repayment():
    payload = request.get_json(silent=True) or {}
    try:
        features = validate_repayment_payload(payload)
        result = _service().score_repayment(features)
    except ValidationError as exc:
        return jsonify({"error": "invalid_request", "details": exc.errors}), 400
    except ModelNotLoadedError as exc:
        return jsonify({"error": "model_unavailable", "message": str(exc)}), 503
    return jsonify(result), 200


@score_bp.post("/score")
def predict_combined():
    """
    Convenience endpoint combining both models in one call. Expects a body
    shaped like:
        {"battery": {...battery features...}, "repayment": {...repayment features...}}

    The Go API (internal/risk/mlclient.go) is free to call /predict/battery
    and /predict/repayment separately instead and do the CytoScore weighting
    itself (internal/risk/scoring.go) — this endpoint exists for quick
    manual testing and for partners who want a single round trip.
    """
    payload = request.get_json(silent=True) or {}
    battery_payload = payload.get("battery", {})
    repayment_payload = payload.get("repayment", {})

    try:
        battery_features = validate_battery_payload(battery_payload)
        repayment_features = validate_repayment_payload(repayment_payload)
        result = _service().combined_score(battery_features, repayment_features)
    except ValidationError as exc:
        return jsonify({"error": "invalid_request", "details": exc.errors}), 400
    except ModelNotLoadedError as exc:
        return jsonify({"error": "model_unavailable", "message": str(exc)}), 503
    return jsonify(result), 200