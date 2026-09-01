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
    if not isinstance(telemetry, list) or not isinstance(repayments, list):
        return jsonify({"error": "invalid_request", "details": {"_": "telemetry and repayments must be arrays"}}), 400
    if not telemetry and not repayments:
        return jsonify({"error": "invalid_request", "details": {"_": "telemetry or repayments is required"}}), 400

    flags = detect(telemetry=telemetry, repayments=repayments)
    return jsonify({"flags": flags}), 200
