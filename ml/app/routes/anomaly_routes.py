from __future__ import annotations

from flask import Blueprint, jsonify, request

from ml.app.services.anomaly_service import detect

anomaly_bp = Blueprint("anomaly", __name__, url_prefix="/predict")


@anomaly_bp.post("/anomaly")
def detect_anomalies():
    """Detect anomalies in raw telemetry/repayment records (non-gating)."""
    payload = request.get_json(silent=True) or {}
    if not isinstance(payload, dict):
        return jsonify({"error": "invalid_request", "details": {"_": "request body must be a JSON object"}}), 400

    telemetry = payload.get("telemetry") or []
    repayments = payload.get("repayments") or []
    swaps = payload.get("swaps") or []
    if not isinstance(telemetry, list) or not isinstance(repayments, list) or not isinstance(swaps, list):
        return jsonify({"error": "invalid_request", "details": {"_": "telemetry, repayments, and swaps must be arrays"}}), 400
    if not telemetry and not repayments and not swaps:
        return jsonify({"error": "invalid_request", "details": {"_": "telemetry, repayments, or swaps is required"}}), 400

    flags = detect(telemetry=telemetry, repayments=repayments, swaps=swaps)
    return jsonify({"flags": flags}), 200
